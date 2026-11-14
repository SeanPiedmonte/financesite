// Package handler is the HTTP API. Every route returns JSON.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"financesite/main/internal/categories"
	"financesite/main/internal/config"
	"financesite/main/internal/model"
	"financesite/main/internal/plaid"
	"financesite/main/internal/service"
	"financesite/main/internal/store"
)

type API struct {
	Cfg   config.Config
	Store *store.Store
	Plaid *plaid.Client // nil in demo mode
}

// Routes registers every /api endpoint on mux.
func (a *API) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/status", a.status)
	mux.HandleFunc("GET /api/categories", a.categories)

	mux.HandleFunc("GET /api/spending", a.spending)
	mux.HandleFunc("GET /api/transactions", a.transactions)
	mux.HandleFunc("PATCH /api/transactions/{id}", a.categorize)
	mux.HandleFunc("POST /api/transactions/import", a.importCSV)

	mux.HandleFunc("GET /api/rules", a.rules)
	mux.HandleFunc("POST /api/rules", a.addRule)
	mux.HandleFunc("DELETE /api/rules/{id}", a.deleteRule)

	mux.HandleFunc("GET /api/accounts", a.accounts)
	mux.HandleFunc("POST /api/accounts/manual", a.saveManualAccount)
	mux.HandleFunc("PUT /api/accounts/manual/{id}", a.saveManualAccount)
	mux.HandleFunc("DELETE /api/accounts/manual/{id}", a.deleteManualAccount)

	mux.HandleFunc("POST /api/properties", a.saveProperty)
	mux.HandleFunc("PUT /api/properties/{id}", a.saveProperty)
	mux.HandleFunc("DELETE /api/properties/{id}", a.deleteProperty)

	mux.HandleFunc("GET /api/connections", a.connections)
	mux.HandleFunc("POST /api/plaid/link-token", a.linkToken)
	mux.HandleFunc("POST /api/plaid/exchange", a.exchange)
	mux.HandleFunc("POST /api/plaid/sync", a.sync)
	mux.HandleFunc("DELETE /api/connections/{id}", a.disconnect)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writing response: %v", err)
	}
}

func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func failStore(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "not found")
		return
	}
	fail(w, http.StatusBadRequest, err.Error())
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		fail(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func (a *API) status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"demo":     a.Plaid == nil,
		"plaidEnv": a.Cfg.PlaidEnv,
	})
}

func (a *API) categories(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, categories.All())
}

func (a *API) spending(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, service.Spending(a.Store.Transactions(), r.URL.Query().Get("month")))
}

// transactions lists transactions, optionally filtered by ?month=YYYY-MM
// and ?category=ID.
func (a *API) transactions(w http.ResponseWriter, r *http.Request) {
	month, category := r.URL.Query().Get("month"), r.URL.Query().Get("category")
	out := []model.Transaction{}
	for _, t := range a.Store.Transactions() {
		if month != "" && !strings.HasPrefix(t.Date, month) {
			continue
		}
		if category != "" && t.Category != category {
			continue
		}
		out = append(out, t)
	}
	writeJSON(w, http.StatusOK, out)
}

// categorize re-files one transaction. With "remember" it instead saves a
// rule so every transaction from that merchant follows.
func (a *API) categorize(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Category string `json:"category"` // empty clears the override
		Remember bool   `json:"remember"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if body.Category != "" && !categories.Valid(body.Category) {
		fail(w, http.StatusBadRequest, "unknown category")
		return
	}
	id := r.PathValue("id")
	tx, err := a.Store.SetOverride(id, body.Category)
	if err != nil {
		failStore(w, err)
		return
	}
	if body.Remember && body.Category != "" {
		match := tx.Merchant
		if match == "" {
			match = tx.Name
		}
		if _, err := a.Store.AddRule(match, body.Category); err != nil {
			failStore(w, err)
			return
		}
	}
	for _, t := range a.Store.Transactions() {
		if t.ID == id {
			tx = t
		}
	}
	writeJSON(w, http.StatusOK, tx)
}

// importCSV accepts a statement export as multipart field "file".
func (a *API) importCSV(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		fail(w, http.StatusBadRequest, `expected a CSV upload in form field "file" (10 MB max)`)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		fail(w, http.StatusBadRequest, "could not read the upload")
		return
	}

	const acctID = "csv_import"
	txs, err := service.ImportCSV(data, acctID)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := a.Store.UpsertTransactions(txs); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"imported": len(txs), "file": header.Filename})
}

func (a *API) rules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Store.Rules())
}

func (a *API) addRule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Match    string `json:"match"`
		Category string `json:"category"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if len(strings.TrimSpace(body.Match)) < 2 || !categories.Valid(body.Category) {
		fail(w, http.StatusBadRequest, "a rule needs match text of at least 2 characters and a known category")
		return
	}
	rule, err := a.Store.AddRule(body.Match, body.Category)
	if err != nil {
		failStore(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (a *API) deleteRule(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.DeleteRule(r.PathValue("id")); err != nil {
		failStore(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) accounts(w http.ResponseWriter, r *http.Request) {
	accts, props := a.Store.Accounts(), a.Store.Properties()
	writeJSON(w, http.StatusOK, map[string]any{
		"accounts": accts,
		"summary":  service.Worth(accts, props),
	})
}

var manualGroups = map[string]bool{
	model.GroupCash: true, model.GroupCredit: true, model.GroupInvestment: true,
	model.GroupRetirement: true, model.GroupTSP: true, model.GroupMortgage: true,
}

// saveManualAccount creates or updates an account you maintain by hand,
// such as a TSP balance. If holdings are given, the balance is their sum.
func (a *API) saveManualAccount(w http.ResponseWriter, r *http.Request) {
	var body model.Account
	if !readJSON(w, r, &body) {
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || !manualGroups[body.Group] || body.Balance < 0 {
		fail(w, http.StatusBadRequest, "an account needs a name, a valid group and a non-negative balance")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		id = store.NewID("manual")
	} else {
		found := false
		for _, old := range a.Store.Accounts() {
			if old.ID == id && old.Manual {
				found = true
			}
		}
		if !found {
			fail(w, http.StatusNotFound, "not found")
			return
		}
	}
	acct := model.Account{
		ID: id, Institution: strings.TrimSpace(body.Institution), Name: body.Name, Group: body.Group,
		Subtype: body.Subtype, Balance: service.RoundFloat(body.Balance, 2), Manual: true,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339), Holdings: []model.Holding{}, Mortgage: body.Mortgage,
	}
	var total float64
	for _, h := range body.Holdings {
		if h.Value < 0 || strings.TrimSpace(h.Name) == "" {
			fail(w, http.StatusBadRequest, "each holding needs a name and a non-negative value")
			return
		}
		if h.Value == 0 {
			continue
		}
		acct.Holdings = append(acct.Holdings, model.Holding{Name: strings.TrimSpace(h.Name), Value: service.RoundFloat(h.Value, 2)})
		total += h.Value
	}
	if len(acct.Holdings) > 0 {
		acct.Balance = service.RoundFloat(total, 2)
	}
	if err := a.Store.UpsertAccounts([]model.Account{acct}); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, acct)
}

func (a *API) deleteManualAccount(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.DeleteAccount(r.PathValue("id")); err != nil {
		failStore(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) saveProperty(w http.ResponseWriter, r *http.Request) {
	var body model.Property
	if !readJSON(w, r, &body) {
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || body.Value < 0 {
		fail(w, http.StatusBadRequest, "a property needs a name and a non-negative value")
		return
	}
	if body.MortgageAccountID != "" {
		ok := false
		for _, acct := range a.Store.Accounts() {
			if acct.ID == body.MortgageAccountID && acct.Group == model.GroupMortgage {
				ok = true
			}
		}
		if !ok {
			fail(w, http.StatusBadRequest, "mortgageAccountId is not a mortgage account")
			return
		}
	}
	body.ID = r.PathValue("id")
	if body.ID == "" {
		body.ID = store.NewID("prop")
	} else {
		found := false
		for _, p := range a.Store.Properties() {
			if p.ID == body.ID {
				found = true
			}
		}
		if !found {
			fail(w, http.StatusNotFound, "not found")
			return
		}
	}
	body.Value = service.RoundFloat(body.Value, 2)
	body.ValueAsOf = time.Now().Format("2006-01-02")
	if err := a.Store.UpsertProperty(body); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (a *API) deleteProperty(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.DeleteProperty(r.PathValue("id")); err != nil {
		failStore(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Plaid ----

type connection struct {
	ID          string `json:"id"`
	Institution string `json:"institution"`
	Kind        string `json:"kind"`
	LastSync    string `json:"lastSync,omitempty"`
}

// connections lists linked institutions. Access tokens are never returned.
func (a *API) connections(w http.ResponseWriter, r *http.Request) {
	out := []connection{}
	for _, it := range a.Store.Items() {
		out = append(out, connection{it.ID, it.Institution, it.Kind, it.LastSync})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) requirePlaid(w http.ResponseWriter) bool {
	if a.Plaid == nil {
		fail(w, http.StatusServiceUnavailable, "Plaid is not configured: set PLAID_CLIENT_ID and PLAID_SECRET and restart")
		return false
	}
	return true
}

func plaidFail(w http.ResponseWriter, err error) {
	log.Printf("plaid: %v", err)
	var perr *plaid.Error
	if errors.As(err, &perr) {
		fail(w, http.StatusBadGateway, perr.Message+" ("+perr.Code+")")
		return
	}
	fail(w, http.StatusBadGateway, "could not reach Plaid")
}

// linkToken starts a Plaid Link session for {"kind": bank|investment|mortgage}.
func (a *API) linkToken(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlaid(w) {
		return
	}
	var body struct {
		Kind string `json:"kind"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	products, ok := service.ProductsFor(body.Kind)
	if !ok {
		fail(w, http.StatusBadRequest, "kind must be bank, investment or mortgage")
		return
	}
	// Single-user app: one fixed Plaid user id.
	token, err := a.Plaid.CreateLinkToken(r.Context(), "financesite-local-user", products)
	if err != nil {
		plaidFail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"linkToken": token})
}

// exchange finishes Link: swaps the public token for an access token, stores
// the institution, and runs its first sync.
func (a *API) exchange(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlaid(w) {
		return
	}
	var body struct {
		PublicToken string `json:"publicToken"`
		Institution string `json:"institution"`
		Kind        string `json:"kind"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if _, ok := service.ProductsFor(body.Kind); !ok || body.PublicToken == "" {
		fail(w, http.StatusBadRequest, "publicToken and a valid kind are required")
		return
	}
	access, itemID, err := a.Plaid.ExchangePublicToken(r.Context(), body.PublicToken)
	if err != nil {
		plaidFail(w, err)
		return
	}
	name := strings.TrimSpace(body.Institution)
	if name == "" {
		name = "Linked institution"
	}
	it := model.Item{ID: itemID, AccessToken: access, Institution: name, Kind: body.Kind}
	if err := a.Store.UpsertItem(it); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := connection{ID: it.ID, Institution: it.Institution, Kind: it.Kind}
	if err := service.SyncItem(r.Context(), a.Plaid, a.Store, it); err != nil {
		// The link itself succeeded; data can lag by a few seconds on a
		// brand-new item, so report it rather than failing the connection.
		log.Printf("first sync of %s: %v", it.Institution, err)
		writeJSON(w, http.StatusOK, map[string]any{"connection": resp, "syncError": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connection": resp})
}

// sync refreshes every linked institution and reports per-institution errors.
func (a *API) sync(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlaid(w) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	failures := map[string]string{}
	items := a.Store.Items()
	for _, it := range items {
		if err := service.SyncItem(ctx, a.Plaid, a.Store, it); err != nil {
			log.Printf("sync %s: %v", it.Institution, err)
			failures[it.Institution] = err.Error()
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"synced": len(items) - len(failures), "errors": failures})
}

func (a *API) disconnect(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlaid(w) {
		return
	}
	it, err := a.Store.DeleteItem(r.PathValue("id"))
	if err != nil {
		failStore(w, err)
		return
	}
	if err := a.Plaid.RemoveItem(r.Context(), it.AccessToken); err != nil {
		// Local data is already gone; the token just stays valid at Plaid.
		log.Printf("removing item at Plaid: %v", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

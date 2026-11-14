package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"financesite/main/internal/model"
	"financesite/main/internal/plaid"
	"financesite/main/internal/store"
)

func TestImportCSVSkipsPreambleAndFlipsSign(t *testing.T) {
	csv := "Account summary,,\nBeginning balance,,\"1,000.00\"\n\nDate,Description,Amount,Running Bal.\n" +
		"01/05/2026,STARBUCKS #123,-4.50,995.50\n" +
		"01/06/2026,PAYROLL ACME,\"2,000.00\",2995.50\n" +
		"01/07/2026,MYSTERY SHOP,(20.00),2975.50\n"
	txs, err := ImportCSV([]byte(csv), "acct")
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 3 {
		t.Fatalf("got %d transactions, want 3", len(txs))
	}
	if txs[0].Amount != 4.50 || txs[0].Date != "2026-01-05" || txs[0].SourceCategory != "FOOD_AND_DRINK" {
		t.Errorf("coffee row parsed as %+v", txs[0])
	}
	if txs[1].Amount != -2000 || txs[1].SourceCategory != "INCOME" {
		t.Errorf("payroll row parsed as %+v", txs[1])
	}
	if txs[2].Amount != 20 || txs[2].SourceCategory != "OTHER" {
		t.Errorf("parenthesised row parsed as %+v", txs[2])
	}

	again, _ := ImportCSV([]byte(csv), "acct")
	if again[0].ID != txs[0].ID {
		t.Error("IDs are not stable across re-imports")
	}
}

func TestImportCSVDebitCreditColumns(t *testing.T) {
	csv := "Date,Payee,Debit,Credit\n2026-02-01,Rent,1500.00,\n2026-02-02,Refund,,25.00\n"
	txs, err := ImportCSV([]byte(csv), "acct")
	if err != nil {
		t.Fatal(err)
	}
	if txs[0].Amount != 1500 || txs[1].Amount != -25 {
		t.Errorf("got amounts %v and %v", txs[0].Amount, txs[1].Amount)
	}
}

func TestImportCSVRejectsFilesWithoutHeader(t *testing.T) {
	if _, err := ImportCSV([]byte("a,b,c\n1,2,3\n"), "acct"); err == nil {
		t.Error("expected an error for a file with no recognisable header")
	}
}

func TestSpendingExcludesIncomeAndTransfersAndNetsRefunds(t *testing.T) {
	txs := []model.Transaction{
		{Date: "2026-03-02", Merchant: "Grocer", Amount: 100, Category: "FOOD_AND_DRINK"},
		{Date: "2026-03-09", Merchant: "Grocer", Amount: 50, Category: "FOOD_AND_DRINK"},
		{Date: "2026-03-10", Merchant: "Shop", Amount: 80, Category: "GENERAL_MERCHANDISE"},
		{Date: "2026-03-11", Merchant: "Shop", Amount: -30, Category: "GENERAL_MERCHANDISE"}, // refund
		{Date: "2026-03-15", Merchant: "Employer", Amount: -3000, Category: "INCOME"},
		{Date: "2026-03-16", Merchant: "Savings", Amount: 500, Category: "TRANSFER_OUT"},
		{Date: "2026-02-01", Merchant: "Grocer", Amount: 70, Category: "FOOD_AND_DRINK"},
	}
	s := Spending(txs, "")
	if s.Month != "2026-03" {
		t.Fatalf("default month = %q, want the latest", s.Month)
	}
	if s.Spent != 200 || s.Income != 3000 {
		t.Errorf("spent %v income %v, want 200 and 3000", s.Spent, s.Income)
	}
	if len(s.ByCategory) != 2 || s.ByCategory[0].Category != "FOOD_AND_DRINK" || s.ByCategory[0].Amount != 150 || s.ByCategory[0].Share != 0.75 {
		t.Errorf("byCategory = %+v", s.ByCategory)
	}
	if len(s.Trend) != 2 || s.Trend[0].Month != "2026-02" || s.Trend[0].Spent != 70 {
		t.Errorf("trend = %+v", s.Trend)
	}
	if s.TopMerchants[0].Name != "Grocer" || s.TopMerchants[0].Count != 2 {
		t.Errorf("top merchants = %+v", s.TopMerchants)
	}
}

func TestWorth(t *testing.T) {
	accts := []model.Account{
		{ID: "c", Group: model.GroupCash, Balance: 1000},
		{ID: "cc", Group: model.GroupCredit, Balance: 200},
		{ID: "k", Group: model.GroupRetirement, Balance: 5000},
		{ID: "t", Group: model.GroupTSP, Balance: 3000},
		{ID: "m", Group: model.GroupMortgage, Balance: 250000},
	}
	props := []model.Property{{ID: "h", Value: 400000, MortgageAccountID: "m"}}
	nw := Worth(accts, props)
	if nw.Assets != 409000 || nw.Liabilities != 250200 || nw.NetWorth != 158800 {
		t.Errorf("got %+v", nw)
	}
	if nw.Properties[0].Equity != 150000 {
		t.Errorf("equity = %v, want 150000", nw.Properties[0].Equity)
	}
}

func TestGroupFor(t *testing.T) {
	cases := []struct{ typ, sub, inst, want string }{
		{"depository", "checking", "Bank", model.GroupCash},
		{"credit", "credit card", "Bank", model.GroupCredit},
		{"investment", "brokerage", "Broker", model.GroupInvestment},
		{"investment", "401k", "Broker", model.GroupRetirement},
		{"investment", "roth", "Broker", model.GroupRetirement},
		{"investment", "other", "Thrift Savings Plan", model.GroupTSP},
		{"loan", "mortgage", "Servicer", model.GroupMortgage},
	}
	for _, c := range cases {
		if got := GroupFor(c.typ, c.sub, c.inst); got != c.want {
			t.Errorf("GroupFor(%q,%q,%q) = %q, want %q", c.typ, c.sub, c.inst, got, c.want)
		}
	}
}

// fakePlaid answers the endpoints SyncItem uses, paging transactions twice.
func fakePlaid(t *testing.T) *plaid.Client {
	t.Helper()
	mux := http.NewServeMux()
	reply := func(w http.ResponseWriter, body string) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
	mux.HandleFunc("/accounts/get", func(w http.ResponseWriter, r *http.Request) {
		reply(w, `{"accounts":[{"account_id":"a1","name":"Checking","mask":"1111","type":"depository","subtype":"checking","balances":{"current":120.5}}]}`)
	})
	mux.HandleFunc("/transactions/sync", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["client_id"] != "id" || req["secret"] != "secret" || req["access_token"] != "access" {
			t.Errorf("request missing credentials: %v", req)
		}
		switch req["cursor"] {
		case nil:
			reply(w, `{"added":[{"transaction_id":"t1","account_id":"a1","amount":12.5,"date":"2026-04-01","name":"COFFEE","merchant_name":"Coffee Co","personal_finance_category":{"primary":"FOOD_AND_DRINK"}},
				{"transaction_id":"t2","account_id":"a1","amount":9,"date":"2026-04-02","name":"OLD","personal_finance_category":{"primary":"SOMETHING_NEW"}}],
				"next_cursor":"c1","has_more":true}`)
		case "c1":
			reply(w, `{"added":[],"modified":[{"transaction_id":"t1","account_id":"a1","amount":13,"date":"2026-04-01","name":"COFFEE","merchant_name":"Coffee Co","personal_finance_category":{"primary":"FOOD_AND_DRINK"}}],
				"removed":[{"transaction_id":"t2"}],"next_cursor":"c2","has_more":false}`)
		default:
			t.Errorf("unexpected cursor %v", req["cursor"])
		}
	})
	mux.HandleFunc("/investments/holdings/get", func(w http.ResponseWriter, r *http.Request) {
		reply(w, `{"accounts":[{"account_id":"i1","name":"401k","type":"investment","subtype":"401k","balances":{"current":900}}],
			"holdings":[{"account_id":"i1","security_id":"s1","quantity":3,"institution_value":900}],
			"securities":[{"security_id":"s1","name":"Index Fund","ticker_symbol":"IDX"}]}`)
	})
	mux.HandleFunc("/liabilities/get", func(w http.ResponseWriter, r *http.Request) {
		reply(w, `{"accounts":[{"account_id":"m1","name":"Mortgage","type":"loan","subtype":"mortgage","balances":{"current":250000}},
			{"account_id":"s1","name":"Student loan","type":"loan","subtype":"student","balances":{"current":9000}}],
			"liabilities":{"mortgage":[{"account_id":"m1","interest_rate":{"percentage":6.5},"next_monthly_payment":1800,"next_payment_due_date":"2026-05-01",
			"property_address":{"street":"1 Main St","city":"Town","region":"VA","postal_code":"20000"}}]}}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &plaid.Client{BaseURL: srv.URL, ClientID: "id", Secret: "secret", HTTP: srv.Client()}
}

func TestSyncBankPagesAndAppliesModifiedAndRemoved(t *testing.T) {
	pc, st := fakePlaid(t), store.NewMemory()
	it := model.Item{ID: "item", AccessToken: "access", Institution: "Bank", Kind: model.KindBank}
	if err := SyncItem(context.Background(), pc, st, it); err != nil {
		t.Fatal(err)
	}
	txs := st.Transactions()
	if len(txs) != 1 || txs[0].ID != "t1" || txs[0].Amount != 13 || txs[0].Category != "FOOD_AND_DRINK" || txs[0].Merchant != "Coffee Co" {
		t.Errorf("transactions after sync = %+v", txs)
	}
	if items := st.Items(); len(items) != 1 || items[0].Cursor != "c2" || items[0].LastSync == "" {
		t.Errorf("item after sync = %+v", items)
	}
	if a := st.Accounts(); len(a) != 1 || a[0].Balance != 120.5 || a[0].Group != model.GroupCash {
		t.Errorf("accounts after sync = %+v", a)
	}
}

func TestSyncInvestmentAndMortgage(t *testing.T) {
	pc, st := fakePlaid(t), store.NewMemory()
	for _, kind := range []string{model.KindInvestment, model.KindMortgage} {
		if err := SyncItem(context.Background(), pc, st, model.Item{ID: kind, AccessToken: "access", Institution: "Inst", Kind: kind}); err != nil {
			t.Fatal(err)
		}
	}
	byID := map[string]model.Account{}
	for _, a := range st.Accounts() {
		byID[a.ID] = a
	}
	if len(byID) != 2 {
		t.Fatalf("want the 401k and the mortgage only, got %+v", byID)
	}
	k := byID["i1"]
	if k.Group != model.GroupRetirement || len(k.Holdings) != 1 || k.Holdings[0].Ticker != "IDX" || k.Holdings[0].Value != 900 {
		t.Errorf("401k = %+v", k)
	}
	m := byID["m1"]
	if m.Group != model.GroupMortgage || m.Mortgage == nil || m.Mortgage.InterestRate != 6.5 || m.Mortgage.PropertyAddress != "1 Main St, Town, VA 20000" {
		t.Errorf("mortgage = %+v %+v", m, m.Mortgage)
	}
}

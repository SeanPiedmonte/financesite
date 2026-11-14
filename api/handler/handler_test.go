package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"financesite/main/internal/demo"
	"financesite/main/internal/model"
	"financesite/main/internal/service"
	"financesite/main/internal/store"
)

func newDemoServer(t *testing.T) *httptest.Server {
	t.Helper()
	st := store.NewMemory()
	if err := demo.Seed(st, time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	(&API{Store: st}).Routes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func call(t *testing.T, method, url string, body any, out any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		_ = json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

func TestDemoSpendingAndRecategorizeWithRule(t *testing.T) {
	srv := newDemoServer(t)

	var before service.SpendingSummary
	if code := call(t, "GET", srv.URL+"/api/spending?month=2026-05", nil, &before); code != 200 {
		t.Fatalf("status %d", code)
	}
	if before.Spent <= 0 || len(before.ByCategory) < 5 || len(before.Trend) != 5 {
		t.Fatalf("unexpected demo summary: spent=%v cats=%d trend=%d", before.Spent, len(before.ByCategory), len(before.Trend))
	}
	var sum float64
	for _, c := range before.ByCategory {
		sum += c.Amount
	}
	if d := sum - before.Spent; d > 0.01 || d < -0.01 {
		t.Errorf("categories sum to %v but spent is %v", sum, before.Spent)
	}

	var txs []model.Transaction
	call(t, "GET", srv.URL+"/api/transactions?month=2026-05&category=FOOD_AND_DRINK", nil, &txs)
	var coffee model.Transaction
	for _, tx := range txs {
		if tx.Merchant == "Corner Coffee" {
			coffee = tx
		}
	}
	if coffee.ID == "" {
		t.Fatal("no coffee transaction in demo data")
	}

	var updated model.Transaction
	code := call(t, "PATCH", srv.URL+"/api/transactions/"+coffee.ID, map[string]any{"category": "ENTERTAINMENT", "remember": true}, &updated)
	if code != 200 || updated.Category != "ENTERTAINMENT" || updated.CategorySource != "rule" {
		t.Fatalf("recategorize: status %d, %+v", code, updated)
	}
	call(t, "GET", srv.URL+"/api/transactions?month=2026-05", nil, &txs)
	for _, tx := range txs {
		if tx.Merchant == "Corner Coffee" && tx.Category != "ENTERTAINMENT" {
			t.Errorf("rule did not apply to %s", tx.ID)
		}
	}

	var after service.SpendingSummary
	call(t, "GET", srv.URL+"/api/spending?month=2026-05", nil, &after)
	if after.Spent != before.Spent {
		t.Errorf("moving between spending categories changed the total: %v -> %v", before.Spent, after.Spent)
	}

	if code := call(t, "PATCH", srv.URL+"/api/transactions/"+coffee.ID, map[string]any{"category": "NOPE"}, nil); code != 400 {
		t.Errorf("unknown category accepted: %d", code)
	}
	if code := call(t, "PATCH", srv.URL+"/api/transactions/missing", map[string]any{"category": "TRAVEL"}, nil); code != 404 {
		t.Errorf("missing transaction: %d", code)
	}
}

func TestManualTSPAndPropertyFeedNetWorth(t *testing.T) {
	srv := newDemoServer(t)
	type accountsResp struct {
		Accounts []model.Account  `json:"accounts"`
		Summary  service.NetWorth `json:"summary"`
	}
	var before accountsResp
	call(t, "GET", srv.URL+"/api/accounts", nil, &before)

	var tsp model.Account
	code := call(t, "POST", srv.URL+"/api/accounts/manual", map[string]any{
		"name": "Second TSP", "institution": "Thrift Savings Plan", "group": "tsp",
		"holdings": []map[string]any{{"name": "C Fund", "value": 1000}, {"name": "G Fund", "value": 500}, {"name": "F Fund", "value": 0}},
	}, &tsp)
	if code != 200 || tsp.Balance != 1500 || len(tsp.Holdings) != 2 || !tsp.Manual {
		t.Fatalf("create TSP: %d %+v", code, tsp)
	}

	var home model.Property
	code = call(t, "PUT", srv.URL+"/api/properties/demo_home", map[string]any{"name": "Primary residence", "value": 435000, "mortgageAccountId": "demo_mortgage"}, &home)
	if code != 200 || home.Value != 435000 {
		t.Fatalf("update property: %d %+v", code, home)
	}

	var after accountsResp
	call(t, "GET", srv.URL+"/api/accounts", nil, &after)
	if got, want := after.Summary.NetWorth-before.Summary.NetWorth, 11500.0; got != want {
		t.Errorf("net worth moved by %v, want %v", got, want)
	}
	if after.Summary.Properties[0].Equity != 435000-318400 {
		t.Errorf("equity = %v", after.Summary.Properties[0].Equity)
	}

	if code := call(t, "POST", srv.URL+"/api/properties", map[string]any{"name": "Cabin", "value": 1, "mortgageAccountId": "demo_checking"}, nil); code != 400 {
		t.Errorf("linking a non-mortgage account was accepted: %d", code)
	}
	if code := call(t, "DELETE", srv.URL+"/api/accounts/manual/demo_checking", nil, nil); code != 400 {
		t.Errorf("deleting a non-manual account: %d", code)
	}
	if code := call(t, "DELETE", srv.URL+"/api/accounts/manual/"+tsp.ID, nil, nil); code != 204 {
		t.Errorf("deleting the manual account: %d", code)
	}
}

func TestPlaidRoutesRefuseInDemoMode(t *testing.T) {
	srv := newDemoServer(t)
	var body map[string]string
	if code := call(t, "POST", srv.URL+"/api/plaid/link-token", map[string]string{"kind": "bank"}, &body); code != 503 || !strings.Contains(body["error"], "PLAID_CLIENT_ID") {
		t.Errorf("link-token in demo mode: %d %v", code, body)
	}
}

func TestImportCSVEndpoint(t *testing.T) {
	srv := newDemoServer(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "statement.csv")
	_, _ = fw.Write([]byte("Date,Description,Amount\n2026-06-10,UNIQUE IMPORT SHOP,-12.34\n"))
	_ = mw.Close()
	resp, err := http.Post(srv.URL+"/api/transactions/import", mw.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("import status %d", resp.StatusCode)
	}
	var txs []model.Transaction
	call(t, "GET", srv.URL+"/api/transactions?month=2026-06", nil, &txs)
	found := false
	for _, tx := range txs {
		if tx.Name == "UNIQUE IMPORT SHOP" && tx.Amount == 12.34 {
			found = true
		}
	}
	if !found {
		t.Error("imported transaction not listed")
	}
}

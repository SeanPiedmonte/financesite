package store

import (
	"os"
	"path/filepath"
	"testing"

	"financesite/main/internal/model"
)

func seed(t *testing.T, st *Store) {
	t.Helper()
	err := st.UpsertTransactions([]model.Transaction{
		{ID: "1", Date: "2026-01-02", Name: "SQ *CORNER COFFEE", Merchant: "Corner Coffee", Amount: 5, SourceCategory: "FOOD_AND_DRINK"},
		{ID: "2", Date: "2026-01-03", Name: "CORNER COFFEE 2", Merchant: "Corner Coffee", Amount: 6, SourceCategory: "FOOD_AND_DRINK"},
		{ID: "3", Date: "2026-01-04", Name: "Mystery", Merchant: "Mystery", Amount: 7, SourceCategory: "NOT_A_CATEGORY"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func category(st *Store, id string) (string, string) {
	for _, t := range st.Transactions() {
		if t.ID == id {
			return t.Category, t.CategorySource
		}
	}
	return "", ""
}

func TestCategoryPrecedenceOverrideThenRuleThenSource(t *testing.T) {
	st := NewMemory()
	seed(t, st)

	if c, src := category(st, "3"); c != "OTHER" || src != "source" {
		t.Errorf("unknown source category resolved to %s/%s, want OTHER/source", c, src)
	}

	if _, err := st.SetOverride("1", "ENTERTAINMENT"); err != nil {
		t.Fatal(err)
	}
	if c, src := category(st, "1"); c != "ENTERTAINMENT" || src != "manual" {
		t.Errorf("override gave %s/%s", c, src)
	}
	if c, _ := category(st, "2"); c != "FOOD_AND_DRINK" {
		t.Errorf("override leaked to another transaction: %s", c)
	}

	if _, err := st.AddRule("corner coffee", "TRAVEL"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"1", "2"} {
		if c, src := category(st, id); c != "TRAVEL" || src != "rule" {
			t.Errorf("tx %s after rule = %s/%s, want TRAVEL/rule", id, c, src)
		}
	}

	// Same match text replaces rather than duplicates.
	if _, err := st.AddRule("Corner Coffee", "MEDICAL"); err != nil {
		t.Fatal(err)
	}
	if len(st.Rules()) != 1 {
		t.Errorf("rules = %+v, want one", st.Rules())
	}
	if err := st.DeleteRule(st.Rules()[0].ID); err != nil {
		t.Fatal(err)
	}
	if c, src := category(st, "2"); c != "FOOD_AND_DRINK" || src != "source" {
		t.Errorf("after deleting the rule = %s/%s", c, src)
	}
}

func TestUpsertKeepsManualOverride(t *testing.T) {
	st := NewMemory()
	seed(t, st)
	_, _ = st.SetOverride("1", "TRAVEL")
	_ = st.UpsertTransactions([]model.Transaction{{ID: "1", Date: "2026-01-02", Name: "CORNER COFFEE", Amount: 5.5, SourceCategory: "FOOD_AND_DRINK"}})
	if c, _ := category(st, "1"); c != "TRAVEL" {
		t.Errorf("a re-sync dropped the manual category: %s", c)
	}
}

func TestPersistenceAndFileMode(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	seed(t, st)
	_ = st.UpsertItem(model.Item{ID: "i", AccessToken: "secret-token", Institution: "Bank", Kind: model.KindBank})

	info, err := os.Stat(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("state file mode = %v, want 0600 (it holds access tokens)", info.Mode().Perm())
	}
	again, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Transactions()) != 3 || len(again.Items()) != 1 {
		t.Error("state was not reloaded from disk")
	}
}

func TestDeleteItemCascades(t *testing.T) {
	st := NewMemory()
	_ = st.UpsertItem(model.Item{ID: "i", Kind: model.KindMortgage})
	_ = st.UpsertAccounts([]model.Account{{ID: "m", ItemID: "i", Group: model.GroupMortgage}, {ID: "keep", Manual: true}})
	_ = st.UpsertTransactions([]model.Transaction{{ID: "t", AccountID: "m"}, {ID: "u", AccountID: "keep"}})
	_ = st.UpsertProperty(model.Property{ID: "p", MortgageAccountID: "m"})
	if _, err := st.DeleteItem("i"); err != nil {
		t.Fatal(err)
	}
	if len(st.Accounts()) != 1 || len(st.Transactions()) != 1 || st.Properties()[0].MortgageAccountID != "" {
		t.Errorf("cascade left accounts=%v txs=%v props=%v", st.Accounts(), st.Transactions(), st.Properties())
	}
}

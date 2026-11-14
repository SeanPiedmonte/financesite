// Package store keeps all application state in memory and, when given a
// path, persists it to a single JSON file.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"financesite/main/internal/categories"
	"financesite/main/internal/model"
)

var ErrNotFound = errors.New("not found")

type state struct {
	Items        []model.Item        `json:"items"`
	Accounts     []model.Account     `json:"accounts"`
	Transactions []model.Transaction `json:"transactions"`
	Rules        []model.Rule        `json:"rules"`
	Properties   []model.Property    `json:"properties"`
}

type Store struct {
	mu   sync.RWMutex
	path string // empty: memory only (demo mode)
	s    state
}

// NewMemory returns a store that is never written to disk.
func NewMemory() *Store { return &Store{} }

// Open loads state from dir/state.json, creating the directory if needed.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	st := &Store{path: filepath.Join(dir, "state.json")}
	raw, err := os.ReadFile(st.path)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &st.s); err != nil {
		return nil, err
	}
	return st, nil
}

func NewID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

// save must be called with the write lock held. The file holds Plaid access
// tokens, so it is written 0600 and replaced atomically.
func (st *Store) save() error {
	if st.path == "" {
		return nil
	}
	raw, err := json.MarshalIndent(st.s, "", "  ")
	if err != nil {
		return err
	}
	tmp := st.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, st.path)
}

// ---- transactions ----

func resolve(t model.Transaction, rules []model.Rule) model.Transaction {
	t.SourceCategory = categories.Normalize(t.SourceCategory)
	if t.Override != "" && categories.Valid(t.Override) {
		t.Category, t.CategorySource = t.Override, "manual"
		return t
	}
	hay := strings.ToLower(t.Merchant + " " + t.Name)
	for _, r := range rules {
		if r.Match != "" && strings.Contains(hay, strings.ToLower(r.Match)) {
			t.Category, t.CategorySource = r.Category, "rule"
			return t
		}
	}
	t.Category, t.CategorySource = t.SourceCategory, "source"
	return t
}

// Transactions returns every transaction with its effective category,
// newest first.
func (st *Store) Transactions() []model.Transaction {
	st.mu.RLock()
	defer st.mu.RUnlock()
	out := make([]model.Transaction, len(st.s.Transactions))
	for i, t := range st.s.Transactions {
		out[i] = resolve(t, st.s.Rules)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date > out[j].Date
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// UpsertTransactions adds or replaces by ID, keeping any manual override.
func (st *Store) UpsertTransactions(txs []model.Transaction) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	index := make(map[string]int, len(st.s.Transactions))
	for i, t := range st.s.Transactions {
		index[t.ID] = i
	}
	for _, t := range txs {
		t.Category, t.CategorySource = "", ""
		if i, ok := index[t.ID]; ok {
			t.Override = st.s.Transactions[i].Override
			st.s.Transactions[i] = t
		} else {
			index[t.ID] = len(st.s.Transactions)
			st.s.Transactions = append(st.s.Transactions, t)
		}
	}
	return st.save()
}

func (st *Store) RemoveTransactions(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	drop := make(map[string]bool, len(ids))
	for _, id := range ids {
		drop[id] = true
	}
	kept := st.s.Transactions[:0]
	for _, t := range st.s.Transactions {
		if !drop[t.ID] {
			kept = append(kept, t)
		}
	}
	st.s.Transactions = kept
	return st.save()
}

// SetOverride sets (or, with an empty category, clears) a manual category.
func (st *Store) SetOverride(id, category string) (model.Transaction, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i := range st.s.Transactions {
		if st.s.Transactions[i].ID == id {
			st.s.Transactions[i].Override = category
			if err := st.save(); err != nil {
				return model.Transaction{}, err
			}
			return resolve(st.s.Transactions[i], st.s.Rules), nil
		}
	}
	return model.Transaction{}, ErrNotFound
}

// ---- rules ----

func (st *Store) Rules() []model.Rule {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return append([]model.Rule{}, st.s.Rules...)
}

// AddRule saves a rule, replacing any existing rule with the same match text.
// Manual overrides on transactions the rule now covers are cleared so the
// rule is the single source of truth for that merchant.
func (st *Store) AddRule(match, category string) (model.Rule, error) {
	match = strings.TrimSpace(match)
	st.mu.Lock()
	defer st.mu.Unlock()
	rule := model.Rule{ID: NewID("rule"), Match: match, Category: category}
	replaced := false
	for i, r := range st.s.Rules {
		if strings.EqualFold(r.Match, match) {
			rule.ID = r.ID
			st.s.Rules[i] = rule
			replaced = true
		}
	}
	if !replaced {
		st.s.Rules = append(st.s.Rules, rule)
	}
	needle := strings.ToLower(match)
	for i, t := range st.s.Transactions {
		if t.Override != "" && strings.Contains(strings.ToLower(t.Merchant+" "+t.Name), needle) {
			st.s.Transactions[i].Override = ""
		}
	}
	return rule, st.save()
}

func (st *Store) DeleteRule(id string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, r := range st.s.Rules {
		if r.ID == id {
			st.s.Rules = append(st.s.Rules[:i], st.s.Rules[i+1:]...)
			return st.save()
		}
	}
	return ErrNotFound
}

// ---- accounts ----

func (st *Store) Accounts() []model.Account {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return append([]model.Account{}, st.s.Accounts...)
}

// UpsertAccounts adds or replaces by ID. A refresh that carries no holdings
// or mortgage details keeps the ones already stored.
func (st *Store) UpsertAccounts(accts []model.Account) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, a := range accts {
		found := false
		for i, old := range st.s.Accounts {
			if old.ID == a.ID {
				if a.Holdings == nil {
					a.Holdings = old.Holdings
				}
				if a.Mortgage == nil {
					a.Mortgage = old.Mortgage
				}
				st.s.Accounts[i] = a
				found = true
				break
			}
		}
		if !found {
			st.s.Accounts = append(st.s.Accounts, a)
		}
	}
	return st.save()
}

func (st *Store) DeleteAccount(id string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, a := range st.s.Accounts {
		if a.ID == id {
			if !a.Manual {
				return errors.New("linked accounts are removed by disconnecting the institution")
			}
			st.s.Accounts = append(st.s.Accounts[:i], st.s.Accounts[i+1:]...)
			for j := range st.s.Properties {
				if st.s.Properties[j].MortgageAccountID == id {
					st.s.Properties[j].MortgageAccountID = ""
				}
			}
			return st.save()
		}
	}
	return ErrNotFound
}

// ---- properties ----

func (st *Store) Properties() []model.Property {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return append([]model.Property{}, st.s.Properties...)
}

func (st *Store) UpsertProperty(p model.Property) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, old := range st.s.Properties {
		if old.ID == p.ID {
			st.s.Properties[i] = p
			return st.save()
		}
	}
	st.s.Properties = append(st.s.Properties, p)
	return st.save()
}

func (st *Store) DeleteProperty(id string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, p := range st.s.Properties {
		if p.ID == id {
			st.s.Properties = append(st.s.Properties[:i], st.s.Properties[i+1:]...)
			return st.save()
		}
	}
	return ErrNotFound
}

// ---- Plaid items ----

func (st *Store) Items() []model.Item {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return append([]model.Item{}, st.s.Items...)
}

func (st *Store) UpsertItem(it model.Item) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, old := range st.s.Items {
		if old.ID == it.ID {
			st.s.Items[i] = it
			return st.save()
		}
	}
	st.s.Items = append(st.s.Items, it)
	return st.save()
}

// DeleteItem removes an institution with its accounts and transactions.
func (st *Store) DeleteItem(id string) (model.Item, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, it := range st.s.Items {
		if it.ID != id {
			continue
		}
		st.s.Items = append(st.s.Items[:i], st.s.Items[i+1:]...)
		gone := map[string]bool{}
		accts := st.s.Accounts[:0]
		for _, a := range st.s.Accounts {
			if a.ItemID == id {
				gone[a.ID] = true
			} else {
				accts = append(accts, a)
			}
		}
		st.s.Accounts = accts
		txs := st.s.Transactions[:0]
		for _, t := range st.s.Transactions {
			if !gone[t.AccountID] {
				txs = append(txs, t)
			}
		}
		st.s.Transactions = txs
		for j := range st.s.Properties {
			if gone[st.s.Properties[j].MortgageAccountID] {
				st.s.Properties[j].MortgageAccountID = ""
			}
		}
		return it, st.save()
	}
	return model.Item{}, ErrNotFound
}

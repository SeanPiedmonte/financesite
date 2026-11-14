package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"financesite/main/internal/model"
	"financesite/main/internal/plaid"
	"financesite/main/internal/store"
)

// ProductsFor maps a connection kind to the Plaid products requested in Link.
func ProductsFor(kind string) ([]string, bool) {
	switch kind {
	case model.KindBank:
		return []string{"transactions"}, true
	case model.KindInvestment:
		return []string{"investments"}, true
	case model.KindMortgage:
		return []string{"liabilities"}, true
	}
	return nil, false
}

var retirementSubtypes = map[string]bool{
	"401a": true, "401k": true, "403b": true, "457b": true, "ira": true,
	"roth": true, "roth 401k": true, "sep ira": true, "simple ira": true,
	"pension": true, "retirement": true, "keogh": true, "profit sharing plan": true,
	"thrift savings plan": true,
}

// GroupFor decides where a Plaid account sits in the net-worth view.
func GroupFor(accountType, subtype, institution string) string {
	switch accountType {
	case "credit":
		return model.GroupCredit
	case "loan":
		return model.GroupMortgage
	case "investment", "brokerage":
		s := strings.ToLower(subtype)
		if s == "thrift savings plan" || strings.Contains(strings.ToLower(institution), "thrift savings") {
			return model.GroupTSP
		}
		if retirementSubtypes[s] {
			return model.GroupRetirement
		}
		return model.GroupInvestment
	}
	return model.GroupCash
}

func toAccount(a plaid.Account, it model.Item, now string) model.Account {
	var bal float64
	if a.Balances.Current != nil {
		bal = *a.Balances.Current
	} else if a.Balances.Available != nil {
		bal = *a.Balances.Available
	}
	return model.Account{
		ID:          a.AccountID,
		ItemID:      it.ID,
		Institution: it.Institution,
		Name:        a.Name,
		Mask:        a.Mask,
		Group:       GroupFor(a.Type, a.Subtype, it.Institution),
		Subtype:     a.Subtype,
		Balance:     RoundFloat(bal, 2),
		UpdatedAt:   now,
	}
}

func toTransaction(t plaid.Transaction) model.Transaction {
	out := model.Transaction{
		ID:        t.TransactionID,
		AccountID: t.AccountID,
		Date:      t.Date,
		Name:      t.Name,
		Merchant:  t.MerchantName,
		Amount:    t.Amount,
		Pending:   t.Pending,
	}
	if out.Merchant == "" {
		out.Merchant = t.Name
	}
	if t.Category != nil {
		out.SourceCategory = t.Category.Primary
	}
	return out
}

// SyncItem refreshes one linked institution according to its kind.
func SyncItem(ctx context.Context, pc *plaid.Client, st *store.Store, it model.Item) error {
	now := time.Now().UTC().Format(time.RFC3339)

	switch it.Kind {
	case model.KindBank:
		accts, err := pc.Accounts(ctx, it.AccessToken)
		if err != nil {
			return fmt.Errorf("accounts: %w", err)
		}
		converted := make([]model.Account, 0, len(accts))
		for _, a := range accts {
			converted = append(converted, toAccount(a, it, now))
		}
		if err := st.UpsertAccounts(converted); err != nil {
			return err
		}
		// Apply a whole sync as one unit: the cursor is only saved after
		// every page has been stored, so a failure mid-way is retried from
		// the old cursor rather than skipping data.
		cursor := it.Cursor
		for {
			page, err := pc.TransactionsSync(ctx, it.AccessToken, cursor)
			if err != nil {
				return fmt.Errorf("transactions: %w", err)
			}
			changed := make([]model.Transaction, 0, len(page.Added)+len(page.Modified))
			for _, t := range page.Added {
				changed = append(changed, toTransaction(t))
			}
			for _, t := range page.Modified {
				changed = append(changed, toTransaction(t))
			}
			if err := st.UpsertTransactions(changed); err != nil {
				return err
			}
			removed := make([]string, 0, len(page.Removed))
			for _, r := range page.Removed {
				removed = append(removed, r.TransactionID)
			}
			if err := st.RemoveTransactions(removed); err != nil {
				return err
			}
			cursor = page.NextCursor
			if !page.HasMore {
				break
			}
		}
		it.Cursor = cursor

	case model.KindInvestment:
		h, err := pc.InvestmentHoldings(ctx, it.AccessToken)
		if err != nil {
			return fmt.Errorf("holdings: %w", err)
		}
		type sec struct{ name, ticker string }
		secs := map[string]sec{}
		for _, s := range h.Securities {
			secs[s.SecurityID] = sec{s.Name, s.TickerSymbol}
		}
		byAccount := map[string][]model.Holding{}
		for _, hd := range h.Holdings {
			s := secs[hd.SecurityID]
			byAccount[hd.AccountID] = append(byAccount[hd.AccountID], model.Holding{
				Name: s.name, Ticker: s.ticker, Quantity: hd.Quantity, Value: RoundFloat(hd.InstitutionValue, 2),
			})
		}
		converted := make([]model.Account, 0, len(h.Accounts))
		for _, a := range h.Accounts {
			acct := toAccount(a, it, now)
			acct.Holdings = byAccount[a.AccountID]
			if acct.Holdings == nil {
				acct.Holdings = []model.Holding{}
			}
			converted = append(converted, acct)
		}
		if err := st.UpsertAccounts(converted); err != nil {
			return err
		}

	case model.KindMortgage:
		l, err := pc.Liabilities(ctx, it.AccessToken)
		if err != nil {
			return fmt.Errorf("liabilities: %w", err)
		}
		details := map[string]*model.MortgageDetails{}
		for _, m := range l.Liabilities.Mortgage {
			addr := strings.Trim(strings.Join([]string{m.PropertyAddress.Street, m.PropertyAddress.City, m.PropertyAddress.Region + " " + m.PropertyAddress.PostalCode}, ", "), ", ")
			details[m.AccountID] = &model.MortgageDetails{
				InterestRate:    m.InterestRate.Percentage,
				NextPayment:     m.NextMonthlyPayment,
				NextPaymentDue:  m.NextPaymentDueDate,
				OriginalBalance: m.OriginationPrincipalAmount,
				PropertyAddress: addr,
			}
		}
		var converted []model.Account
		for _, a := range l.Accounts {
			d, ok := details[a.AccountID]
			if !ok {
				continue // only mortgages; skip cards or student loans at the same servicer
			}
			acct := toAccount(a, it, now)
			acct.Group = model.GroupMortgage
			acct.Mortgage = d
			converted = append(converted, acct)
		}
		if err := st.UpsertAccounts(converted); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown connection kind %q", it.Kind)
	}

	it.LastSync = now
	return st.UpsertItem(it)
}

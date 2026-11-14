package service

import (
	"sort"

	"financesite/main/internal/categories"
	"financesite/main/internal/model"
)

type CategoryTotal struct {
	Category string  `json:"category"`
	Label    string  `json:"label"`
	Amount   float64 `json:"amount"`
	Count    int     `json:"count"`
	Share    float64 `json:"share"` // 0..1 of the month's spending
}

type MonthTotal struct {
	Month  string  `json:"month"` // YYYY-MM
	Spent  float64 `json:"spent"`
	Income float64 `json:"income"`
}

type MerchantTotal struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type SpendingSummary struct {
	Month        string          `json:"month"`
	Months       []string        `json:"months"` // every month with data, newest first
	Spent        float64         `json:"spent"`
	Income       float64         `json:"income"`
	ByCategory   []CategoryTotal `json:"byCategory"`
	Trend        []MonthTotal    `json:"trend"` // up to 12 months ending at Month, oldest first
	TopMerchants []MerchantTotal `json:"topMerchants"`
}

// Spending summarises one month (YYYY-MM). An empty month means the most
// recent month that has transactions.
//
// "Spent" is net spending: refunds in a spending category reduce it. Income
// and transfers are excluded, and pending transactions are included.
func Spending(txs []model.Transaction, month string) SpendingSummary {
	perMonth := map[string]*MonthTotal{}
	for _, t := range txs {
		if len(t.Date) < 7 {
			continue
		}
		m := t.Date[:7]
		mt := perMonth[m]
		if mt == nil {
			mt = &MonthTotal{Month: m}
			perMonth[m] = mt
		}
		switch {
		case categories.IsSpending(t.Category):
			mt.Spent += t.Amount
		case t.Category == "INCOME":
			mt.Income += -t.Amount
		}
	}

	months := make([]string, 0, len(perMonth))
	for m := range perMonth {
		months = append(months, m)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(months)))

	out := SpendingSummary{Months: months, ByCategory: []CategoryTotal{}, Trend: []MonthTotal{}, TopMerchants: []MerchantTotal{}}
	if month == "" && len(months) > 0 {
		month = months[0]
	}
	out.Month = month
	if month == "" {
		return out
	}

	byCat := map[string]*CategoryTotal{}
	byMerchant := map[string]*MerchantTotal{}
	for _, t := range txs {
		if len(t.Date) < 7 || t.Date[:7] != month || !categories.IsSpending(t.Category) {
			continue
		}
		ct := byCat[t.Category]
		if ct == nil {
			ct = &CategoryTotal{Category: t.Category}
			byCat[t.Category] = ct
		}
		ct.Amount += t.Amount
		ct.Count++

		name := t.Merchant
		if name == "" {
			name = t.Name
		}
		mt := byMerchant[name]
		if mt == nil {
			mt = &MerchantTotal{Name: name}
			byMerchant[name] = mt
		}
		mt.Amount += t.Amount
		mt.Count++
	}

	if mt := perMonth[month]; mt != nil {
		out.Spent, out.Income = RoundFloat(mt.Spent, 2), RoundFloat(mt.Income, 2)
	}
	for _, c := range categories.All() {
		ct := byCat[c.ID]
		if ct == nil {
			continue
		}
		ct.Label = c.Label
		ct.Amount = RoundFloat(ct.Amount, 2)
		if out.Spent > 0 {
			ct.Share = ct.Amount / out.Spent
		}
		out.ByCategory = append(out.ByCategory, *ct)
	}
	sort.SliceStable(out.ByCategory, func(i, j int) bool { return out.ByCategory[i].Amount > out.ByCategory[j].Amount })

	for _, mt := range byMerchant {
		mt.Amount = RoundFloat(mt.Amount, 2)
		out.TopMerchants = append(out.TopMerchants, *mt)
	}
	sort.Slice(out.TopMerchants, func(i, j int) bool {
		a, b := out.TopMerchants[i], out.TopMerchants[j]
		if a.Amount != b.Amount {
			return a.Amount > b.Amount
		}
		return a.Name < b.Name
	})
	if len(out.TopMerchants) > 8 {
		out.TopMerchants = out.TopMerchants[:8]
	}

	for i := len(months) - 1; i >= 0; i-- {
		if months[i] > month {
			continue
		}
		mt := *perMonth[months[i]]
		mt.Spent, mt.Income = RoundFloat(mt.Spent, 2), RoundFloat(mt.Income, 2)
		out.Trend = append(out.Trend, mt)
	}
	if len(out.Trend) > 12 {
		out.Trend = out.Trend[len(out.Trend)-12:]
	}
	return out
}

type PropertyView struct {
	model.Property
	MortgageBalance float64 `json:"mortgageBalance"`
	Equity          float64 `json:"equity"`
}

type NetWorth struct {
	Assets      float64            `json:"assets"`
	Liabilities float64            `json:"liabilities"`
	NetWorth    float64            `json:"netWorth"`
	Groups      map[string]float64 `json:"groups"` // total per account group, plus "property"
	Properties  []PropertyView     `json:"properties"`
}

// Worth totals accounts and properties. Credit-card and mortgage balances
// are liabilities; everything else, including home values, is an asset.
func Worth(accts []model.Account, props []model.Property) NetWorth {
	nw := NetWorth{Groups: map[string]float64{}, Properties: []PropertyView{}}
	balance := map[string]float64{}
	for _, a := range accts {
		balance[a.ID] = a.Balance
		nw.Groups[a.Group] += a.Balance
		if a.Group == model.GroupCredit || a.Group == model.GroupMortgage {
			nw.Liabilities += a.Balance
		} else {
			nw.Assets += a.Balance
		}
	}
	for _, p := range props {
		owed := balance[p.MortgageAccountID]
		nw.Properties = append(nw.Properties, PropertyView{Property: p, MortgageBalance: owed, Equity: RoundFloat(p.Value-owed, 2)})
		nw.Groups["property"] += p.Value
		nw.Assets += p.Value
	}
	for k, v := range nw.Groups {
		nw.Groups[k] = RoundFloat(v, 2)
	}
	nw.Assets, nw.Liabilities = RoundFloat(nw.Assets, 2), RoundFloat(nw.Liabilities, 2)
	nw.NetWorth = RoundFloat(nw.Assets-nw.Liabilities, 2)
	return nw
}

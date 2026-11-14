// Package demo seeds a store with made-up data so the dashboard is usable
// before any bank is connected. None of it is real.
package demo

import (
	"fmt"
	"math/rand"
	"time"

	"financesite/main/internal/model"
	"financesite/main/internal/store"
)

type recurring struct {
	day      int
	merchant string
	category string
	amount   float64
	account  string
}

type variable struct {
	merchant string
	category string
	min, max float64
	perMonth int
	account  string
}

const (
	checking = "demo_checking"
	card     = "demo_card"
)

var fixed = []recurring{
	{1, "Lakeview Mortgage", "LOAN_PAYMENTS", 2140.00, checking},
	{3, "City Power & Light", "RENT_AND_UTILITIES", 132.40, checking},
	{5, "Fiber Internet Co", "RENT_AND_UTILITIES", 70.00, checking},
	{7, "Mobile Wireless", "RENT_AND_UTILITIES", 85.00, card},
	{9, "Auto Insurance Mutual", "GENERAL_SERVICES", 118.00, checking},
	{12, "StreamFlix", "ENTERTAINMENT", 17.99, card},
	{14, "Tunes Premium", "ENTERTAINMENT", 11.99, card},
	{18, "Iron Works Gym", "PERSONAL_CARE", 45.00, card},
	{1, "Employer Payroll", "INCOME", -3650.00, checking},
	{15, "Employer Payroll", "INCOME", -3650.00, checking},
	{16, "Transfer to Savings", "TRANSFER_OUT", 500.00, checking},
}

var varied = []variable{
	{"Greenleaf Market", "FOOD_AND_DRINK", 45, 160, 6, card},
	{"Corner Coffee", "FOOD_AND_DRINK", 4, 9, 9, card},
	{"Taqueria Luna", "FOOD_AND_DRINK", 14, 48, 4, card},
	{"Noodle House", "FOOD_AND_DRINK", 18, 60, 2, card},
	{"Fuel Stop", "TRANSPORTATION", 38, 62, 3, card},
	{"City Transit", "TRANSPORTATION", 2.75, 2.75, 6, card},
	{"Everything Store", "GENERAL_MERCHANDISE", 12, 140, 5, card},
	{"Big Box Outlet", "GENERAL_MERCHANDISE", 25, 180, 2, card},
	{"Neighborhood Pharmacy", "MEDICAL", 8, 55, 1, card},
	{"Hardware Depot", "HOME_IMPROVEMENT", 20, 210, 1, card},
	{"Downtown Cinema", "ENTERTAINMENT", 14, 36, 1, card},
}

func round2(v float64) float64 { return float64(int64(v*100+0.5)) / 100 }

// Seed fills st with six months of transactions ending today, plus accounts
// for every group and one property.
func Seed(st *store.Store, today time.Time) error {
	// Compare calendar dates only, in the caller's time zone.
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	rng := rand.New(rand.NewSource(7)) // fixed seed: the demo looks the same every run
	var txs []model.Transaction
	n := 0
	add := func(day time.Time, merchant, category string, amount float64, account string) {
		if day.After(today) {
			return
		}
		n++
		txs = append(txs, model.Transaction{
			ID: fmt.Sprintf("demo_tx_%04d", n), AccountID: account, Date: day.Format("2006-01-02"),
			Name: merchant, Merchant: merchant, Amount: round2(amount), SourceCategory: category,
		})
	}

	first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	for back := 5; back >= 0; back-- {
		start := first.AddDate(0, -back, 0)
		daysInMonth := start.AddDate(0, 1, -1).Day()
		at := func(d int) time.Time {
			if d > daysInMonth {
				d = daysInMonth
			}
			return start.AddDate(0, 0, d-1)
		}
		for _, f := range fixed {
			add(at(f.day), f.merchant, f.category, f.amount, f.account)
		}
		for _, v := range varied {
			for i := 0; i < v.perMonth; i++ {
				add(at(1+rng.Intn(daysInMonth)), v.merchant, v.category, v.min+rng.Float64()*(v.max-v.min), v.account)
			}
		}
		// A few one-offs so months differ.
		switch back {
		case 4:
			add(at(11), "Skyway Airlines", "TRAVEL", 412.60, card)
			add(at(20), "Harbor Hotel", "TRAVEL", 389.00, card)
		case 2:
			add(at(8), "Family Dental", "MEDICAL", 240.00, card)
		case 1:
			add(at(22), "County Tax Office", "GOVERNMENT_AND_NON_PROFIT", 96.00, checking)
			add(at(25), "Everything Store", "GENERAL_MERCHANDISE", -42.18, card) // a refund
		}
	}
	if err := st.UpsertTransactions(txs); err != nil {
		return err
	}

	asOf := today.Format(time.RFC3339)
	accounts := []model.Account{
		{ID: checking, Institution: "Demo Bank", Name: "Everyday Checking", Mask: "0001", Group: model.GroupCash, Subtype: "checking", Balance: 4820.55},
		{ID: "demo_savings", Institution: "Demo Bank", Name: "High-Yield Savings", Mask: "0002", Group: model.GroupCash, Subtype: "savings", Balance: 18250.00},
		{ID: card, Institution: "Demo Bank", Name: "Rewards Card", Mask: "4242", Group: model.GroupCredit, Subtype: "credit card", Balance: 1436.20},
		{ID: "demo_brokerage", Institution: "Demo Brokerage", Name: "Individual Brokerage", Mask: "7781", Group: model.GroupInvestment, Subtype: "brokerage", Balance: 42310.00,
			Holdings: []model.Holding{
				{Name: "Total Market Index Fund", Ticker: "TMKT", Quantity: 120, Value: 31200.00},
				{Name: "International Index Fund", Ticker: "INTL", Quantity: 150, Value: 9110.00},
				{Name: "Cash", Value: 2000.00},
			}},
		{ID: "demo_401k", Institution: "Demo Retirement Services", Name: "Employer 401(k)", Mask: "3300", Group: model.GroupRetirement, Subtype: "401k", Balance: 86540.00,
			Holdings: []model.Holding{
				{Name: "Target Date 2060 Fund", Value: 86540.00},
			}},
		{ID: "demo_tsp", Institution: "Thrift Savings Plan", Name: "TSP", Group: model.GroupTSP, Subtype: "thrift savings plan", Balance: 54200.00, Manual: true,
			Holdings: []model.Holding{
				{Name: "C Fund", Value: 32520.00},
				{Name: "S Fund", Value: 10840.00},
				{Name: "I Fund", Value: 5420.00},
				{Name: "G Fund", Value: 5420.00},
			}},
		{ID: "demo_mortgage", Institution: "Lakeview Mortgage", Name: "Home Loan", Mask: "9087", Group: model.GroupMortgage, Subtype: "mortgage", Balance: 318400.00,
			Mortgage: &model.MortgageDetails{InterestRate: 6.125, NextPayment: 2140.00, NextPaymentDue: first.AddDate(0, 1, 0).Format("2006-01-02"), OriginalBalance: 340000.00, PropertyAddress: "12 Sample Street, Springfield"}},
	}
	for i := range accounts {
		accounts[i].UpdatedAt = asOf
	}
	if err := st.UpsertAccounts(accounts); err != nil {
		return err
	}
	return st.UpsertProperty(model.Property{
		ID: "demo_home", Name: "Primary residence", Address: "12 Sample Street, Springfield",
		Value: 425000.00, ValueAsOf: today.Format("2006-01-02"), MortgageAccountID: "demo_mortgage",
	})
}

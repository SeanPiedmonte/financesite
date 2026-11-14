// Package categories defines the spending taxonomy. It is Plaid's
// personal_finance_category primary list, used as-is so linked-bank data
// needs no remapping.
package categories

import "strings"

type Category struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Spending is false for income and transfers, which are not counted
	// as money spent.
	Spending bool `json:"spending"`
}

const Other = "OTHER"

var all = []Category{
	{"RENT_AND_UTILITIES", "Rent & Utilities", true},
	{"FOOD_AND_DRINK", "Food & Drink", true},
	{"GENERAL_MERCHANDISE", "Shopping", true},
	{"TRANSPORTATION", "Transportation", true},
	{"TRAVEL", "Travel", true},
	{"ENTERTAINMENT", "Entertainment", true},
	{"MEDICAL", "Medical", true},
	{"PERSONAL_CARE", "Personal Care", true},
	{"HOME_IMPROVEMENT", "Home Improvement", true},
	{"GENERAL_SERVICES", "Services", true},
	{"LOAN_PAYMENTS", "Loan Payments", true},
	{"GOVERNMENT_AND_NON_PROFIT", "Government & Non-profit", true},
	{"BANK_FEES", "Bank Fees", true},
	{Other, "Other", true},
	{"INCOME", "Income", false},
	{"TRANSFER_IN", "Transfer In", false},
	{"TRANSFER_OUT", "Transfer Out", false},
}

var byID = func() map[string]Category {
	m := make(map[string]Category, len(all))
	for _, c := range all {
		m[c.ID] = c
	}
	return m
}()

func All() []Category { return all }

func Valid(id string) bool { _, ok := byID[id]; return ok }

func IsSpending(id string) bool { return byID[id].Spending }

// Normalize maps anything unknown (including an empty Plaid category) to Other.
func Normalize(id string) string {
	id = strings.ToUpper(strings.TrimSpace(id))
	if Valid(id) {
		return id
	}
	return Other
}

// keywords back the guess used for CSV imports, where no category is supplied.
var keywords = []struct {
	category string
	words    []string
}{
	{"INCOME", []string{"payroll", "direct dep", "salary", "interest paid", "dividend"}},
	{"TRANSFER_OUT", []string{"transfer to", "zelle to", "venmo", "online transfer"}},
	{"TRANSFER_IN", []string{"transfer from", "zelle from"}},
	{"LOAN_PAYMENTS", []string{"mortgage", "loan pmt", "loan payment", "student loan", "card payment"}},
	{"RENT_AND_UTILITIES", []string{"rent", "electric", "water", "gas co", "comcast", "xfinity", "verizon", "at&t", "t-mobile", "internet", "utility"}},
	{"FOOD_AND_DRINK", []string{"grocery", "market", "kroger", "safeway", "trader joe", "whole foods", "aldi", "restaurant", "cafe", "coffee", "starbucks", "doordash", "ubereats", "grubhub", "chipotle", "mcdonald", "pizza"}},
	{"TRANSPORTATION", []string{"uber", "lyft", "shell", "exxon", "chevron", "bp ", "parking", "transit", "metro", "toll"}},
	{"TRAVEL", []string{"airline", "delta", "united", "southwest", "hotel", "marriott", "hilton", "airbnb", "expedia"}},
	{"ENTERTAINMENT", []string{"netflix", "spotify", "hulu", "disney", "steam", "cinema", "theater", "ticketmaster"}},
	{"MEDICAL", []string{"pharmacy", "cvs", "walgreens", "doctor", "dental", "hospital", "clinic"}},
	{"PERSONAL_CARE", []string{"gym", "fitness", "salon", "barber", "spa"}},
	{"HOME_IMPROVEMENT", []string{"home depot", "lowe's", "lowes", "ace hardware"}},
	{"GENERAL_SERVICES", []string{"insurance", "geico", "state farm", "usaa", "tuition", "daycare"}},
	{"GOVERNMENT_AND_NON_PROFIT", []string{"irs", "dmv", "tax", "donation", "charity"}},
	{"BANK_FEES", []string{"overdraft", "atm fee", "service fee", "late fee"}},
	{"GENERAL_MERCHANDISE", []string{"amazon", "target", "walmart", "costco", "best buy", "ebay"}},
}

// Guess picks a category from a transaction description. Negative amounts
// (money in) with no keyword match are treated as income.
func Guess(description string, amount float64) string {
	d := strings.ToLower(description)
	for _, k := range keywords {
		for _, w := range k.words {
			if strings.Contains(d, w) {
				return k.category
			}
		}
	}
	if amount < 0 {
		return "INCOME"
	}
	return Other
}

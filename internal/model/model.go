// Package model holds the types shared by the store, services and API.
package model

// Account groups drive the net-worth view.
const (
	GroupCash       = "cash"
	GroupCredit     = "credit"
	GroupInvestment = "investment"
	GroupRetirement = "retirement" // 401(k), IRA, pension
	GroupTSP        = "tsp"        // Thrift Savings Plan
	GroupMortgage   = "mortgage"
)

// Connection kinds: which Plaid product a linked institution is used for.
const (
	KindBank       = "bank"       // checking, savings, credit cards -> transactions
	KindInvestment = "investment" // brokerage, 401(k), IRA -> holdings
	KindMortgage   = "mortgage"   // mortgage servicer -> liabilities
)

type Transaction struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	Date      string `json:"date"` // YYYY-MM-DD
	Name      string `json:"name"`
	Merchant  string `json:"merchant"`
	// Amount follows Plaid's convention: positive is money leaving the account.
	Amount  float64 `json:"amount"`
	Pending bool    `json:"pending"`

	// SourceCategory is what the data source said (Plaid, or a keyword guess for CSV).
	SourceCategory string `json:"sourceCategory"`
	// Override is a category the user picked for this one transaction.
	Override string `json:"override,omitempty"`

	// Category and CategorySource are computed on read: override > rule > source.
	Category       string `json:"category"`
	CategorySource string `json:"categorySource"` // manual | rule | source
}

type Holding struct {
	Name     string  `json:"name"`
	Ticker   string  `json:"ticker,omitempty"`
	Quantity float64 `json:"quantity,omitempty"`
	Value    float64 `json:"value"`
}

type MortgageDetails struct {
	InterestRate    float64 `json:"interestRate,omitempty"` // percent
	NextPayment     float64 `json:"nextPayment,omitempty"`
	NextPaymentDue  string  `json:"nextPaymentDue,omitempty"`
	OriginalBalance float64 `json:"originalBalance,omitempty"`
	PropertyAddress string  `json:"propertyAddress,omitempty"`
}

type Account struct {
	ID          string `json:"id"`
	ItemID      string `json:"itemId,omitempty"`
	Institution string `json:"institution"`
	Name        string `json:"name"`
	Mask        string `json:"mask,omitempty"`
	Group       string `json:"group"`
	Subtype     string `json:"subtype,omitempty"`
	// Balance is always positive; credit and mortgage groups are what you owe.
	Balance   float64          `json:"balance"`
	Manual    bool             `json:"manual"`
	UpdatedAt string           `json:"updatedAt,omitempty"`
	Holdings  []Holding        `json:"holdings,omitempty"`
	Mortgage  *MortgageDetails `json:"mortgage,omitempty"`
}

// Rule files every transaction whose merchant or description contains Match
// (case-insensitive) under Category.
type Rule struct {
	ID       string `json:"id"`
	Match    string `json:"match"`
	Category string `json:"category"`
}

// Property is a home you own. Value is entered by hand (see README: no
// aggregator supplies home values); the linked mortgage supplies the balance.
type Property struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Address           string  `json:"address,omitempty"`
	Value             float64 `json:"value"`
	ValueAsOf         string  `json:"valueAsOf,omitempty"`
	MortgageAccountID string  `json:"mortgageAccountId,omitempty"`
}

// Item is one linked institution. AccessToken never leaves the server.
type Item struct {
	ID          string `json:"id"`
	AccessToken string `json:"accessToken"`
	Institution string `json:"institution"`
	Kind        string `json:"kind"`
	Cursor      string `json:"cursor,omitempty"` // transactions/sync cursor
	LastSync    string `json:"lastSync,omitempty"`
}

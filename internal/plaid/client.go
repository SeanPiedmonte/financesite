// Package plaid is a small client for the parts of the Plaid API this app
// uses: Link tokens, token exchange, transactions, balances, investment
// holdings and mortgage liabilities. It talks JSON over HTTPS directly so the
// project has no third-party Go dependencies.
package plaid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL  string
	ClientID string
	Secret   string
	HTTP     *http.Client
}

func BaseURL(env string) string {
	if env == "production" {
		return "https://production.plaid.com"
	}
	return "https://sandbox.plaid.com"
}

func New(env, clientID, secret string) *Client {
	return &Client{
		BaseURL:  BaseURL(env),
		ClientID: clientID,
		Secret:   secret,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Error is Plaid's error envelope.
type Error struct {
	Status  int    `json:"-"`
	Type    string `json:"error_type"`
	Code    string `json:"error_code"`
	Message string `json:"error_message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("plaid %s (%d): %s", e.Code, e.Status, e.Message)
}

func (c *Client) post(ctx context.Context, path string, body map[string]any, out any) error {
	body["client_id"] = c.ClientID
	body["secret"] = c.Secret
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		perr := &Error{Status: resp.StatusCode}
		if json.Unmarshal(data, perr) != nil || perr.Code == "" {
			perr.Code, perr.Message = "HTTP_ERROR", http.StatusText(resp.StatusCode)
		}
		return perr
	}
	return json.Unmarshal(data, out)
}

// CreateLinkToken starts a Plaid Link session for the given products
// ("transactions", "investments" or "liabilities").
func (c *Client) CreateLinkToken(ctx context.Context, userID string, products []string) (string, error) {
	var out struct {
		LinkToken string `json:"link_token"`
	}
	err := c.post(ctx, "/link/token/create", map[string]any{
		"client_name":   "financesite",
		"language":      "en",
		"country_codes": []string{"US"},
		"user":          map[string]string{"client_user_id": userID},
		"products":      products,
	}, &out)
	return out.LinkToken, err
}

// ExchangePublicToken turns Link's short-lived public token into a
// long-lived access token for one institution ("item").
func (c *Client) ExchangePublicToken(ctx context.Context, publicToken string) (accessToken, itemID string, err error) {
	var out struct {
		AccessToken string `json:"access_token"`
		ItemID      string `json:"item_id"`
	}
	err = c.post(ctx, "/item/public_token/exchange", map[string]any{"public_token": publicToken}, &out)
	return out.AccessToken, out.ItemID, err
}

func (c *Client) RemoveItem(ctx context.Context, accessToken string) error {
	var out struct{}
	return c.post(ctx, "/item/remove", map[string]any{"access_token": accessToken}, &out)
}

type Account struct {
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
	Mask      string `json:"mask"`
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	Balances  struct {
		Current   *float64 `json:"current"`
		Available *float64 `json:"available"`
	} `json:"balances"`
}

func (c *Client) Accounts(ctx context.Context, accessToken string) ([]Account, error) {
	var out struct {
		Accounts []Account `json:"accounts"`
	}
	err := c.post(ctx, "/accounts/get", map[string]any{"access_token": accessToken}, &out)
	return out.Accounts, err
}

type Transaction struct {
	TransactionID string  `json:"transaction_id"`
	AccountID     string  `json:"account_id"`
	Amount        float64 `json:"amount"`
	Date          string  `json:"date"`
	Name          string  `json:"name"`
	MerchantName  string  `json:"merchant_name"`
	Pending       bool    `json:"pending"`
	Category      *struct {
		Primary  string `json:"primary"`
		Detailed string `json:"detailed"`
	} `json:"personal_finance_category"`
}

type SyncPage struct {
	Added    []Transaction `json:"added"`
	Modified []Transaction `json:"modified"`
	Removed  []struct {
		TransactionID string `json:"transaction_id"`
	} `json:"removed"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

// TransactionsSync fetches one page of changes since cursor (empty for the
// first call). Callers loop while HasMore is true.
func (c *Client) TransactionsSync(ctx context.Context, accessToken, cursor string) (SyncPage, error) {
	var out SyncPage
	body := map[string]any{"access_token": accessToken, "count": 500}
	if cursor != "" {
		body["cursor"] = cursor
	}
	err := c.post(ctx, "/transactions/sync", body, &out)
	return out, err
}

type Holdings struct {
	Accounts []Account `json:"accounts"`
	Holdings []struct {
		AccountID        string  `json:"account_id"`
		SecurityID       string  `json:"security_id"`
		Quantity         float64 `json:"quantity"`
		InstitutionValue float64 `json:"institution_value"`
	} `json:"holdings"`
	Securities []struct {
		SecurityID   string `json:"security_id"`
		Name         string `json:"name"`
		TickerSymbol string `json:"ticker_symbol"`
	} `json:"securities"`
}

func (c *Client) InvestmentHoldings(ctx context.Context, accessToken string) (Holdings, error) {
	var out Holdings
	err := c.post(ctx, "/investments/holdings/get", map[string]any{"access_token": accessToken}, &out)
	return out, err
}

type Mortgage struct {
	AccountID    string `json:"account_id"`
	InterestRate struct {
		Percentage float64 `json:"percentage"`
	} `json:"interest_rate"`
	NextMonthlyPayment         float64 `json:"next_monthly_payment"`
	NextPaymentDueDate         string  `json:"next_payment_due_date"`
	OriginationPrincipalAmount float64 `json:"origination_principal_amount"`
	PropertyAddress            struct {
		Street     string `json:"street"`
		City       string `json:"city"`
		Region     string `json:"region"`
		PostalCode string `json:"postal_code"`
	} `json:"property_address"`
}

type Liabilities struct {
	Accounts    []Account `json:"accounts"`
	Liabilities struct {
		Mortgage []Mortgage `json:"mortgage"`
	} `json:"liabilities"`
}

func (c *Client) Liabilities(ctx context.Context, accessToken string) (Liabilities, error) {
	var out Liabilities
	err := c.post(ctx, "/liabilities/get", map[string]any{"access_token": accessToken}, &out)
	return out, err
}

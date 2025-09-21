package config

/*
 * Struct to track Get Request Response
 * TransType is the transaction type so either a credit or a debit
 * TransOrigin is where the transaction originated from could be a bank or
 * vendor
 * Amount is the total amount from the origin
 */
type Response struct {
	TransType string `json:"transType"`
	TransOrigin string `json:"transOrigin"`
	Amount float64 `json:"amount"`
}

// Package service holds the business logic: CSV import, spending summaries,
// net worth, and syncing linked institutions from Plaid.
package service

import (
	"bytes"
	"crypto/sha1"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"financesite/main/internal/categories"
	"financesite/main/internal/model"
)

// RoundFloat rounds val to the given number of decimal places.
func RoundFloat(val float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(val*factor) / factor
}

func parseMoney(s string) (float64, error) {
	s = strings.TrimSpace(s)
	negative := strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")")
	s = strings.NewReplacer(",", "", "$", "", "(", "", ")", "").Replace(s)
	if s == "" {
		return 0, errors.New("empty amount")
	}
	v, err := strconv.ParseFloat(s, 64)
	if negative {
		v = -v
	}
	return v, err
}

var dateLayouts = []string{"2006-01-02", "01/02/2006", "1/2/2006", "01/02/06", "1/2/06", "2006/01/02"}

func parseDate(s string) (string, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), true
		}
	}
	return "", false
}

// ImportCSV parses a bank or card statement export. Banks put a varying
// number of summary lines above the table, so the header is the first row
// with a "description" column plus an amount (or debit/credit) column.
//
// Statement amounts are negative for money out; they are flipped to the
// app's convention (positive is spending).
func ImportCSV(data []byte, accountID string) ([]model.Transaction, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}

	date, desc, amount, debit, credit := -1, -1, -1, -1, -1
	header := -1
	for i, row := range rows {
		date, desc, amount, debit, credit = -1, -1, -1, -1, -1
		for j, cell := range row {
			c := strings.ToLower(strings.TrimSpace(cell))
			switch {
			case strings.Contains(c, "desc") || c == "payee" || c == "name" || c == "merchant":
				if desc == -1 {
					desc = j
				}
			case strings.Contains(c, "date") && date == -1:
				date = j
			case c == "amount" || strings.HasPrefix(c, "amount"):
				amount = j
			case c == "debit" || c == "withdrawal" || c == "withdrawals":
				debit = j
			case c == "credit" || c == "deposit" || c == "deposits":
				credit = j
			}
		}
		if desc != -1 && (amount != -1 || debit != -1 || credit != -1) {
			header = i
			break
		}
	}
	if header == -1 {
		return nil, errors.New("no header row found: need a description column and an amount (or debit/credit) column")
	}

	cell := func(row []string, i int) string {
		if i < 0 || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	var out []model.Transaction
	for n, row := range rows[header+1:] {
		name := cell(row, desc)
		if name == "" {
			continue
		}
		var amt float64
		if amount != -1 {
			v, err := parseMoney(cell(row, amount))
			if err != nil {
				continue // balance rows, totals, blank lines
			}
			amt = -v
		} else {
			d, derr := parseMoney(cell(row, debit))
			c, cerr := parseMoney(cell(row, credit))
			if derr != nil && cerr != nil {
				continue
			}
			amt = math.Abs(d) - math.Abs(c)
		}
		day, ok := parseDate(cell(row, date))
		if !ok {
			continue
		}
		// A stable ID makes re-uploading the same file a no-op. The row
		// number keeps two identical same-day purchases apart.
		sum := sha1.Sum([]byte(fmt.Sprintf("%s|%s|%s|%.2f|%d", accountID, day, name, amt, n)))
		out = append(out, model.Transaction{
			ID:             "csv_" + hex.EncodeToString(sum[:8]),
			AccountID:      accountID,
			Date:           day,
			Name:           name,
			Merchant:       name,
			Amount:         RoundFloat(amt, 2),
			SourceCategory: categories.Guess(name, amt),
		})
	}
	if len(out) == 0 {
		return nil, errors.New("no transactions found below the header row")
	}
	return out, nil
}

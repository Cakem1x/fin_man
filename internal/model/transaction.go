package model

import "time"

// Transaction represents a canonical financial transaction in the domain model.
type Transaction struct {
	ID          string
	Date        time.Time
	Payee       string
	AmountCents int64 // Represents the amount in minor units (e.g., cents) to avoid floating-point issues
	Currency    string
	Memo        string
}

package model

import "time"

// Transaction represents a canonical financial transaction in the domain model.
type Transaction struct {
	ID              string
	Date            time.Time
	Payee           string
	AmountCents     int64 // Represents the amount in minor units (e.g., cents) to avoid floating-point issues
	Currency        string
	Memo            string
	ArchiveFilePath *string `json:"archive_file_path,omitempty"`
	CategoryID      *string // Foreign key to Category
	CategoryName    *string // Hydrated for convenience
	Tags            []Tag   // Hydrated many-to-many relationship
	IsReviewed      bool    // Indicates if the transaction has been reviewed by the user
}

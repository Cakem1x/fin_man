package importer

import (
	"io"

	"github.com/Cakem1x/fin_man/internal/model"
)

// TransactionImporter defines the interface for parsing financial records into canonical domain models.
type TransactionImporter interface {
	// Import reads from the provided reader and returns a slice of Transactions.
	Import(r io.Reader) ([]model.Transaction, error)
}

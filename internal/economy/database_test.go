package economy

import (
	"path/filepath"
	"testing"
)

func TestDatabaseRecordsAndListsTransactions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "economy.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase failed: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	if err := db.RecordTransaction(Transaction{
		ID:      "tx_test_1",
		Amount:  42,
		Type:    "MINT",
		Hash:    "hash_123",
		Reason:  "Ported Go mint",
		Address: "bob_address",
	}); err != nil {
		t.Fatalf("RecordTransaction failed: %v", err)
	}

	transactions, err := db.ListTransactions()
	if err != nil {
		t.Fatalf("ListTransactions failed: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(transactions))
	}
	if transactions[0].Type != "MINT" {
		t.Fatalf("unexpected transaction type: %s", transactions[0].Type)
	}
	if transactions[0].Reason != "Ported Go mint" {
		t.Fatalf("unexpected transaction reason: %s", transactions[0].Reason)
	}
}

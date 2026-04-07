package badger

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	badgerapi "github.com/dgraph-io/badger/v4"
)

func TestMigrateBadgerEncryptionCopiesDataToStableKey(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "badger.db")
	legacyKey := testKey("legacy-key")
	stableKey := testKey("stable-key")

	legacyDB, err := openBadger(dbPath, legacyKey)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}

	if err := legacyDB.Update(func(txn *badgerapi.Txn) error {
		return txn.Set([]byte("cfg:api_key"), []byte("secret"))
	}); err != nil {
		t.Fatalf("seed legacy db: %v", err)
	}

	migratedDB, err := migrateBadgerEncryption(dbPath, legacyDB, stableKey)
	if err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	defer migratedDB.Close()

	var got string
	if err := migratedDB.View(func(txn *badgerapi.Txn) error {
		item, err := txn.Get([]byte("cfg:api_key"))
		if err != nil {
			return err
		}
		return item.Value(func(value []byte) error {
			got = string(value)
			return nil
		})
	}); err != nil {
		t.Fatalf("read migrated db: %v", err)
	}

	if got != "secret" {
		t.Fatalf("migrated value = %q, want %q", got, "secret")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read temp dir: %v", err)
	}
	foundBackup := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "badger.db.legacy-backup-") {
			foundBackup = true
			break
		}
	}
	if !foundBackup {
		t.Fatal("legacy backup directory was not created")
	}
}

func testKey(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

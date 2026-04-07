package badger

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

var (
	encryptionKey []byte // must be 32 bytes
	db            *badger.DB
	rootDir       string
	once          sync.Once
)

const fingerprintSalt = "CrowsNest-static-salt-value"

func GetHardwareEntropy() ([]byte, error) {
	source, machineID, err := getMachineID()
	if err != nil {
		return nil, err
	}

	fingerprint := strings.Join([]string{
		"v2",
		runtime.GOOS,
		source,
		machineID,
		fingerprintSalt,
	}, ":")

	return hashFingerprint(fingerprint), nil
}

func GetLegacyHardwareEntropy() []byte {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
		log.Printf("Error getting hostname: %v", err)
	}
	if legacyHostname := strings.TrimSpace(os.Getenv("CROWSNEST_LEGACY_HOSTNAME")); legacyHostname != "" {
		hostname = legacyHostname
	}

	// Get username
	currentUser, err := user.Current()
	username := "unknown-user"
	if err == nil && currentUser != nil {
		username = currentUser.Username
	}
	if legacyUsername := strings.TrimSpace(os.Getenv("CROWSNEST_LEGACY_USERNAME")); legacyUsername != "" {
		username = legacyUsername
	}

	// Get OS and architecture info
	osInfo := runtime.GOOS + "-" + runtime.GOARCH
	if legacyOSInfo := strings.TrimSpace(os.Getenv("CROWSNEST_LEGACY_OSINFO")); legacyOSInfo != "" {
		osInfo = legacyOSInfo
	}

	// Combine all information for a unique but consistent fingerprint
	fingerprint := strings.Join([]string{
		hostname,
		username,
		osInfo,
		// You could add a static salt here for additional security
		fingerprintSalt,
	}, ":")

	// Hash the fingerprint to get a 32-byte key
	return hashFingerprint(fingerprint)
}

func hashFingerprint(fingerprint string) []byte {
	sum := sha256.Sum256([]byte(fingerprint))
	return sum[:]
}

func getMachineID() (string, string, error) {
	switch runtime.GOOS {
	case "darwin":
		return getDarwinMachineID()
	case "linux":
		return getLinuxMachineID()
	case "windows":
		return getWindowsMachineID()
	default:
		return "", "", fmt.Errorf("stable machine id is not implemented for %s", runtime.GOOS)
	}
}

func getDarwinMachineID() (string, string, error) {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return "", "", fmt.Errorf("run ioreg: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "IOPlatformUUID") {
			continue
		}
		if id := normalizeMachineID(lastQuotedValue(line)); id != "" {
			return "darwin-ioplatformuuid", id, nil
		}
	}

	return "", "", errors.New("IOPlatformUUID not found")
}

func getLinuxMachineID() (string, string, error) {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		out, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if id := normalizeMachineID(string(out)); id != "" {
			return "linux-machine-id", id, nil
		}
	}

	return "", "", errors.New("machine-id not found")
}

func getWindowsMachineID() (string, string, error) {
	out, err := exec.Command("reg", "query", `HKLM\SOFTWARE\Microsoft\Cryptography`, "/v", "MachineGuid").Output()
	if err != nil {
		return "", "", fmt.Errorf("query MachineGuid: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && strings.EqualFold(fields[0], "MachineGuid") {
			if id := normalizeMachineID(fields[len(fields)-1]); id != "" {
				return "windows-machineguid", id, nil
			}
		}
	}

	return "", "", errors.New("MachineGuid not found")
}

func lastQuotedValue(line string) string {
	values := strings.Split(line, "\"")
	if len(values) < 4 {
		return ""
	}
	return values[len(values)-2]
}

func normalizeMachineID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func Start(dirPath string) *badger.DB {
	var err error

	zap.L().Info("Starting Badger DB", zap.String("directory", dirPath))
	zap.L().Info("Badger DB Directory Path", zap.String("directory", dirPath))

	once.Do(func() {
		if !strings.HasSuffix(dirPath, "db") {
			dirPath = filepath.Join(dirPath, "db")
		}
		rootDir = dirPath

		encryptionKey, err = GetHardwareEntropy()
		if err != nil {
			zap.L().Fatal("get_encryption_key",
				zap.String("message", "failed to get encryption key"),
				zap.Error(err),
			)
		}

		badgerDB := filepath.Join(rootDir, "badger.db")
		db, err = openBadger(badgerDB, encryptionKey)
		if err != nil {
			zap.L().Warn("open_badger_db",
				zap.String("message", "failed to open badger database with stable machine key; trying legacy key"),
				zap.Error(err),
			)
			db, err = openBadgerWithLegacyMigration(badgerDB, encryptionKey)
			if err != nil {
				zap.L().Fatal("new_badger_db",
					zap.String("message", "failed to open badger database"),
					zap.Error(err),
				)
			}
		}
	})

	return db
}

func openBadger(dbPath string, key []byte) (*badger.DB, error) {
	opts := badger.DefaultOptions(dbPath).
		WithEncryptionKey(key).
		WithIndexCacheSize(10 << 20). // 10MB
		WithLoggingLevel(badger.ERROR)
	return badger.Open(opts)
}

func openBadgerWithLegacyMigration(dbPath string, stableKey []byte) (*badger.DB, error) {
	legacyKey := GetLegacyHardwareEntropy()
	legacyDB, err := openBadger(dbPath, legacyKey)
	if err != nil {
		return nil, fmt.Errorf("stable key failed and legacy key failed: %w", err)
	}

	migratedDB, err := migrateBadgerEncryption(dbPath, legacyDB, stableKey)
	if err != nil {
		if closeErr := legacyDB.Close(); closeErr != nil {
			zap.L().Error("close_legacy_badger_db", zap.Error(closeErr))
		}
		return nil, err
	}

	return migratedDB, nil
}

func migrateBadgerEncryption(dbPath string, legacyDB *badger.DB, stableKey []byte) (*badger.DB, error) {
	parentDir := filepath.Dir(dbPath)
	timestamp := time.Now().Format("20060102-150405")
	migrationPath := filepath.Join(parentDir, fmt.Sprintf(".%s.migrating-%s", filepath.Base(dbPath), timestamp))
	backupPath := filepath.Join(parentDir, fmt.Sprintf("%s.legacy-backup-%s", filepath.Base(dbPath), timestamp))

	newDB, err := openBadger(migrationPath, stableKey)
	if err != nil {
		return nil, fmt.Errorf("open migration badger db: %w", err)
	}

	if err := copyBadgerData(legacyDB, newDB); err != nil {
		_ = newDB.Close()
		_ = os.RemoveAll(migrationPath)
		return nil, fmt.Errorf("copy legacy badger data: %w", err)
	}

	if err := legacyDB.Close(); err != nil {
		_ = newDB.Close()
		_ = os.RemoveAll(migrationPath)
		return nil, fmt.Errorf("close legacy badger db: %w", err)
	}

	if err := newDB.Close(); err != nil {
		_ = os.RemoveAll(migrationPath)
		return nil, fmt.Errorf("close migration badger db: %w", err)
	}

	if err := os.Rename(dbPath, backupPath); err != nil {
		_ = os.RemoveAll(migrationPath)
		return nil, fmt.Errorf("backup legacy badger db: %w", err)
	}

	if err := os.Rename(migrationPath, dbPath); err != nil {
		if restoreErr := os.Rename(backupPath, dbPath); restoreErr != nil {
			return nil, fmt.Errorf("promote migrated badger db: %w; restore legacy backup: %v", err, restoreErr)
		}
		return nil, fmt.Errorf("promote migrated badger db: %w", err)
	}

	db, err := openBadger(dbPath, stableKey)
	if err != nil {
		return nil, fmt.Errorf("open migrated badger db: %w", err)
	}

	zap.L().Info("migrated_badger_encryption",
		zap.String("backup", backupPath),
		zap.String("path", dbPath),
	)
	return db, nil
}

func copyBadgerData(src *badger.DB, dst *badger.DB) error {
	return src.View(func(srcTxn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iter := srcTxn.NewIterator(opts)
		defer iter.Close()

		return dst.Update(func(dstTxn *badger.Txn) error {
			for iter.Rewind(); iter.Valid(); iter.Next() {
				item := iter.Item()
				if item.IsDeletedOrExpired() {
					continue
				}

				key := item.KeyCopy(nil)
				value, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}

				entry := badger.NewEntry(key, value).WithMeta(item.UserMeta())
				entry.ExpiresAt = item.ExpiresAt()
				if err := dstTxn.SetEntry(entry); err != nil {
					return err
				}
			}
			return nil
		})
	})
}

func Close() {
	err := db.Close()
	if err != nil {
		zap.L().Fatal("new_badger_db",
			zap.String("message", "failed to close badger database"),
			zap.Error(err),
		)
	}
}

func GetDehashedKey() string {
	var apiKey string

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("cfg:api_key"))
		if err != nil {
			return err // could be ErrKeyNotFound
		}
		return item.Value(func(val []byte) error {
			apiKey = string(val)
			return nil
		})
	})

	if err != nil {
		zap.L().Error("get_api_key",
			zap.String("message", "failed to get api_key"),
			zap.Error(err),
		)
	}

	return apiKey
}

func GetHunterKey() string {
	var apiKey string

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("cfg:hunter_api_key"))
		if err != nil {
			return err // could be ErrKeyNotFound
		}
		return item.Value(func(val []byte) error {
			apiKey = string(val)
			return nil
		})
	})

	if err != nil {
		zap.L().Error("get_hunter_api_key",
			zap.String("message", "failed to get hunter_api_key"),
			zap.Error(err),
		)
	}
	return apiKey
}

func GetUseLocalDB() bool {
	var useLocal bool

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("cfg:use_local_db"))
		if err != nil {
			// If key not found, set default to false and return nil
			if errors.Is(err, badger.ErrKeyNotFound) {
				// Store the default value for future use
				err = StoreUseLocalDB(false)
				if err != nil {
					zap.L().Error("store_use_local_db",
						zap.String("message", "failed to store use_local_db"),
						zap.Error(err),
					)
					return err
				}
				return nil
			}
			// Return other errors
			return err
		}
		return item.Value(func(val []byte) error {
			useLocal = val[0] == 1
			return nil
		})
	})

	if err != nil {
		zap.L().Error("get_use_local_db",
			zap.String("message", "failed to get use_local_db"),
			zap.Error(err),
		)
	}

	return useLocal
}

func StoreDehashedKey(apiKey string) error {
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("cfg:api_key"), []byte(apiKey))
	})
	if err != nil {
		zap.L().Error("set_api_key",
			zap.String("message", "failed to set dehashed api_key"),
			zap.Error(err),
		)
	}
	return err
}

func StoreHunterKey(apiKey string) error {
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("cfg:hunter_api_key"), []byte(apiKey))
	})
	if err != nil {
		zap.L().Error("set_api_key",
			zap.String("message", "failed to set hunter api_key"),
			zap.Error(err),
		)
	}
	return err
}

func StoreUseLocalDB(useLocal bool) error {
	var local byte
	if useLocal {
		local = 1
	} else {
		local = 0
	}

	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("cfg:use_local_db"), []byte{local})
	})
	if err != nil {
		zap.L().Error("set_use_local_db",
			zap.String("message", "failed to set use_local_db"),
			zap.Error(err),
		)
	}
	return err
}

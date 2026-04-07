package sqlite

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strings"

	sql "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes the database connection
func InitDB(dbPath string) (*gorm.DB, error) {
	zap.L().Info("Initializing database", zap.String("path", dbPath))

	// Check if the path is a file or directory
	fileInfo, err := os.Stat(dbPath)
	var finalDbPath string

	// If path doesn't exist or is a directory
	if os.IsNotExist(err) || (err == nil && fileInfo.IsDir()) {
		// Treat as directory path
		if err := os.MkdirAll(dbPath, 0755); err != nil {
			zap.L().Error("Failed to create database directory", zap.Error(err))
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
		finalDbPath = filepath.Join(dbPath, "crowsnest.sqlite")
	} else {
		// Treat as file path
		// Ensure the directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			zap.L().Error("Failed to create parent directory for database", zap.Error(err))
			return nil, fmt.Errorf("failed to create parent directory for database: %w", err)
		}
		finalDbPath = dbPath
	}

	zap.L().Info("Opening database", zap.String("finalPath", finalDbPath))
	db, err := gorm.Open(sql.Open(finalDbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		zap.L().Error("Failed to connect to database", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate your models
	err = db.AutoMigrate(&Result{}, &User{}, &QueryOptions{}, &User{}, &WhoisRecord{}, &HistoryRecord{},
		&LookupResult{}, &HunterDomainData{}, &HunterEmail{}, &PersonData{}, &Subdomain{})
	if err != nil {
		zap.L().Error("Failed to migrate database", zap.Error(err))
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	DB = db
	return db, nil
}

// GetDB returns the database connection
func GetDB() *gorm.DB {
	if DB == nil {
		zap.L().Error("database not initialized")
		fmt.Println("sqlite database not initialized")
		os.Exit(1)
	}
	return DB
}

type Table int64

const (
	ResultsTable Table = iota
	RunsTable
	CredsTable
	WhoIsTable
	SubdomainsTable
	HistoryTable
	LookupTable
	HunterDomainTable
	HunterEmailTable
	PersonTable
	UnknownTable
)

func GetTable(userInput string) Table {
	switch strings.ToLower(userInput) {
	case "dehashed", "results":
		return ResultsTable
	case "runs":
		return RunsTable
	case "users", "creds":
		return CredsTable
	case "whois":
		return WhoIsTable
	case "subdomains":
		return SubdomainsTable
	case "history":
		return HistoryTable
	case "lookup":
		return LookupTable
	case "hunter_domain":
		return HunterDomainTable
	case "hunter_email":
		return HunterEmailTable
	case "person":
		return PersonTable
	default:
		return UnknownTable
	}
}

func (t Table) Object() interface{} {
	switch t {
	case ResultsTable:
		return Result{}
	case RunsTable:
		return QueryOptions{}
	case CredsTable:
		return User{}
	case WhoIsTable:
		return WhoisRecord{}
	case SubdomainsTable:
		return SubdomainRecord{}
	case HistoryTable:
		return HistoryRecord{}
	case LookupTable:
		return LookupResult{}
	case HunterDomainTable:
		return HunterDomainData{}
	case HunterEmailTable:
		return HunterEmail{}
	case PersonTable:
		return PersonData{}
	default:
		return nil
	}
}

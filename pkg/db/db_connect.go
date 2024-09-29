package db

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var sql_schema string

// Database starts at version 0.  First migration brings us to version 1
var MIGRATIONS = []string{}
var ENGINE_DATABASE_VERSION = len(MIGRATIONS)

var (
	ErrTargetExists = errors.New("target already exists")
)

type DB struct {
	DB *sqlx.DB
}


func DBCreate(path string) (DB, error) {
	// First check if the path already exists
	_, err := os.Stat(path)
	if err == nil {
		return DB{}, ErrTargetExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return DB{}, fmt.Errorf("path error: %w", err)
	}

	// Create DB file
	fmt.Printf("Creating: %s\n", path)
	db := sqlx.MustOpen("sqlite3", path+"?_foreign_keys=on")
	db.MustExec(sql_schema)

	for k, v := range EVENT_NAMES {
		_, err := db.Exec(
			"insert into event_types (contract_address, hashed_name, name) values (?, ?, ?)",
			common.HexToAddress("223c067f8cf28ae173ee5cafea60ca44c335fecb"), k, v,
		)
		if err != nil {
			fmt.Println(k, v)
			panic(err)
		}
	}

	return DB{db}, nil
}

func DBConnect(path string) (DB, error) {
	db := sqlx.MustOpen("sqlite3", fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL", path))
	ret := DB{db}
	err := ret.CheckAndUpdateVersion()
	return ret, err
}

/**
 * Colors for terminal output
 */

const (
	COLOR_RESET  = "\033[0m"
	COLOR_RED    = "\033[31m"
	COLOR_GREEN  = "\033[32m"
	COLOR_YELLOW = "\033[33m"
	COLOR_BLUE   = "\033[34m"
	COLOR_PURPLE = "\033[35m"
	COLOR_CYAN   = "\033[36m"
	COLOR_GRAY   = "\033[37m"
	COLOR_WHITE  = "\033[97m"
)

func (db DB) CheckAndUpdateVersion() error {
	var version int
	err := db.DB.Get(&version, "select version from db_version")
	if err != nil {
		return fmt.Errorf("couldn't check database version: %w", err)
	}

	if version > ENGINE_DATABASE_VERSION {
		return VersionMismatchError{ENGINE_DATABASE_VERSION, version}
	}

	if ENGINE_DATABASE_VERSION > version {
		fmt.Print(COLOR_YELLOW)
		fmt.Printf("================================================\n")
		fmt.Printf("Database version is out of date.  Upgrading database from version %d to version %d!\n", version,
			ENGINE_DATABASE_VERSION)
		fmt.Print(COLOR_RESET)
		return db.UpgradeFromXToY(version, ENGINE_DATABASE_VERSION)
	}

	return nil
}

// Run all the migrations from version X to version Y, and update the `database_version` table's `version_number`
func (db DB) UpgradeFromXToY(x int, y int) error {
	for i := x; i < y; i++ {
		fmt.Print(COLOR_CYAN)
		fmt.Println(MIGRATIONS[i])
		fmt.Print(COLOR_RESET)

		db.DB.MustExec(MIGRATIONS[i])
		db.DB.MustExec("update database_version set version_number = ?", i+1)

		fmt.Print(COLOR_YELLOW)
		fmt.Printf("Now at database schema version %d.\n", i+1)
		fmt.Print(COLOR_RESET)
	}
	fmt.Print(COLOR_GREEN)
	fmt.Printf("================================================\n")
	fmt.Printf("Database version has been upgraded to version %d.\n", y)
	fmt.Print(COLOR_RESET)
	return nil
}

type VersionMismatchError struct {
	EngineVersion   int
	DatabaseVersion int
}

func (e VersionMismatchError) Error() string {
	return fmt.Sprintf(
		`This profile was created with database schema version %d, which is newer than this application's database schema version, %d.
Please upgrade this application to a newer version to use this profile.  Or downgrade the profile's schema version, somehow.`,
		e.DatabaseVersion, e.EngineVersion,
	)
}

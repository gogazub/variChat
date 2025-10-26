package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Init(dsn string) error {
    var err error
    DB, err = sql.Open("mysql", dsn)
    if err != nil {
        return err
    }

    DB.SetMaxOpenConns(20)
    DB.SetMaxIdleConns(10)

    if err := DB.Ping(); err != nil {
        return fmt.Errorf("failed to ping MySQL: %w", err)
    }
    return nil
}

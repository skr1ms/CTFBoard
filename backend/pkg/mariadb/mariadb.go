package mariadb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/skr1ms/CTFBoard/config"
)

const (
	maxPoolSize  = 10
	connAttempts = 10
	connTimeout  = time.Second
)

func New(cfg *config.DB) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 0; i < connAttempts; i++ {
		db, err = sql.Open("mysql", cfg.URL)
		if err == nil {
			if err = db.PingContext(context.Background()); err == nil {
				db.SetMaxOpenConns(maxPoolSize)
				db.SetMaxIdleConns(maxPoolSize)
				db.SetConnMaxLifetime(5 * time.Minute)

				return db, nil
			}
		}
		time.Sleep(connTimeout)
	}

	return nil, fmt.Errorf("mariadb - New - connAttempts failed: %w", err)
}

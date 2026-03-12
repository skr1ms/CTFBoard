package migrator

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/TakuyaYagam1/AstroCTFb/config"
)

func Run(cfg *config.DB) error {
	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return fmt.Errorf("migrator - Run - sql.Open: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrator - Run - goose.SetDialect: %w", err)
	}
	if err := goose.Up(db, cfg.MigrationsPath); err != nil {
		return fmt.Errorf("migrator - Run - goose.Up: %w", err)
	}
	return nil
}

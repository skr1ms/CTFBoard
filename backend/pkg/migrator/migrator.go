package migrator

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/skr1ms/CTFBoard/config"
)

func Run(cfg *config.DB) error {
	migrationDBURL := fmt.Sprintf("mysql://%s", cfg.URL)

	m, err := migrate.New(
		"file://"+cfg.MigrationsPath,
		migrationDBURL,
	)

	if err != nil {
		return fmt.Errorf("migrator - Run - migrate.New: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("migrator - Run - m.Up: %w", err)
	}

	return nil
}

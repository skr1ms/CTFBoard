package migrator

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/skr1ms/CTFBoard/config"
)

func Run(cfg *config.DB) error {
	m, err := migrate.New(
		"file://"+cfg.MigrationsPath,
		cfg.URL,
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

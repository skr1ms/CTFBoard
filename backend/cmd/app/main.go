package main

import (
	"fmt"
	"os"

	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/app"
)

func main() {
	l, err := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger initialization failed: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.New()
	if err != nil {
		l.WithError(err).Fatal("Config initialization failed")
	}

	l.Info("Configuration loaded successfully")

	app.Run(cfg, l)
}

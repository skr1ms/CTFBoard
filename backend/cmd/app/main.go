package main

import (
	"log"

	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/app"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Config initialization failed: %v", err)
	}

	l, err := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	if err != nil {
		log.Fatalf("Logger initialization failed: %v", err)
	}

	l.Info("Configuration loaded successfully")

	app.Run(cfg, l)
}

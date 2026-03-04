package main

import (
	"log"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/app"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Config initialization failed: %v", err)
	}

	l := logger.New(&logger.Options{
		Level:  logger.InfoLevel,
		Output: logger.ConsoleOutput,
	})
	l.Info("Configuration loaded successfully")

	app.Run(cfg, l)
}

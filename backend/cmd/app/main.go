package main

import (
	"os"

	"github.com/skr1ms/CTFBoard/config"
	"github.com/skr1ms/CTFBoard/internal/app"
	"github.com/skr1ms/CTFBoard/pkg/logger"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		println("Config initialization failed: " + err.Error())
		os.Exit(1)
	}

	l := logger.New(&logger.Options{
		Level:  logger.InfoLevel,
		Output: logger.ConsoleOutput,
	})
	l.Info("Configuration loaded successfully")

	app.Run(cfg, l)
}

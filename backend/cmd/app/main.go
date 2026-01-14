package main

import (
	"log"

	"github.com/skr1ms/CTFBoard/config"
	"github.com/skr1ms/CTFBoard/internal/app"
	"github.com/skr1ms/CTFBoard/pkg/logger"

	_ "github.com/skr1ms/CTFBoard/docs"
)

// @title           CTFBoard API
// @version         1.0.0
// @description     REST API for managing CTF competition
// @termsOfService  https://ctfleague.ru/terms

// @contact.name   API Support
// @contact.email  skr1ms13666@gmail.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      api.ctfleague.ru
// @BasePath  /api/v1
// @schemes   https http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT access token. Format: "Bearer {token}"

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Config initialization failed: %v", err)
	}

	l := logger.New(cfg.LogLevel, cfg.ChiMode)
	l.Info("Configuration loaded successfully", nil)

	app.Run(cfg, l)
}

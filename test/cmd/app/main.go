package main

import (
	"log/slog"
	"os"
	config "test/org/config"
	providers "test/org/internal/providers"

	"go.uber.org/fx"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("configuration loaded", "environment", cfg.Env)

	app := fx.New(
		fx.Supply(cfg),
		fx.Provide(
			fxutil.Logger,
			providers.DB(true),
		),
		fx.Invoke(
			fxutil.Logger,
		),
	)

	app.Run()
}

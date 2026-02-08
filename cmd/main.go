package main

import (
	"time"

	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/ctx"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	"github.com/shanth1/support-bot/internal/app"
	"github.com/shanth1/support-bot/internal/config"
)

func main() {
	ctx, shutdownCtx, cancel, shutdownCancel := ctx.WithGracefulShutdown(10 * time.Second)
	defer cancel()
	defer shutdownCancel()

	logger := log.New()
	logger.Info().Msg("starting service")

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("load config")
	}

	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("invalid configuration")
	}

	logger = logger.WithOptions(log.WithConfig(log.Config{
		Level:        cfg.Logger.Level,
		App:          cfg.Logger.App,
		Service:      cfg.Logger.Service,
		UDPAddress:   cfg.Logger.UDPAddress,
		EnableCaller: cfg.Logger.EnableCaller,
		Console:      cfg.Env != consts.EnvProd,
		JSONOutput:   cfg.Env == consts.EnvProd,
	}))

	logger.Info().Any(logkeys.Env, cfg.Env).Msg("application has been successfully configured")

	ctx = log.NewContext(ctx, logger)

	app, err := app.New(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create app")
	}

	app.Run(ctx, shutdownCtx)
}

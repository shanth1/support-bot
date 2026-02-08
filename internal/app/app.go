package app

import (
	"context"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	"github.com/shanth1/support-bot/internal/bot"
	"github.com/shanth1/support-bot/internal/config"
	"github.com/shanth1/support-bot/internal/server"
	"github.com/shanth1/support-bot/internal/storage"
	"golang.org/x/sync/errgroup"
)

type Worker interface {
	Run(ctx context.Context, shutdownCtx context.Context) error
}

type App struct {
	logger   log.Logger
	workers  []Worker
	cleanups []func()
}

func New(cfg *config.Config, logger log.Logger) (Worker, error) {
	app := &App{
		logger: logger,
	}

	store, err := storage.New(cfg.Storage.Path, logger)
	if err != nil {
		return nil, err
	}
	app.onShutdown(func() {
		logger.Debug().Msg("closing storage...")
		_ = store.Close()
	})

	tgBot, err := bot.New(cfg, store, logger)
	if err != nil {
		app.closeAll()
		return nil, err
	}
	app.workers = append(app.workers, tgBot)

	if cfg.Server.Enabled && cfg.Server.Addr != "" {
		srv := server.New(cfg.Server.Addr, cfg.Server.APIKey, tgBot, logger)
		app.workers = append(app.workers, srv)
	}

	return app, nil
}

func (app *App) onShutdown(fn func()) {
	app.cleanups = append([]func(){fn}, app.cleanups...)
}

func (app *App) closeAll() {
	for _, fn := range app.cleanups {
		fn()
	}
}

func (app *App) Run(ctx context.Context, shutdownCtx context.Context) error {
	defer app.closeAll()

	g, runCtx := errgroup.WithContext(ctx)

	for _, w := range app.workers {
		worker := w
		g.Go(func() error {
			return worker.Run(runCtx, shutdownCtx)
		})
	}

	app.logger.Info().Int(logkeys.Amount, len(app.workers)).Msg("app workers started")

	err := g.Wait()

	if err != nil && err != context.Canceled {
		app.logger.Error().Err(err).Msg("app stopped with error")
		return err
	}

	app.logger.Info().Msg("app stopped gracefully")
	return nil
}

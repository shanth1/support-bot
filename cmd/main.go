package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shanth1/support-bot/internal/bot"
	"github.com/shanth1/support-bot/internal/config"
	"github.com/shanth1/support-bot/internal/server"
	"github.com/shanth1/support-bot/internal/storage"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.MustLoad()

	// DB
	store, err := storage.New(cfg.DBPath)
	if err != nil {
		logger.Error("Storage init failed", "err", err)
		os.Exit(1)
	}

	// Bot
	tgBot, err := bot.New(cfg.BotToken, cfg.AdminGroupID, store)
	if err != nil {
		logger.Error("Bot init failed", "err", err)
		os.Exit(1)
	}

	go tgBot.Start()
	logger.Info("Bot started")

	// Server
	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: server.NewMux(tgBot, cfg.APIKey),
	}

	go func() {
		logger.Info("Server listening", "port", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "err", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = srv.Shutdown(ctx)
	logger.Info("Exited")
}

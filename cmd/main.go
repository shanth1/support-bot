package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/shanth1/support-bot/internal/bot"
	"github.com/shanth1/support-bot/internal/config"
	"github.com/shanth1/support-bot/internal/server"
	"github.com/shanth1/support-bot/internal/storage"
)

func main() {
	_ = os.Mkdir("data", 0755)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.MustLoad()
	store, _ := storage.New(cfg.DBPath)
	tgBot, _ := bot.New(cfg, store, logger)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: server.NewMux(tgBot, cfg.APIKey),
	}

	go tgBot.Start()

	logger.Info("Server started", "port", cfg.Port)
	srv.ListenAndServe()
}

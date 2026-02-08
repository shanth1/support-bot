package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
)

type BotService interface {
	SendNotification(title, message string) error
}

type Server struct {
	httpServer *http.Server
	bot        BotService
	apiKey     string
	log        log.Logger
}

func New(addr string, apiKey string, bot BotService, l log.Logger) *Server {
	mux := http.NewServeMux()
	s := &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		bot:    bot,
		apiKey: apiKey,
		log:    l,
	}
	mux.HandleFunc("/notify", s.handleNotify)
	return s
}

func (s *Server) Run(ctx context.Context, shutdownCtx context.Context) error {
	s.log.Info().Str("addr", s.httpServer.Addr).Msg("HTTP Server starting")

	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		s.log.Info().Msg("http server received stop signal")
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.log.Warn().Err(err).Msg("HTTP Server shutdown error")
			return s.httpServer.Close()
		}
		return nil
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

type NotifyReq struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

func (s *Server) handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.apiKey != "" {
		token := r.Header.Get(consts.HeaderAuthorization)
		if token != "Bearer "+s.apiKey {
			s.log.Warn().
				Str(logkeys.RemoteAddr, r.RemoteAddr).
				Msg("unauthorized api attempt")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req NotifyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := s.bot.SendNotification(req.Title, req.Message); err != nil {
		s.log.Error().Err(err).Msg("Failed to send notification via bot")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set(consts.HeaderContentType, string(consts.ContentTypeJSON))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"sent"}`))
}

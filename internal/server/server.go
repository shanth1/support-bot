package server

import (
	"encoding/json"
	"net/http"

	"github.com/shanth1/support-bot/internal/bot"
)

type NotificationReq struct {
	Project string `json:"project"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
	TopicID int    `json:"topic_id"`
}

func NewMux(b *bot.Bot, apiKey string) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /notify", func(w http.ResponseWriter, r *http.Request) {
		if apiKey != "" && r.Header.Get("Authorization") != "Bearer "+apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req NotificationReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		err := b.SendNotification(req.Project, req.Type, req.Name, req.Email, req.Message, req.TopicID)
		if err != nil {
			http.Error(w, "Bot Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	return mux
}

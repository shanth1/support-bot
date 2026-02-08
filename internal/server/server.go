package server

import (
	"encoding/json"
	"net/http"
)

type NotificationReq struct {
	Project string            `json:"project"`
	Type    string            `json:"type"`
	Name    string            `json:"name"`
	Email   string            `json:"email"`
	Message string            `json:"message"`
	Meta    map[string]string `json:"meta"`
}

type BotService interface {
	SendNotification(project, msgType, name, email, message string, meta map[string]string) error
}

func NewMux(bot BotService, apiKey string) *http.ServeMux {
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

		err := bot.SendNotification(req.Project, req.Type, req.Name, req.Email, req.Message, req.Meta)
		if err != nil {
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"sent"}`))
	})

	return mux
}

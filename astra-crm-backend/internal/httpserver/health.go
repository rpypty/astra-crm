package httpserver

import (
	"context"
	"net/http"
	"time"
)

type ReadyPinger interface {
	Ping(ctx context.Context) error
}

type HealthResponse struct {
	Status string `json:"status"`
}

func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
	}
}

func ReadyHandler(pinger ReadyPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if pinger == nil {
			WriteJSON(w, http.StatusServiceUnavailable, HealthResponse{Status: "not_ready"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pinger.Ping(ctx); err != nil {
			WriteJSON(w, http.StatusServiceUnavailable, HealthResponse{Status: "not_ready"})
			return
		}

		WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
	}
}

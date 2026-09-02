package api

import (
  "net/http"
  "github.com/go-chi/chi/v5"
  "github.com/example/warp-server/internal/db"
)

func NewRouter(database *db.DB) http.Handler {
  r := chi.NewRouter()
  r.Get("/health", HealthHandler)
  return r
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  w.Write([]byte(`{"status":"ok"}`))
}

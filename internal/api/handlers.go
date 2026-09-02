package api

import (
  "encoding/json"
  "net/http"
  "github.com/go-chi/chi/v5"
  "github.com/example/warp-server/internal/db"
)

func NewRouter(database *db.DB) http.Handler {
  r := chi.NewRouter()
  r.Get("/health", HealthHandler)
  r.Get("/config", ConfigHandler(database))
  r.Get("/api/identities", IdentitiesHandler(database))
  r.Get("/api/endpoints", EndpointsHandler(database))
  return r
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  w.Write([]byte(`{"status":"ok"}`))
}

func IdentitiesHandler(database *db.DB) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    identities, err := database.GetIdentities()
    if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(identities)
  }
}

func EndpointsHandler(database *db.DB) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    endpoints, err := database.GetAliveEndpoints(0)
    if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(endpoints)
  }
}

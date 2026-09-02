package main

import (
  "log"
  "net/http"
  "github.com/example/warp-server/internal/api"
  "github.com/example/warp-server/internal/db"
)

func main() {
  database, err := db.New("warp.db")
  if err != nil {
    log.Fatal(err)
  }
  defer database.Close()
  
  router := api.NewRouter(database)
  log.Println("Server listening on :8080")
  http.ListenAndServe(":8080", router)
}

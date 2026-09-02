package main

import (
  "context"
  "log"
  "os"
  "time"
  "github.com/example/warp-server/internal/db"
  "github.com/example/warp-server/internal/warp"
  "github.com/example/warp-server/internal/scanner"
)

func main() {
  if len(os.Args) < 2 {
    log.Fatal("Usage: worker <register|scan|probe>")
  }
  
  cmd := os.Args[1]
  ctx := context.Background()
  
  database, err := db.New("warp.db")
  if err != nil {
    log.Fatalf("Failed to open DB: %v", err)
  }
  defer database.Close()
  
  switch cmd {
  case "register":
    log.Println("Starting register worker")
    MaintainIdentityPool(ctx, database, 10, 5*time.Minute)
  case "scan":
    log.Println("Starting scan worker")
    ScanPeriodically(ctx, database, 10*time.Minute)
  case "probe":
    log.Println("Starting probe worker")
    ProbeEndpointsPeriodically(ctx, database, 2*time.Minute)
  default:
    log.Fatalf("Unknown command: %s", cmd)
  }
}

func MaintainIdentityPool(ctx context.Context, database *db.DB, targetCount int, interval time.Duration) {
  ticker := time.NewTicker(interval)
  defer ticker.Stop()
  
  for {
    select {
    case <-ctx.Done():
      return
    case <-ticker.C:
      identities, _ := database.GetIdentities()
      if len(identities) < targetCount {
        log.Printf("Registering new identity (%d/%d)", len(identities), targetCount)
        
        identity, err := warp.RegisterIdentity(ctx)
        if err != nil {
          log.Printf("Registration failed: %v", err)
          continue
        }
        
        if err := database.InsertIdentity(identity); err != nil {
          log.Printf("Failed to save identity: %v", err)
        }
      }
    }
  }
}

func ScanPeriodically(ctx context.Context, database *db.DB, interval time.Duration) {
  ticker := time.NewTicker(interval)
  defer ticker.Stop()
  
  for {
    select {
    case <-ctx.Done():
      return
    case <-ticker.C:
      log.Println("Starting endpoint scan")
      
      prefixes := warp.WarpPrefixes()
      ports := warp.WarpPorts()
      
      endpoints, err := scanner.ScanEndpoints(ctx, prefixes, ports, 50)
      if err != nil {
        log.Printf("Scan failed: %v", err)
        continue
      }
      
      for _, ep := range endpoints {
        if err := database.UpsertEndpoint(&ep); err != nil {
          log.Printf("Failed to save endpoint: %v", err)
        }
      }
      
      log.Printf("Scan complete: %d endpoints found", len(endpoints))
    }
  }
}

func ProbeEndpointsPeriodically(ctx context.Context, database *db.DB, interval time.Duration) {
  ticker := time.NewTicker(interval)
  defer ticker.Stop()
  
  for {
    select {
    case <-ctx.Done():
      return
    case <-ticker.C:
      log.Println("Starting endpoint probing")
      
      endpoints, _ := database.GetAliveEndpoints(0.5)
      identities, _ := database.GetIdentities()
      
      if len(identities) == 0 {
        continue
      }
      
      identity := identities[0]
      keys := &scanner.IdentityKeys{
        PrivateKey: identity.PrivateKey,
        PublicKey:  identity.PublicKey,
      }
      
      for _, ep := range endpoints {
        rtt, err := scanner.ProbeEndpoint(ctx, *ep, keys)
        success := err == nil
        
        if success {
          log.Printf("Probed %s:%d - RTT %dms", ep.Host, ep.Port, rtt)
        }
      }
    }
  }
}

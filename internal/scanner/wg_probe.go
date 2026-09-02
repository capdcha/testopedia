package scanner

import (
  "context"
  "time"
)

type IdentityKeys struct {
  PrivateKey string
  PublicKey  string
}

func ProbeEndpoint(ctx context.Context, endpoint Endpoint, keys *IdentityKeys) (int, error) {
  start := time.Now()
  
  // TODO: Реальный WireGuard handshake с использованием golang.zx2c4.com/wireguard
  // Сейчас заглушка
  time.Sleep(50 * time.Millisecond)
  
  return int(time.Since(start).Milliseconds()), nil
}

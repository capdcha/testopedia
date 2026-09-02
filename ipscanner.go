package scanner

import (
  "context"
  "net"
  "strconv"
  "time"
)

type Endpoint struct {
  Host string
  Port int
  RTT  int
}

func ScanEndpoints(ctx context.Context, prefixes []string, ports []int, maxResults int) ([]Endpoint, error) {
  var results []Endpoint
  
  // Простейший TCP-скан (для полноценного нужен bepass-org/warp-plus ipscanner)
  for _, prefix := range prefixes {
    ip, ipnet, _ := net.ParseCIDR(prefix)
    for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
      for _, port := range ports {
        start := time.Now()
        conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip.String(), strconv.Itoa(port)), 2*time.Second)
        if err == nil {
          conn.Close()
          rtt := int(time.Since(start).Milliseconds())
          results = append(results, Endpoint{Host: ip.String(), Port: port, RTT: rtt})
          if len(results) >= maxResults {
            return results, nil
          }
        }
      }
    }
  }
  
  return results, nil
}

func inc(ip net.IP) {
  for j := len(ip) - 1; j >= 0; j-- {
    ip[j]++
    if ip[j] > 0 {
      break
    }
  }
}

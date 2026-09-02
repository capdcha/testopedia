package scanner

import (
  "context"
  "net"
  "sort"
  "strconv"
  "sync"
  "time"
)

type Endpoint struct {
  ID   int64
  Host string
  Port int
  RTT  int
}

type job struct {
  host string
  port int
}

func ScanEndpoints(ctx context.Context, prefixes []string, ports []int, maxResults int) ([]Endpoint, error) {
  if len(prefixes) == 0 || len(ports) == 0 {
    return []Endpoint{}, nil
  }

  if d, ok := ctx.Deadline(); !ok || time.Until(d) > 60*time.Second {
    var cancel context.CancelFunc
    ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
    defer cancel()
  }

  var jobs []job
  for _, prefix := range prefixes {
    ip, ipnet, err := net.ParseCIDR(prefix)
    if err != nil {
      continue
    }
    for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
      host := ip.String()
      for _, port := range ports {
        jobs = append(jobs, job{host: host, port: port})
      }
    }
  }

  jobCh := make(chan job)
  var (
    mu      sync.Mutex
    results []Endpoint
    wg      sync.WaitGroup
  )

  const workers = 256

  for i := 0; i < workers; i++ {
    wg.Add(1)
    go func() {
      defer wg.Done()
      dialer := &net.Dialer{Timeout: 2 * time.Second}
      for j := range jobCh {
        if ctx.Err() != nil {
          return
        }
        start := time.Now()
        conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(j.host, strconv.Itoa(j.port)))
        if err != nil {
          continue
        }
        conn.Close()
        rtt := int(time.Since(start).Milliseconds())

        mu.Lock()
        results = append(results, Endpoint{Host: j.host, Port: j.port, RTT: rtt})
        mu.Unlock()
      }
    }()
  }

sendLoop:
  for _, j := range jobs {
    select {
    case jobCh <- j:
    case <-ctx.Done():
      break sendLoop
    }
  }
  close(jobCh)
  wg.Wait()

  mu.Lock()
  defer mu.Unlock()
  sort.Slice(results, func(i, j int) bool {
    return results[i].RTT < results[j].RTT
  })
  if len(results) > maxResults {
    results = results[:maxResults]
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

package db

import (
  "github.com/example/warp-server/internal/scanner"
  "time"
)

func (db *DB) UpsertEndpoint(e *scanner.Endpoint) error {
  _, err := db.conn.Exec(`
    INSERT INTO endpoints (host, port, rtt_ms, last_seen)
    VALUES (?, ?, ?, ?)
    ON CONFLICT(host, port) DO UPDATE SET
      rtt_ms = excluded.rtt_ms,
      last_seen = excluded.last_seen`,
    e.Host, e.Port, e.RTT, time.Now(),
  )
  return err
}

func (db *DB) GetAllEndpoints() ([]*scanner.Endpoint, error) {
  rows, err := db.conn.Query(`
    SELECT id, host, port, rtt_ms FROM endpoints
    ORDER BY rtt_ms ASC`)
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  result := []*scanner.Endpoint{}
  for rows.Next() {
    var e scanner.Endpoint
    rows.Scan(&e.ID, &e.Host, &e.Port, &e.RTT)
    result = append(result, &e)
  }
  return result, nil
}

func (db *DB) GetAliveEndpoints(minSuccessRate float64) ([]*scanner.Endpoint, error) {
  rows, err := db.conn.Query(`
    SELECT id, host, port, rtt_ms FROM endpoints
    WHERE datetime(last_seen) > datetime('now', '-1 hour')
      AND (success_count + fail_count = 0
           OR (success_count * 1.0) / (success_count + fail_count) >= ?)
    ORDER BY rtt_ms ASC`,
    minSuccessRate,
  )
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  result := []*scanner.Endpoint{}
  for rows.Next() {
    var e scanner.Endpoint
    rows.Scan(&e.ID, &e.Host, &e.Port, &e.RTT)
    result = append(result, &e)
  }
  return result, nil
}

func (db *DB) UpdateEndpointMetrics(id int64, rtt int, success bool) error {
  var query string
  var args []interface{}

  if success {
    query = "UPDATE endpoints SET rtt_ms = ?, success_count = success_count + 1, last_checked = ? WHERE id = ?"
    args = []interface{}{rtt, time.Now(), id}
  } else {
    query = "UPDATE endpoints SET fail_count = fail_count + 1, last_checked = ? WHERE id = ?"
    args = []interface{}{time.Now(), id}
  }

  _, err := db.conn.Exec(query, args...)
  return err
}

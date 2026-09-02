# WARP Server: Промпты для выполнения агентами

Каждый промпт — изолированная задача для отдельного агента. Передавайте промпты последовательно согласно графу зависимостей.

---

## ЗАДАЧА 1: Схема БД и миграции

```
Создай SQL-схему для SQLite базы данных с тремя таблицами:

1. `identities` - хранение WARP identity:
   - id (TEXT PRIMARY KEY)
   - private_key (TEXT NOT NULL)
   - public_key (TEXT NOT NULL)
   - client_id (TEXT NOT NULL)
   - token (TEXT NOT NULL)
   - addresses_v4 (TEXT NOT NULL, JSON-массив)
   - addresses_v6 (TEXT NOT NULL, JSON-массив)
   - peer_public_key (TEXT NOT NULL)
   - created_at (DATETIME DEFAULT CURRENT_TIMESTAMP)

2. `endpoints` - хранение проверенных WARP-узлов:
   - id (INTEGER PRIMARY KEY AUTOINCREMENT)
   - host (TEXT NOT NULL)
   - port (INTEGER NOT NULL)
   - rtt_ms (INTEGER)
   - success_count (INTEGER DEFAULT 0)
   - fail_count (INTEGER DEFAULT 0)
   - last_seen (DATETIME)
   - last_checked (DATETIME)
   - UNIQUE(host, port)

3. `configs` - история сгенерированных конфигов (опционально):
   - id (INTEGER PRIMARY KEY AUTOINCREMENT)
   - identity_id (TEXT FK -> identities.id)
   - endpoint_id (INTEGER FK -> endpoints.id)
   - config_text (TEXT NOT NULL)
   - created_at (DATETIME DEFAULT CURRENT_TIMESTAMP)

Добавь индексы:
- idx_endpoints_alive: (success_count, fail_count, last_seen)
- idx_endpoints_rtt: (rtt_ms)
- idx_identities_created: (created_at)

Выходной файл: `schema.sql`

Проверка: выполни `sqlite3 warp.db < schema.sql` и убедись, что все таблицы созданы.
```

---

## ЗАДАЧА 2: HTTP API сервер (скелет)

**Зависимости:** Задача 1

```
Создай HTTP API сервер на Go со следующей структурой:

- `cmd/server/main.go`: точка входа, подключение к SQLite БД (`warp.db`), запуск HTTP-сервера на :8080
- `internal/api/handlers.go`: chi router с одним эндпоинтом GET /health → {"status":"ok"}
- `internal/db/db.go`: структура DB с методами New(path string), Close()

Стек:
- github.com/go-chi/chi/v5 для роутинга
- github.com/mattn/go-sqlite3 для SQLite
- database/sql стандартная библиотека

go.mod:
```
module github.com/example/warp-server
go 1.21
```

Проверка: запусти сервер, выполни `curl http://localhost:8080/health`, должен вернуть `{"status":"ok"}`.
```

---

## ЗАДАЧА 3: Модуль генерации X25519 keypair

**Зависимости:** нет

```
Создай Go-модуль для генерации X25519 ключевых пар:

- Файл: `internal/crypto/keypair.go`
- Функция: `GenerateX25519() (privateKey, publicKey []byte, err error)`
- Используй: `crypto/rand` + `golang.org/x/crypto/curve25519`

Приватный ключ: 32 случайных байта
Публичный ключ: curve25519.X25519(privateKey, curve25519.Basepoint)

Добавь unit-тест `internal/crypto/keypair_test.go`:
- Проверь, что оба ключа по 32 байта
- Проверь, что из приватного ключа восстанавливается тот же публичный

Проверка: `go test ./internal/crypto` должен пройти.
```

---

## ЗАДАЧА 4: Модуль WARP-регистрации

**Зависимости:** Задача 3

```
Создай модуль регистрации WARP identity через Cloudflare API:

- Файл: `internal/warp/register.go`
- Функция: `RegisterIdentity(ctx context.Context) (*Identity, error)`

Структура Identity:
```go
type Identity struct {
  ID             string
  PrivateKey     string // base64
  PublicKey      string // base64
  ClientID       string
  Token          string
  AddressesV4    []string
  AddressesV6    []string
  PeerPublicKey  string // base64
}
```

Алгоритм:
1. Генерируй пару ключей через `crypto.GenerateX25519()`
2. Отправь POST https://api.cloudflareclient.com/v0a4471/reg
   - Headers: 
     - Content-Type: application/json
     - CF-Client-Version: a-6.35-4471
     - User-Agent: okhttp/3.12.1
   - Body (JSON):
     ```json
     {
       "key": "<base64 публичного ключа>",
       "install_id": "",
       "fcm_token": "",
       "tos": "2023-01-01T00:00:00.000Z",
       "type": "Android",
       "locale": "en_US"
     }
     ```
3. Парси ответ:
   - result.id → Identity.ID
   - result.token → Identity.Token
   - result.config.client_id → Identity.ClientID
   - result.config.peers[0].public_key → Identity.PeerPublicKey
   - result.config.peers[0].endpoint.(v4/v6/host) → игнорируй (берём из сканера)
   - result.config.interface.addresses.(v4/v6) → Identity.Addresses*

Проверка: создай unit-тест с mock HTTP сервером (golden response JSON).
```

---

## ЗАДАЧА 5: Модуль сохранения identity в БД

**Зависимости:** Задача 1, Задача 4

```
Создай модуль для работы с таблицей `identities`:

- Файл: `internal/db/identities.go`
- Методы (на структуре DB):
  - `InsertIdentity(id *warp.Identity) error` - вставка identity в БД
  - `GetIdentities() ([]*warp.Identity, error)` - получение всех identity
  - `GetIdentityByID(id string) (*warp.Identity, error)` - получение по ID

При вставке сериализуй массивы AddressesV4/V6 в JSON (encoding/json).
При чтении десериализуй обратно.

Добавь integration-тест:
- Создай временную БД
- Вставь тестовую identity
- Прочитай обратно
- Сравни все поля

Проверка: `go test ./internal/db` проходит.
```

---

## ЗАДАЧА 6: Модуль префиксов WARP

**Зависимости:** нет

```
Создай модуль с константами CIDR-префиксов WARP:

- Файл: `internal/warp/prefixes.go`
- Функция: `WarpPrefixes() []string`

Возвращай минимум 10 CIDR:
- 162.159.192.0/24
- 162.159.193.0/24
- 162.159.195.0/24
- 188.114.96.0/24
- 188.114.97.0/24
- 188.114.98.0/24
- 188.114.99.0/24
- 162.159.36.0/24
- 162.159.46.0/24
- 162.159.138.0/24

Unit-тест: проверь, что функция возвращает >= 10 элементов и содержит известные префиксы.
```

---

## ЗАДАЧА 7: Модуль портов WARP

**Зависимости:** нет

```
Создай модуль с константами портов WARP:

- Файл: `internal/warp/ports.go`
- Функция: `WarpPorts() []int`

Возвращай ~55 портов (из анализа Nova-Android):
500, 854, 859, 864, 878, 880, 890, 891, 894, 903, 908, 928, 934, 939,
942, 943, 945, 946, 955, 968, 987, 988, 1002, 1010, 1014, 1018, 1070,
1074, 1180, 1387, 1701, 1843, 2371, 2408, 2506, 3138, 3476, 3581, 3854,
4177, 4198, 4233, 4500, 5279, 5956, 7103, 7152, 7156, 7281, 7559, 8319,
8742, 8854, 8886

Unit-тест: проверь, что возвращается >= 50 портов и что 500, 2408 присутствуют.
```

---

## ЗАДАЧА 8: Модуль ipscanner

**Зависимости:** Задача 6, Задача 7

```
Создай модуль сканирования WARP endpoints:

- Файл: `internal/scanner/ipscanner.go`
- Структура:
  ```go
  type Endpoint struct {
    Host string
    Port int
    RTT  int // milliseconds
  }
  ```
- Функция: `ScanEndpoints(ctx context.Context, prefixes []string, ports []int, maxResults int) ([]Endpoint, error)`

Алгоритм (упрощённая версия):
1. Для каждого CIDR в prefixes, разверни все IP
2. Для каждого IP × каждый порт:
   - Выполни TCP-connect с таймаутом 2 секунды
   - Если успех → замерь RTT, добавь в результаты
   - Если набрано maxResults → вернуть
3. Отсортируй результаты по RTT (по возрастанию)

Опционально: интегрируй github.com/bepass-org/warp-plus/ipscanner для полноценного ICMP пинга.

Integration-тест: запусти скан на реальной сети (с таймаутом 30 сек), ожидай >= 1 живой endpoint.

Проверка: `go test ./internal/scanner -timeout 60s`
```

---

## ЗАДАЧА 9: Модуль neighbor expansion

**Зависимости:** нет

```
Создай модуль для расширения IP соседями:

- Файл: `internal/scanner/neighbors.go`
- Функция: `ExpandNeighbors(baseIP string, rangeSize int) []string`

Алгоритм:
1. Парси baseIP (IPv4)
2. Извлеки последний октет (например, для 162.159.192.5 → 5)
3. Генерируй IP с последним октетом от (5 - rangeSize) до (5 + rangeSize)
4. Ограничь диапазон 0..255
5. Верни массив IP-адресов

Пример: ExpandNeighbors("162.159.192.5", 3) → ["162.159.192.2", "162.159.192.3", ..., "162.159.192.8"]

Unit-тест: проверь граничные случаи (0, 255, отрицательные).
```

---

## ЗАДАЧА 10: Модуль WireGuard handshake probe

**Зависимости:** нет (но использует структуры из Задачи 4)

```
Создай модуль для проверки endpoint через WireGuard handshake:

- Файл: `internal/scanner/wg_probe.go`
- Функция: `ProbeEndpoint(ctx context.Context, endpoint Endpoint, identity *warp.Identity) (rttMs int, err error)`

Алгоритм (упрощённый для MVP):
1. Отправь WireGuard handshake initiation на endpoint.Host:endpoint.Port
   - Используй golang.zx2c4.com/wireguard/conn или wireguard-go
   - Приватный ключ из identity.PrivateKey
   - Peer public key из identity.PeerPublicKey
2. Ожидай response с таймаутом 5 секунд
3. Если получен response → возврати RTT в миллисекундах
4. Если таймаут/ошибка → возврати error

Для MVP можно сделать заглушку (TCP-connect + sleep 50ms).

Unit-тест: mock UDP socket.

Проверка: `go test ./internal/scanner`
```

---

## ЗАДАЧА 11: Модуль сохранения endpoints в БД

**Зависимости:** Задача 1, Задача 8

```
Создай модуль для работы с таблицей `endpoints`:

- Файл: `internal/db/endpoints.go`
- Методы (на структуре DB):
  - `UpsertEndpoint(e *scanner.Endpoint) error` - вставка/обновление endpoint (по host+port)
  - `GetAliveEndpoints(minSuccessRate float64) ([]*scanner.Endpoint, error)` - получение живых endpoints (success_count/(success_count+fail_count) >= minSuccessRate, last_seen < 1 час назад), сортировка по rtt_ms
  - `UpdateEndpointMetrics(id int64, rtt int, success bool) error` - обновление метрик после пробы

При Upsert: ON CONFLICT(host, port) DO UPDATE SET rtt_ms, last_seen.

Integration-тест:
- Вставь endpoint
- Обнови метрики несколько раз (успех/неудача)
- Получи alive endpoints с разными minSuccessRate
- Проверь фильтрацию

Проверка: `go test ./internal/db`
```

---

## ЗАДАЧА 12: Генератор AmneziaWG конфига

**Зависимости:** Задача 4 (Identity), Задача 8 (Endpoint)

```
Создай генератор текстового AmneziaWG конфига:

- Файл: `internal/warp/config.go`
- Функция: `GenerateAmneziaConfig(identity *Identity, endpoint *scanner.Endpoint) string`

Формат .conf (INI-like):
```
[Interface]
PrivateKey = <identity.PrivateKey>
Address = <identity.AddressesV4[0]>, <identity.AddressesV6[0]>
DNS = 1.1.1.1, 1.0.0.1
Jc = 4
Jmin = 40
Jmax = 70
H1 = <случайное число 0..4294967295>
H2 = <случайное число>
H3 = <случайное число>
H4 = <случайное число>

[Peer]
PublicKey = <identity.PeerPublicKey>
Endpoint = <endpoint.Host>:<endpoint.Port>
AllowedIPs = 0.0.0.0/0, ::/0
```

Параметры обфускации (Jc, Jmin, Jmax, H1-H4) — константы из анализа Nova (см. warp_verified_seeds.json).

Unit-тест с golden файлом: сгенерируй конфиг, сравни с ожидаемым шаблоном.

Проверка: `go test ./internal/warp`
```

---

## ЗАДАЧА 13: API эндпоинт GET /config

**Зависимости:** Задача 2, Задача 5, Задача 11, Задача 12

```
Создай HTTP handler для генерации конфига:

- Файл: `internal/api/config.go`
- Handler: `ConfigHandler(database *db.DB) http.HandlerFunc`
- Эндпоинт: GET /config

Логика:
1. Получи все identity из БД (GetIdentities)
2. Получи alive endpoints (GetAliveEndpoints(0.8))
3. Если identity == 0 или endpoints == 0 → верни 503 Service Unavailable
4. Выбери случайную identity и endpoint с минимальным RTT
5. Сгенерируй конфиг через warp.GenerateAmneziaConfig
6. Верни с Content-Type: text/plain

Добавь в NewRouter (internal/api/handlers.go): r.Get("/config", ConfigHandler(database))

Integration-тест:
- Заполни БД тестовыми данными (identity + endpoint)
- Сделай запрос GET /config
- Проверь, что ответ содержит [Interface] и [Peer]

Проверка: `go test ./internal/api`
```

---

## ЗАДАЧА 14: API эндпоинты CRUD для identities

**Зависимости:** Задача 2, Задача 5

```
Создай REST API для работы с identity:

- Файл: `internal/api/identities.go`
- Эндпоинты:
  - GET /api/identities → JSON массив всех identity
  - POST /api/identities → регистрирует новую identity (вызывает warp.RegisterIdentity), возвращает JSON
  - GET /api/identities/:id → JSON одной identity

Добавь в NewRouter.

API-тесты с httptest:
- Создай identity через POST
- Получи список через GET
- Получи по ID

Проверка: `go test ./internal/api`
```

---

## ЗАДАЧА 15: API эндпоинты CRUD для endpoints

**Зависимости:** Задача 2, Задача 11

```
Создай REST API для работы с endpoints:

- Файл: `internal/api/endpoints.go`
- Эндпоинты:
  - GET /api/endpoints?alive=true → JSON массив endpoints (фильтр по alive опционален)
  - POST /api/endpoints → добавляет endpoint вручную (JSON body: {host, port, rtt})
  - PATCH /api/endpoints/:id → обновляет метрики (JSON body: {rtt, success})

Добавь в NewRouter.

API-тесты с httptest:
- Добавь endpoint через POST
- Получи список через GET
- Обнови метрики через PATCH
- Проверь фильтр ?alive=true

Проверка: `go test ./internal/api`
```

---

## ЗАДАЧА 16: Воркер регистрации identity (пул)

**Зависимости:** Задача 4, Задача 5

```
Создай фоновый воркер для поддержания пула identity:

- Файл: `cmd/worker/register.go`
- Функция: `MaintainIdentityPool(ctx context.Context, database *db.DB, targetCount int, interval time.Duration)`

Алгоритм:
1. Каждые `interval` (например, 5 минут):
2. Получи текущее количество identity из БД
3. Если count < targetCount:
   - Регистрируй новую identity через warp.RegisterIdentity
   - Сохрани в БД через database.InsertIdentity
   - Добавь задержку между регистрациями (rate-limit: 10 секунд)
4. Логируй успех/ошибки

Добавь CLI команду в cmd/worker/main.go:
```go
func main() {
  if len(os.Args) < 2 {
    log.Fatal("Usage: worker <command>")
  }
  
  cmd := os.Args[1]
  database, _ := db.New("warp.db")
  defer database.Close()
  
  ctx := context.Background()
  
  switch cmd {
  case "register":
    MaintainIdentityPool(ctx, database, 10, 5*time.Minute)
  default:
    log.Fatal("Unknown command")
  }
}
```

Integration-тест: запусти воркер в goroutine на 30 секунд, проверь, что identity появились в БД.

Проверка: `go test ./cmd/worker`
```

---

## ЗАДАЧА 17: Воркер сканирования endpoints

**Зависимости:** Задача 8, Задача 11

```
Создай фоновый воркер для периодического сканирования endpoints:

- Файл: `cmd/worker/scanner.go`
- Функция: `ScanPeriodically(ctx context.Context, database *db.DB, interval time.Duration)`

Алгоритм:
1. Каждые `interval` (например, 10 минут):
2. Получи префиксы через warp.WarpPrefixes()
3. Получи порты через warp.WarpPorts()
4. Запусти scanner.ScanEndpoints(ctx, prefixes, ports, 50)
5. Для каждого найденного endpoint:
   - database.UpsertEndpoint(&endpoint)
6. Логируй количество найденных endpoints

Добавь в cmd/worker/main.go:
```go
case "scan":
  ScanPeriodically(ctx, database, 10*time.Minute)
```

Integration-тест: запусти воркер, подожди один цикл (с коротким интервалом), проверь БД.

Проверка: `go test ./cmd/worker -timeout 5m`
```

---

## ЗАДАЧА 18: Воркер пробинга (health check)

**Зависимости:** Задача 10, Задача 11

```
Создай фоновый воркер для проверки живости endpoints:

- Файл: `cmd/worker/prober.go`
- Функция: `ProbeEndpointsPeriodically(ctx context.Context, database *db.DB, interval time.Duration)`

Алгоритм:
1. Каждые `interval` (например, 2 минуты):
2. Получи alive endpoints через database.GetAliveEndpoints(0.5)
3. Получи любую identity из БД (для handshake)
4. Для каждого endpoint:
   - Вызови scanner.ProbeEndpoint(ctx, endpoint, identity)
   - Если успех → database.UpdateEndpointMetrics(endpoint.ID, rtt, true)
   - Если ошибка → database.UpdateEndpointMetrics(endpoint.ID, 0, false)
5. Удали endpoints с fail_count > 10 (SQL: DELETE FROM endpoints WHERE fail_count > 10)

Добавь в cmd/worker/main.go:
```go
case "probe":
  ProbeEndpointsPeriodically(ctx, database, 2*time.Minute)
```

Integration-тест: добавь тестовые endpoints в БД, запусти воркер, проверь обновление метрик.

Проверка: `go test ./cmd/worker`
```

---

## ЗАДАЧА 19: Dockerfile и entrypoint

**Зависимости:** Задача 2, Задача 16, Задача 17, Задача 18

```
Создай Dockerfile для мультикоманды (server + worker):

- Файл: `Dockerfile`

Структура:
1. Stage 1: builder
   - FROM golang:1.21-alpine
   - COPY go.mod, go.sum
   - RUN go mod download
   - COPY . .
   - RUN go build -o /server ./cmd/server
   - RUN go build -o /worker ./cmd/worker

2. Stage 2: runtime
   - FROM alpine:latest
   - RUN apk add --no-cache sqlite ca-certificates
   - COPY --from=builder /server /server
   - COPY --from=builder /worker /worker
   - COPY schema.sql /schema.sql
   - EXPOSE 8080
   - ENTRYPOINT ["/server"]

Добавь .dockerignore:
```
*.db
.git
tmp/
```

Проверка: 
- `docker build -t warp-server .` собирается
- `docker run warp-server` запускает API-сервер
- `docker run warp-server /worker register` запускает воркер регистрации
```

---

## ЗАДАЧА 20: docker-compose для локального деплоя

**Зависимости:** Задача 19

```
Создай docker-compose.yml для запуска всех компонентов:

- Файл: `docker-compose.yml`

Сервисы:
1. api: 
   - build: .
   - command: /server
   - ports: ["8080:8080"]
   - volumes: ["./data:/data"]
   - environment: {DB_PATH: /data/warp.db}

2. worker-register:
   - build: .
   - command: /worker register
   - volumes: ["./data:/data"]
   - environment: {DB_PATH: /data/warp.db, TARGET_COUNT: 10, INTERVAL: 5m}

3. worker-scanner:
   - build: .
   - command: /worker scan
   - volumes: ["./data:/data"]
   - environment: {DB_PATH: /data/warp.db, INTERVAL: 10m}

4. worker-prober:
   - build: .
   - command: /worker probe
   - volumes: ["./data:/data"]
   - environment: {DB_PATH: /data/warp.db, INTERVAL: 2m}

Volumes: {data: {}}

Добавь init-script для создания БД при первом запуске (в entrypoint или в api-сервисе).

Проверка:
- `docker-compose up -d` запускает все сервисы
- `curl http://localhost:8080/health` отвечает
- Через 5 минут в data/warp.db появляются записи
```

---

## ЗАДАЧА 21: README с инструкциями

**Зависимости:** всё выше

```
Создай подробный README.md для проекта:

Разделы:
1. **Описание** - что делает проект (генератор WARP/AmneziaWG конфигов)
2. **Архитектура** - схема компонентов (API + 3 воркера), таблица интерфейсов
3. **Быстрый старт**:
   ```bash
   git clone <repo>
   cd warp-server
   docker-compose up -d
   curl http://localhost:8080/config > warp.conf
   ```
4. **API документация**:
   - GET /health
   - GET /config
   - GET /api/identities, POST /api/identities
   - GET /api/endpoints?alive=true
5. **Разработка**:
   - Требования (Go 1.21+, Docker, SQLite)
   - Сборка: `go build ./cmd/server`
   - Тесты: `go test ./...`
   - Локальный запуск: `./server`
6. **Конфигурация** - переменные окружения (DB_PATH, TARGET_COUNT, INTERVAL)
7. **Troubleshooting** - частые проблемы (429 от Cloudflare, пустой пул endpoints)

Проверка: следуй инструкциям из README на чистой системе, убедись, что всё работает.
```

---

## ЗАДАЧА 22: E2E тест

**Зависимости:** Задача 20

```
Создай E2E тест для проверки полного цикла работы:

- Файл: `tests/e2e_test.sh` (bash) или `tests/e2e_test.go`

Сценарий:
1. Запусти `docker-compose up -d`
2. Жди готовности API (цикл curl /health до успеха, таймаут 30 сек)
3. Жди появления identity в БД (воркер register должен создать, ~60 сек)
4. Запроси GET /config
5. Сохрани в файл /tmp/test_warp.conf
6. Проверь, что файл содержит обязательные поля:
   - [Interface]
   - PrivateKey = ...
   - Address = ...
   - [Peer]
   - Endpoint = ...
7. (Опционально) Попробуй поднять WireGuard-туннель в привилегированном контейнере
8. Teardown: `docker-compose down -v`

Проверка:
```bash
cd tests
./e2e_test.sh
```

Вернуть exit code 0 при успехе, 1 при ошибке.
```

---

## Порядок выполнения промптов

### Волна 1 (параллельно):
- Задача 1, 3, 6, 7, 9

### Волна 2:
- Задача 2, 4, 5, 10

### Волна 3:
- Задача 8, 11, 12

### Волна 4:
- Задача 13, 14, 15, 16, 17, 18

### Волна 5:
- Задача 19

### Волна 6:
- Задача 20

### Волна 7:
- Задача 21, 22

---

## Шаблон для передачи агенту

```
Ты — специализированный агент для выполнения изолированной задачи разработки.

Контекст: Создаём серверную часть для генерации WARP/AmneziaWG конфигураций на Go.

Твоя задача: [ВСТАВИТЬ ПРОМПТ ИЗ ЗАДАЧИ N]

Условие завершения: [ИЗ СЕКЦИИ "Проверка"]

Выходные файлы: [СПИСОК ФАЙЛОВ ИЗ ПРОМПТА]

Не выполняй другие задачи. Не создавай файлы, не упомянутые в задаче.
Когда закончишь, выведи чеклист выполненных шагов и команду для проверки.
```

---

**Примечание:** Каждый промпт — атомарная задача. Передавай их последовательно, проверяя условие завершения перед следующим. Граф зависимостей в начале файла `warp-server-plan.md`.

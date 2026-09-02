# План серверной части WARP-генератора: изолированные задачи

## Условия функционального объединения

### Интерфейсы взаимодействия между модулями:

1. **БД SQLite** (единое хранилище):
   - Таблица `identities`: id, private_key, public_key, client_id, token, addresses_v4, addresses_v6, peer_public_key, created_at
   - Таблица `endpoints`: id, host, port, rtt_ms, success_count, fail_count, last_seen, last_checked
   - Таблица `configs`: id, identity_id, endpoint_id, config_text, created_at

2. **HTTP API контракт** (внутренний для воркеров, внешний для клиентов):
   - `GET /api/identities` → JSON массив identity
   - `GET /api/endpoints?alive=true` → JSON живых endpoints
   - `POST /api/identities` → создать новую identity
   - `POST /api/endpoints` → добавить endpoint
   - `PATCH /api/endpoints/:id` → обновить метрики (rtt, success/fail)
   - `GET /config` → сгенерировать готовый AmneziaWG конфиг
   - `GET /health` → статус сервиса

3. **Форматы данных**:
   ```json
   Identity: {
     "id": "uuid",
     "private_key": "base64",
     "public_key": "base64",
     "client_id": "hex",
     "token": "string",
     "addresses_v4": ["10.2.0.x/32"],
     "addresses_v6": ["2606:4700:110::/128"],
     "peer_public_key": "base64",
     "created_at": "ISO8601"
   }
   
   Endpoint: {
     "id": "int",
     "host": "162.159.x.x",
     "port": 500,
     "rtt_ms": 45,
     "success_count": 10,
     "fail_count": 0,
     "last_seen": "ISO8601",
     "last_checked": "ISO8601"
   }
   ```

4. **Проверки совместимости**:
   - Identity без endpoint → скан должен найти рабочий узел
   - Endpoint без identity → регистрация должна создать identity
   - Каждый модуль должен работать, если БД существует и схема актуальна
   - API должен возвращать 503 если нет ни одной рабочей пары identity+endpoint

---

## Изолированные задачи

### Задача 1: Схема БД и миграции
**Вход:** нет  
**Выход:** `schema.sql`, `migrate.go` (или SQL-скрипт)  
**Условие завершения:** `sqlite3 warp.db < schema.sql` создаёт все таблицы с индексами

### Задача 2: HTTP API сервер (скелет)
**Вход:** `schema.sql` (из Задачи 1)  
**Выход:** `cmd/server/main.go`, `internal/api/handlers.go`, `internal/db/db.go`  
**Условие завершения:** `curl localhost:8080/health` → `{"status":"ok"}`, `go test ./internal/api` проходит

### Задача 3: Модуль генерации X25519 keypair
**Вход:** нет  
**Выход:** `internal/crypto/keypair.go` с функцией `GenerateX25519()` → (privateKey, publicKey []byte)  
**Условие завершения:** unit-тест создаёт пару ключей 32 байта каждый, приватный восстанавливает публичный

### Задача 4: Модуль WARP-регистрации
**Вход:** функция `GenerateX25519()` (из Задачи 3), схема `identities` (из Задачи 1)  
**Выход:** `internal/warp/register.go` с функцией `RegisterIdentity(ctx) (*Identity, error)`  
**Условие завершения:** 
- `POST https://api.cloudflareclient.com/v0a4471/reg` с корректными headers
- Парсинг JSON ответа (config.peers[0].endpoint, addresses, client_id, token)
- Возврат структуры Identity
- Unit-тест с mock HTTP (golden response)

### Задача 5: Модуль сохранения identity в БД
**Вход:** структура `Identity` (из Задачи 4), `internal/db/db.go` (из Задачи 2)  
**Выход:** `internal/db/identities.go` с функциями `InsertIdentity(*Identity)`, `GetIdentities()`, `GetIdentityByID(id)`  
**Условие завершения:** integration-тест вставляет identity и читает обратно, все поля совпадают

### Задача 6: Модуль префиксов WARP (WarpPrefixes)
**Вход:** нет  
**Выход:** `internal/warp/prefixes.go` с функцией `WarpPrefixes() []string` → массив CIDR (162.159.192.0/24, 188.114.96.0/24, ...)  
**Условие завершения:** возвращает >= 10 CIDR, unit-тест проверяет наличие известных префиксов

### Задача 7: Модуль портов WARP
**Вход:** нет  
**Выход:** `internal/warp/ports.go` с функцией `WarpPorts() []int` → [500, 854, 859, 864, ...]  
**Условие завершения:** возвращает ~55 портов, unit-тест проверяет наличие 500, 2408

### Задача 8: Модуль ipscanner (интеграция warp-plus или свой)
**Вход:** `WarpPrefixes()` (Задача 6), `WarpPorts()` (Задача 7)  
**Выход:** `internal/scanner/ipscanner.go` с функцией `ScanEndpoints(ctx, prefixes, ports, maxResults) ([]Endpoint, error)`  
**Условие завершения:** 
- Сканирует CIDR + порты, возвращает endpoints с rtt_ms
- Сортировка по RTT
- Integration-тест находит >= 1 живой endpoint (реальный скан с таймаутом)

### Задача 9: Модуль neighbor expansion
**Вход:** `Endpoint` (базовый IP)  
**Выход:** `internal/scanner/neighbors.go` с функцией `ExpandNeighbors(baseIP string, range int) []string` → [baseIP±1..range]  
**Условие завершения:** для 162.159.192.5 с range=3 возвращает [162.159.192.2, .3, .4, .5, .6, .7, .8], unit-тест

### Задача 10: Модуль WireGuard handshake probe
**Вход:** `Endpoint` (host, port), `Identity` (keys)  
**Выход:** `internal/scanner/wg_probe.go` с функцией `ProbeEndpoint(ctx, endpoint, identity) (rttMs int, err error)`  
**Условие завершения:** отправляет WireGuard handshake initiation, получает response, измеряет RTT, unit-тест с mock

### Задача 11: Модуль сохранения endpoints в БД
**Вход:** массив `Endpoint` (из Задачи 8), `internal/db/db.go` (из Задачи 2)  
**Выход:** `internal/db/endpoints.go` с функциями `UpsertEndpoint(*Endpoint)`, `GetAliveEndpoints(minSuccessRate)`, `UpdateEndpointMetrics(id, rtt, success)`  
**Условие завершения:** integration-тест upsert'ит endpoint, обновляет метрики, фильтрует по success_rate

### Задача 12: Генератор AmneziaWG конфига
**Вход:** `Identity`, `Endpoint`, константы обфускации (Jc, Jmin, Jmax, H1-H4, I1)  
**Выход:** `internal/warp/config.go` с функцией `GenerateAmneziaConfig(identity, endpoint) (string, error)` → текст .conf  
**Условие завершения:** 
- Генерирует валидный `[Interface]` + `[Peer]` блок
- Unit-тест проверяет наличие всех полей (PrivateKey, Address, Jc=4, Endpoint=host:port)
- Golden-тест с фикстурой

### Задача 13: API эндпоинт GET /config
**Вход:** `GetIdentities()` (Задача 5), `GetAliveEndpoints()` (Задача 11), `GenerateAmneziaConfig()` (Задача 12)  
**Выход:** handler в `internal/api/config.go`  
**Условие завершения:** 
- `curl /config` возвращает .conf текст с заголовком `Content-Type: text/plain`
- Если нет рабочих пар → 503 Service Unavailable
- Integration-тест с preseeded БД

### Задача 14: API эндпоинты CRUD для identities
**Вход:** `internal/db/identities.go` (Задача 5)  
**Выход:** handlers `GET /api/identities`, `POST /api/identities`, `GET /api/identities/:id`  
**Условие завершения:** API-тесты проходят для всех эндпоинтов, JSON schema валидация

### Задача 15: API эндпоинты CRUD для endpoints
**Вход:** `internal/db/endpoints.go` (Задача 11)  
**Выход:** handlers `GET /api/endpoints`, `POST /api/endpoints`, `PATCH /api/endpoints/:id`  
**Условие завершения:** API-тесты проходят, фильтрация по ?alive=true работает

### Задача 16: Воркер регистрации identity (пул)
**Вход:** `RegisterIdentity()` (Задача 4), `InsertIdentity()` (Задача 5)  
**Выход:** `cmd/worker/register.go` с функцией `MaintainIdentityPool(ctx, targetCount, interval)`  
**Условие завершения:** 
- Проверяет текущее количество identity в БД
- Если < targetCount → регистрирует новые с интервалом (rate-limit защита от 429)
- Логирует ошибки регистрации
- Integration-тест запускает воркер, ждёт N identity

### Задача 17: Воркер сканирования endpoints
**Вход:** `ScanEndpoints()` (Задача 8), `UpsertEndpoint()` (Задача 11)  
**Выход:** `cmd/worker/scanner.go` с функцией `ScanPeriodically(ctx, interval)`  
**Условие завершения:** 
- Запускает скан каждые N минут
- Найденные endpoints сохраняет в БД
- Integration-тест запускает воркер, ждёт появления endpoints в БД

### Задача 18: Воркер пробинга (health check)
**Вход:** `GetAliveEndpoints()` (Задача 11), `ProbeEndpoint()` (Задача 10), `UpdateEndpointMetrics()` (Задача 11)  
**Выход:** `cmd/worker/prober.go` с функцией `ProbeEndpointsPeriodically(ctx, interval)`  
**Условие завершения:** 
- Берёт все живые endpoints из БД
- Пробует каждый с одной identity
- Обновляет метрики (rtt, success/fail count)
- Удаляет endpoints с fail_count > threshold
- Integration-тест запускает пробинг, проверяет обновление метрик

### Задача 19: Dockerfile и entrypoint
**Вход:** `cmd/server/main.go`, `cmd/worker/*.go`  
**Выход:** `Dockerfile`, `docker-entrypoint.sh`  
**Условие завершения:** 
- `docker build -t warp-server .` собирается
- `docker run warp-server server` запускает HTTP сервер
- `docker run warp-server worker register` запускает воркер регистрации
- Healthcheck работает

### Задача 20: docker-compose для локального деплоя
**Вход:** `Dockerfile` (Задача 19)  
**Выход:** `docker-compose.yml` с сервисами `api`, `worker-register`, `worker-scanner`, `worker-prober`  
**Условие завершения:** 
- `docker-compose up` поднимает все сервисы
- Общая БД `warp.db` на volume
- `curl localhost:8080/health` отвечает
- Через 5 минут в БД появляются identity и endpoints

### Задача 21: README с инструкциями
**Вход:** вся кодовая база  
**Выход:** `README.md` с разделами: архитектура, установка, запуск, API документация, примеры  
**Условие завершения:** человек может склонировать репозиторий и запустить сервис по README

### Задача 22: E2E тест
**Вход:** docker-compose (Задача 20)  
**Выход:** `tests/e2e_test.sh` или `tests/e2e_test.go`  
**Условие завершения:** 
- Запускает docker-compose
- Ждёт готовности API (/health)
- Запрашивает GET /config
- Сохраняет конфиг в файл
- Пытается поднять WireGuard туннель (в контейнере с правами)
- Проверяет связность (curl через туннель)
- Teardown

---

## Граф зависимостей задач

```
1 (schema) ─┬─→ 2 (HTTP скелет) ─┬─→ 13 (GET /config)
            │                      ├─→ 14 (API identities)
            │                      └─→ 15 (API endpoints)
            │
            ├─→ 5 (DB identities) ─┬─→ 13
            │                       ├─→ 14
            │                       └─→ 16 (worker register)
            │
            └─→ 11 (DB endpoints) ─┬─→ 13
                                    ├─→ 15
                                    ├─→ 17 (worker scanner)
                                    └─→ 18 (worker prober)

3 (X25519) ──→ 4 (WARP register) ─→ 16

6 (prefixes) ─┬─→ 8 (ipscanner) ──→ 17
7 (ports) ────┘

9 (neighbors) ─→ 17 (опционально, расширение после базового скана)

10 (WG probe) ─→ 18

4 + 5 + 11 + 12 ──→ 13 (собирается всё в /config endpoint)

12 (config gen) ─→ 13

2 + 13 + 14 + 15 ──→ 19 (Dockerfile для server)
16 + 17 + 18 ──────→ 19 (Dockerfile для workers)

19 ──→ 20 (docker-compose)

20 ──→ 21 (README)
     └──→ 22 (E2E тест)
```

---

## Порядок выполнения (топологическая сортировка)

**Волна 1** (параллельно):
- Задача 1 (schema)
- Задача 3 (X25519)
- Задача 6 (prefixes)
- Задача 7 (ports)
- Задача 9 (neighbors)

**Волна 2** (после Волны 1):
- Задача 2 (HTTP скелет) ← требует 1
- Задача 4 (WARP register) ← требует 3
- Задача 5 (DB identities) ← требует 1
- Задача 10 (WG probe) ← независимая

**Волна 3** (после Волны 2):
- Задача 8 (ipscanner) ← требует 6, 7
- Задача 11 (DB endpoints) ← требует 1, 2
- Задача 12 (config gen) ← независимая (можно в Волне 1)

**Волна 4** (после Волны 3):
- Задача 13 (GET /config) ← требует 2, 5, 11, 12
- Задача 14 (API identities) ← требует 2, 5
- Задача 15 (API endpoints) ← требует 2, 11
- Задача 16 (worker register) ← требует 4, 5
- Задача 17 (worker scanner) ← требует 8, 11
- Задача 18 (worker prober) ← требует 10, 11

**Волна 5** (после Волны 4):
- Задача 19 (Dockerfile)

**Волна 6** (после Волны 5):
- Задача 20 (docker-compose)

**Волна 7** (после Волны 6):
- Задача 21 (README)
- Задача 22 (E2E тест)


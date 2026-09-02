# WARP Server

Автономный сервер для генерации WARP/AmneziaWG конфигураций.

## Структура проекта

```
warp-server/
├── cmd/
│   ├── server/        # HTTP API сервер
│   └── worker/        # Фоновые воркеры
├── internal/
│   ├── api/           # HTTP handlers
│   ├── crypto/        # X25519 keypair генерация
│   ├── db/            # SQLite операции
│   ├── scanner/       # IP scanner, WG probe
│   └── warp/          # WARP регистрация, конфиг генерация
├── tests/             # E2E тесты
├── docs/              # Документация
├── schema.sql         # Схема БД
├── warp-server-plan.md    # Детальный план задач
├── tasks-prompts.md       # Промпты для агентов
└── tasks-bash.sh          # Bash-скрипт выполнения
```

## Быстрый старт

```bash
docker-compose up -d
curl http://localhost:8080/config > warp.conf
```

## API

- `GET /health` - статус сервиса
- `GET /config` - получить AmneziaWG конфиг
- `GET /api/identities` - список identity
- `GET /api/endpoints?alive=true` - живые endpoints

## Разработка

См. `warp-server-plan.md` для детального плана задач и `tasks-prompts.md` для промптов агентам.

## Лицензия

MIT

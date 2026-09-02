# WARP Server

Автономный сервер для генерации WARP/AmneziaWG конфигураций.

## Быстрый старт

```bash
docker-compose up -d
curl http://localhost:8080/config > warp.conf
```

## API

- `GET /health` - статус сервиса
- `GET /config` - получить готовый AmneziaWG конфиг
- `GET /api/identities` - список зарегистрированных identity
- `GET /api/endpoints?alive=true` - живые endpoints

## Архитектура

- **API Server** - HTTP API на порту 8080
- **Worker Register** - регистрация WARP identity (пул 10 шт)
- **Worker Scanner** - сканирование WARP endpoints каждые 10 минут
- **Worker Prober** - проверка живых endpoints каждые 2 минуты

## Разработка

```bash
go test ./...
go build ./cmd/server
./server
```

# Distributed URL Shortener

Сервис сокращения ссылок с аналитикой кликов в реальном времени.

## Стек

`Go 1.22` `PostgreSQL 16` `Redis 7` `Kafka` `ClickHouse` `gRPC` `chi` `Docker`

> **Текущая версия:** MVP — базовые операции с URL (создание, редирект, удаление).  
> В разработке: Kafka + ClickHouse аналитика, gRPC, JWT аутентификация, Prometheus метрики.

---

## Быстрый старт

### Локальная разработка

```bash
# 1. Склонировать репозиторий
git clone https://github.com/ashcloud/url-shortener
cd url-shortener

# 2. Скопировать конфиг
cp .env.example .env

# 3. Поднять PostgreSQL + Redis
make docker-dev

# 4. Применить миграции и запустить сервер
make migrate-up
make run
```

### Docker Compose (всё вместе)

```bash
cp .env.example .env
docker compose -f deployments/docker-compose.yml up -d
```

---

## API

### Создать короткую ссылку

```bash
curl -X POST http://localhost:8080/api/urls \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/very/long/path"}'
```

```json
{
	"short_code": "aB3xKp7",
	"short_url": "http://localhost:8080/aB3xKp7",
	"original_url": "https://example.com/very/long/path",
	"created_at": "2024-01-15T12:00:00Z"
}
```

### Редирект

```
GET http://localhost:8080/aB3xKp7
→ 301 Redirect → https://example.com/very/long/path
```

### Список ссылок пользователя

```bash
curl http://localhost:8080/api/urls?user_id=1
```

### Удалить ссылку

```bash
curl -X DELETE http://localhost:8080/api/urls/aB3xKp7?user_id=1
```

### Health check

```bash
curl http://localhost:8080/api/health
# {"status":"ok"}
```

---

## Архитектура

```
HTTP Layer (chi router)
       │
       ├── GET  /{code}         → Redirect
       ├── POST /api/urls       → Create
       ├── GET  /api/urls       → List
       └── DELETE /api/urls/{code} → Delete
               │
         Service Layer
         (url_service.go)
               │
       ┌───────┴────────┐
       │                │
  PostgreSQL          Redis
  (urls table)     (short_code → original_url, TTL 24h)
```

**Логика резолвинга (Read path):**

1. Проверить Redis кэш → если есть, редиректим + инкрементим счётчик в фоне
2. Если кэш промах → PostgreSQL → обновить кэш → редирект

---

## Структура проекта

```
├── cmd/api/          — точка входа HTTP сервера
├── internal/
│   ├── config/       — конфигурация через env
│   ├── domain/       — модели и ошибки
│   ├── service/      — бизнес-логика
│   ├── storage/
│   │   ├── postgres/ — репозиторий URL
│   │   └── redis/    — кэш
│   └── handler/http/ — HTTP handlers (chi)
├── pkg/
│   ├── hasher/       — генерация short code (base62, crypto/rand)
│   └── logger/       — slog обёртка
└── migrations/       — SQL миграции (golang-migrate)
```

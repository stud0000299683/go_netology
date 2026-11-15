# Ledger

Сервис хранит бюджеты и расходы в PostgreSQL. Сводки по категориям кешируются в Redis на 30 секунд.

## База и миграции

Нужны PostgreSQL и Redis. Локально их можно поднять так:

```powershell
docker run -d --name cashapp-postgres -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=cashapp -p 5432:5432 postgres:16-alpine
docker run -d --name cashcraft-redis -p 6379:6379 redis:7-alpine
```

По умолчанию Ledger подключается к `localhost:5432`, пользователь `postgres`, пароль `postgres`, база `cashapp`, `sslmode=disable`. Своя строка задаётся через `DATABASE_URL`.

Из корня репозитория:

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
goose -dir ./homework7/ledger/migrations postgres $env:DATABASE_URL up
```

Откат последней миграции:

```powershell
goose -dir ./homework7/ledger/migrations postgres $env:DATABASE_URL down
```

## Запуск

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
$env:REDIS_ADDR = "localhost:6379"
$env:REDIS_DB = "0"
cd homework7/ledger
go run ./cmd/ledger
```

`REDIS_PASSWORD` нужен только если у Redis задан пароль. При успешном старте в журнале появляются сообщения о подключении к PostgreSQL и Redis. Если база или Redis недоступны, процесс завершается с ошибкой.

Миграции применяются до первого запуска и после каждого изменения схемы.

# Ledger

Учёт разделён на слои. Gateway получает готовый сервис и не видит SQL.

```text
internal/domain            сущности, валидация и интерфейсы репозиториев
internal/repository/pg     PostgreSQL: upsert бюджета, выборки, сумма, insert расхода
internal/service           лимит, отчёт, пакетный импорт на каналах и кеш
internal/app               чтение окружения, пул базы, Redis, сборка сервиса
internal/db                строка подключения и sql.Open
internal/cache             клиент Redis
```

`ledger.New` вызывает фабрику `internal/app`. Она возвращает интерфейс сервиса, функцию закрытия соединений и ошибку. Миграции фабрика не применяет.

## База и миграции

```powershell
docker run -d --name cashapp-postgres -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=cashapp -p 5432:5432 postgres:16-alpine
docker run -d --name cashcraft-redis -p 6379:6379 redis:7-alpine
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
goose -dir ./homework11/ledger/migrations postgres $env:DATABASE_URL up
```

Откат: `goose -dir ./homework11/ledger/migrations postgres $env:DATABASE_URL down`.

Если `DATABASE_URL` нет, адрес собирается из `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS` и `DB_NAME`. По умолчанию база `cashapp` на `localhost:5432`, пользователь и пароль `postgres`. Redis: `REDIS_ADDR` (`localhost:6379`), `REDIS_DB` (`0`), необязательный `REDIS_PASSWORD`.

## Запуск

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
$env:REDIS_ADDR = "localhost:6379"
cd homework11/ledger
go run ./cmd/ledger
```

При успешном старте процесс пишет, что подключился к PostgreSQL и Redis. Если соединение не удалось, он завершается с ошибкой.

Образ Ledger тоже двухэтапный, но в compose он не запускается: Gateway встраивает Ledger как библиотеку.

```powershell
docker build -f homework11/ledger/Dockerfile -t cashapp-ledger homework11/ledger
```

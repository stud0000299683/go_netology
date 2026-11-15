# go_netology

Homework 7: Ledger хранит бюджеты и расходы в PostgreSQL, а сводку по категориям кеширует в Redis. Gateway отдаёт прежние HTTP-контракты и отчёт. Задания 2–6 лежат в папках `homework2`–`homework6`.

Нужен Go 1.26 или новее и утилита [Goose](https://github.com/pressly/goose).

## База данных и Redis

```powershell
docker run -d --name cashapp-postgres -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=cashapp -p 5432:5432 postgres:16-alpine
docker run -d --name cashcraft-redis -p 6379:6379 redis:7-alpine
```

Если `DATABASE_URL` не задана, Ledger собирает адрес из `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS` и `DB_NAME`. По умолчанию это `localhost:5432`, пользователь `postgres`, пароль `postgres`, база `cashapp`, `sslmode=disable`. Redis: `REDIS_ADDR` (`localhost:6379`), `REDIS_DB` (`0`), необязательный `REDIS_PASSWORD`.

Применить миграции из корня репозитория:

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
goose -dir ./homework7/ledger/migrations postgres $env:DATABASE_URL up
```

Откатить:

```powershell
goose -dir ./homework7/ledger/migrations postgres $env:DATABASE_URL down
```

Схема применяется до первого запуска Ledger и после каждого изменения таблиц.

## Запуск

В обоих терминалах должны быть заданы `DATABASE_URL` и `REDIS_ADDR`.

```powershell
cd homework7/ledger
go run ./cmd/ledger
```

```powershell
cd homework7/gateway
go run .
```

Ledger при старте пишет, что подключился к PostgreSQL и Redis, и завершается, если соединение не удалось. Gateway слушает `http://localhost:8080`.

## Эндпоинты

- `POST /api/budgets` — создать или обновить бюджет, ответ `201`
- `GET /api/budgets` — список бюджетов, ответ `200` или `[]`
- `POST /api/transactions` — создать расход, ответ `201`; неверные данные — `400`; превышение лимита — `409` и `{"error":"budget exceeded"}`
- `GET /api/transactions` — список расходов, ответ `200` или `[]`
- `GET /api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` — суммы по категориям за интервал

Ответы идут с заголовком `Content-Type: application/json; charset=utf-8`.

## Примеры

```powershell
curl -X POST http://localhost:8080/api/budgets -H "Content-Type: application/json" -d "{\"category\":\"еда\",\"limit\":5000}"
curl -X POST http://localhost:8080/api/transactions -H "Content-Type: application/json" -d "{\"amount\":450,\"category\":\"еда\",\"description\":\"ланч\",\"date\":\"2025-09-10\"}"
curl http://localhost:8080/api/budgets
curl http://localhost:8080/api/transactions
curl "http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30"
curl "http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30"
```

Первый запрос отчёта считает данные в PostgreSQL и кладёт JSON в Redis на 30 секунд. Второй запрос с теми же датами читает кеш. После перезапуска Ledger записи остаются в базе.

## Тесты

Проверки полей не требуют базы. Сценарии с записью пропускаются, если PostgreSQL или Redis недоступны.

```powershell
cd homework7
go test -p 1 ./gateway/... ./ledger/...
```

Флаг `-p 1` нужен, потому что сценарии с записью используют одну и ту же базу и не должны идти одновременно.

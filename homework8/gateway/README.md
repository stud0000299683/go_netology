# Gateway

HTTP-слой на порту `8080`. При старте `main` вызывает `ledger.New`: фабрика Ledger сама открывает PostgreSQL и Redis и возвращает интерфейс сервиса и функцию закрытия. Обработчики получают этот сервис в конструкторе `api.New` и вызывают его методы. Пакеты `database/sql`, `pgx` и `repository` Gateway не импортирует.

Контракты прежние: `POST/GET /api/budgets`, `POST/GET /api/transactions`, `GET /api/reports/summary`. Ошибки по-прежнему `400`, `409 {"error":"budget exceeded"}` и `500`.

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
$env:REDIS_ADDR = "localhost:6379"
cd homework8/gateway
go run .
```

Проверка:

```powershell
curl -X POST http://localhost:8080/api/budgets -H "Content-Type: application/json" -d "{\"category\":\"еда\",\"limit\":5000}"
curl -X POST http://localhost:8080/api/transactions -H "Content-Type: application/json" -d "{\"amount\":450,\"category\":\"еда\",\"description\":\"ланч\",\"date\":\"2025-09-10\"}"
curl http://localhost:8080/api/budgets
curl http://localhost:8080/api/transactions
curl "http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30"
curl "http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30"
```

Первый запрос отчёта считается по базе, второй в течение 30 секунд читается из Redis.

# Gateway

HTTP-слой на порту `8080`. При старте `main` вызывает `ledger.New`: фабрика Ledger сама открывает PostgreSQL и Redis и возвращает интерфейс сервиса и функцию закрытия. Обработчики получают этот сервис в конструкторе `api.New` и вызывают его методы. Пакеты `database/sql`, `pgx` и `repository` Gateway не импортирует.

Контракты бюджетов и одиночных транзакций прежние. `GET /api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` возвращает объект сумм, например `{"еда":450}`. `POST /api/transactions/bulk?workers=4` принимает массив транзакций и возвращает `200` со сводкой `{ "accepted": 1, "rejected": 1, "errors": [ { "index": 1, "error": "..." } ] }`. Без `workers` берётся 4. Ошибки: `400` для неверных дат и неверного `workers`, `409 {"error":"budget exceeded"}` для лимита одиночной транзакции, `500` для сбоя и `504 {"error":"request timeout"}`, если запрос не уложился в таймаут. В пакетном импорте превышение лимита попадает в `errors`, а ответ остаётся `200`.

Таймаут по умолчанию — 2 секунды, middleware стоит на всех маршрутах. Переменная `REQUEST_TIMEOUT` меняет его, например `$env:REQUEST_TIMEOUT="50ms"`.

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
$env:REDIS_ADDR = "localhost:6379"
cd homework10/gateway
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
curl -X POST "http://localhost:8080/api/transactions/bulk?workers=4" -H "Content-Type: application/json" -d "[{\"amount\":450,\"category\":\"еда\",\"description\":\"ланч\",\"date\":\"2025-09-10\"},{\"amount\":0,\"category\":\"еда\",\"description\":\"пусто\",\"date\":\"2025-09-10\"}]"
```

Первый запрос отчёта считается по базе, второй в течение 30 секунд читается из Redis.

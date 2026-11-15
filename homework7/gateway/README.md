# Gateway

HTTP-шлюз на порту `8080`. Контракты бюджетов и транзакций прежние. Данные читает и пишет пакет Ledger: PostgreSQL для хранения, Redis для короткого кеша отчётов и списка бюджетов.

Перед запуском примените миграции и поднимите PostgreSQL и Redis. Переменные те же, что у Ledger: `DATABASE_URL` или `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAME`, а также `REDIS_ADDR`, `REDIS_DB`, `REDIS_PASSWORD`.

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
$env:REDIS_ADDR = "localhost:6379"
cd homework7/gateway
go run .
```

## Проверка

Создать бюджет:

```powershell
curl -X POST http://localhost:8080/api/budgets -H "Content-Type: application/json" -d "{\"category\":\"еда\",\"limit\":5000}"
```

Транзакция в пределах лимита:

```powershell
curl -X POST http://localhost:8080/api/transactions -H "Content-Type: application/json" -d "{\"amount\":450,\"category\":\"еда\",\"description\":\"ланч\",\"date\":\"2025-09-10\"}"
```

Транзакция сверх лимита возвращает `409` и `{"error":"budget exceeded"}`.

Списки:

```powershell
curl http://localhost:8080/api/budgets
curl http://localhost:8080/api/transactions
```

Отчёт. Первый запрос считает сумму в базе, второй в течение 30 секунд читает Redis. В журнале Gateway это видно как «промах» и затем «попадание».

```powershell
curl "http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30"
curl "http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30"
```

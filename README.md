# go_netology

Homework 10: пакетный импорт расходов идёт через каналы и пул воркеров. Сводка по категориям по-прежнему считается параллельно и прерывается по таймауту запроса. Задания 2–9 лежат в папках `homework2`–`homework9`.

Нужен Go 1.26 или новее и утилита [Goose](https://github.com/pressly/goose).

## Слои Ledger

- `homework10/ledger/internal/domain` — транзакция, бюджет, валидация и интерфейсы репозиториев
- `homework10/ledger/internal/repository/pg` — SQL для PostgreSQL
- `homework10/ledger/internal/service` — лимит категории, параллельный отчёт, пакетный импорт и кеш Redis
- `homework10/ledger/internal/app` — чтение окружения, подключение к базе и Redis, сборка сервиса

`ledger.New(ctx)` возвращает сервис, функцию закрытия соединений и ошибку. Миграции эта функция не запускает. В Gateway `main` вызывает её и передаёт сервис в `api.New`.

## База данных и Redis

```powershell
docker run -d --name cashapp-postgres -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=cashapp -p 5432:5432 postgres:16-alpine
docker run -d --name cashcraft-redis -p 6379:6379 redis:7-alpine
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
goose -dir ./homework10/ledger/migrations postgres $env:DATABASE_URL up
```

Откат:

```powershell
goose -dir ./homework10/ledger/migrations postgres $env:DATABASE_URL down
```

Если `DATABASE_URL` не задана, адрес собирается из `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS` и `DB_NAME`. По умолчанию это `localhost:5432`, пользователь `postgres`, пароль `postgres`, база `cashapp`. Redis: `REDIS_ADDR` (`localhost:6379`), `REDIS_DB` (`0`), необязательный `REDIS_PASSWORD`.

## Запуск

```powershell
cd homework10/ledger
go run ./cmd/ledger
```

```powershell
cd homework10/gateway
go run .
```

В обоих процессах должны быть заданы `DATABASE_URL` и `REDIS_ADDR`. При ошибке подключения процесс пишет причину и завершается. Gateway слушает `http://localhost:8080`.

## Эндпоинты

- `POST /api/budgets` — создать или обновить бюджет, ответ `201`
- `GET /api/budgets` — список бюджетов, ответ `200` или `[]`
- `POST /api/transactions` — создать расход, ответ `201`; неверные данные — `400`; превышение лимита — `409` и `{"error":"budget exceeded"}`
- `GET /api/transactions` — список расходов, ответ `200` или `[]`
- `GET /api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` — суммы по категориям за интервал, объект `{ "еда": 450 }`
- `POST /api/transactions/bulk?workers=4` — пакетный импорт массива транзакций, ответ `200` со сводкой даже если часть элементов отклонена

Ответы идут с заголовком `Content-Type: application/json; charset=utf-8`.

## Примеры

```powershell
curl -X POST http://localhost:8080/api/budgets -H "Content-Type: application/json" -d "{\"category\":\"еда\",\"limit\":5000}"
curl -X POST http://localhost:8080/api/transactions -H "Content-Type: application/json" -d "{\"amount\":450,\"category\":\"еда\",\"description\":\"ланч\",\"date\":\"2025-09-10\"}"
curl http://localhost:8080/api/budgets
curl http://localhost:8080/api/transactions
curl "http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30"
curl "http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30"
curl -X POST "http://localhost:8080/api/transactions/bulk?workers=4" -H "Content-Type: application/json" -d "[{\"amount\":450,\"category\":\"еда\",\"description\":\"ланч\",\"date\":\"2025-09-10\"},{\"amount\":0,\"category\":\"еда\",\"description\":\"пусто\",\"date\":\"2025-09-10\"}]"
```

Пример ответа:

```json
{"еда":450}
```

Для каждой категории сумма считается в отдельной горутине. Пока расчёт идёт, раз в 400 мс в журнал пишется, что он ещё выполняется. Первый запрос кладёт JSON в Redis на 30 секунд, второй с теми же датами читает кеш.

Некорректные `from` или `to` дают `400`. Если даты перепутаны местами, тоже `400` и `{"error":"to is before from"}`.

## Пакетный импорт

`POST /api/transactions/bulk` принимает JSON-массив в том же формате, что и одна транзакция. Параметр `workers` задаёт число воркеров. Если его нет, используется 4. Значение меньше 1 или не число даёт `400`. Сервис не поднимает больше 32 воркеров и не больше, чем элементов в пакете.

Ledger кладёт элементы в канал заданий. Воркеры читают его, проверяют поля и лимит категории и пишут результат в канал результатов. После последней задачи канал заданий закрывается. Когда воркеры заканчивают работу, закрывается канал результатов. Отдельная горутина собирает сводку: `accepted` и `rejected` увеличиваются атомарно, список ошибок пополняется только в этой горутине. Отклонённые элементы не превращают весь ответ в ошибку.

```json
{"accepted":1,"rejected":1,"errors":[{"index":1,"error":"transaction amount must be positive"}]}
```

Если запрос отменён или истёк таймаут, воркеры останавливаются по `ctx.Done()`, а клиент получает `504` и `{"error":"request timeout"}`.

## Таймауты запросов

На все маршруты надет middleware `WithTimeout`. Он создаёт `context.WithTimeout` и передаёт этот контекст в обработчик. По умолчанию предел — 2 секунды. Другое значение задаётся переменной `REQUEST_TIMEOUT` в формате Go, например `50ms` или `2s`.

Если запрос не успел завершиться, клиент получает `504` и `{"error":"request timeout"}`. Расчёт отчёта при этом останавливается по `ctx.Done()`, тикер heartbeat прекращается, горутины по категориям не остаются ждать.

## Тесты

Проверки валидации, правила бюджета, формата отчёта и таймаута не требуют запущенной базы.

```powershell
cd homework10
go test -race ./gateway/... ./ledger/...
```

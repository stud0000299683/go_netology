# go_netology

Homework 6: HTTP-шлюз вызывает функции модуля Ledger. Транзакции и бюджеты хранятся в памяти. Предыдущие задания лежат в папках `homework2`–`homework5`.

## Запуск

Нужен Go 1.22 или новее. Команды выполняются из корня репозитория.

```powershell
cd homework6/gateway
go run .
```

Сервер слушает `http://localhost:8080`. В `gateway/go.mod` модуль `ledger` подключён через `replace` на соседнюю папку `../ledger`.

Все ответы приходят с заголовком `Content-Type: application/json; charset=utf-8`. Ошибки имеют вид `{"error":"..."}`: `400` при неверных данных, `409` при превышении бюджета, `500` при внутренней ошибке.

## Эндпоинты

- `POST /api/transactions` — создать транзакцию, ответ `201`
- `GET /api/transactions` — список транзакций, ответ `200` или `[]`
- `POST /api/budgets` — создать или обновить бюджет, ответ `201`
- `GET /api/budgets` — список бюджетов, ответ `200` или `[]`

## Примеры

Создать бюджет:

```powershell
curl -X POST http://localhost:8080/api/budgets -H "Content-Type: application/json" -d "{\"category\":\"еда\",\"limit\":5000}"
```

Создать транзакцию:

```powershell
curl -X POST http://localhost:8080/api/transactions -H "Content-Type: application/json" -d "{\"amount\":450,\"category\":\"еда\",\"description\":\"ланч\",\"date\":\"2025-09-10\"}"
```

Список транзакций:

```powershell
curl http://localhost:8080/api/transactions
```

Список бюджетов:

```powershell
curl http://localhost:8080/api/budgets
```

## Тесты

Команды выполняются из папки `homework6`. Файл `go.work` объединяет модули `gateway` и `ledger`.

```powershell
cd homework6
go test ./gateway/... ./ledger/... -cover -coverprofile=cover.out
go tool cover -html=cover.out -o cover.html
```

Покрытие тестами ~90%. Пакет `ledger` покрыт полностью, HTTP-слой Gateway — примерно на 93%. `ledger.Reset()` очищает память перед каждым сценарием.

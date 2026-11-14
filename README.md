# go_netology

Homework 5: HTTP-шлюз вызывает функции модуля Ledger. Транзакции и бюджеты хранятся в памяти.

Задания 2–4 лежат в папках `homework2`, `homework3` и `homework4`.

## Запуск

Нужен Go 1.22 или новее. Команды выполняются из корня репозитория.

```powershell
cd homework5/gateway
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

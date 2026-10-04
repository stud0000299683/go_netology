# Gateway и Ledger

Два независимых Go-модуля в одном репозитории:

- `gateway` — HTTP-шлюз
- `ledger` — бизнес-логика финансовых транзакций (пока без сетевого сервера)

## Требования

- Go 1.22 или новее

## Gateway

HTTP-сервер на порту `8080`. `GET /ping` возвращает `pong` со статусом `200 OK`.

```powershell
cd gateway
go run .
```

Проверка из другого терминала:

```powershell
curl http://localhost:8080/ping
```

Ожидаемый ответ: `pong`.

## Ledger

Сервис печатает `Ledger service started`, добавляет три тестовые транзакции в память и выводит их список.

```powershell
cd ledger
go run .
```

Ожидаемый вывод содержит строку запуска и три транзакции с категориями `food`, `transport` и `utilities`.

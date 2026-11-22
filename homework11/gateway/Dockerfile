# Контекст сборки — каталог homework11: модуль ledger подключается через replace.
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY gateway/go.mod gateway/go.sum gateway/
COPY ledger/go.mod ledger/go.sum ledger/

WORKDIR /src/gateway
RUN go mod download

WORKDIR /src
COPY gateway gateway
COPY ledger ledger

WORKDIR /src/gateway
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/gateway .

FROM alpine:latest

WORKDIR /app
COPY --from=builder /out/gateway /app/gateway
EXPOSE 8080
CMD ["/app/gateway"]

FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/ledger ./cmd/ledger

FROM alpine:latest

WORKDIR /app
COPY --from=builder /out/ledger /app/ledger
CMD ["/app/ledger"]

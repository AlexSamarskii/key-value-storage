# === Сборка ===
FROM golang:1.24-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git make ca-certificates

# Рабочая директория
WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем статически скомпилированный бинарник
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -o /go/bin/keyvalue-server \
    ./cmd/server

# === Финальный образ ===
FROM alpine:latest

# Установка необходимых пакетов и создание пользователя
RUN apk --no-cache add ca-certificates && \
    adduser -D -u 1000 -g 1000 keyvalue && \
    mkdir -p /data /app && \
    chown -R keyvalue:keyvalue /data /app

# Копируем бинарник с правами пользователя
COPY --from=builder --chown=keyvalue:keyvalue /go/bin/keyvalue-server /usr/local/bin/keyvalue-server

# Устанавливаем рабочую директорию
WORKDIR /app

# Пользователь без прав root
USER keyvalue

# Переменные окружения
ENV PORT=6379
ENV AOF_FILENAME=/data/database.aof

# Экспорт порта Redis
EXPOSE 6379

# Точка входа и команда по умолчанию
ENTRYPOINT ["/usr/local/bin/keyvalue-server"]
CMD []
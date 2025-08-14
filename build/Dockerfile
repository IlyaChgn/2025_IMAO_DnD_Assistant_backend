# Этап сборки
FROM golang:1.23-alpine3.19 AS build

WORKDIR /app

COPY ../prod.env prod.env

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем бинарник
RUN go build -o main ./cmd/app/main.go

# Финальный образ
FROM alpine:edge AS prod

# Устанавливаем bash и нужные утилиты
RUN apk add --no-cache bash ca-certificates

WORKDIR /root

# Копируем бинарник
COPY --from=build /app/main .

# Копируем конфиги, если нужно
COPY --from=build /app/internal/pkg/config/config.yaml ./internal/pkg/config/config.yaml

COPY --from=build /app/prod.env ./prod.env

# Открываем нужный порт
EXPOSE 8085

# Устанавливаем shell по умолчанию
SHELL ["/bin/bash", "-c"]

# Точка входа
ENTRYPOINT ["./main", "-prod"]

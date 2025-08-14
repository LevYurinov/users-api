# Этап сборки
FROM golang:1.23-alpine AS build

WORKDIR /app

# Копируем go-модули и устанавливаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Сборка Go-приложения
RUN go build -o main .

# Финальный образ (минимальный)
FROM alpine:latest

WORKDIR /app

# Копируем бинарник из стадии сборки
COPY --from=build /app/main .

# Устанавливаем переменные окружения (опционально)
ENV PORT=8080

# Открываем порт (для Render не обязателен, но можно оставить)
EXPOSE 8080

# Запускаем приложение
CMD ["./main"]
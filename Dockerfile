# 1. Этап тестирования
FROM golang:1.23-alpine AS test

WORKDIR /app

# Копируем модули и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь код
COPY . .

# Запускаем тесты (если хоть один не пройдет — сборка остановится)
RUN go test ./...

# 2. Этап сборки боевого бинарника
FROM golang:1.23-alpine AS build

WORKDIR /app

# Копируем зависимости и исходники из первого этапа
COPY --from=test /go/pkg /go/pkg
COPY . .

# Сборка бинарника
RUN go build -o main .

# 3. Финальный минимальный образ для запуска
FROM alpine:latest

WORKDIR /app

# Копируем бинарь из предыдущего этапа
COPY --from=build /app/main .

# Запуск сервера
CMD ["./main"]
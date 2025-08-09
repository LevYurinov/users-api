# users-api

Простой CRUD-сервер на Go для управления пользователями.  
Проект использует PostgreSQL, Docker и Swagger для генерации документации.  

## Технологии

- **Go (Golang)** — язык программирования
- **PostgreSQL** — база данных
- **Docker / Docker Compose** — контейнеризация
- **Gorilla Mux** — маршрутизация
- **Swagger** — автогенерация документации
- **Testify** — тестирование
- **Zap** — логирование

---

## Как запустить локально

### 1. Клонировать репозиторий
```bash
git clone https://github.com/LevYurinov/users-api.git
cd users-api

2. Запустить через Docker
docker-compose up --build

3. Эндпоинты и документация
API будет доступно на:
http://localhost:8081

Swagger UI:
http://localhost:8081/swagger/index.html

Эндпоинты API:
Метод	Путь	Описание
GET	/users	Получить всех пользователей
GET	/users/{id}	Получить пользователя по ID
POST	/users	Создать нового пользователя
PUT	/users/{id}	Обновить пользователя
PATCH	/users/{id}	Частичное обновление
DELETE	/users/{id}	Удалить пользователя

Тестирование
Запуск всех тестов:
go test ./...

Swagger документация
Swagger UI доступен после запуска:
http://localhost:8081/swagger/index.html

Исходники:
docs/swagger.yaml
docs/swagger.json

4. Логирование
В проекте используется пакет `internal/logger` для создания и настройки логгера на базе `zap`.

## Использование

1. Конфигурация логгера берётся из переменных окружения:

- `APP_ENV` — окружение приложения (`dev` или `prod`)
- `LOG_LEVEL` — уровень логирования (`debug`, `info`, `warn`, `error`)
- `LOG_WITH_STACK` — нужно ли выводить стек ошибок (`true` или `false`)

2. В коде:

```go
cfg := config.LoadConfig()
log := logger.New(cfg.Logger)
defer log.Sync()

log.Info("Logger initialized")


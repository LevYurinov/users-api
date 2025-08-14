package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
	"time"
)

// переменные для создания токенов
var (
	JWTSecret       = []byte("super-secret-key")
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 1200 * time.Hour
)

// Config хранит настройки приложения, включая настройки базы данных и логгера
type Config struct { // единая точка загрузки
	PostgresDSN string       // Строка подключения к PostgreSQL
	Logger      LoggerConfig // Настройки логгера
}

// LoggerConfig хранит конфигурацию логгера: уровень, среду выполнения и вывод стека ошибок
type LoggerConfig struct {
	AppEnv       string // Окружение приложения: dev или prod
	LogLevel     string // Уровень логирования: debug, info, warn, error
	LogWithStack bool   // // Включение/отключение вывода стека ошибок: true или false
}

// LoadConfig загружает конфигурацию из переменных окружения.
// При отсутствии необходимых переменных завершает работу программы с ошибкой.
func LoadConfig() *Config {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(".env file not found")
	}

	inputPostgresDSN := os.Getenv("POSTGRES_DSN")
	if inputPostgresDSN == "" {
		log.Fatal("PostgresDSN is not set") // если оставить, приложение падает локально

		// TODO: переделать настройки ENV перед выкатыванием на сервер
	}

	inputAppEnv := os.Getenv("APP_ENV")
	if inputAppEnv != "dev" && inputAppEnv != "prod" {
		log.Fatalf("Invalid APP_ENV: %s (must be dev or prod)", inputAppEnv)

		// TODO: переделать настройки ENV перед выкатыванием на сервер
	}

	inputLogLevel := os.Getenv("LOG_LEVEL")
	if inputLogLevel != "debug" && inputLogLevel != "info" && inputLogLevel != "warn" && inputLogLevel != "error" {
		log.Fatalf("Invalid LOG_LEVEL: %s (must be debug or info or warn or error)", inputLogLevel)

		// TODO: переделать настройки ENV перед выкатыванием на сервер
	}

	inputLogWithStack := os.Getenv("LOG_WITH_STACK")
	if inputLogWithStack != "true" && inputLogWithStack != "false" {
		log.Fatalf("Invalid LOG_WITH_STACK: %s (must be true or false)", inputLogWithStack)

		// TODO: переделать настройки ENV перед выкатыванием на сервер

	}

	var boolLogWithStack bool
	if strings.ToLower(inputLogWithStack) == "true" { // принимаем любой регистр
		boolLogWithStack = true
	}

	cfg := Config{
		PostgresDSN: inputPostgresDSN,
		Logger: LoggerConfig{
			AppEnv:       inputAppEnv,
			LogLevel:     inputLogLevel,
			LogWithStack: boolLogWithStack,
		},
	}

	return &cfg
}

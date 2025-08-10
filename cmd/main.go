// @title           Users API
// @version         1.0
// @description     Документация API для управления пользователями (CRUD + авторизация).
// @termsOfService  http://swagger.io/terms/
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
// @host      localhost:8080
// @BasePath  /

package main

import (
	"go.uber.org/zap"
	"os"
	"pet/config"
	"pet/internal/database"
	"pet/internal/logger"
	"pet/internal/repository"
	"pet/internal/server"
	"runtime/debug"
)

func main() {
	// загружает настройки из конфиг-файла
	cfg := config.LoadConfig()

	// инициализирует новый логгер с помощью настроек из конфига LoggerConfig
	log := logger.New(cfg.Logger)
	defer log.Sync()

	// глобальный перехватчик паник
	defer func() {
		r := recover()
		if r != nil {
			log.Error(
				"panic recovered",
				zap.Any("error", r),
				zap.ByteString("stack", debug.Stack()), // для паник всегда логирую стек
			)
		}
	}()

	// Подключается к БД
	dbUsers, err := database.ConnectDB("app_db")
	if err != nil {
		log.Error(
			"cannot connect to DB",
			zap.Error(err),
			zap.String("db_name", "app_db"),
			zap.String("env", cfg.Logger.AppEnv),
			zap.String("component", "database"),
			zap.String("operation", "connect"),
		)
		os.Exit(1)
	}
	defer dbUsers.Close()

	repo := repository.NewUserRepository(dbUsers, log)
	server.StartServer(repo, log)

	// client.Run(errorsLogger)
}

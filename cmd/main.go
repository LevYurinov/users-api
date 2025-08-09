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
	"fmt"
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
			log.Error("panic recovered", zap.Any("error", r), zap.ByteString("stack", debug.Stack()))
		}
	}()

	dbUsers, err := database.ConnectDB("app_db")
	if err != nil {
		errorsLogger.Println("[DB] не удалось подключиться к БД:", err)
		fmt.Println("[DB] подробности ошибки:", err)
		os.Exit(1) // <--- добавь это, чтобы программа завершилась при ошибке подключения
	}
	defer dbUsers.Close()

	repo := repository.NewUserRepository(dbUsers, errorsLogger)

	server.StartServer(repo, errorsLogger)
	// client.Run(errorsLogger)

	fmt.Println("Сервер завершил работу") // <- если ты это видишь — значит сервер НЕ запустился
}

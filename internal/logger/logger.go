package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"pet/config"
)

// New создает новый экземпляр zap.Logger на основе переданных настроек логгера.
//
// В режиме "dev" логгирование идет в консоль в читаемом формате,
// при этом добавляется информация о месте вызова (файл, строка).
//
// В режиме "prod" логирование идет в JSON-формате, удобном для систем агрегирования логов.
//
// Уровень логирования задается через cfg.LogLevel (debug, info, warn, error).
//
// Опция LogWithStack управляет выводом стека вызовов для ошибок:
// true — стек выводится при ошибках уровня Error и выше,
// false — только для фатальных ошибок.
//
// Возвращает готовый к использованию *zap.Logger.
func New(cfg config.LoggerConfig) *zap.Logger {

	var encoderCfg zapcore.EncoderConfig
	if cfg.AppEnv == "dev" {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderCfg = zap.NewProductionEncoderConfig()
	}

	// задаю одинаковый формат времени для dev и prod
	encoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")

	var level zapcore.Level

	switch cfg.LogLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// опции для логгера
	var opts []zap.Option

	if cfg.AppEnv == "dev" {
		opts = append(opts, zap.AddCaller()) // добавляет в лог строку с файлом и номером, откуда был вызван лог
	}

	if cfg.LogWithStack {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel)) // печатает стек при логах с уровнем ≥ заданного
	} else {
		opts = append(opts, zap.AddStacktrace(zapcore.FatalLevel)) // только для фатальных
	}

	// сборка ядра логгера
	var encoder zapcore.Encoder
	if cfg.AppEnv == "dev" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg) // вывод лог-сообщения в формате строки
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg) // вывод в JSON
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	// куда писать логи: стандартный вывод в терминал + будет фильтровать сообщения ниже заданного уровня

	return zap.New(core, opts...) // оборачивает ядро и опции в один логгер-объект типа *zap. Logger
}

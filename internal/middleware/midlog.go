package middleware

import (
	"go.uber.org/zap"
	"net/http"
	"time"
)

// midLogResponseWriter — обёртка над http.ResponseWriter,
// которая перехватывает статус ответа для последующего логирования
type midLogResponseWriter struct {
	http.ResponseWriter // интерфейс ResponseWriter встроен в структуру
	statusCode          int
}

// WriteHeader сохраняет статус-код и вызывает оригинальный WriteHeader
func (lrw *midLogResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// MidLog возвращает middleware для логирования всех HTTP-запросов.
// Логирует метод, путь, статус-код и время выполнения.
// Уровень логирования зависит от кода ответа:
//   - Info: для 2xx–3xx
//   - Warn: для 4xx
//   - Error: для 5xx
func MidLog() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {

			start := time.Now()

			// Оборачиваем ResponseWriter до передачи в хендлер
			// Здесь создаётся указатель на loggingResponseWriter.
			// Мы явно передаём в него wr — оригинальный http.ResponseWriter, пришедший в хендлер.
			// statusCode: http.StatusOK — просто значение по умолчанию, на случай, если хендлер ничего не вызовет (это допустимо)
			lrw := &midLogResponseWriter{
				ResponseWriter: wr,
				statusCode:     http.StatusOK, // ← на случай, если WriteHeader вообще не вызовут
			}

			// Вызываем следующий хендлер, передаём ему нашу обёртку
			next.ServeHTTP(lrw, req)

			duration := time.Since(start)

			log := LoggerFromContext(req.Context())

			switch {
			case lrw.statusCode >= 500:
				log.Error("HTTP request completed",
					zap.Int("http.status_code", lrw.statusCode),
					zap.String("http.method", req.Method),
					zap.String("http.target", req.URL.Path),
					zap.Duration("duration", duration),
					zap.String("component", "middleware"),
					zap.String("operation", "response"),
				)
			case lrw.statusCode >= 400:
				log.Warn("HTTP request completed",
					zap.Int("http.status_code", lrw.statusCode),
					zap.String("http.method", req.Method),
					zap.String("http.target", req.URL.Path),
					zap.Duration("duration", duration),
					zap.String("component", "middleware"),
					zap.String("operation", "response"),
				)
			default:
				log.Info("HTTP request completed",
					zap.Int("http.status_code", lrw.statusCode),
					zap.String("http.method", req.Method),
					zap.String("http.target", req.URL.Path),
					zap.Duration("duration", duration),
					zap.String("component", "middleware"),
					zap.String("operation", "response"),
				)
			}
		})
	}
}

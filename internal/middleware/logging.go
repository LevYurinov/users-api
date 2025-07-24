package middleware

import (
	"log"
	"net/http"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter // интерфейс встроен (embedded) в структуру loggingResponseWriter
	statusCode          int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func Logging(logger *log.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {

			start := time.Now()

			// Оборачиваем ResponseWriter до передачи в хендлер
			// Здесь создаётся указатель на loggingResponseWriter.
			// Мы явно передаём в него wr — оригинальный http.ResponseWriter, пришедший в хендлер.
			// statusCode: http.StatusOK — просто значение по умолчанию, на случай, если хендлер ничего не вызовет (это допустимо)
			lrw := &loggingResponseWriter{
				ResponseWriter: wr,
				statusCode:     http.StatusOK, // ← на случай, если WriteHeader вообще не вызовут
			}

			// Вызываем следующий хендлер, передаём ему нашу обёртку
			next.ServeHTTP(lrw, req)

			duration := time.Since(start)
			// Логгируем завершение запроса с кодом и временем
			logger.Printf("Completed %s %s with %d in %v", req.Method, req.URL.Path, lrw.statusCode, duration)
		})
	}
}

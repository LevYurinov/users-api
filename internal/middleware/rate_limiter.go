package middleware

import (
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"sync"
	"time"
)

// лимит: 5 запросов в секунду, burst (максимальный "взрыв") — 10
const (
	reqPerSecond = 5
	burstSize    = 10
)

// client - структура для хранения лимитеров по IP
type client struct {
	limiter  *rate.Limiter // ограничитель запросов
	lastSeen time.Time     // время последнего запроса
}

// Эта карта (map) хранит лимитеры (rate.Limiter) по IP-адресу пользователя
//
// Карты не потокобезопасны в Go.
// Одновременная запись/чтение без синхронизации вызовет панику: concurrent map writes.
// sync.Mutex используется для блокировки доступа к карте, пока в нее пишут или читают

var clients = make(map[string]*client)
var mu sync.Mutex

// RateLimiterMiddleware — middleware для ограничения запросов
func RateLimiterMiddleware() func(http.Handler) http.Handler {
	// запускаем фоновую очистку старых IP
	go cleanupClients()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// r.RemoteAddr - это поле из http.Request, IP-адрес клиента + порт, с которого он обратился к серверу.
			// строка вида "IP:порт", например,"192.168.1.42:54321"
			// net.SplitHostPort разделяет эту строку на части: IP и порт:
			ip, _, err := net.SplitHostPort(r.RemoteAddr)

			if err != nil {
				// Если ошибка возникла (например, RemoteAddr был кривой), то мы возвращаем 500 с сообщением:
				http.Error(w, "Unable to parse IP", http.StatusInternalServerError)
				return
			}

			// регулирует, сколько запросов в секунду (или минуту) можно сделать с этого IP.
			limiter := getLimiter(ip)

			// Allow - основной метод, проверяющий, можно ли сейчас выполнить запрос, согласно лимиту?
			if !limiter.Allow() { // если лимит исчерпан на текущий момент, то
				w.Header().Set("Content-Type", "application/json") // заголовок, что ответ будет в JSON
				http.Error(w, `{"error": "Too Many Requests"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// очистка старых IP-лимитеров (например, раз в минуту), чтобы не держать их лимитеры в памяти
func cleanupClients() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, c := range clients {
			if time.Since(c.lastSeen) > 3*time.Minute {
				delete(clients, ip)
			}
		}
		mu.Unlock()
	}
}

// getLimiter возвращает лимитер для IP (создает, если нет)
func getLimiter(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	user, exists := clients[ip]
	if !exists { // если юзера нет в базе
		// создаем новый лимитер: requestsPerSecond - частота, burstSize - "всплеск"
		limiter := rate.NewLimiter(rate.Limit(reqPerSecond), burstSize)
		clients[ip] = &client{limiter, time.Now()}
		return limiter
	}

	// если юзер есть в базе, просто обновляем время последнего запроса
	user.lastSeen = time.Now()
	return user.limiter
}

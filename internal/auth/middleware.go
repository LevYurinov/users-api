package auth

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"pet/config"
	"strings"
)

// contextKey — уникальный собственный тип для ключа контекста,
// чтобы безопасно использовать контекст без риска "переписать" чужие значения
type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware — middleware для аутентификации по заголовку Authorization
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// Простейшая проверка на наличие токена в формате Bearer ...
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized: missing or invalid token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ") // удаляет Bearer из записи с токеном

		//	надо распарсить и проверить только access-токен
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			// Проверка, что используется правильный метод подписи
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("неожиданный метод подписи %v", token.Header["alg"])
			}
			return config.JWTSecret, nil
		})
		if err != nil || !token.Valid {
			log.Printf("[AUTH] ошибка разбора токена: %v", err)
			http.Error(w, "токен невалидный", http.StatusUnauthorized)
			return
		}

		sub, ok := claims["sub"]
		if !ok {
			log.Printf("[AUTH] ошибка в claims токена: %v", err)
			http.Error(w, "поле sub отсутствует", http.StatusUnauthorized)
			return
		}

		userIDFloat, ok := sub.(float64)
		if !ok {
			http.Error(w, "поле sub имеет неверный тип", http.StatusUnauthorized)
			return
		}

		role, _ := claims["role"]

		// Добавляем userID в context
		ctx := context.WithValue(r.Context(), userIDKey, int(userIDFloat))
		ctx = context.WithValue(ctx, "role", role)

		// Передаём дальше с новым контекстом
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext — извлекает userID из context.Context
func GetUserIDFromContext(r *http.Request) (int, bool) {
	val := r.Context().Value(userIDKey)
	userID, ok := val.(int)
	return userID, ok
}

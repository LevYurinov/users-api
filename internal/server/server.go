package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"pet/config"
	"pet/internal/auth"
	"pet/internal/middleware"
	"pet/internal/model"
	"pet/internal/repository"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	_ "pet/docs" // важно для инициализации swagger-доков
)

// errorsLogger - отдельный логгер для тестов из пакета test
var errorsLogger *log.Logger

// InitLogger - отдельная функция для логирования тестов с помощью стандартного пакета
func InitLogger(l *log.Logger) {
	errorsLogger = l
}

var validate *validator.Validate

// InitValidator инициирует валидатор для проверки входных данных
// в http-запросах внутри ручкех (хендлеров), например логина и пароля
func InitValidator() {
	validate = validator.New()
}

// StartServer - главная функция по запуску сервера, передается в main
// @Summary Запускает сервер, настраивает роутер и хендлеры
// @Description Производит запуск сервера на localhost:8080.
// Запускает и настраивает роутер для страниц, где происходят CRUD-операции с пользователями и БД.
func StartServer(repo *repository.UserRepository, log *zap.Logger) {
	InitValidator()

	var handler http.Handler = SetupRoutes(repo) // явно указываю тип

	handler = middleware.WithLogger(log)(handler)         // кладем логгер в контекст для исп. в ручках
	handler = middleware.Recoverer()(handler)             // сначала обработка panic()
	handler = middleware.MidLog()(handler)                // логирует метод, путь, статус-код и время выполнения запроса
	handler = middleware.Cors()(handler)                  // обработка предзапросов браузера
	handler = middleware.DefaultHeaders()(handler)        // системные заголовки по умолчанию
	handler = middleware.RateLimiterMiddleware()(handler) // добавил лимит запросов 5-10 в секунду
	//handler = auth.AuthMiddleware(handler)              // проверка токена и передачи ID через контекст

	log.Info("Starting HTTP-server on :8080")

	err := runServer(handler) // запуск сервера с проверкой
	if err != nil {
		// Если сервер остановился с ошибкой, отличной от http.ErrServerClosed — это реально ошибка
		if errors.Is(err, http.ErrServerClosed) {
			// graceful shutdown — нормальное завершение, логируем как info
			log.Info("Server stopped gracefully")
		} else {
			// неожиданная ошибка — логируем с ошибкой и стеком
			log.Error(
				"Server stopped with error",
				zap.Error(err),
				zap.ByteString("stack", debug.Stack()),
			)
		}
		return
	}

	log.Info("Server stopped gracefully")
}

// SetupRoutes - настройки роутера и хендлеров
func SetupRoutes(repo *repository.UserRepository) *mux.Router {
	fmt.Println("[DEBUG] SetupRoutes: начало")
	router := mux.NewRouter()

	// Способ как оборачивать в middleware отдельные ручки
	// Handle(...) используется, потому что ты передаёшь http.Handler, а не просто функцию
	// router.Handle("/users", middleware.Logging(http.HandlerFunc(handlePostUsers(repo)))).Methods(http.MethodPost)

	protected := router.Methods(http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete).Subrouter()
	protected.Use(auth.AuthMiddleware)
	protected.Use(middleware.AdminOnly)
	protected.Use(middleware.EditorOnly)

	// Публичные маршруты или эндпоинты
	router.HandleFunc("/ready", ReadyHandler).Methods(http.MethodGet)
	router.HandleFunc("/users", GetUsersHandler(repo)).Methods(http.MethodGet)
	router.HandleFunc("/users/{id}", GetUserByIDFromURLHandler(repo)).Methods(http.MethodGet)
	router.HandleFunc("/me", GetUserByIDFromContextHandler(repo)).Methods(http.MethodGet)

	router.HandleFunc("/register", RegisterHandler(repo)).Methods(http.MethodPost)
	router.HandleFunc("/login", LoginHandler(repo)).Methods(http.MethodPost) // вместо GET !!!

	// Маршруты / эндпоинты, защищенные авторизацией и правами доступа
	protected.HandleFunc("/users", PostUserHandler(repo)).Methods(http.MethodPost)
	protected.HandleFunc("/users/{id}", PutUserHandler(repo)).Methods(http.MethodPut)
	protected.HandleFunc("/users/{id}", PatchUserHandler(repo)).Methods(http.MethodPatch)
	protected.HandleFunc("/users/{id}", DeleteUserHandler(repo)).Methods(http.MethodDelete)

	//Маршруты / эндпоинты для документации:
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	fmt.Println("[DEBUG] SetupRoutes: конец, router =", router)
	if router == nil {
		fmt.Println("[FATAL] router is nil!")
	}
	return router
}

// runServer - запускает сервер на :8080
func runServer(handler http.Handler) error {
	//go func() {
	err := http.ListenAndServe(":8081", handler)
	if err != nil {
		return err
	}
	//}()

	//err := waitForServer("http://localhost:8080/ready", 5*time.Second)
	//if err != nil {
	//	return fmt.Errorf("[SERVER] сервер не запустился вовремя: %v", err)
	//}
	return nil
}

// ReadyHandler проверяет, работает ли сервер
// @Summary Проверка готовности сервера
// @Tags health
// @Success 200 {object} map[string]string
// @Router /ready [get]
func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	//_, _ = w.Write([]byte(`{"status":"ok"}`))
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// PostUserHandler добавляет нового пользователя.
// @Summary Создать пользователя
// @Tags users
// @Accept json
// @Produce json
// @Param user body model.User true "Информация о пользователе"
// @Success 201 {object} model.User
// @Failure 400 {string} string "Неверный JSON или ошибка валидации"
// @Failure 422 {string} string "Ошибка бизнес-валидации (например, обязательные поля)"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /users [post]
func PostUserHandler(repo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log := middleware.LoggerFromContext(r.Context())

		var newUser model.User

		defer r.Body.Close()
		err := json.NewDecoder(r.Body).Decode(&newUser)
		if err != nil {
			http.Error(w, "[SERVER] неверный формат JSON", http.StatusBadRequest)
			return
		}

		err = validate.Struct(newUser)
		if err != nil {
			http.Error(w, "[SERVER] JSON не прошел валидацию по полям: "+err.Error(), http.StatusBadRequest)
			return
		}

		postUser, err := repo.PostUser(newUser)
		if err != nil {
			// Проверим, ошибка ли это валидации (ошибка пользователя)
			if strings.Contains(err.Error(), "обязательны для заполнения") {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}

			// Иначе — это внутренняя ошибка сервера
			http.Error(w, "[SERVER] ошибка при добавлении нового пользователя в БД", http.StatusInternalServerError)
			return
		}

		// вернуть статус и заголовки, удалим как будет готов ErrorHandler
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		// вернуть добавленного пользователя
		err = json.NewEncoder(w).Encode(postUser)
		if err != nil {
			log.Error(
				"encoding error",
				zap.Error(err),
				zap.String("event", "UserCreated"),
			)
			return
		}

		log.Info("user added successfully",
			zap.String("event", "UserCreated"),
			zap.Int("user.id", postUser.ID),
			zap.String("user.email", postUser.Email),
		)
	}
}

// RegisterHandler регистрирует нового пользователя.
// @Summary Зарегистрировать нового пользователя
// @Description Декодирует, валидирует поля JSON, генерирует хеш-пароль, добавляет в БД, возвращает ответ
// @Tags users
// @Accept json
// @Produce json
// @Param user body model.RegisterRequest true "Информация о пользователе"
// @Success 201 {object} model.User
// @Failure 400 {string} string "Неверный JSON или ошибка валидации"
// @Failure 422 {string} string "Ошибка бизнес-валидации (например, обязательные поля)"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /register [post]
func RegisterHandler(repo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var registerUser model.RegisterRequest

		defer r.Body.Close()

		err := json.NewDecoder(r.Body).Decode(&registerUser)
		if err != nil {
			http.Error(w, "[SERVER] неверный формат JSON", http.StatusBadRequest)
			return
		}

		err = validate.Struct(registerUser)
		if err != nil {
			http.Error(w, "[SERVER] JSON не прошел валидацию по полям: "+err.Error(), http.StatusBadRequest)
			return
		}

		// хеширование пароля
		hash, err := bcrypt.GenerateFromPassword([]byte(registerUser.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "[SERVER] ошибка при хешировании пароля:"+err.Error(), http.StatusInternalServerError)
		}

		var newUser model.User
		newUser.Name = registerUser.Name
		newUser.Age = registerUser.Age
		newUser.Email = registerUser.Email
		newUser.HashedPassword = string(hash)

		postUser, err := repo.PostUser(newUser)
		if err != nil {
			// Проверим, ошибка ли это валидации (ошибка пользователя)
			if strings.Contains(err.Error(), "обязательны для заполнения") {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			// Иначе — это внутренняя ошибка сервера
			http.Error(w, "[SERVER] ошибка при добавлении нового пользователя в БД", http.StatusInternalServerError)
			return
		}

		log := middleware.LoggerFromContext(r.Context())

		// вернуть статус и заголовки. Нужны ли еще какие-либо заголовки?
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		// вернуть добавленного пользователя
		err = json.NewEncoder(w).Encode(postUser)
		if err != nil {
			log.Error(
				"encoding error",
				zap.Error(err),
				zap.String("event", "UserRegistered"),
			)
			return
		}

		log.Info("user added successfully",
			zap.String("event", "UserCreated"),
			zap.Int("user.id", postUser.ID),
			zap.String("user.email", postUser.Email),
		)
	}
}

// GetUsersHandler получает список всех пользователей из БД.
// @Summary Получить всех пользователей из БД
// @Description делает запрос в БД, получает слайс со всем пользователями, инкодирует в JSON и возвращает ответ
// @Tags users
// @Produce json
// @Success 200 {array} model.User
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /users [get]
func GetUsersHandler(repo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getUsers, err := repo.GetAllUsers()
		if err != nil {
			http.Error(w, "[SERVER] ошибка при получении списка пользователей", http.StatusInternalServerError)
			return
		}

		log := middleware.LoggerFromContext(r.Context())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(getUsers)
		if err != nil {
			log.Error(
				"encoding error",
				zap.Error(err),
				zap.String("event", "GetUsers"),
			)
			return
		}

		log.Info("users got successfully",
			zap.String("event", "GetUsers"),
		)
	}
}

// PutUserHandler полностью обновляет нового пользователя.
// @Summary Полностью обновить пользователя
// @Description парсит ID из URL, декодирует новые данные о пользователе, валидирует, обновляет в БД, отправляет ответ
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя"
// @Param user body model.User true "Информация о пользователе"
// @Success 200 {object} model.User
// @Failure 400 {string} string "Неверный JSON или ошибка валидации"
// @Failure 404 {string} string "Пользователь не найден в БД"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /users/{id} [put]
func PutUserHandler(repo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log := middleware.LoggerFromContext(r.Context())

		id, err := parseIDFromRequest(r)
		if err != nil {
			http.Error(w, "[SERVER]"+err.Error(), http.StatusBadRequest)
		}

		var updatedUser model.User
		err = json.NewDecoder(r.Body).Decode(&updatedUser)
		if err != nil {
			http.Error(w, "[SERVER] неверный формат JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if updatedUser.ID != id {
			http.Error(w, "[SERVER] ID из URL и тела запроса не совпадают", http.StatusBadRequest)
			return
		}

		// Простая проверка на пустые поля
		if updatedUser.Name == "" || updatedUser.Email == "" || updatedUser.Age == 0 {
			http.Error(w, "некорректные данные: пустые поля", http.StatusBadRequest)
			return
		}

		err = validate.Struct(updatedUser)
		if err != nil {
			http.Error(w, "[SERVER] JSON не прошел валидацию по полям: "+err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: Разобрать ошибку и возвращать 409 только при нарушении уникальности
		// http.Error(w, "[SERVER] ошибка при PUT-обновлении пользователя в БД", http.StatusConflict)

		putUser, err := repo.PutUser(updatedUser)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, fmt.Sprintf("пользователь с ID %d не найден", updatedUser.ID), http.StatusNotFound)
				return
			}

			http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(putUser)
		if err != nil {
			log.Error("encoding error",
				zap.Error(err),
				zap.String("event", "UserPut"),
			)
			return
		}

		log.Info("user added successfully",
			zap.String("event", "UserPut"),
			zap.Int("user.id", putUser.ID),
			zap.String("user.email", putUser.Email),
		)
	}
}

// PatchUserHandler частично обновляет нового пользователя.
// @Summary Частично обновить пользователя
// @Description Извлекает ID из URL, парсит JSON, валидирует поля, обновляет данные пользователя в БД
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя"
// @Param user body model.PartialUser true "Информация о пользователе"
// @Success 200 {object} model.User
// @Failure 400 {string} string "Неверный JSON или ошибка валидации"
// @Failure 404 {string} string "Пользователь не найден в БД"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /users/{id} [patch]
func PatchUserHandler(repo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.LoggerFromContext(r.Context())

		id, err := parseIDFromRequest(r)
		if err != nil {
			http.Error(w, "[SERVER]"+err.Error(), http.StatusBadRequest)
		}

		var updatedUser model.PartialUser
		err = json.NewDecoder(r.Body).Decode(&updatedUser)
		if err != nil {
			http.Error(w, "Некорректный JSON", http.StatusBadRequest)
			return
		}

		if updatedUser.ID != id {
			http.Error(w, "[SERVER] ID в URL не соответствует ID в теле PATCH-запроса", http.StatusBadRequest)
			return
		}

		//// убрал эту логику
		//if updatedUser.Name == nil && updatedUser.Age == nil && updatedUser.Email == nil {
		//	http.Error(w, "[SERVER] необходимо передать хотя бы одно поле для обновления", http.StatusBadRequest)
		//	return
		//}

		err = validate.Struct(updatedUser)
		if err != nil {
			http.Error(w, "[SERVER] JSON не прошел валидацию по полям: "+err.Error(), http.StatusBadRequest)
			return
		}

		patchUser, err := repo.PatchUser(updatedUser)
		if err != nil {
			switch {
			case err.Error() == "нет полей для обновления":
				http.Error(w, "[SERVER] необходимо передать хотя бы одно поле для обновления", http.StatusBadRequest)
				return
			case errors.Is(err, sql.ErrNoRows):
				http.Error(w, "[SERVER] пользователь с таким ID не найден", http.StatusNotFound)
				return
			default:
				http.Error(w, "[SERVER] ошибка при PATCH-обновлении пользователя в БД", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(patchUser)
		if err != nil {
			log.Error("encoding error",
				zap.Error(err),
				zap.String("event", "UserPatch"),
			)
			return
		}

		log.Info("user added successfully",
			zap.String("event", "UserPatch"),
			zap.Int("user.id", patchUser.ID),
			zap.String("user.email", patchUser.Email),
		)
	}
}

// DeleteUserHandler удаляет пользователя.
// @Summary Удалить пользователя
// @Description Извлекает ID из URL, удаляет данные о пользователе из БД
// @Tags users
// @Param id path int true "ID пользователя"
// @Success 204 "Пользователь успешно удалён"
// @Failure 400 {string} string "Неверный ID"
// @Failure 404 {string} string "Пользователь не найден в БД"
// @Router /users/{id} [delete]
func DeleteUserHandler(repo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log := middleware.LoggerFromContext(r.Context())

		id, err := parseIDFromRequest(r)
		if err != nil {
			http.Error(w, "[SERVER]"+err.Error(), http.StatusBadRequest)
			return
		}

		err = repo.DeleteUser(id)
		if err != nil {
			http.Error(w, "[SERVER] ID пользователя не найден в БД", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// GetUserByIDFromURLHandler получает пользователя по ID.
// @Summary Получить пользователя по ID
// @Description делает запрос в БД, получает пользователя, инкодирует в JSON и возвращает ответ
// @Tags users
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} model.User
// @Failure 400 {string} string "ошибка при получении ID"
// @Failure 404 {string} string "Пользователь не найден в БД"
// @Router /users/{id} [get]
func GetUserByIDFromURLHandler(repo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log := middleware.LoggerFromContext(r.Context())

		id, err := parseIDFromRequest(r)
		if err != nil {
			http.Error(w, "[SERVER]"+err.Error(), http.StatusBadRequest)
			return
		}

		getUser, err := repo.GetUserByID(id)
		if err != nil {
			http.Error(w, "[SERVER] не удалось найти пользователя в БД по ID", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(getUser)
	}
}

// GetUserByIDFromContextHandler получает пользователя по ID из контекста запроса.
// @Summary Получить пользователя по ID из контекста (текущий пользователь)
// @Description Извлекает ID пользователя из контекста (установленного middleware), ищет пользователя в БД, возвращает JSON
// @Tags users
// @Produce json
// @Success 200 {object} model.User
// @Failure 404 {string} string "Пользователь не найден"
// @Failure 500 {string} string "Ошибка сервера при извлечении ID из контекста"
// @Router /me [get]
func GetUserByIDFromContextHandler(repo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log := middleware.LoggerFromContext(r.Context())

		id, ok := auth.GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "[SERVER] ID не был передан с контекстом из middleware", http.StatusInternalServerError)
			return
		}

		getUser, err := repo.GetUserByID(id)
		if err != nil {
			errorsLogger.Printf("[SERVER] ошибка при поиске пользователя: %v", err)
			http.Error(w, "[SERVER] не удалось найти пользователя в БД по ID", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(getUser)
	}
}

// LoginHandler авторизует пользователя.
// @Summary Авторизовать пользователя и получить JWT токены
// @Description Принимает email и пароль, валидирует, проверяет в БД, создает JWT access и refresh токены, возвращает access токен в JSON и refresh токен в HTTP-only cookie
// @Tags users
// @Accept json
// @Produce json
// @Param user body model.LoginRequest true "Данные для авторизации (email и пароль)"
// @Success 200 {object} map[string]string "access-token и сообщение об успешной авторизации"
// @Failure 400 {string} string "Неверный JSON или ошибка валидации"
// @Failure 401 {string} string "Неверный email или пароль"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /login [post]
func LoginHandler(repo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log := middleware.LoggerFromContext(r.Context())

		var user model.LoginRequest
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "[SERVER]"+err.Error(), http.StatusBadRequest) // не StatusUnauthorized
			return
		}

		err = validate.Struct(user)
		if err != nil {
			http.Error(w, "[SERVER] логин или пароль были введены некорректно", http.StatusBadRequest) // не StatusUnauthorized
			return
		}

		loginUser, err := repo.GetUserByEmail(user.Email)
		if err != nil {
			http.Error(w, "[SERVER] не удалось найти пользователя по e-mail", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(loginUser.HashedPassword), []byte(user.Password)) // сначала пароль из БД
		if err != nil {
			http.Error(w, "[SERVER] неправильный пароль", http.StatusUnauthorized)
			return
		}

		// Создать JWT access-токен
		accessTokenString, err := getAccessToken(loginUser)
		if err != nil {
			http.Error(w, "ошибка при создании токена"+err.Error(), http.StatusInternalServerError)
			return
		}

		// Создать JWT refresh-токен
		refreshTokenString, err := getRefreshToken(loginUser)
		if err != nil {
			http.Error(w, "ошибка при создании токена"+err.Error(), http.StatusInternalServerError)
			return
		}

		// здесь (в ручке) передаю только refresh-токен в cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh-token",
			Value:    refreshTokenString,
			Path:     "/",
			Secure:   false,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   int(config.RefreshTokenTTL.Seconds()),
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(map[string]string{
			"message":      "Успешная авторизация",
			"access-token": accessTokenString,
		})
		if err != nil {
			log.Error(
				"encoding error",
				zap.Error(err),
				zap.String("event", "UserLogin"),
			)
			return
		}

		log.Info("user added successfully",
			zap.String("event", "UserLogin"),
			zap.Int("user.id", loginUser.ID),
			zap.String("user.email", loginUser.Email),
		)
	}
}

func parseIDFromRequest(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		return 0, fmt.Errorf("ID пользователя не указан в ссылке")

	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("ID не является числом")
	}
	return id, nil
}

func getAccessToken(user model.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{ // только HEADER.PAYLOAD
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(config.AccessTokenTTL).Unix(),
	})

	tokenString, err := token.SignedString(config.JWTSecret) // добавление секретного ключа
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func getRefreshToken(user model.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(config.RefreshTokenTTL).Unix(),
	})

	tokenString, err := token.SignedString(config.JWTSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("[SERVER] сервер не ответил на %s. Таймаут: %v", url, timeout)
}

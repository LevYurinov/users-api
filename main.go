// @title           Users API
// @version         1.0
// @description     Документация API для управления пользователями (CRUD + авторизация).
// @termsOfService  http://swagger.io/terms/

// @contact.name   Иван Иванов
// @contact.email  example@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

package main

import (
	"fmt"
	"log"
	"os"
	"pet/internal/database"
	"pet/internal/repository"
	"pet/internal/server"
)

func main() {
	errorsLogger := initLogger()
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

func initLogger() *log.Logger {
	file, err := os.OpenFile("main_logs.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("ошибка при открытии лог-файла: %v", err)
	}
	return log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

//package main
//
//import (
//	"bytes"
//	"context"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"io"
//	"log"
//	"net/http"
//	"os"
//	"time"
//)
//
//var errorLogger *log.Logger
//
//type User struct {
//	Name  string `json:"name"`
//	Age   int    `json:"age"`
//	Email string `json:"email"`
//}
//
//type DynamicUsers struct {
//	JSON json.RawMessage `json:"json"`
//}
//
//func main() {
//	initLogger()
//	ageLimit, err := getAgeLimit()
//	if err != nil {
//		fmt.Println("Вы вышли из программы!")
//		return
//	}
//
//	users := createUsers()
//	filteredUsers, err := filterUsers(users, ageLimit)
//	if err != nil {
//		errorLogger.Println(err)
//		return
//	}
//
//	usersBytes, err := convertToJson(filteredUsers)
//	if err != nil {
//		errorLogger.Println(err)
//		return
//	}
//
//	timeout := 10 * time.Second
//	url := "https://httpbin.org/post"
//	resp, err := sendRequest(usersBytes, timeout, url)
//	if err != nil {
//		errorLogger.Println(err)
//		return
//	}
//	if resp != nil {
//		defer resp.Body.Close()
//	}
//
//	err = responseDecoding(resp.Body)
//	if err != nil {
//		errorLogger.Println(err)
//		return
//	}
//}
//
//func initLogger() {
//	file, err := os.OpenFile("app4.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
//	if err != nil {
//		log.Fatalf("ошибка при открытии лог-файла: %v\n", err)
//	}
//	errorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
//}
//
//func getAgeLimit() (int, error) {
//
//	var ageLimit int
//	for {
//		fmt.Println("Введите цифру ограничения по возрасту от 1 до 130.")
//		fmt.Println("Или нажмите 0 для выхода из программы.")
//		fmt.Print("Введите данные: ")
//
//		_, err := fmt.Scan(&ageLimit)
//		if err != nil {
//			fmt.Println("Неподходящий формат данных.")
//			continue
//		}
//		if ageLimit == 0 {
//			return -1, fmt.Errorf("пользователь выходит из программы")
//		}
//		if ageLimit < 0 || ageLimit > 130 {
//			fmt.Println("Невалидный возраст.")
//			continue
//		}
//		break
//	}
//	return ageLimit, nil
//}
//
//func createUsers() any {
//	// создайте одного (User) или нескольких пользователей ([]User)
//	users := []User{
//		{"Lev", 32, "lev@gmail.com"},
//		{"Timur", 30, "timur@gmail.com"},
//		{"Klera", 84, "klera@gmail.com"},
//		{"Marina", 16, "marina@gmail.com"},
//	}
//	return users
//}
//
//func filterUsers(users any, ageLimit int) (any, error) {
//	switch input := users.(type) {
//	case User:
//		if input.Age >= ageLimit {
//			return input, nil
//		}
//		return nil, nil
//
//	case []User:
//		filteredUsers := make([]User, 0)
//
//		for _, value := range input {
//			if value.Age >= ageLimit {
//				filteredUsers = append(filteredUsers, value)
//			}
//		}
//		return filteredUsers, nil
//
//	default:
//		return nil, fmt.Errorf("неподдерживаемый тип: %T", input)
//	}
//}
//
//func convertToJson(filteredUsers any) ([]byte, error) {
//	usersBytes, err := json.Marshal(filteredUsers)
//	if err != nil {
//		return nil, fmt.Errorf("ошибка при конвертации в JSON: %w", err)
//	}
//	return usersBytes, nil
//}
//
//func sendRequest(usersBytes []byte, timeout time.Duration, url string) (*http.Response, error) {
//	ctx, cancel := context.WithTimeout(context.Background(), timeout)
//	defer cancel()
//
//	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(usersBytes))
//	if err != nil {
//		return nil, fmt.Errorf("ошибка при создании POST-запроса: %v", err)
//	}
//	req.Header.Set("Content-Type", "application/json")
//	req.Header.Set("Accept", "application/json")
//
//	client := &http.Client{}
//	resp, err := client.Do(req)
//	if err != nil {
//		if errors.Is(err, context.DeadlineExceeded) {
//			return nil, fmt.Errorf("время ожидания ответа истекло. Таймаут: %v секунд", timeout)
//		} else {
//			return nil, fmt.Errorf("ошибка при получении ответа: %v", err)
//		}
//	}
//	if resp.StatusCode != http.StatusOK {
//		return nil, fmt.Errorf("ошибка. Неожиданный статус отправки запроса: %v (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
//	}
//	fmt.Println("Статус отправки запроса: ", http.StatusText(resp.StatusCode))
//	return resp, nil
//}
//
//func responseDecoding(body io.Reader) error {
//	bodyBytes, err := io.ReadAll(body)
//	if err != nil {
//		return fmt.Errorf("ошибка чтения тела ответа %v", err)
//	}
//
//	var dynamicUsers DynamicUsers
//
//	err = json.Unmarshal(bodyBytes, &dynamicUsers)
//	if err != nil {
//		return fmt.Errorf("ошибка декодирования контейнера DynamicUsers: %w", err)
//	}
//
//	if len(dynamicUsers.JSON) == 0 {
//		fmt.Println("Нет пользователей, удовлетворяющих возрастным ограничениям")
//		return nil
//	}
//
//	var users []User
//	err = json.Unmarshal(dynamicUsers.JSON, &users)
//
//	if err == nil {
//		fmt.Println()
//		fmt.Println("Получен список пользователей:")
//		for i, user := range users {
//			fmt.Printf("Пользователь %d:\n", i+1)
//			fmt.Printf("Имя: %s\nВозраст: %d\nEmail: %s\n\n", user.Name, user.Age, user.Email)
//		}
//		return nil
//	}
//
//	var user User
//	err = json.Unmarshal(dynamicUsers.JSON, &user)
//	fmt.Println()
//	fmt.Println("Получены данные пользователя:")
//	fmt.Printf("Имя: %s\nВозраст: %d\nEmail: %s\n\n", user.Name, user.Age, user.Email)
//	return nil
//}

//package main
//
//import (
//	"bytes"
//	"context"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"io"
//	"log"
//	"net/http"
//	"os"
//	"time"
//)
//
//var errorLogger *log.Logger
//
//type User struct {
//	Name  string `json:"name"`
//	Email string `json:"email"`
//	Age   uint   `json:"age"`
//}
//
//type UsersBack struct {
//	JSON []User `json:"json"`
//}
//
//func main() {
//	initLogger()
//	users := buildUsers()
//	usersBytes, err := encodeToJSON(users)
//	if err != nil {
//		errorLogger.Println(err)
//		return
//	}
//
//	url := "https://httpbin.org/post"
//	timeout := 5 * time.Second
//	resp, err := sendRequest(usersBytes, url, timeout)
//	if err != nil {
//		errorLogger.Println(err)
//		return
//	}
//	defer resp.Body.Close()
//
//	usersBack, err := parseUsersResponse(resp.Body)
//	if err != nil {
//		errorLogger.Println(err)
//	}
//
//	fmt.Println("Информация о пользователях:")
//	fmt.Println()
//
//	count := 1
//	for _, user := range usersBack {
//		fmt.Printf("Пользователь %d:\nИмя: %s\nВозраст: %d\nEmail: %s\n\n", count, user.Name, user.Age, user.Email)
//		count++
//	}
//}
//
//func initLogger() {
//	file, err := os.OpenFile("app3.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
//	if err != nil {
//		log.Fatalf("Ошибка при открытии лог-файла: %v\n", err)
//	}
//	errorLogger = log.New(file, "Error: ", log.Ldate|log.Ltime|log.Lshortfile|log.Llongfile)
//}
//
//func buildUsers() []User {
//	users := []User{
//		{Name: "Lev", Age: 32, Email: "lev@gmail.com"},
//		{Name: "Timur", Age: 30, Email: "timur@gmail.com"},
//		{Name: "Klera", Age: 84, Email: "klera@gmail.com"},
//	}
//	return users
//}
//
//func encodeToJSON(users []User) ([]byte, error) {
//	usersBytes, err := json.Marshal(users)
//	if err != nil {
//		return nil, fmt.Errorf("Ошибка при инкодировании JSON: %v\n", err)
//	}
//	return usersBytes, nil
//}
//
//func sendRequest(usersBytes []byte, url string, timeout time.Duration) (*http.Response, error) {
//	ctx, cancel := context.WithTimeout(context.Background(), timeout)
//	defer cancel()
//
//	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(usersBytes))
//	if err != nil {
//		return nil, fmt.Errorf("Ошибка при создании запроса: %v\n", err)
//	}
//	req.Header.Set("Content-Type", "application/json")
//	req.Header.Set("Accept", "application/json")
//
//	client := &http.Client{}
//	resp, err := client.Do(req)
//	if err != nil {
//		if errors.Is(err, context.DeadlineExceeded) {
//			return nil, fmt.Errorf("Время ответа сервера истекло. Таймаут: %v\n", timeout)
//		}
//		return nil, fmt.Errorf("Ошибка при отправке POST-запроса: %v\n", err)
//	}
//
//	fmt.Println("Статус отправки POST-запроса: ", http.StatusText(resp.StatusCode))
//	return resp, nil
//}
//
//func parseUsersResponse(body io.Reader) ([]User, error) {
//	var usersBack UsersBack
//
//	err := json.NewDecoder(body).Decode(&usersBack)
//	if err != nil {
//		return nil, fmt.Errorf("Ошибка при декодировании тела ответа: %v\n", err)
//	}
//	return usersBack.JSON, nil
//}

//package main
//
//import (
//	"bytes"
//	"context"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"log"
//	"net/http"
//	"os"
//	"time"
//)
//
//var errorLogger *log.Logger
//
//type User struct {
//	Name  string `json:"name"`
//	Email string `json:"email"`
//	Age   uint   `json:"age"`
//}
//
//type UserBack struct {
//	User `json:"json"`
//}
//
//func main() {
//	initLogger()
//
//	// Создать структуру User с данными
//	user := User{
//		Name:  "Lev",
//		Email: "lev@gmail.com",
//		Age:   32,
//	}
//
//	// Сериализовать структуру в JSON
//	output, err := json.Marshal(user)
//	if err != nil {
//		errorLogger.Printf("Ошибка в структуре данных при сериализации в JSON")
//		return
//	}
//
//	// Создание POST-запроса
//	timeout := 5 * time.Second
//	ctx, cancel := context.WithTimeout(context.Background(), timeout)
//	defer cancel()
//
//	link := "https://httpbin.org/post"
//	req, err := http.NewRequestWithContext(ctx, http.MethodPost, link, bytes.NewBuffer(output))
//	if err != nil {
//		errorLogger.Printf("Ошибка при создании POST-запроса: %v\n", err)
//		return
//	}
//	req.Header.Set("Content-Type", "application/json")
//
//	// Отправка POST-запроса
//	client := &http.Client{}
//	resp, err := client.Do(req)
//	if err != nil {
//		if errors.Is(err, context.DeadlineExceeded) {
//			errorLogger.Printf("Время ожидания запроса истекло. Таймаут: %v\n", timeout)
//		} else {
//			errorLogger.Printf("Ошибка при отправке POST-запроса на сервер: %v\n", err)
//		}
//		return
//	}
//	defer resp.Body.Close()
//
//	// Обработать ответ
//	fmt.Println("Ответ сервера: ", resp.StatusCode, http.StatusText(resp.StatusCode))
//
//	//bodyBytes, err := io.ReadAll(resp.Body)
//	//if err != nil {
//	//	errorLoger.Printf("Ошибка чтения тела ответа JSON: %v\n", err)
//	//	return
//	//}
//	//
//	//fmt.Println(string(bodyBytes))
//
//	// Распаршиваю ответ
//	var userBack UserBack
//	err = json.NewDecoder(resp.Body).Decode(&userBack)
//	if err != nil {
//		errorLogger.Printf("Ошибка при декодировании JSON: %v\n", err)
//		return
//	}
//
//	fmt.Printf("Данные пользователя в теле ответа JSON:\nИмя: %s\nВозраст: %d\nEmail: %s\n", userBack.Name, userBack.Age, userBack.Email)
//}
//
//func initLogger() {
//	file, err := os.OpenFile("app2.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
//	if err != nil {
//		log.Fatalf("Не удалось открыть лог-файл: %v\n", err)
//	}
//
//	errorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile|log.Llongfile)
//}

// Полный GET-запрос (3-я задача)
//package main
//
//import (
//	"context"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"log"
//	"net/http"
//	"os"
//	"time"
//)
//
//var errorLogger *log.Logger
//
//type Joke struct {
//	Type      string `json:"type"`
//	Setup     string `json:"setup"`
//	Punchline string `json:"punchline"`
//	ID        int    `json:"id"`
//}
//
//func main() {
//	initLogger()
//
//	// создаю контекст
//	timeout := 5 * time.Second
//	ctx, cancel := context.WithTimeout(context.Background(), timeout)
//	defer cancel()
//
//	// создаю GET-запрос
//	link := "https://official-joke-api.appspot.com/jokes/ten"
//	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
//	if err != nil {
//		errorLogger.Printf("Ошибка при создании GET-запроса: %v\n", err)
//		return
//	}
//
//	resp, err := http.DefaultClient.Do(req)
//	if err != nil {
//		if errors.Is(err, context.DeadlineExceeded) {
//			errorLogger.Printf("Время ожидания ответа истекло. Таймаут: %v\n", timeout)
//		} else {
//			errorLogger.Printf("Ошибка при получении ответа GET-запроса %v\n", err)
//		}
//		return
//	}
//	defer resp.Body.Close()
//
//	// Проверка статуса запроса
//	if resp.StatusCode != http.StatusOK {
//		errorLogger.Printf("Неожиданный статус ответа сервера: %v\n", resp.StatusCode)
//		return
//	}
//
//	// Выясняю структуру JSON-ответа
//	//bodyBytes, err := io.ReadAll(resp.Body)
//	//if err != nil {
//	//	errorLogger.Printf("Ошибка при чтении JSON: %v\n", err)
//	//}
//	//fmt.Println(string(bodyBytes))
//
//	// Распаршиваю JSON-ответ в слайс структур Joke
//	var jokes []Joke
//	err = json.NewDecoder(resp.Body).Decode(&jokes)
//	if err != nil {
//		errorLogger.Printf("Ошибка при декодировании JSON: %v\n", err)
//		return
//	}
//
//	// Разбиваю шутки на группы по полю Type
//	jokesMap := make(map[string][]Joke)
//	for _, joke := range jokes {
//		jokesMap[joke.Type] = append(jokesMap[joke.Type], joke)
//	}
//
//	// Вывод
//	for typeJoke, jokesValues := range jokesMap {
//		fmt.Printf("Тип: %s\n", typeJoke)
//
//		for _, joke := range jokesValues {
//			fmt.Printf("ID: %d. Setup: %s. Punchline: %s\n", joke.ID, joke.Setup, joke.Punchline)
//		}
//		fmt.Println()
//	}
//}
//
//func initLogger() {
//	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
//	if err != nil {
//		log.Fatalf("не удалось открыть лог-файл: %v", err)
//	}
//
//	errorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile|log.)
//}

//package main
//
//import (
//	"context"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"net/http"
//	"sort"
//	"time"
//)
//
//type Joke struct {
//	Type      string `json:"type"`
//	Setup     string `json:"setup"`
//	Punchline string `json:"punchline"`
//	ID        int    `json:"id"`
//}
//
//func main() {
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	// создание GET-запроса
//	link := "https://official-joke-api.appspot.com/jokes/ten"
//	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
//	if err != nil {
//		fmt.Printf("Ошибка при создании запроса: %v\n", err)
//		return
//	}
//	// Отправка GET-запроса + обработка ошибок
//	resp, err := http.DefaultClient.Do(req)
//	if err != nil {
//		if errors.Is(err, context.DeadlineExceeded) {
//			fmt.Println("Время ожидания ответа истекло:", err)
//		} else {
//			fmt.Println("Ошибка при выполнении запроса:", err)
//		}
//		return
//	}
//
//	defer resp.Body.Close()
//
//	// Проверка статуса ответа
//	if resp.StatusCode != http.StatusOK {
//		fmt.Println("Неожиданный статус ответа:", resp.StatusCode, http.StatusText(resp.StatusCode))
//		return
//	}
//
//	// Прочитать Body
//	//bodyBytes, err := io.ReadAll(resp.Body)
//	//fmt.Println(string(bodyBytes)) // прочитал и закомитил
//
//	// Распарсить JSON как массив структур.
//	jokes := make([]Joke, 0)
//	err = json.NewDecoder(resp.Body).Decode(&jokes)
//	if err != nil {
//		fmt.Println("Ошибка при декодировании JSON ответа:", err)
//		return
//	}
//
//	// Отсортировать массив по joke.ID
//	sort.Slice(jokes, func(i, j int) bool {
//		return jokes[i].ID < jokes[j].ID
//	})
//
//	// Вывести в консоль каждую шутку: setup + punchline
//	for _, joke := range jokes {
//		fmt.Printf("ID: %d\n%s — %s\n\n", joke.ID, joke.Setup, joke.Punchline)
//	}
//}

//package main
//
//import (
//	"context"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"io"
//	"net/http"
//	"time"
//)
//
//type Joke struct {
//	Type      string `json:"type"`
//	Setup     string `json:"setup"`
//	Punchline string `json:"punchline"`
//	ID        int    `json:"id"`
//}
//
//func main() {
//	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
//	defer cancel()
//
//	api := "https://official-joke-api.appspot.com/jokes/random"
//	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
//	if err != nil {
//		fmt.Println("Ошибка при создании запроса", err)
//		return
//	}
//
//	resp, err := http.DefaultClient.Do(req)
//	if err != nil {
//		if errors.Is(err, context.DeadlineExceeded) {
//			fmt.Println("Ошибка при выполнении запроса. Превышен таймаут")
//		} else {
//			fmt.Println("Ошибка при получении ответа:", err)
//		}
//		return
//	}
//	defer resp.Body.Close()
//	fmt.Println("Запрос получен успешно. Статус:", resp.Status)
//
//	respBytes, err := io.ReadAll(resp.Body)
//	if err != nil {
//		fmt.Println("Ошибка при обработке содержимого JSON файла", err)
//		return
//	}
//	fmt.Println(string(respBytes))
//
//	var joke Joke
//	err = json.NewDecoder(resp.Body).Decode(&joke)
//	if err != nil {
//		fmt.Println("Ошибка при декодировании JSON", err)
//		return
//	}
//
//	fmt.Printf("Setup: %s\nPunchline: %s/n", joke.Setup, joke.Punchline)
//}

//	// получение GET-запроса
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	link := "https://official-joke-api.appspot.com/jokes/random"
//	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
//	if err != nil {
//		fmt.Println("Ошибка при создании запроса")
//		return
//	}
//
//	resp, err := http.DefaultClient.Do(req)
//	if err != nil {
//		if errors.Is(err, context.DeadlineExceeded) {
//			fmt.Println("Контекст отменён: превышен таймаут")
//		} else {
//			fmt.Println("Ошибка при выполнении запроса")
//		}
//		return
//	}
//
//	defer resp.Body.Close()
//	fmt.Println("Успешный ответ. Статус:", resp.Status)
//
//	// обработка GET-запроса
//	bodyBytes, err := io.ReadAll(resp.Body)
//	if err != nil {
//		fmt.Println("Ошибка при чтении тела ответа:", err)
//		return
//	}
//	fmt.Println(string(bodyBytes))
//
//	var joke Joke
//	err = json.NewDecoder(resp.Body).Decode(&joke)
//	if err != nil {
//		fmt.Println("Ошибка при декодировании JSON", err)
//		return
//	}
//	fmt.Printf("Setup: %s\nPunchline: %s\n", joke.Setup, joke.Punchline)
//}

//package main
//
//import (
//	"context"
//	"fmt"
//	"net/http"
//	"time"
//)
//
//func main() {
//	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
//	defer cancel()
//
//	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://watta.market/", nil)
//	if err != nil {
//		return nil, fmt.Errorf("failed to create request with ctx: %w", err)
//	}
//}

//ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
//defer cancel()
//
//url := "https://httpbin.org/delay/3"
//req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
//if err != nil {
//	fmt.Println("Ошибка при создании запроса")
//	return
//}
//
//resp, err := http.DefaultClient.Do(req)
//if err != nil {
//	if errors.Is(err, context.DeadlineExceeded) {
//		fmt.Println("Контекст отменён: превышен таймаут")
//	} else {
//		fmt.Println("Ошибка при выполнении запроса")
//	}
//	return
//}
//defer resp.Body.Close()
//
//fmt.Println("Успешный ответ. Статус:", resp.Status)

// теория из книги по каналам
//package main
//
//import (
//	"fmt"
//	"io/ioutil"
//	"log"
//	"net/http"
//)
//
//func main() {
//	sizes := make(chan int)
//
//	go responseSize("https://watta.market/", sizes)
//	go responseSize("https://voltag.ru/", sizes)
//	go responseSize("https://tstarter.ru/", sizes)
//
//	fmt.Println(<-sizes)
//	fmt.Println(<-sizes)
//	fmt.Println(<-sizes)
//
//}
//
//func responseSize(url string, sizes chan int) {
//	fmt.Println("Getting", url)
//
//	response, err := http.Get(url)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer response.Body.Close()
//
//	body, err := ioutil.ReadAll(response.Body)
//	if err != nil {
//		log.Fatal(err)
//	}
//	sizes <- len(body)
//}

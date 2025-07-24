package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"pet/internal/model"
	"strconv"
	"time"
)

var errorsLogger *log.Logger

var client = &http.Client{}

// Run - главная функция пакета client, которая передается в main.go
func Run(l *log.Logger) {
	errorsLogger = l

	newUser := createUser()
	bytesNewUser, err := marshalUser(newUser)
	if err != nil {
		errorsLogger.Println(err)
		return
	}

	// Отправка POST-запроса и обработка ответа сервера
	resp, err := sendPostNewUser(bytesNewUser)
	if err != nil {
		errorsLogger.Println(err)
		return
	}
	if resp != nil {
		defer resp.Body.Close()
		err = outputRespBody(resp.Body, "POST")
		if err != nil {
			errorsLogger.Println(err)
			return
		}
	} else {
		fmt.Println("Тело ответа POST-запроса вернулось пустым")
	}

	// Отправка GET-запроса и обработка ответа сервера
	err = sendGetUsers()
	if err != nil {
		errorsLogger.Println(err)
		return
	}

	// Отправка PUT-запроса и обработка ответа сервера
	bytesPutUser, err := marshalUser(model.UpdatedUserPut)
	if err != nil {
		errorsLogger.Println(err)
		return
	}

	resp, err = sendPutUser(bytesPutUser, model.UpdatedUserPut.ID)
	if err != nil {
		errorsLogger.Println(err)
		return
	}
	if resp != nil {
		defer resp.Body.Close()

		err = outputRespBody(resp.Body, "PUT")
		if err != nil {
			errorsLogger.Println(err)
			return
		}
	} else {
		fmt.Println("Тело ответа PUT-запроса вернулось пустым")
	}

	// Отправка PATCH-запроса и обработка ответа сервера
	patchUpdatedUser, err := json.Marshal(model.UpdatedUserPatch)
	if err != nil {
		errorsLogger.Println(err)
		return
	}
	resp, err = sendPatchUser(patchUpdatedUser, model.UpdatedUserPatch.ID)
	if err != nil {
		errorsLogger.Println(err)
		return
	}
	if resp != nil {
		defer resp.Body.Close()

		err = outputRespBody(resp.Body, "PATCH")
		if err != nil {
			errorsLogger.Println(err)
			return
		}
	} else {
		fmt.Println("Тело ответа PATCH-запроса вернулось пустым")
	}

	// Отправка DELETE-запроса и обработка ответа сервера
	resp, err = sendDeleteUser(model.DeletedUser.ID)
	if err != nil {
		errorsLogger.Println(err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	fmt.Printf("Пользователь с ID %d удален из БД\n", model.DeletedUser.ID)

	// Отправка PUT-запроса с ID и обработка ответа сервера
	getUser, err := sendGetUser(model.GetUserByID.ID)
	// дальнейшая работа с полученным пользователем
	fmt.Printf("Получен пользователь ID %d:\n", getUser.ID)
	fmt.Printf("Имя: %s\nВозраст: %d\nE-mail: %s\n", getUser.Name, getUser.Age, getUser.Email)
}

func createUser() model.User {
	newUser := model.User{
		Name: "Lev", Age: 32, Email: "lev@gmail.com"}
	return newUser
}

// Функция декодирования и вывода в консоль тела ответа
func outputRespBody(body io.ReadCloser, method string) error {
	defer body.Close()

	updatedUser, err := unmarshalUser(body)
	if err != nil {
		return fmt.Errorf("[CLIENT] ошибка при декодировании тела ответа из JSON: %v", err)
	}
	fmt.Printf("Методом %v обновлены данные пользователя с ID %d:\n", method, updatedUser.ID)
	fmt.Printf("Имя: %s\nВозраст: %d\nE-mail: %s\n", updatedUser.Name, updatedUser.Age, updatedUser.Email)
	return nil
}

// Функция декодирования из JSON
func marshalUser(user model.User) ([]byte, error) {
	bytesUser, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("[CLIENT] ошибка при инкодировании пользователей в JSON: %w", err)
	}
	return bytesUser, nil
}

// Функция добавления нового пользователя в БД
func sendPostNewUser(bytesNewUser []byte) (*http.Response, error) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:8080/users", bytes.NewBuffer(bytesNewUser))
	if err != nil {
		return nil, fmt.Errorf("[CLIENT] ошибка при создании POST-запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("[CLIENT] время ожидания ответа сервера истекло. Таймаут %v секунд", timeout)
		} else {
			return nil, fmt.Errorf("[CLIENT] ошибка при отправке POST-запроса: %w", err)
		}
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("[CLIENT] вернулся неожиданный статус ответа: %v", resp.StatusCode)
	}

	return resp, nil
}

// Функция получения списка пользователей из БД
func sendGetUsers() error {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/users", nil)
	if err != nil {
		return fmt.Errorf("[CLIENT] ошибка при создании GET-запроса %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("[CLIENT] время ожидания ответа сервера истекло. Таймаут %v секунд", timeout)
		} else {
			return fmt.Errorf("[CLIENT] ошибка при отправке запроса: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[CLIENT] неожиданный статус ответа: %v: %v", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var usersList []model.User
	err = json.NewDecoder(resp.Body).Decode(&usersList)
	if err != nil {
		return fmt.Errorf("[CLIENT] ошибка при декодировании ответа, %w", err)
	}

	return nil
}

// Функция обновления пользователя в БД
func sendPutUser(bytesUpdatedUser []byte, ID int) (*http.Response, error) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	url := "http://localhost:8080/users/" + strconv.Itoa(ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(bytesUpdatedUser))
	if err != nil {
		return nil, fmt.Errorf("[CLIENT] ошибка при создании PUT-запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("[CLIENT] время ожидания ответа сервера истекло. Таймаут %v секунд", timeout)
		} else {
			return nil, fmt.Errorf("[CLIENT] ошибка при отправке запроса: %w", err)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[CLIENT] неожиданный статус ответа: %v: %v", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return resp, nil
}

// Функция инкодирования в JSON
func unmarshalUser(body io.Reader) (model.User, error) {
	var updatedUser model.User
	err := json.NewDecoder(body).Decode(&updatedUser)
	if err != nil {
		return updatedUser, fmt.Errorf("[CLIENT] ошибка при декодировании ответа PUT-запроса %w", err)
	}
	return updatedUser, nil
}

// Функция частичного обновления пользователя в БД
func sendPatchUser(bytesUpdatedUser []byte, ID int) (*http.Response, error) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	urlID := "http://localhost:8080" + "/users/" + strconv.Itoa(ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, urlID, bytes.NewBuffer(bytesUpdatedUser))
	if err != nil {
		return nil, fmt.Errorf("[CLIENT] ошибка при создании запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("[CLIENT] время ожидания ответа сервера истекло. Таймаут %v секунд", timeout)
		} else {
			return nil, fmt.Errorf("[CLIENT] ошибка при отправке запроса: %w", err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[CLIENT] неожиданный статус ответа сервера: %v", resp.StatusCode)
	}

	return resp, nil
}

// Функция удаления пользователя в БД
func sendDeleteUser(ID int) (*http.Response, error) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	urlID := "http://localhost:8080" + "/users/" + strconv.Itoa(ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, urlID, nil)
	if err != nil {
		return nil, fmt.Errorf("[CLIENT] ошибка при создании запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("[CLIENT] время ожидания ответа сервера истекло. Таймаут %v секунд", timeout)
		} else {
			return nil, fmt.Errorf("[CLIENT] ошибка при отправке запроса: %w", err)
		}
	}
	if resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("[CLIENT] неожиданный статус ответа сервера: %v", resp.StatusCode)
	}
	return resp, nil
}

// Функция получения пользователя из БД по ID
func sendGetUser(ID int) (*model.User, error) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	url := "http://localhost:8080" + "/users/" + strconv.Itoa(ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("[CLIENT] ошибка при создании GET-запроса %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("[CLIENT] время ожидания ответа сервера истекло. Таймаут %v секунд", timeout)
		} else {
			return nil, fmt.Errorf("[CLIENT] ошибка при отправке запроса: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[CLIENT] неожиданный статус ответа: %v: %v", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var getUser model.User
	err = json.NewDecoder(resp.Body).Decode(&getUser)
	if err != nil {
		return nil, fmt.Errorf("[CLIENT] ошибка при декодировании JSON: %w", err)
	}
	return &getUser, nil
}

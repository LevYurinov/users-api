package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"pet/model"
	"pet/repository"
	"pet/server"
	"strconv"
	"strings"
	"testing"
)

// setupTestServer - создаёт временный HTTP-сервер, который автоматически запускается в фоне
func setupTestServer() *httptest.Server {
	testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	testRepo := repository.NewUserRepository(TestDB, testLogger)

	// Инициализируем глобальный логгер в пакете server
	server.InitLogger(testLogger)

	router := server.SetupRoutes(testRepo)

	return httptest.NewServer(router)
}

func TestGetAllUsersHandler(t *testing.T) {
	deleteTestUsers(TestDB)
	expected, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/users")
	if err != nil {
		t.Fatalf("ошибка при запросе: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ожидался статус ответа: 200, а пришел: %v", resp.StatusCode)
	}

	var users []model.User
	err = json.NewDecoder(resp.Body).Decode(&users)
	if err != nil {
		t.Fatalf("ошибка при декодировании тела ответа в JSON: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("ожидалось 2 пользователя, получено %d", len(users))
	}

	for _, user := range users {
		exp, ok := expected[user.Email]
		if !ok {
			t.Errorf("неожиданный пользователь в ответе: %v", user.Email)
			continue
		}
		if user.Name != exp.Name || user.Age != exp.Age {
			t.Errorf("данные не совпадают для %s: ожидали %+v, получили %+v", user.Email, exp, user)
		}
	}
}

func TestGetUserByIDHandler(t *testing.T) {
	deleteTestUsers(TestDB)
	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	id := users["alice@example.com"].ID

	resp, err := http.Get(testServer.URL + "/users/" + strconv.Itoa(id))
	if err != nil {
		t.Fatalf("ошибка при запросе: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ожидался успешный статус ответа 200, а вернулся %v", resp.StatusCode)
	}

	var user model.User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		t.Fatalf("ошибка при декодировании пользователя из JSON: %v", err)
	}

	if user.ID != id {
		t.Errorf("ожидался ID %d, получен %d", id, user.ID)
	}

	expUser := users["alice@example.com"]
	if expUser.Name != user.Name || expUser.Age != user.Age || expUser.Email != user.Email {
		t.Errorf("ожидался пользователь %+v, а вернулся %+v", expUser, user)
	}
}

func TestPostUserHandler(t *testing.T) {
	deleteTestUsers(TestDB)
	_, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	newUser := model.User{Name: "New", Age: 1, Email: "new@example.com"}
	bytesNewUser, err := json.Marshal(newUser)
	if err != nil {
		t.Fatalf("ошибка при инкодировании пользователя JSON: %v", err)
	}

	url := testServer.URL + "/users"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bytesNewUser))
	if err != nil {
		t.Fatalf("ошибка при создании запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("ожидаемый статус в ответе 201, а вернулся: %v", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("ожидался Content-Type application/json, но получен: %s", resp.Header.Get("Content-Type"))
	}

	var user model.User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		t.Fatalf("ошибка при декодировании пользователя в JSON: %v", err)
	}

	if user.ID < 1 {
		t.Errorf("ожидаемый ID нового пользователя >1, а вернулся: %d", user.ID)
	}

	if user.Name != newUser.Name || user.Age != newUser.Age || user.Email != newUser.Email {
		t.Errorf("ожидаемый новый пользователь: %v, а вернулся: %v", newUser, user)
	}

	// проверка, что результат POST действительно записан в БД, а не просто вернулся "из воздуха"
	getResp, err := http.Get(fmt.Sprintf("%s/users/%d", testServer.URL, user.ID))
	if err != nil {
		t.Fatalf("ошибка при GET-запросе созданного пользователя: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("ожидался статус 200, а получен %d", getResp.StatusCode)
	}

	var getUser model.User
	err = json.NewDecoder(getResp.Body).Decode(&getUser)
	if err != nil {
		t.Fatalf("ошибка при декодировании ответа GET: %v", err)
	}

	if getUser != user {
		t.Errorf("пользователь, полученный через GET, не совпадает с тем, что вернул POST. POST: %+v, GET: %+v", user, getUser)
	}
}

func TestPutUserHandler(t *testing.T) {
	deleteTestUsers(TestDB)
	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	id := users["alice@example.com"].ID

	testServer := setupTestServer()
	defer testServer.Close()

	updatedUser := model.User{ID: id, Name: "New", Age: 1, Email: "new@example.com"}
	bytesNewUser, err := json.Marshal(updatedUser)
	if err != nil {
		t.Fatalf("ошибка при инкодировании пользователя JSON: %v", err)
	}

	url := testServer.URL + "/users/" + strconv.Itoa(id)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(bytesNewUser))
	if err != nil {
		t.Fatalf("ошибка при создании PUT-запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при выполнении PUT-запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ожидаемый статус в ответе 200, а вернулся: %v", resp.StatusCode)
	}

	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		t.Errorf("ожидался Content-Type application/json, но получен: %s", resp.Header.Get("Content-Type"))
	}

	var user model.User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		t.Fatalf("ошибка при декодировании пользователя в JSON: %v", err)
	}

	if user.ID != updatedUser.ID {
		t.Errorf("ожидаемый ID пользователя: %d, а вернулся: %d", updatedUser.ID, user.ID)
	}

	if user.Name != updatedUser.Name || user.Age != updatedUser.Age || user.Email != updatedUser.Email {
		t.Errorf("ожидаемый новый пользователь: %+v, а вернулся: %+v", updatedUser, user)
	}

	// проверка, что результат PUT действительно записан в БД, а не просто вернулся "из воздуха"
	getResp, err := http.Get(fmt.Sprintf("%s/users/%d", testServer.URL, user.ID))
	if err != nil {
		t.Fatalf("ошибка при GET-запросе обновленного пользователя: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("ожидался статус 200, а получен %d", getResp.StatusCode)
	}

	var getUser model.User
	err = json.NewDecoder(getResp.Body).Decode(&getUser)
	if err != nil {
		t.Fatalf("ошибка при декодировании ответа GET: %v", err)
	}

	if getUser != user {
		t.Errorf("пользователь, полученный через GET, не совпадает с тем, что вернул PUT. PUT: %+v, GET: %+v", user, getUser)
	}
}

func TestPatchUserHandler(t *testing.T) {
	deleteTestUsers(TestDB)
	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}
	originalUser := users["alice@example.com"]
	id := originalUser.ID

	testServer := setupTestServer()
	defer testServer.Close()

	updatedUser := model.PartialUser{ID: id, Email: model.StrPtr("new@example.com")}
	bytesNewUser, err := json.Marshal(updatedUser)
	if err != nil {
		t.Fatalf("ошибка при инкодировании пользователя JSON: %v", err)
	}

	url := testServer.URL + "/users/" + strconv.Itoa(id)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(bytesNewUser))
	if err != nil {
		t.Fatalf("ошибка при создании PATCH-запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при выполнении PATCH-запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ожидаемый статус в ответе 200, а вернулся: %v", resp.StatusCode)
	}

	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		t.Errorf("ожидался Content-Type application/json, но получен: %s", resp.Header.Get("Content-Type"))
	}

	var user model.User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		t.Fatalf("ошибка при декодировании пользователя в JSON: %v", err)
	}

	if user.ID != updatedUser.ID {
		t.Errorf("ожидаемый ID пользователя: %d, а вернулся: %d", updatedUser.ID, user.ID)
	}

	if user.Name != originalUser.Name || user.Age != originalUser.Age || user.Email != *updatedUser.Email {
		t.Errorf("ожидаемый новый пользователь: %+v, а вернулся: %+v", updatedUser, user)
	}

	// проверка, что результат PATCH действительно записан в БД, а не просто вернулся "из воздуха"
	getResp, err := http.Get(fmt.Sprintf("%s/users/%d", testServer.URL, user.ID))
	if err != nil {
		t.Fatalf("ошибка при GET-запросе обновленного пользователя: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("ожидался статус 200, а получен %d", getResp.StatusCode)
	}

	var getUser model.User
	err = json.NewDecoder(getResp.Body).Decode(&getUser)
	if err != nil {
		t.Fatalf("ошибка при декодировании ответа GET: %v", err)
	}

	if getUser != user {
		t.Errorf("пользователь, полученный через GET, не совпадает с тем, что вернул PATCH. PATCH: %+v, GET: %+v", user, getUser)
	}
}

func TestPatchUser_PartialUpdate_NoChange(t *testing.T) {
	deleteTestUsers(TestDB)
	users, _ := seedTestUsers(TestDB)

	original := users["alice@example.com"]

	testServer := setupTestServer()
	defer testServer.Close()

	// PATCH-запрос: ничего не передаём кроме ID
	patch := model.PartialUser{ID: original.ID}
	body, _ := json.Marshal(patch)

	req, err := http.NewRequest(http.MethodPatch, testServer.URL+"/users/"+strconv.Itoa(original.ID), bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("ошибка при создании PATCH-запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при выполнении PATCH-запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ожидался статус 200, а получен: %d", resp.StatusCode)
	}

	var updated model.User
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("ошибка при декодировании JSON: %v", err)
	}

	// Проверка: все поля остались прежними
	if updated.Name != original.Name {
		t.Errorf("имя не должно было измениться. ожидалось: %s, получено: %s", original.Name, updated.Name)
	}
	if updated.Age != original.Age {
		t.Errorf("возраст не должен был измениться. ожидалось: %d, получено: %d", original.Age, updated.Age)
	}
	if updated.Email != original.Email {
		t.Errorf("email не должен был измениться. ожидалось: %s, получено: %s", original.Email, updated.Email)
	}
}

func TestDeleteUserHandler(t *testing.T) {
	deleteTestUsers(TestDB)
	users, _ := seedTestUsers(TestDB)

	deletedID := users["alice@example.com"].ID

	testServer := setupTestServer()
	defer testServer.Close()

	url := testServer.URL + "/users/" + strconv.Itoa(deletedID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("ошибка при создании DELETE-запроса: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при выполнении DELETE-запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Fatalf("ожидался статус 204 или 200, а получен: %d", resp.StatusCode)
	}

	// можно проверить, что сервер не возвращает тело, если используется 204
	if resp.StatusCode == http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			t.Errorf("при статусе 204 тело ответа должно быть пустым, но получено: %s", body)
		}
	}

	getResp, err := http.Get(fmt.Sprintf("%s/users/%d", testServer.URL, deletedID))
	if err != nil {
		t.Fatalf("ошибка при GET-запросе удаленного пользователя: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("ожидался статус 404, что пользователь не найден, а получен %d", getResp.StatusCode)
	}
}

func TestGetUserByID_Negative_IDNotFound(t *testing.T) {
	deleteTestUsers(TestDB)

	row := TestDB.QueryRow("SELECT COALESCE(max(id), 0) + 1000 FROM users")
	var nonExistID int
	err := row.Scan(&nonExistID)
	if err != nil {
		t.Fatalf("ошибка при считывании данных из БД %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/users/" + strconv.Itoa(nonExistID))
	if err != nil {
		t.Fatalf("ошибка при запросе: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ожидался статус 404 (пользователь не найден), а вернулся %v", resp.StatusCode)
	}
}

func TestGetUserByID_Negative_InvalidID(t *testing.T) {
	deleteTestUsers(TestDB)
	_, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/users/abc")
	if err != nil {
		t.Fatalf("ошибка при запросе: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ожидался статус 400 или 404, а вернулся %v", resp.StatusCode)
	}
}

func TestPostUser_Negative_MissingFields(t *testing.T) {
	deleteTestUsers(TestDB)

	testServer := setupTestServer()
	defer testServer.Close()

	userCheckNotNull := model.User{}
	bytesUserCheckNotNull, err := json.Marshal(userCheckNotNull)
	if err != nil {
		t.Fatalf("ошибка при конвертации в JSON: %v", err)
	}

	url := testServer.URL + "/users"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bytesUserCheckNotNull))
	if err != nil {
		t.Fatalf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("ожидался статус 400 или 422, но получен: %v", resp.StatusCode)
	}
}

func TestPostUser_Negative_UniqueFields(t *testing.T) {
	deleteTestUsers(TestDB)
	_, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	userCheckEmail := model.User{Name: "New", Age: 1, Email: "alice@example.com"}
	bytesUserCheckEmail, err := json.Marshal(userCheckEmail)
	if err != nil {
		t.Fatalf("ошибка при конвертации в JSON: %v", err)
	}

	url := testServer.URL + "/users"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bytesUserCheckEmail))
	if err != nil {
		t.Fatalf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("ожидался статус 409 или 500, но получен: %v", resp.StatusCode)
	}
}

func TestPostUser_Negative_InvalidJSON(t *testing.T) {
	testServer := setupTestServer()
	defer testServer.Close()

	// Невалидный JSON — просто текст
	invalidJSON := []byte(`{invalid json...`)

	url := testServer.URL + "/users"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(invalidJSON))
	if err != nil {
		t.Fatalf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("ожидался статус 400, но получен: %d", resp.StatusCode)
	}
}

func TestPutUser_Negative_IDNotFound(t *testing.T) {
	deleteTestUsers(TestDB)

	row := TestDB.QueryRow("SELECT COALESCE(max(id), 0) + 1000 FROM users")
	var nonExistID int
	err := row.Scan(&nonExistID)
	if err != nil {
		t.Fatalf("ошибка при считывании данных из БД %v", err)
	}
	testUser := model.User{ID: nonExistID, Name: "New", Age: 1, Email: "new@example.com"}
	bytesTestUser, err := json.Marshal(testUser)
	if err != nil {
		t.Fatalf("ошибка при конвертации в JSON: %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	url := testServer.URL + "/users/" + strconv.Itoa(nonExistID)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(bytesTestUser))
	if err != nil {
		t.Fatalf("ошибка при создании запроса: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ожидался статус 404 (не найден ID %d), а получен %v", nonExistID, resp.StatusCode)
	}
}

func TestPutUser_Negative_EmptyFields(t *testing.T) {
	deleteTestUsers(TestDB)
	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}
	id := users["alice@example.com"].ID

	testServer := setupTestServer()
	defer testServer.Close()

	// PUT-запрос: ничего не передаём кроме ID
	alice := model.User{ID: id}
	body, err := json.Marshal(alice)
	if err != nil {
		t.Fatalf("ошибка при конвертации в JSON: %v", err)
	}

	url := testServer.URL + "/users/" + strconv.Itoa(id)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("ошибка при создании запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("ожидался статус 400 либо 422, а получен: %d", resp.StatusCode)
	}
}

func TestPutUser_Negative_UniqueFields(t *testing.T) {
	deleteTestUsers(TestDB)
	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	id := users["bob@example.com"].ID
	bob := model.User{
		ID:    id,
		Name:  "Bob", // добавь имя
		Age:   30,    // добавь возраст
		Email: "alice@example.com",
	}
	body, err := json.Marshal(bob)
	if err != nil {
		t.Fatalf("ошибка при конвертации в JSON: %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	url := testServer.URL + "/users/" + strconv.Itoa(id)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("ошибка при создании запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	// TODO: Разобрать ошибку и возвращать 409 только при нарушении уникальности
	// http.Error(w, "[SERVER] ошибка при PUT-обновлении пользователя в БД", http.StatusConflict)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("ожидался статус 409 или 500, но получен: %v", resp.StatusCode)
	}
}

func TestPatchUser_Negative_NoUpdatableFields(t *testing.T) {
	deleteTestUsers(TestDB)
	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}
	id := users["alice@example.com"].ID

	testServer := setupTestServer()
	defer testServer.Close()

	alice := model.PartialUser{ID: id}
	body, err := json.Marshal(alice)
	if err != nil {
		t.Fatalf("ошибка при конвертации в JSON: %v", err)
	}

	url := testServer.URL + "/users/" + strconv.Itoa(id)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("ошибка при создании запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	// теперь 200 OK — значит, PATCH ничего не изменил, и это допустимо
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ожидался статус 200, а получен: %v", resp.StatusCode)
	}
}

func TestPatchUser_Negative_UniqueFields(t *testing.T) {
	deleteTestUsers(TestDB)
	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	checkEmail := "alice@example.com"

	id := users["bob@example.com"].ID
	bob := model.PartialUser{ID: id, Email: &checkEmail}
	body, err := json.Marshal(bob)
	if err != nil {
		t.Fatalf("ошибка при конвертации в JSON: %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	url := testServer.URL + "/users/" + strconv.Itoa(id)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("ошибка при создании запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("ожидался статус 409 или 500, но получен: %v", resp.StatusCode)
	}
}

func TestPatchUser_Negative_IDNotFound(t *testing.T) {
	deleteTestUsers(TestDB)

	row := TestDB.QueryRow("SELECT COALESCE(max(id), 0) + 1000 FROM users")
	var nonExistID int
	err := row.Scan(&nonExistID)
	if err != nil {
		t.Fatalf("ошибка при считывании данных из БД %v", err)
	}

	name := "New"
	age := 1
	email := "new@example.com"

	testUser := model.PartialUser{ID: nonExistID, Name: &name, Age: &age, Email: &email}
	bytesTestUser, err := json.Marshal(testUser)
	if err != nil {
		t.Fatalf("ошибка при конвертации в JSON: %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	url := testServer.URL + "/users/" + strconv.Itoa(nonExistID)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(bytesTestUser))
	if err != nil {
		t.Fatalf("ошибка при создании запроса: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ожидался статус 404 (не найден ID %d), а получен %v", nonExistID, resp.StatusCode)
	}
}

func TestDeleteUser_Negative_IDNotFound(t *testing.T) {
	deleteTestUsers(TestDB)

	row := TestDB.QueryRow("SELECT COALESCE(max(id), 0) + 1000 FROM users")
	var nonExistID int
	err := row.Scan(&nonExistID)
	if err != nil {
		t.Fatalf("ошибка при считывании данных из БД %v", err)
	}

	testServer := setupTestServer()
	defer testServer.Close()

	url := testServer.URL + "/users/" + strconv.Itoa(nonExistID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("ошибка при создании запроса: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ожидался статус 404 (не найден ID %d), а получен %v", nonExistID, resp.StatusCode)
	}
}

func TestDelete_Negative_InvalidID(t *testing.T) {
	deleteTestUsers(TestDB)

	testServer := setupTestServer()
	defer testServer.Close()

	url := testServer.URL + "/users/abc"
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("ошибка при создании запроса: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ожидался статус 400 или 404, а вернулся %v", resp.StatusCode)
	}
}

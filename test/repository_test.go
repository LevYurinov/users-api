package test

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"pet/model"
	"pet/repository"
	"testing"
)

var logger = log.New(os.Stdout, "[TEST]", log.LstdFlags)

// TestDatabaseConnection проверяет, что тестовый сервер работает и БД подключена
func TestDatabaseConnection(t *testing.T) {
	if TestDB == nil {
		t.Fatal("TestDB не инициализирована")
	}
	err := TestDB.Ping()
	if err != nil {
		t.Fatalf("Ошибка подключения к тестовой БД: %v", err)
	}
}

// seedTestUsers - добавляет данные в таблицу тестовой БД
func seedTestUsers(testBD *sql.DB) (map[string]model.User, error) {
	query := `
	INSERT INTO users (name, age, email)
	VALUES
	    ('Alice', 30, 'alice@example.com'),
		('Bob', 25, 'bob@example.com')
	ON CONFLICT (email) DO NOTHING
	RETURNING id, name, age, email ;
`
	rows, err := testBD.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expected := make(map[string]model.User)

	for rows.Next() {
		var id, age int
		var name, email string

		err = rows.Scan(&id, &name, &age, &email)
		if err != nil {
			return nil, err
		}
		expected[email] = model.User{id, name, age, email}
	}

	if rows.Err() != nil {
		return nil, err
	}

	return expected, nil
}

// deleteTestUsers - очищает таблицу тестовой БД
func deleteTestUsers(testBD *sql.DB) {
	query := `
	DELETE FROM users;
	`

	_, err := testBD.Exec(query)
	if err != nil {
		log.Fatalf("не удалось очистить таблицу users: %v", err)
	}
}

// testRepo := repository.NewUserRepository(TestDB, testLogger) - в каждой тестовой функции

func TestGetUsers(t *testing.T) {
	deleteTestUsers(TestDB)

	expected, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("не удалось добавить пользователей в таблицу тестовой БД: %v", err)
	}

	testRepo := repository.NewUserRepository(TestDB, logger)

	users, err := testRepo.GetAllUsers()
	if err != nil {
		t.Fatalf("не удалось получить пользователей из таблицы тестовой БД: %v", err)
	}

	for _, user := range users {
		exp, ok := expected[user.Email]
		if !ok {
			t.Errorf("неожидаемый e-mail: %s", user.Email)
			continue
		}

		if user.Name != exp.Name || user.Age != exp.Age {
			t.Errorf("данные не совпадают для: %s. Получили name=%s, age=%d", user.Email, user.Name, user.Age)
		}
	}
}

func TestGetUserByID(t *testing.T) {
	deleteTestUsers(TestDB)

	expected, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД %v", err)
	}

	testRepo := repository.NewUserRepository(TestDB, logger)

	expectedUser := expected["alice@example.com"]
	user, err := testRepo.GetUserByID(expectedUser.ID)
	if err != nil {
		t.Errorf("не удалось получить пользователя по ID: %v", err)
		return
	}

	if user.Name != expectedUser.Name || user.Age != expectedUser.Age {
		t.Errorf("полученные данные не совпадают: ожидалось %+v, получено %+v", expected, user)
	}
}

func TestPostUser(t *testing.T) {
	deleteTestUsers(TestDB)

	testRepo := repository.NewUserRepository(TestDB, logger)
	user := model.User{
		Name: "Alice", Age: 30, Email: "alice@example.com",
	}

	newUser, err := testRepo.PostUser(user)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователя в таблицу тестовой БД: %v", err)
	}

	if newUser.ID == 0 {
		t.Error("ID не был установлен после добавления пользователя")
	}

	if newUser.Name != user.Name || newUser.Age != user.Age || newUser.Email != user.Email {
		t.Errorf("полученные данные не совпадают. Ожидалось %+v, получено %+v", user, newUser)
	}
}

func TestPutUser(t *testing.T) {
	deleteTestUsers(TestDB)

	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД %v", err)
	}

	updatedID := users["alice@example.com"].ID
	updatedUser := model.User{
		ID: updatedID, Name: "Put", Age: 100, Email: "put@example.com",
	}

	testRepo := repository.NewUserRepository(TestDB, logger)

	putUser, err := testRepo.PutUser(updatedUser)
	if err != nil {
		t.Fatalf("ошибка при получении пользователя из таблицы тестовой БД %v", err)
	}

	if putUser.ID != updatedUser.ID || putUser.Name != updatedUser.Name || putUser.Age != updatedUser.Age || putUser.Email != updatedUser.Email {
		t.Errorf("не совпадают данные пользователя. Ожидалось: %+v. Получили: %+v", updatedUser, putUser)
	}
}

func TestPatchUser(t *testing.T) {
	deleteTestUsers(TestDB)

	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}
	updatedID := users["alice@example.com"].ID

	testRepo := repository.NewUserRepository(TestDB, logger)

	var updatedUser = model.PartialUser{
		ID:    updatedID,
		Email: model.StrPtr("new@gmail.com"),
	}

	patchUser, err := testRepo.PatchUser(updatedUser)
	if err != nil {
		t.Fatalf("ошибка при обновлении пользователя в таблице тестовой БД: %v", err)
	}

	if patchUser.ID != updatedUser.ID || patchUser.Email != *updatedUser.Email {
		t.Errorf("не совпадают данные пользователя.\nОжидалось: ID=%v, email=%v.\nПолучили: ID=%v, email=%v.\n", updatedUser.ID, *updatedUser.Email, patchUser.ID, patchUser.Email)
	}

	if updatedUser.Name != nil && updatedUser.Age != nil {
		if patchUser.Name != *updatedUser.Name || patchUser.Age != *updatedUser.Age {
			t.Errorf("PATCH не должен менять поля name/age. Было: name=%v age=%v. Стало: name=%v age=%v",
				*updatedUser.Name, *updatedUser.Age, patchUser.Name, patchUser.Age)
		}
	}
}

func TestDeleteUser(t *testing.T) {
	deleteTestUsers(TestDB)

	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	deletedID := users["alice@example.com"].ID

	testRepo := repository.NewUserRepository(TestDB, logger)
	err = testRepo.DeleteUser(deletedID)
	if err != nil {
		t.Fatalf("ошибка при удалении пользователя из таблицы тестовой БД: %v", err)
	}

	deletedUser, err := testRepo.GetUserByID(deletedID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("ожидалась ошибка sql.ErrNoRows, но получили: %v", err)
	}

	if err == nil {
		t.Errorf("Пользователь с ID %d должен быть удалён, но найден: %+v", deletedID, deletedUser)

	}

	// Если хочешь ещё больше уверенности, можно добавить проверку, что 2-й пользователь остался:
	otherID := users["bob@example.com"].ID
	_, err = testRepo.GetUserByID(otherID)
	if err != nil {
		t.Errorf("пользователь с ID %d (не удаляемый) должен остаться, но не найден: %v", otherID, err)
	}
}

func TestGetUser_Negative_IsEmpty(t *testing.T) {
	deleteTestUsers(TestDB)

	testRepo := repository.NewUserRepository(TestDB, logger)
	users, err := testRepo.GetAllUsers()
	if err != nil {
		t.Fatalf("ошибка при получении списка пользователей из таблицы тестовой БД: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("ожидается пустой спискок {} без пользователей, а получили: %v", users)
	}
}

func TestGetUserByID_Negative_NotFound(t *testing.T) {
	deleteTestUsers(TestDB)

	_, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	row := TestDB.QueryRow("SELECT COALESCE(max(id), 0) + 1000 FROM users")
	var nonExistID int
	err = row.Scan(&nonExistID)

	testRepo := repository.NewUserRepository(TestDB, logger)
	_, err = testRepo.GetUserByID(nonExistID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Метод не возвращает ошибку, если пользователя не найдено в БД. Ожидалось: %v, вернулось %v", sql.ErrNoRows, err)
	}
}

func TestPostUser_Negative_Unique(t *testing.T) {
	deleteTestUsers(TestDB)
	_, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	userEmailChecking := model.User{Email: "alice@example.com"}

	testRepo := repository.NewUserRepository(TestDB, logger)
	_, err = testRepo.PostUser(userEmailChecking)
	if err == nil {
		t.Fatal("ожидалась ошибка из-за дублирования email, но err == nil")
	}
}

func TestPostUser_Negative_NotNull(t *testing.T) {
	deleteTestUsers(TestDB)
	_, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	testRepo := repository.NewUserRepository(TestDB, logger)

	userCheckNotNull := model.User{}
	_, err = testRepo.PostUser(userCheckNotNull)
	if err == nil {
		t.Fatal("ожидалась ошибка из-за добавлений пустых значений в NOT NULL поля, но err == nil")
	}
}

// TestPutUser_Negative_ID тест, что нет обновления по несуществующему ID
func TestPutUser_Negative_ID(t *testing.T) {
	deleteTestUsers(TestDB)

	nonExistentID := 99999 // ID, которого точно нет
	updatedUser := model.User{
		ID:    nonExistentID,
		Name:  "Ghost",
		Age:   42,
		Email: "ghost@example.com",
	}

	testRepo := repository.NewUserRepository(TestDB, logger)
	_, err := testRepo.PutUser(updatedUser)
	if err == nil {
		t.Fatalf("ожидалась ошибка при попытке обновить пользователя с несуществующим ID (%d), но err == nil", nonExistentID)
	}
}

func TestPutUser_Negative_Email(t *testing.T) {
	deleteTestUsers(TestDB)
	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	// Берём существующие email и id
	alice := users["alice@example.com"]
	bob := users["bob@example.com"]

	// Пытаемся присвоить Bob'у email от Alice
	conflictUser := model.User{
		ID:    bob.ID,
		Name:  "Bob Updated",
		Age:   35,
		Email: alice.Email, // дублирующий email
	}

	testRepo := repository.NewUserRepository(TestDB, logger)
	_, err = testRepo.PutUser(conflictUser)
	if err == nil {
		t.Fatalf("ожидалась ошибка дублирования e-mail, но err == nil")
	}
}

func TestPatchUser_Negative_Unique(t *testing.T) {
	deleteTestUsers(TestDB)
	users, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	checkID := users["alice@example.com"].ID
	duplicateEmail := users["bob@example.com"].Email
	userCheckID := model.PartialUser{
		ID:    checkID,
		Email: &duplicateEmail,
	}

	testRepo := repository.NewUserRepository(TestDB, logger)
	_, err = testRepo.PatchUser(userCheckID)
	if err == nil {
		t.Fatalf("ожидалась ошибка дублирования e-mail, но err == nil")
	}
}

// func TestPatchUser_Negative_Nil(t *testing.T) {
//	deleteTestUsers(TestDB)
//	users, err := seedTestUsers(TestDB)
//	if err != nil {
//		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
//	}
//
//	checkID := users["alice@example.com"].ID
//	userCheckID := model.PartialUser{
//		ID:    checkID,
//		Name:  nil,
//		Age:   nil,
//		Email: nil,
//	}
//
//	testRepo := repository.NewUserRepository(TestDB, logger)
//	_, err = testRepo.PatchUser(userCheckID)
//	if err != nil {
//		t.Fatalf("не ожидалась ошибка, но получена: %v", err) // поправил, убрал старую логику
//	}
//
//	//if err == nil {
//	//	t.Fatal("ожидалась ошибка (нет полей для обновления), но она не получена")
//	//}
//}

func TestDeleteUser_Negative_ID(t *testing.T) {
	deleteTestUsers(TestDB)

	_, err := seedTestUsers(TestDB)
	if err != nil {
		t.Fatalf("ошибка при добавлении пользователей в таблицу тестовой БД: %v", err)
	}

	var id int
	row := TestDB.QueryRow("SELECT COALESCE(max(id), 0) + 1000 FROM users")
	err = row.Scan(&id)
	if err != nil {
		t.Fatalf("ошибка при получении несуществующего ID из таблицы тестовой БД: %v", err)
	}

	testRepo := repository.NewUserRepository(TestDB, logger)
	err = testRepo.DeleteUser(id)

	if err == nil {
		t.Fatalf("ожидалась ошибка при удалении несуществующего ID, но err == nil")
	}

	expected := fmt.Sprintf("пользователь с id %d не найден", id)
	if err.Error() != expected {
		t.Errorf("ошибка не соответствует ожидаемой.\nОжидали: %q\nПолучили: %q", expected, err.Error())
	}
}

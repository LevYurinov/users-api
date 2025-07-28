package test

import (
	"database/sql"
	"log"
	"os"
	"pet/internal/database"
	"testing"
)

// Глобальная переменная TestDB,
// чтобы к ней можно было обращаться из разных тестов внутри test пакета.
// Это экземпляр подключения к test_users базе.
var TestDB *sql.DB

func TestMain(m *testing.M) { // m - менеджер тестов
	var err error
	TestDB, err = database.ConnectDB("test_users")
	if err != nil {
		log.Fatalf("не удалось подключиться к тестовой БД: %v", err)
	}

	err = setupTestSchema(TestDB)
	if err != nil {
		log.Fatalf("не удалось создать схему таблицы users в тестовой БД %v", err)
	}

	defer TestDB.Close()
	os.Exit(m.Run())
}

func setupTestSchema(testDB *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
	       id SERIAL PRIMARY KEY,
	       name TEXT NOT NULL,
	       age INT NOT NULL,
	       email TEXT UNIQUE NOT NULL
	)
`
	_, err := testDB.Exec(query)
	return err
}

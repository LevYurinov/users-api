package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func ConnectDB(dbName string) (*sql.DB, error) {
	fmt.Println("[DB] Попытка подключения к базе", dbName) // отладка

	connStr := fmt.Sprintf("host=localhost port=5432 user=postgres password=postgres dbname=%s sslmode=disable", dbName)
	fmt.Println("[DB] строка подключения:", connStr)

	connStr = os.Getenv("POSTGRES_DSN")
	if connStr == "" {
		return nil, fmt.Errorf("переменная POSTGRES_DSN не установлена")
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("[DATA BASE] ошибка открытия соединения в БД %v: %w", dbName, err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("[DATA BASE] ошибка соединения с БД %v: %w", dbName, err)
	}

	fmt.Printf("✅ Соединение с БД %v установлено!", dbName)
	return db, nil
}

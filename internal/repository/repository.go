// repository будет предоставлять методы работы с БД
package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"pet/internal/model"
	"strings"
)

// UserRepository — это уровень доступа к данным (Data Access Layer).
// Он знает, как общаться с базой, выполнять CRUD операции, но не знает бизнес-правил
type UserRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewUserRepository(db *sql.DB, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		db:  db,
		log: logger,
	}
}

// GetAllUsers - получает весь список пользователей
func (r *UserRepository) GetAllUsers() ([]model.User, error) {
	query := `
SELECT id, name, age, email 
FROM users
`
	rows, err := r.db.Query(query)
	if err != nil {
		r.log.Error("failed to execute SELECT users",
			zap.Error(err),
			zap.String("component", "repository"),
			zap.String("event", "GetAllUsers"))

		return nil, fmt.Errorf("repository/GetAllUsers: %w", err)
	}
	defer func() {
		err = rows.Close()
		if err != nil {
			r.log.Warn("failed to close rows",
				zap.String("component", "repository"),
				zap.String("event", "GetAllUsers"))
		}
	}()

	var users []model.User

	for rows.Next() {
		var user model.User

		err = rows.Scan(&user.ID, &user.Name, &user.Age, &user.Email)
		if err != nil {
			r.log.Error("failed to scan user row",
				zap.Error(err),
				zap.String("component", "repository"),
				zap.String("event", "GetAllUsers"))

			return nil, fmt.Errorf("repository/GetAllUsers: %w", err)
		}
		users = append(users, user)
	}

	err = rows.Err()
	if err != nil {
		r.log.Error("rows iteration error",
			zap.Error(err),
			zap.String("component", "repository"),
			zap.String("event", "GetAllUsers"))

		return nil, fmt.Errorf("repository/GetAllUsers: %w", err)
	}

	return users, nil
}

// GetUserByID получает пользователя по его ID
func (r *UserRepository) GetUserByID(id int) (model.User, error) {
	query := `
	SELECT id, name, age, email, password
	FROM users
	WHERE id = $1
`
	row := r.db.QueryRow(query, id)

	var user model.User
	err := row.Scan(&user.ID, &user.Name, &user.Age, &user.Email, &user.HashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // если польз. не найден в БД
			r.log.Info("user not found",
				zap.Int("id", id),
				zap.String("component", "repository"),
				zap.String("event", "GetUserByID"))

			return model.User{}, fmt.Errorf("user with id %d not found", id)
		}

		r.log.Error("failed to scan user ID", // если ошибка по другой причине
			zap.Error(err),
			zap.String("component", "repository"),
			zap.String("event", "GetUserByID"))

		return model.User{}, fmt.Errorf("repository/GetUserByID: %w", err)
	}

	return user, nil
}

// GetUserByEmail получает пользователя по e-mail
func (r *UserRepository) GetUserByEmail(email string) (model.User, error) {
	query := `
	SELECT id, name, age, email, password
	FROM users
	WHERE email = $1
`
	row := r.db.QueryRow(query, email)

	var loginUser model.User
	err := row.Scan(&loginUser.ID, &loginUser.Name, &loginUser.Age, &loginUser.Email, &loginUser.HashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // если польз. не найден в БД
			r.log.Info("user not found",
				zap.String("email", loginUser.Email),
				zap.String("component", "repository"),
				zap.String("event", "GetUserByEmail"))

			return model.User{}, fmt.Errorf("user with email %s not found", loginUser.Email)
		}

		r.log.Error("failed to scan user email", // если ошибка по другой причине
			zap.Error(err),
			zap.String("component", "repository"),
			zap.String("event", "GetUserByEmail"))

		return model.User{}, fmt.Errorf("repository/GetUserByEmail: %w", err)
	}
	return loginUser, nil
}

// PostUser добавляет пользователя в БД
func (r *UserRepository) PostUser(createUser model.User) (model.User, error) {

	if createUser.Name == "" || createUser.Email == "" || createUser.Age <= 0 || createUser.HashedPassword == "" {
		r.log.Info("not all fields filled",
			zap.String("component", "repository"),
			zap.String("event", "PostUser"))

		return model.User{}, fmt.Errorf("not all fields fill")
	}

	query := `
	INSERT INTO users (name, age, email, password)
	VALUES ($1, $2, $3, $4)
	RETURNING id
`
	row := r.db.QueryRow(query,
		createUser.Name,
		createUser.Age,
		createUser.Email,
		createUser.HashedPassword)

	var id int
	err := row.Scan(&id)
	if err != nil {
		r.log.Error("failed to insert user",
			zap.Error(err),
			zap.String("component", "repository"),
			zap.String("event", "PostUser"))

		return model.User{}, fmt.Errorf("repository/PostUser: %w", err)
	}

	backUser, _ := r.GetUserByID(id)
	backUser.HashedPassword = ""

	return backUser, nil
}

// PutUser полностью обновляет пользователя в БД
func (r *UserRepository) PutUser(updateUser model.User) (model.User, error) {
	_, err := r.GetUserByID(updateUser.ID)
	if err != nil {
		r.log.Error("user not found by ID", // если ошибка по другой причине
			zap.Error(err),
			zap.String("component", "repository"),
			zap.String("event", "PutUser"))

		return model.User{}, fmt.Errorf("repository/PutUser: %w", err)
	}

	query := `
	UPDATE users
	SET name = $1, age = $2, email = $3
	WHERE id = $4
	RETURNING id, name, age, email
`
	var user model.User

	err = r.db.QueryRow(query, updateUser.Name, updateUser.Age, updateUser.Email, updateUser.ID).
		Scan(&user.ID, &user.Name, &user.Age, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			r.log.Info("user not found",
				zap.String("component", "repository"),
				zap.String("event", "PutUser"))

			return model.User{}, fmt.Errorf("repository/PutUser: %w", err)
		}

		r.log.Error("failed to update user",
			zap.Error(err),
			zap.String("component", "repository"),
			zap.String("event", "PutUser"))

		return model.User{}, fmt.Errorf("repository/PutUser: %w", err)
	}

	return user, nil
}

// PatchUser частично обновляет пользователя в БД
func (r *UserRepository) PatchUser(updateUser model.PartialUser) (model.User, error) {
	setParts := []string{} // фрагменты SQL типа name = $1
	args := []any{}        // значения на место $1, $2
	argIdx := 1            // индекс для SQL-плейсхолдеров ($1, $2, ...)

	if updateUser.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *updateUser.Name)
		argIdx++
	}

	if updateUser.Age != nil {
		setParts = append(setParts, fmt.Sprintf("age = $%d", argIdx))
		args = append(args, *updateUser.Age)
		argIdx++
	}

	if updateUser.Email != nil {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, *updateUser.Email)
		argIdx++
	}

	// Если PATCH не содержит новых полей, логичнее не падать с ошибкой,
	// а просто вернуть текущую версию пользователя (ничего ведь не изменилось).
	if len(setParts) == 0 {
		return r.GetUserByID(updateUser.ID)
	}

	//// вернул проверку
	//if len(setParts) == 0 {
	//	return model.User{}, fmt.Errorf("нет полей для обновления")
	//}

	query := fmt.Sprintf(`
	UPDATE users
	SET %s
	WHERE id = $%d
`, strings.Join(setParts, ", "), argIdx)

	args = append(args, updateUser.ID)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		r.log.Error("failed to update user",
			zap.Error(err),
			zap.String("component", "repository"),
			zap.String("event", "PatchUser"))

		return model.User{}, fmt.Errorf("repository/PatchUser: %w", err)
	}

	updatedUser, err := r.GetUserByID(updateUser.ID)
	if err != nil {
		r.log.Error("user not found by ID",
			zap.Error(err),
			zap.String("component", "repository"),
			zap.String("event", "PatchUser"))

		return model.User{}, fmt.Errorf("repository/PatchUser: %w", err)
	}
	return updatedUser, nil // пользователь частично обновлен
}

// DeleteUser удаляет пользователя в БД
func (r *UserRepository) DeleteUser(id int) error {
	query := `
	DELETE FROM users
	WHERE id = $1
`
	result, err := r.db.Exec(query, id)
	if err != nil {
		r.log.Error("user not found",
			zap.Error(err),
			zap.Int("id", id),
			zap.String("component", "repository"),
			zap.String("event", "DeleteUser"))

		return fmt.Errorf("repository/DeleteUser: %s", "user not found")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.log.Info("user not found",
			zap.Int("id", id),
			zap.String("component", "repository"),
			zap.String("event", "DeleteUser"))

		return fmt.Errorf("repository/DeleteUser: %w", err)
	}
	return nil
}

func (r *UserRepository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

// WithdrawBalance списывает средства со счета
func (r *UserRepository) WithdrawBalance(tx *sql.Tx, senderID int, amount float64) error {
	// tx типа *sql.Tx — специальный объект, через который нужно делать SQL-запросы внутри транзакции

	var currentBalance float64

	row := tx.QueryRow("SELECT balance FROM users WHERE id = $1", senderID)

	err := row.Scan(&currentBalance)
	if err != nil {
		r.log.Error("user not found",
			zap.Error(err),
			zap.Int("id", senderID),
			zap.String("component", "repository"),
			zap.String("event", "WithdrawBalance"))

		return fmt.Errorf("repository/WithdrawBalance: %w", err)
	}

	if currentBalance < amount {
		r.log.Info("user has no enough founds",
			zap.Int("id", senderID),
			zap.String("component", "repository"),
			zap.String("event", "WithdrawBalance"))

		return fmt.Errorf("repository/WithdrawBalance: user has no enough founds: %w", err)
	}

	_, err = tx.Exec("UPDATE users SET balance = balance - $1 WHERE id = $2", amount, senderID)
	if err != nil {
		r.log.Error("withdraw error",
			zap.Error(err),
			zap.Int("id", senderID),
			zap.String("component", "repository"),
			zap.String("event", "WithdrawBalance"))

		return fmt.Errorf("repository/WithdrawBalance: %w", err)
	}

	return nil
}

// DepositBalance зачисляет средства на счет
func (r *UserRepository) DepositBalance(tx *sql.Tx, receiverID int, amount float64) error {

	var currentBalance float64

	row := tx.QueryRow("SELECT balance FROM users WHERE id = $1", receiverID)

	err := row.Scan(&currentBalance)
	if err != nil {
		r.log.Error("user not found",
			zap.Error(err),
			zap.Int("id", receiverID),
			zap.String("component", "repository"),
			zap.String("event", "DepositBalance"))

		return fmt.Errorf("repository/DepositBalance: %w", err)
	}

	_, err = tx.Exec("UPDATE users SET balance = balance + $1 WHERE id = $2", amount, receiverID)
	if err != nil {
		r.log.Error("deposit error",
			zap.Error(err),
			zap.Int("id", receiverID),
			zap.String("component", "repository"),
			zap.String("event", "DepositBalance"))

		return fmt.Errorf("repository/DepositBalance: %w", err)
	}

	return nil
}

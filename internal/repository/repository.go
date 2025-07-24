// repository будет предоставлять методы работы с БД
package repository

import (
	"database/sql"
	"fmt"
	"log"
	"pet/internal/model"
	"strings"
)

// UserRepository — это уровень доступа к данным (Data Access Layer).
// Он знает, как общаться с базой, выполнять CRUD операции, но не знает бизнес-правил
type UserRepository struct {
	db           *sql.DB
	errorsLogger *log.Logger
}

func NewUserRepository(db *sql.DB, errorsLogger *log.Logger) *UserRepository {
	return &UserRepository{
		db:           db,
		errorsLogger: errorsLogger,
	}
}

// GetAllUsers получает весь список пользователей
func (r *UserRepository) GetAllUsers() ([]model.User, error) {
	query := `
SELECT id, name, age, email 
FROM users
`
	rows, err := r.db.Query(query)
	if err != nil {
		r.errorsLogger.Printf("[DB ERROR] не удалось выполнить SELECT %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []model.User

	for rows.Next() {
		var user model.User

		err = rows.Scan(&user.ID, &user.Name, &user.Age, &user.Email)
		if err != nil {
			r.errorsLogger.Printf("[DB ERROR] ошибка при сканировании строки SELECT-запроса %v", err)
			return nil, err
		}
		users = append(users, user)
	}

	err = rows.Err()
	if err != nil {
		r.errorsLogger.Printf("[DB ERROR] ошибка при считывании данных из БД %v", err)
		return nil, err
	}

	return users, nil
}

// GetUserByID получает весь список пользователей
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
		r.errorsLogger.Printf("[DB ERROR] ошибка при выполнении SELECT-запроса: %v", err)
		return model.User{}, err
	}
	return user, nil
}

// GetUserByEmail возвращает пользователя по e-mail
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
		r.errorsLogger.Printf("[DB ERROR] ошибка при выполнении SELECT-запроса: %v", err)
		return model.User{}, err
	}
	return loginUser, nil
}

// PostUser добавляет пользователя в БД
func (r *UserRepository) PostUser(createUser model.User) (model.User, error) {

	if createUser.Name == "" || createUser.Email == "" || createUser.Age <= 0 || createUser.HashedPassword == "" {
		return model.User{}, fmt.Errorf("все поля обязательны для заполнения")
	}

	query := `
	INSERT INTO users (name, age, email, password)
	VALUES ($1, $2, $3, $4)
	RETURNING id
`
	row := r.db.QueryRow(query, createUser.Name, createUser.Age, createUser.Email, createUser.HashedPassword)

	var id int
	err := row.Scan(&id)
	if err != nil {
		r.errorsLogger.Printf("[DB ERROR] ошибка при добавлении пользователя в БД: %v", err)
		return model.User{}, err
	}

	backUser, _ := r.GetUserByID(id)
	backUser.HashedPassword = ""

	return backUser, nil
}

// PutUser полностью обновляет пользователя в БД
func (r *UserRepository) PutUser(updateUser model.User) (model.User, error) {
	_, err := r.GetUserByID(updateUser.ID)
	if err != nil {
		r.errorsLogger.Printf("[DB ERROR] не найден ID обновляемого пользователя в БД: %v", err)
		return model.User{}, err
	}

	query := `
	UPDATE users
	SET name = $1, age = $2, email = $3
	WHERE id = $4
`
	_, err = r.db.Exec(query, updateUser.Name, updateUser.Age, updateUser.Email, updateUser.ID)
	if err != nil {
		r.errorsLogger.Printf("[DB ERROR] ошибка при PUT-обновлении пользователя: %v", err)
		return model.User{}, err
	}

	updatedUser, err := r.GetUserByID(updateUser.ID)
	if err != nil { // хотя, эта ошибка уже обрабатывается на более раннем этапе, лучше убрать обр. ошибки?
		r.errorsLogger.Printf("[DB ERROR] ошибка при выполнении SELECT-запроса: %v", err)
	}

	return updatedUser, nil
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
		r.errorsLogger.Printf("[DB ERROR] ошибка при PATCH: %v", err)
		return model.User{}, err
	}

	updatedUser, err := r.GetUserByID(updateUser.ID)
	if err != nil {
		r.errorsLogger.Printf("[DB ERROR] ошибка при SELECT запросе PATCH-обновления пользователя: %v", err)
		return model.User{}, err
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
		r.errorsLogger.Printf("[DB ERROR] удаляемый пользователь не найден в БД: %v", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("пользователь с id %d не найден", id)
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
		return fmt.Errorf("пользователь не найден: %w", err)
	}

	if currentBalance < amount {
		return fmt.Errorf("недостаточно средств у пользователя %d", senderID)
	}

	_, err = tx.Exec("UPDATE users SET balance = balance - $1 WHERE id = $2", amount, senderID)
	if err != nil {
		return fmt.Errorf("ошибка при списании средств: %w", err)
	}

	return nil
}

// DepositBalance зачисляет средства на счет
func (r *UserRepository) DepositBalance(tx *sql.Tx, receiverID int, amount float64) error {
	var currentBalance float64
	row := tx.QueryRow("SELECT balance FROM users WHERE id = $1", receiverID)
	err := row.Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("пользователь не найден: %w", err)
	}

	_, err = tx.Exec("UPDATE users SET balance = balance + $1 WHERE id = $2", amount, receiverID)
	if err != nil {
		return fmt.Errorf("ошибка при зачислении средств: %w", err)
	}

	return nil
}

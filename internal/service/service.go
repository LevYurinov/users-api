package service

import (
	"database/sql"
	"fmt"
	"go.uber.org/zap"
	"pet/internal/model"
)

const op = "users.service"

// UserRepository определяет контракт для взаимодействия с хранилищем пользователей.
// Он абстрагирует слой сервиса от конкретной реализации репозитория - PostgreSQL.
type UserRepository interface {
	GetAllUsers() ([]model.User, error)
	GetUserByID(id int) (model.User, error)
	PostUser(createUser model.User) (model.User, error)
	PutUser(updateUser model.User) (model.User, error)
	PatchUser(updateUser model.PartialUser) (model.User, error)
	DeleteUser(id int) error
	BeginTx() (*sql.Tx, error)
	WithdrawBalance(tx *sql.Tx, senderID int, amount float64) error
	DepositBalance(tx *sql.Tx, receiverID int, amount float64) error
	// другие методы...
}

// UserService реализует бизнес-логику работы с пользователями.
// Он использует UserRepository для доступа к данным и инкапсулирует операции,
// которые выходят за рамки простых CRUD-методов (например, перевод средств).
// А также содержит экземпляр глобального логгера
type UserService struct {
	repo UserRepository
	log  *zap.Logger
}

// NewUserService создаёт и возвращает новый экземпляр UserService.
// Принимает реализацию UserRepository и логгер zap для ведения логов.
func NewUserService(repo UserRepository, logger *zap.Logger) *UserService {
	return &UserService{
		repo: repo,
		log:  logger,
	}
}

// TransferFunds переводит указанную сумму со счёта отправителя на счёт получателя.
// Операция выполняется в транзакции и либо полностью завершается, либо полностью откатывается.
// Возвращает ошибку в случае проблем с началом транзакции, списанием, зачислением или коммитом.
func (s *UserService) TransferFunds(senderID int, receiverID int, amount float64) (err error) {
	tx, err := s.repo.BeginTx()
	if err != nil {
		return fmt.Errorf("%s.TransferFunds: begin transaction error: %w", op, err)
	}

	defer func() {
		if err != nil {

			rbErr := tx.Rollback()
			if rbErr != nil {
				s.log.Error("rollback transaction error",
					zap.Error(rbErr),
					zap.String("component", "service"),
					zap.String("event", "TransferFunds"))
			}
		}
	}()

	err = s.repo.WithdrawBalance(tx, senderID, amount)
	if err != nil {
		return fmt.Errorf("%s.TransferFunds: withdraw error: %w", op, err)
	}

	err = s.repo.DepositBalance(tx, receiverID, amount)
	if err != nil {
		return fmt.Errorf("%s.TransferFunds: deposit error: %w", op, err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("%s.TransferFunds: canceled, transaction error : %w", op, err)
	}

	return nil
}

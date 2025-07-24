package service

import (
	"database/sql"
	"fmt"
	"log"
	"pet/model"
)

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

type UserService struct {
	repo UserRepository
}

func (s *UserService) TransferFunds(senderID int, receiverID int, amount float64) (err error) {
	tx, err := s.repo.BeginTx()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}

	defer func() {
		if err != nil {

			rbErr := tx.Rollback()
			if rbErr != nil {
				log.Fatalf("ошибка при откате транзакции: %v", rbErr)
			}
		}
	}()

	err = s.repo.WithdrawBalance(tx, senderID, amount)
	if err != nil {
		return fmt.Errorf("не удалось списать средства: %w", err)
	}

	err = s.repo.DepositBalance(tx, receiverID, amount)
	if err != nil {
		return fmt.Errorf("не удалось зачислить средства: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("отмена. Транзакция не прошла из-за ошибок: %w", err)
	}

	return nil
}

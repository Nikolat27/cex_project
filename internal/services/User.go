package services

import (
	"cex_project/internal/core"
	"context"
	"database/sql"
	"errors"
)

func (s *Services) DoesUserExist(ctx context.Context, userID int64) (bool, error) {
	_, err := s.DB.Models.User.GetByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *Services) CreateUser(ctx context.Context) (*core.User, error) {
	user, err := s.DB.Models.User.Create(ctx)
	if err != nil {
		return nil, err
	}

	newUser := core.NewUser(uint64(user.ID))
	
	if _, err = s.Exchange.CreateUser(newUser.ID); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *Services) Deposit(ctx context.Context, userID int64, asset string, amount int64) error {
	return s.DB.Models.Balance.Deposit(ctx, userID, asset, amount)
}

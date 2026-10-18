package services

import (
	"cex_project/internal/core"
	"cex_project/internal/database"
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
    if err := s.Exchange.Deposit(uint64(userID), core.Asset(asset), amount); err != nil {
		return err
	}

	balance, err := s.DB.Models.Balance.Get(ctx, userID, asset)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if balance == nil {
		if err := s.DB.Models.Balance.Create(ctx, &database.Balance{
			UserID: userID,
			Asset:  asset,
		}); err != nil {
			return err
		}
	}

	if err := s.DB.Models.Balance.Deposit(ctx, userID, asset, amount); err != nil {
		return err
	} 

	return nil
}

func (s *Services) Withdraw(ctx context.Context, userID int64, asset string, amount  int64) error {
	doesUserExist, err := s.DoesUserExist(ctx, userID)
	if err != nil {
		return err
	}

	if !doesUserExist {
		return errors.New("user not found")
	}
	
	if err := s.Exchange.Withdraw(uint64(userID), core.Asset(asset), amount); err != nil {
		return err
	}

    if err := s.DB.Models.Balance.Withdraw(ctx, userID, asset, amount); err != nil {
		return err
	}

	return nil
}

func (s *Services) GetBalance(ctx context.Context, userID int64, asset string) (int64, error) {
	balance, err := s.DB.Models.Balance.Get(ctx, userID, asset)
	if err != nil {
		return 0, err
	}

	return balance.Available, nil
}

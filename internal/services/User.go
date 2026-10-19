package services

import (
	"context"
	"database/sql"
	"errors"
	"cex_project/internal/core"
	"cex_project/internal/database"
	"cex_project/internal/utils"
)

func (s *Services) DoesUserExist(ctx context.Context, userID int64) (bool, error) {
	_, err := s.DB.Models.User.GetByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, utils.DatabaseError(err)
	}

	return true, nil
}

func (s *Services) CreateUser(ctx context.Context, username, password string) (*core.User, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	passwordHash, err := utils.Hash(password)
	if err != nil {
		return nil, err
	}

	user, err := s.DB.Models.User.Create(ctx, username, passwordHash)
	if err != nil {
		return nil, utils.DatabaseError(err)
	}

	newUser := core.NewUser(uint64(user.ID))
	newUser.Username = username

	if _, err = s.Exchange.CreateUser(newUser.ID); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *Services) Deposit(ctx context.Context, userID int64, asset string, amount int64) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if err := s.Exchange.Deposit(uint64(userID), core.Asset(asset), amount); err != nil {
		return err
	}

	balance, err := s.DB.Models.Balance.Get(ctx, userID, asset)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return utils.DatabaseError(err)
	}

	if balance == nil {
		if err := s.DB.Models.Balance.Create(ctx, &database.Balance{
			UserID: userID,
			Asset:  asset,
		}); err != nil {
			return utils.DatabaseError(err)
		}
	}

	if err := s.DB.Models.Balance.Deposit(ctx, userID, asset, amount); err != nil {
		return utils.DatabaseError(err)
	}

	return nil
}

func (s *Services) Withdraw(ctx context.Context, userID int64, asset string, amount int64) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

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
		return utils.DatabaseError(err)
	}

	return nil
}

func (s *Services) GetBalance(ctx context.Context, userID int64,
	asset string) (*core.Balance, error) {

	balance, err := s.DB.Models.Balance.Get(ctx, userID, asset)
	if err != nil {
		return nil, utils.DatabaseError(err)
	}

	return &core.Balance{
		Available: balance.Available,
		Locked:    balance.Locked,
	}, nil
}

func (s *Services) Login(ctx context.Context, username, password string) (string, error) {
	if username == "" || password == "" {
		return "", errors.New("username and password are required")
	}

	user, err := s.DB.Models.User.GetByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("user not found with this username, Please register first")
	}

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", utils.DatabaseError(err)
	}

	if err := utils.VerifyHash(password, user.PasswordHash); err != nil {
		return "", errors.New("invalid password")
	}

    token, err := s.Paseto.CreateToken(user.ID, user.Username)
	if err != nil {
		return "", err
	}

	return token, nil
}

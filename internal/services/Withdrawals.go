package services

import (
	"cex_project/internal/core"
	"cex_project/internal/database"
	"cex_project/internal/utils"
	"context"
	"database/sql"
	"errors"
	"strconv"
)

var ErrBalanceNotFound = errors.New("balance not found")
var ErrInsufficientBalance = errors.New("insufficient balance")

func (s *Services) CreateWithdrawal(ctx context.Context, userID int64, symbol, network,
	 destination string, amount int64) error {
		
	_, err := s.DB.Models.Balance.Get(ctx, userID, symbol)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrBalanceNotFound
	}
	if err != nil {
		return utils.DatabaseError(err)
	}

	withdrawal := &database.Withdrawal{
		UserID:      userID,
		Symbol:      symbol,
		Network:     network,
		Destination: destination,
		Amount:      strconv.FormatInt(amount, 10),
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	user := s.Exchange.Accounts.GetUser(uint64(userID))
	if user == nil {
		return errors.New("user not found")
	}

	if err := user.Lock(core.Asset(symbol), amount); err != nil {
		return ErrInsufficientBalance
	}

	if err := s.DB.Models.Balance.Lock(ctx, userID, symbol, amount); err != nil {
		if unlockErr := user.Unlock(core.Asset(symbol), amount); unlockErr != nil {
			return unlockErr
		}

		if errors.Is(err, utils.ErrRequireOneRowAffected) {
			return ErrInsufficientBalance
		}

		return utils.DatabaseError(err)
	}

	if err := s.DB.Models.Withdrawal.Create(ctx, withdrawal); err != nil {
		if unlockErr := user.Unlock(core.Asset(symbol), amount); unlockErr != nil {
			return unlockErr
		}

		if unlockErr := s.DB.Models.Balance.Unlock(ctx, userID, symbol, amount); unlockErr != nil {
			return utils.DatabaseError(unlockErr)
		}

		return utils.DatabaseError(err)
	}

	return nil
}

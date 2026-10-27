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
var ErrWithdrawalNotFound = errors.New("withdrawal not found")
var ErrWithdrawalNotCancellable = errors.New("withdrawal cannot be cancelled")

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

func (s *Services) ListWithdrawals(ctx context.Context, userID int64) ([]database.Withdrawal, error) {
	withdrawals, err := s.DB.Models.Withdrawal.ListByUserID(ctx, userID)
	if err != nil {
		return nil, utils.DatabaseError(err)
	}

	return withdrawals, nil
}

func (s *Services) CancelWithdrawal(ctx context.Context, userID, withdrawalID int64) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	withdrawal, err := s.DB.Models.Withdrawal.Get(ctx, withdrawalID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrWithdrawalNotFound
	}
	if err != nil {
		return utils.DatabaseError(err)
	}

	if withdrawal.UserID != userID {
		return ErrWithdrawalNotFound
	}

	if withdrawal.Status != "requested" {
		return ErrWithdrawalNotCancellable
	}

	amount, err := utils.StringToInt64(withdrawal.Amount)
	if err != nil {
		return err
	}

	user := s.Exchange.Accounts.GetUser(uint64(userID))
	if user == nil {
		return errors.New("user not found")
	}

	if err := s.DB.Models.Withdrawal.SetStatus(ctx, withdrawalID, "requested", "cancelled"); err != nil {
		if errors.Is(err, utils.ErrRequireOneRowAffected) {
			return ErrWithdrawalNotCancellable
		}
		
		return utils.DatabaseError(err)
	}

	if err := user.Unlock(core.Asset(withdrawal.Symbol), amount); err != nil {
		_ = s.DB.Models.Withdrawal.SetStatus(ctx, withdrawalID, "cancelled", "requested")
		return err
	}

	if err := s.DB.Models.Balance.Unlock(ctx, userID, withdrawal.Symbol, amount); err != nil {
		_ = user.Lock(core.Asset(withdrawal.Symbol), amount)
		_ = s.DB.Models.Withdrawal.SetStatus(ctx, withdrawalID, "cancelled", "requested")
		return utils.DatabaseError(err)
	}

	return nil
}

package services

import (
	"cex_project/internal/database"
	"cex_project/internal/wallet"
	"context"
	"database/sql"
	"errors"
	"os"
)

func (s *Services) GetOrCreateDepositAddress(ctx context.Context, userID int64,
	 symbol, network string) (*database.DepositAddress, error) {

	existing, err := s.DB.Models.DepositAddress.Get(ctx, userID, symbol, network)
	if err == nil {
		return existing, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if symbol != "ETH" || network != "sepolia" {
		return nil, errors.New("unsupported symbol or network")
	}

	index, err := s.DB.Models.DepositAddress.NextIndex(ctx)
	if err != nil {
		return nil, err
	}

    address, err := wallet.EthereumAddress(os.Getenv("ETH_XPUB"), uint32(index))
	if err != nil {
		return nil, err
	}

	if err := s.DB.Models.DepositAddress.Create(ctx, userID,
		 symbol, network, address, index); err != nil {

		return nil, err
	}

    return s.DB.Models.DepositAddress.Get(ctx, userID, symbol, network)
}

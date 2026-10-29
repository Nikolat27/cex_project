package services

import (
	"cex_project/internal/utils"
	"cex_project/internal/wallet"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func (s *Services) SepoliaWatcher(ctx context.Context) error {
	addresses, err := s.DB.Models.DepositAddress.ListByNetwork(ctx, "ETH", "sepolia")
	if err != nil {
		return err
	}

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	if rpcURL == "" {
		return errors.New("SEPOLIA_RPC_URL is not set")
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return err
	}
	defer client.Close()

	latest, err := client.BlockNumber(ctx)
	if err != nil {
		return err
	}

	start, err := s.GetSepoliaBlockNumber(ctx, latest)
	if err != nil {
		return err
	}

	if start > latest {
		return nil
	}

	finalizedNumber, err := s.readLastXBlocks(ctx, latest, start, addresses, client)
	if err != nil {
		return err
	}

	if err := s.creditUserDeposit(ctx, finalizedNumber); err != nil {
		return err
	}

	return s.saveSepoliaCheckpoint(ctx, latest)
}

// SignerWatcher -> Watches for the oldest requested withdrawal and sends it to the signer
func (s *Services) SignerWatcher(ctx context.Context) error {
	if !s.signerMu.TryLock() {
		return nil
	}
	defer s.signerMu.Unlock()

	withdrawal, err := s.DB.Models.Withdrawal.GetOldest(ctx, "ETH", "sepolia", "requested")
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return utils.DatabaseError(err)
	}

	txHash, err := wallet.SendSepoliaTransfer(ctx, os.Getenv("SEPOLIA_RPC_URL"),
		os.Getenv("ETH_HOT_PRIVATE_KEY"), withdrawal.Destination, withdrawal.Amount)

	if err != nil {
		return err
	}

	if err := s.ChangeWithdrawalStatus(ctx, withdrawal.UserID, withdrawal.ID, "broadcast", txHash); err != nil {
		return fmt.Errorf("sent %s but status update failed: %w", txHash, err)
	}

	return nil
}

// BroadcastWatcher -> Watches for the oldest broadcast withdrawal and checks if it is completed or failed
func (s *Services) BroadcastWatcher(ctx context.Context) error {
	if !s.broadcastMu.TryLock() {
		return nil
	}
	defer s.broadcastMu.Unlock()

	withdrawal, err := s.DB.Models.Withdrawal.GetOldest(ctx, "ETH", "sepolia", "broadcast")
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return utils.DatabaseError(err)
	}

	txHash := withdrawal.TxHash

	client, err := ethclient.DialContext(ctx, os.Getenv("SEPOLIA_RPC_URL"))
	if err != nil {
		return err
	}
	defer client.Close()

	receipt, err := client.TransactionReceipt(ctx, common.HexToHash(*txHash))
	if errors.Is(err, ethereum.NotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	if receipt.Status == 0 {
		if err := s.ChangeWithdrawalStatus(ctx, withdrawal.UserID, withdrawal.ID, "failed", ""); err != nil {
			return fmt.Errorf("failed to change withdrawal status: %w", err)
		}

		return nil
	}

	if receipt.Status == 1 {
		finalizedNumber, err := utils.FinalizedBlockNumber(ctx, client)
		if err != nil {
			return err
		}

		if receipt.BlockNumber.Uint64() > finalizedNumber {
			return nil // Wait for the block to be finalized
		}

		if err := s.ChangeWithdrawalStatus(ctx, withdrawal.UserID, withdrawal.ID, "completed", *txHash); err != nil {
			return fmt.Errorf("failed to change withdrawal status: %w", err)
		}

		return nil
	}

	return nil
}

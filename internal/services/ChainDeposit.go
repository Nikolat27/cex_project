package services

import (
	"cex_project/internal/database"
	"cex_project/internal/utils"
	"context"
	"database/sql"
	"errors"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
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

func (s *Services) saveSepoliaCheckpoint(ctx context.Context, latest uint64) error {
	_, err := s.DB.Models.ChainCheckpoints.Get(ctx, "sepolia")
	if errors.Is(err, sql.ErrNoRows) {
		return s.DB.Models.ChainCheckpoints.Create(ctx, "sepolia", int64(latest))
	}
	
	if err != nil {
		return err
	}

	return s.DB.Models.ChainCheckpoints.Update(ctx, "sepolia", int64(latest))
}

func (s *Services) GetSepoliaBlockNumber(ctx context.Context, latest uint64) (uint64, error) {
	checkpoint, err := s.DB.Models.ChainCheckpoints.Get(ctx, "sepolia")
	if errors.Is(err, sql.ErrNoRows) {
		return latest, nil
	}

	if err != nil {
		return 0, err
	}

    return uint64(checkpoint.BlockNumber + 1), nil
}

func (s *Services) readLastXBlocks(ctx context.Context, latest, start uint64,
	addresses []database.DepositAddress, client *ethclient.Client) (uint64, error) {

	finalizedBlock, err := client.BlockByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return 0, err
	}

	finalizedNumber := finalizedBlock.NumberU64()

	for n := start; n <= latest; n++ {
		block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(n))
		if err != nil {
			return 0, err
		}

		txs := block.Transactions()

		for _, address := range addresses {
			for _, tx := range txs {
				if err := s.processTx(ctx, tx, &address, latest,
					block.NumberU64(), block.Hash().String()); err != nil {
					return 0, err
				}
			}
		}
	}

	return finalizedNumber, nil
} 

func (s *Services) processTx(ctx context.Context, tx *types.Transaction,
	address *database.DepositAddress, latest, txBlockNumber uint64, blockHash string) error {

	if tx.To() == nil || !strings.EqualFold(tx.To().Hex(), address.Address) {
		return nil
	}

	if tx.Value().Sign() <= 0 {
		return nil
	}
	
	confirmations := latest - txBlockNumber

	newDeposit := &database.ChainDeposit{
		UserID: address.UserID,
		Symbol: "ETH",
		Network: "sepolia",
		Address: address.Address,
		TxHash: tx.Hash().String(),
		Amount: tx.Value().String(),
		Confirmations: int64(confirmations),
		Credited: false,
		BlockNumber: int64(txBlockNumber),
		BlockHash: blockHash,
	}

	return s.DB.Models.ChainDeposit.Create(ctx, newDeposit)
}

func (s *Services) creditUserDeposit(ctx context.Context, finalizedNumber uint64) error {
	log.Println("Finalized block number: ", finalizedNumber)

	deposits, err := s.DB.Models.ChainDeposit.ListReady(ctx, finalizedNumber)
	if err != nil {
		return err
	}
	
	for _, deposit := range deposits {
		amount, err := utils.StringToInt64(deposit.Amount)
		if err != nil {
			return err
		}

		claimed, err := s.DB.Models.ChainDeposit.MarkCredit(ctx, deposit.ID)
		if err != nil {
			return err
		}
		
		if !claimed {
			continue
		}

		if err := s.Deposit(ctx, deposit.UserID, deposit.Symbol, amount); err != nil {
			if undoErr := s.DB.Models.ChainDeposit.UnmarkCredit(ctx, deposit.ID); undoErr != nil {
				return undoErr
			}

			return err
		}
	}
	
	return nil
}

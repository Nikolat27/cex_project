package services

import (
	"cex_project/internal/database"
	"cex_project/internal/utils"
	"context"
	"errors"
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

	// Read the last 20 blocks
	finalizedNumber, err := s.readLastXBlocks(ctx, latest, 20, addresses, client)
	if err != nil {
		return err
	}

	return s.creditUserDeposit(ctx, finalizedNumber)
}

func (s *Services) readLastXBlocks(ctx context.Context, latest, amountOfBlocks uint64, 
	addresses []database.DepositAddress, client *ethclient.Client) (uint64, error) {

	var start uint64
	if latest > amountOfBlocks {
		start = latest - amountOfBlocks
	}

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

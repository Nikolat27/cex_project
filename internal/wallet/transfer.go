package wallet

import (
	"context"
	"errors"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const sepoliaChainID int64 = 11155111
const ethTransferGas uint64 = 21000

func SendSepoliaTransfer(ctx context.Context, rpcURL, privateKeyHex, to, amountWei string) (string, error) {
	if rpcURL == "" {
		return "", errors.New("SEPOLIA_RPC_URL is not set")
	}
	if privateKeyHex == "" {
		return "", errors.New("ETH_HOT_PRIVATE_KEY is not set")
	}
	if !common.IsHexAddress(to) {
		return "", errors.New("invalid destination address")
	}

	amount, ok := new(big.Int).SetString(amountWei, 10)
	if !ok || amount.Sign() <= 0 {
		return "", errors.New("invalid withdrawal amount")
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return "", err
	}

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return "", err
	}
	defer client.Close()

	from := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", err
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}

	toAddress := common.HexToAddress(to)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &toAddress,
		Value:    amount,
		Gas:      ethTransferGas,
		GasPrice: gasPrice,
	})

	signed, err := types.SignTx(tx, types.LatestSignerForChainID(big.NewInt(sepoliaChainID)), privateKey)
	if err != nil {
		return "", err
	}

	if err := client.SendTransaction(ctx, signed); err != nil {
		return "", err
	}

	log.Println("Sent transaction successfully: ", signed.Hash().Hex())

	return signed.Hash().Hex(), nil
}

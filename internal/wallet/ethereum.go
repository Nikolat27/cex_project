package wallet

import (
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func EthereumAddress(xpub string, index uint32) (string, error) {
	account, err := hdkeychain.NewKeyFromString(xpub)
	if err != nil {
		return "", err
	}

	external, err := account.Derive(0)
	if err != nil {
		return "", err
	}

	child, err := external.Derive(index)
	if err != nil {
		return "", err
	}

	pubKey, err := child.ECPubKey()
	if err != nil {
		return "", err
	}

	hash := crypto.Keccak256(pubKey.SerializeUncompressed()[1:])

	return common.BytesToAddress(hash[12:]).Hex(), nil
}

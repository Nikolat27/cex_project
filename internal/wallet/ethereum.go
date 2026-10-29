package wallet

import (
	"crypto/ecdsa"
	"errors"
	"strings"

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

// EthereumPrivateKey -> Derivces the private key for the targetted address from X_PRV
func EthereumPrivateKey(xprv string, index uint32) (*ecdsa.PrivateKey, error) {
	if strings.TrimSpace(xprv) == "" {
		return nil, errors.New("ETH_XPRV is not set")
	}
	if index >= hdkeychain.HardenedKeyStart {
		return nil, errors.New("derivation index must be non-hardened")
	}

	account, err := hdkeychain.NewKeyFromString(strings.TrimSpace(xprv))
	if err != nil {
		return nil, err
	}

	if !account.IsPrivate() {
		return nil, errors.New("ETH_XPRV must be an extended private key")
	}

	external, err := account.Derive(0)
	if err != nil {
		return nil, err
	}

	child, err := external.Derive(index)
	if err != nil {
		return nil, err
	}

	priv, err := child.ECPrivKey()
	if err != nil {
		return nil, err
	}

	return crypto.ToECDSA(priv.Serialize())
}

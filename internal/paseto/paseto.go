package paseto

import (
	"crypto/sha256"
	"errors"
	"os"
	"github.com/o1egl/paseto"
)

type Maker struct {
	paseto *paseto.V2
    symmetricKey []byte
}

func New() (*Maker, error) {
	symmetricKey, err := getSymmetricKey()
	if err != nil {
		return nil, err
	}

	newMaker := &Maker{
		paseto: paseto.NewV2(),
		symmetricKey: symmetricKey,
	}

	return newMaker, nil
}

func getSymmetricKey() ([]byte, error) {
    symmetricKey := os.Getenv("PASETO_SYMMETRIC_KEY")
    if symmetricKey == "" {
		return nil, errors.New("PASETO_SYMMETRIC_KEY is not set")
	}

	hashedKey := sha256.Sum256([]byte(symmetricKey))

	return hashedKey[:], nil
}

func (maker *Maker) CreateToken(userID int64, username string) (string, error) {
	payload, err := NewPayload(userID, username)
	if err != nil {
		return "", err
	}

    token, err := maker.paseto.Encrypt(maker.symmetricKey, payload, nil)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (maker *Maker) VerifyToken(token string) (*Payload, error) {
	var payload Payload

	if err := maker.paseto.Decrypt(token, maker.symmetricKey, &payload, nil); err != nil {
		return nil, err
	}

	if err := payload.Valid(); err != nil {
		return nil, err
	}

	return &payload, nil
}

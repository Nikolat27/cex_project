package paseto

import (
	"time"
	"github.com/google/uuid"
	"errors"
)

const ExpirationTime = 12 * time.Hour

type Payload struct {
	ID uuid.UUID `json:"id"`
	UserID int64 `json:"user_id"`
	Username string `json:"username"`
	ExpireAt time.Time `json:"expire_at"`
}

func NewPayload(userID int64, username string) (*Payload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	newPayload := &Payload{
		ID: tokenID,
		UserID: userID,
		Username: username,
		ExpireAt: time.Now().Add(ExpirationTime),
	}

	return newPayload, nil
}

func (payload *Payload) valid() error {
	if time.Now().After(payload.ExpireAt) {
		return errors.New("token has expired")
	}

	return nil
}

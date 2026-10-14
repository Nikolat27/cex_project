package services

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Services) DoesUserExist(ctx context.Context, userID int64) (bool, error) {
	_, err := s.DB.Models.User.GetByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

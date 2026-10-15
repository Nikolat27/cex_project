package utils

import (
	"database/sql"
	"errors"
)

// SafeMultiply -> Multiplies 2 integers safely to prevent overflow
func SafeMultiply(a, b int64) (int64, error) {
	const maxInt64 = int64(^uint64(0) >> 1)

	if a <= 0 || b <= 0 {
		return 0, errors.New("values must be positive")
	}
	if a > maxInt64/b {
		return 0, errors.New("integer overflow")
	}

	return a * b, nil
}

// SafeAdd -> Adds 2 integers safely to prevent overflow
func SafeAdd(a, b int64) (int64, error) {
	const maxInt64 = int64(^uint64(0) >> 1)
	if a < 0 || b < 0 {
		return 0, errors.New("values cannot be negative")
	}
	if a > maxInt64-b {
		return 0, errors.New("integer overflow")
	}
	return a + b, nil
}

func RequireOneRowAffected(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rows == 0 {
		return errors.New("database update matched not rows")
	}

	return nil
}

package utils

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
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

func ToMinorUnits(amount float64, asset string) (int64, error) {
    scale, err := scale(asset)
    if err != nil {
        return 0, err
    }
    if amount <= 0 {
        return 0, errors.New("amount must be positive")
    }

    minor := math.Round(amount * float64(scale))
    if minor > float64(math.MaxInt64) {
        return 0, errors.New("integer overflow")
    }

    return int64(minor), nil
}

func scale(asset string) (int64, error) {
	switch asset {
	case "BTC":
		return 100_000_000, nil
	case "USDT": 
		return 1_000_000, nil
	default: 
		return 0, errors.New("unknown asset")
	}
}

func WriteJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, status int, err error) error {
	return WriteJSON(w, status, map[string]string{"error": err.Error()})
}

func ReadJSON(r *http.Request, v any, maxBytes int64) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
	if err != nil {
		return err
	}

	if int64(len(body)) > maxBytes {
		return errors.New("request body too large")
	}

	return json.Unmarshal(body, v)
}

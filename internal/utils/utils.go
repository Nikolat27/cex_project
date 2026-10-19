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

var currentAssets = map[string]int64{
	"BTC":  100_000_000,
	"USDT": 1_000_000,
}

func ValidateAsset(asset string) error {
	if _, ok := currentAssets[asset]; !ok {
		return errors.New("unknown asset")
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

func PriceToMinorUnits(price float64, baseAsset, quoteAsset string) (int64, error) {
	if price <= 0 {
		return 0, errors.New("amount must be positive")
	}

	baseScale, err := scale(baseAsset)
	if err != nil {
		return 0, err
	}

	quoteScale, err := scale(quoteAsset)
	if err != nil {
		return 0, err
	}

	minor := math.Round(price * float64(quoteScale) / float64(baseScale))
	if minor < 1 {
		return 0, errors.New("price is too small")
	}
	if minor > float64(math.MaxInt64) {
		return 0, errors.New("integer overflow")
	}

	return int64(minor), nil
}

func PriceFromMinorUnits(price int64, baseAsset, quoteAsset string) (float64, error) {
	if price <= 0 {
		return 0, errors.New("amount must be positive")
	}

	baseScale, err := scale(baseAsset)
	if err != nil {
		return 0, err
	}

	quoteScale, err := scale(quoteAsset)
	if err != nil {
		return 0, err
	}

	return float64(price) * float64(baseScale) / float64(quoteScale), nil
}

func FromMinorUnits(amount int64, asset string) (float64, error) {
	scale, err := scale(asset)
	if err != nil {
		return 0, err
	}
	if amount < 0 {
		return 0, errors.New("amount cannot be negative")
	}

	return float64(amount) / float64(scale), nil
}

func scale(asset string) (int64, error) {
	assetScale, ok := currentAssets[asset]
	if !ok {
		return 0, errors.New("unknown asset")
	}

	return assetScale, nil
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

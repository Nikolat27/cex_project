package utils

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
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

func PriceToMinorUnits(price float64, quoteAsset string) (int64, error) {
	return ToMinorUnits(price, quoteAsset)
}

func PriceFromMinorUnits(price int64, quoteAsset string) (float64, error) {
	if price <= 0 {
		return 0, errors.New("amount must be positive")
	}

	return FromMinorUnits(price, quoteAsset)
}

func QuoteAmount(price, quantity int64, baseAsset string) (int64, error) {
	if price <= 0 || quantity <= 0 {
		return 0, errors.New("values must be positive")
	}

	baseScale, err := scale(baseAsset)
	if err != nil {
		return 0, err
	}

	var whole int64
	if price/baseScale > 0 {
		whole, err = SafeMultiply(price/baseScale, quantity)
		if err != nil {
			return 0, err
		}
	}

	remainder := price % baseScale
	var partial int64
	if remainder > 0 {
		scaled, err := SafeMultiply(remainder, quantity)
		if err != nil {
			return 0, err
		}
		partial = scaled / baseScale
	}

	if whole == 0 && partial == 0 {
		return 0, errors.New("quote amount is too small")
	}

	return SafeAdd(whole, partial)
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

func DatabaseError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("record does not exist")
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return errors.New("record exists")
	}

	return errors.New("database error")
}

func HashPassword(text string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func VerifyHash(text, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(text))
}

func LoadEnv() error {
	return godotenv.Load(".env")
}

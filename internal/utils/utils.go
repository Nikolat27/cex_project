package utils

import "errors"

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
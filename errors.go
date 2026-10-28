package shorturl

import "errors"

var (
	ErrInvalidConfig = errors.New("shorturl: invalid configuration")
	ErrInvalidURL    = errors.New("shorturl: invalid destination URL")
	ErrInvalidCode   = errors.New("shorturl: invalid code")
	ErrInvalidTTL    = errors.New("shorturl: invalid TTL")
	ErrNotFound      = errors.New("shorturl: link not found")
	ErrExpired       = errors.New("shorturl: link expired")
	ErrRevoked       = errors.New("shorturl: link revoked")
	ErrConflict      = errors.New("shorturl: code already reserved")
	ErrCodeExhausted = errors.New("shorturl: code generation attempts exhausted")
	ErrRandom        = errors.New("shorturl: randomness unavailable")
	ErrStorage       = errors.New("shorturl: storage failure")
)

// storageError preserves errors.Is classification without printing backend details.
type storageError struct{ cause error }

func (e storageError) Error() string { return ErrStorage.Error() }
func (e storageError) Is(target error) bool {
	return target == ErrStorage || errors.Is(e.cause, target)
}

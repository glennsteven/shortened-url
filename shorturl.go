// Package shorturl creates and resolves URL mappings using caller-owned storage.
// It performs no network requests and provides no HTTP or authorization layer.
package shorturl

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"reflect"
	"time"
)

const insertAttempts = 5

// Config controls destination validation. Zero values select HTTPS-only and an
// 8192-byte input limit. MaxURLBytes must be nonnegative; HTTP is opt-in.
type Config struct {
	AllowHTTP   bool
	MaxURLBytes int
}

// Service is safe for concurrent use when its Store satisfies the Store contract.
// Configuration is copied at construction; there are no background jobs or caches.
type Service struct {
	store  Store
	cfg    Config
	now    func() time.Time
	random io.Reader
}

// New uses the supplied store without taking ownership of its lifecycle.
func New(store Store, cfg Config) (*Service, error) {
	if store == nil {
		return nil, ErrInvalidConfig
	}
	v := reflect.ValueOf(store)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			return nil, ErrInvalidConfig
		}
	}
	if cfg.MaxURLBytes < 0 {
		return nil, ErrInvalidConfig
	}
	if cfg.MaxURLBytes == 0 {
		cfg.MaxURLBytes = 8192
	}
	return &Service{store: store, cfg: cfg, now: time.Now, random: rand.Reader}, nil
}

// Shorten creates a new mapping each time. A zero TTL means no expiration;
// negative TTLs are invalid. Codes contain 16 random bytes as unpadded base64url.
// Only insertion conflicts are retried, for at most five total attempts.
func (s *Service) Shorten(ctx context.Context, destination string, ttl time.Duration) (Link, error) {
	if err := ctx.Err(); err != nil {
		return Link{}, err
	}
	if err := s.ValidateURL(destination); err != nil {
		return Link{}, err
	}
	if ttl < 0 {
		return Link{}, ErrInvalidTTL
	}
	link := Link{Destination: destination, CreatedAt: s.now().UTC()}
	if ttl > 0 {
		link.ExpiresAt = link.CreatedAt.Add(ttl)
	}
	for attempt := 0; attempt < insertAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return Link{}, err
		}
		var raw [16]byte
		if _, err := io.ReadFull(s.random, raw[:]); err != nil {
			return Link{}, ErrRandom
		}
		link.Code = base64.RawURLEncoding.EncodeToString(raw[:])
		err := s.store.Insert(ctx, link)
		if ctx.Err() != nil {
			return Link{}, ctx.Err()
		}
		if err == nil {
			return link, nil
		}
		if errors.Is(err, ErrConflict) {
			continue
		}
		return Link{}, storageError{err}
	}
	return Link{}, ErrCodeExhausted
}

// Resolve reads authoritative storage and revalidates the destination.
// Revocation takes precedence over expiration. Expiration is now >= ExpiresAt.
func (s *Service) Resolve(ctx context.Context, code string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := ValidateCode(code); err != nil {
		return "", err
	}
	link, err := s.store.Get(ctx, code)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrNotFound
		}
		return "", storageError{err}
	}
	if link.Code != code {
		return "", storageError{ErrInvalidCode}
	}
	if link.Revoked {
		return "", ErrRevoked
	}
	if !link.ExpiresAt.IsZero() && !s.now().Before(link.ExpiresAt) {
		return "", ErrExpired
	}
	if err := s.ValidateURL(link.Destination); err != nil {
		return "", err
	}
	return link.Destination, nil
}

// Revoke disables an existing link. Repeated revocation succeeds.
func (s *Service) Revoke(ctx context.Context, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateCode(code); err != nil {
		return err
	}
	err := s.store.Revoke(ctx, code)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return storageError{err}
}

// ValidateCode accepts only the canonical encoding of exactly 16 bytes.
func ValidateCode(code string) error {
	if len(code) != 22 {
		return ErrInvalidCode
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(code)
	if err != nil || len(raw) != 16 || base64.RawURLEncoding.EncodeToString(raw) != code {
		return ErrInvalidCode
	}
	return nil
}

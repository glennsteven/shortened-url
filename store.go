package shorturl

import (
	"context"
	"sync"
	"time"
)

// Link is a stored mapping. A zero ExpiresAt means no expiration.
// Destinations and codes must remain immutable after insertion.
type Link struct {
	Code        string
	Destination string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Revoked     bool
}

// Store implementations must support concurrent use and honor context cancellation.
// Insert atomically reserves a code forever, returning ErrConflict on duplicates.
// Get returns ErrNotFound only for missing records and independent value copies.
// Revoke is atomic and idempotent for existing records; missing ones return
// ErrNotFound. Reads starting after Revoke succeeds must observe revocation.
// Never reuse expired/revoked codes; retain records or permanent reservations.
// Return operational failures distinctly from these sentinel errors.
type Store interface {
	Insert(context.Context, Link) error
	Get(context.Context, string) (Link, error)
	Revoke(context.Context, string) error
}

// MemoryStore is concurrency-safe, non-durable storage with unbounded growth.
// Its zero value is usable. It retains all codes, including revoked/expired ones.
type MemoryStore struct {
	mu    sync.RWMutex
	links map[string]Link
}

func (m *MemoryStore) Insert(ctx context.Context, link Link) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, ok := m.links[link.Code]; ok {
		return ErrConflict
	}
	if m.links == nil {
		m.links = make(map[string]Link)
	}
	m.links[link.Code] = link
	return nil
}

func (m *MemoryStore) Get(ctx context.Context, code string) (Link, error) {
	if err := ctx.Err(); err != nil {
		return Link{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return Link{}, err
	}
	link, ok := m.links[code]
	if !ok {
		return Link{}, ErrNotFound
	}
	return link, nil
}

func (m *MemoryStore) Revoke(ctx context.Context, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	link, ok := m.links[code]
	if !ok {
		return ErrNotFound
	}
	link.Revoked = true
	m.links[code] = link
	return nil
}

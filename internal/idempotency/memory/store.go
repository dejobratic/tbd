package memory

import (
	"context"
	"sync"

	"github.com/dejobratic/tbd/internal/orders/ports"
)

// Store retains idempotency responses for replaying duplicate requests.
type Store struct {
	mu    sync.RWMutex
	items map[string]ports.StoredResponse
}

// NewStore creates a new in-memory idempotency store.
func NewStore() *Store {
	return &Store{items: make(map[string]ports.StoredResponse)}
}

// Get returns the stored response for a given key if present.
func (s *Store) Get(_ context.Context, key string) (*ports.StoredResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	copy := value
	return &copy, nil
}

// Save stores or overwrites the response for a key.
func (s *Store) Save(_ context.Context, key string, response ports.StoredResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = response
	return nil
}

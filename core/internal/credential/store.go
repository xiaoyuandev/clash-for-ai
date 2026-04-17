package credential

import (
	"context"
	"fmt"
	"sync"
)

type Store interface {
	Save(ctx context.Context, key string, value string) (string, error)
	Get(ctx context.Context, ref string) (string, error)
	Delete(ctx context.Context, ref string) error
}

type InMemoryStore struct {
	mu    sync.RWMutex
	items map[string]string
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		items: map[string]string{},
	}
}

func (s *InMemoryStore) Save(_ context.Context, key string, value string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ref := fmt.Sprintf("memory://%s", key)
	s.items[ref] = value

	return ref, nil
}

func (s *InMemoryStore) Get(_ context.Context, ref string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.items[ref]
	if !ok {
		return "", fmt.Errorf("credential not found for ref %s", ref)
	}

	return value, nil
}

func (s *InMemoryStore) Delete(_ context.Context, ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, ref)
	return nil
}

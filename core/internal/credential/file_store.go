package credential

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileStore struct {
	mu    sync.RWMutex
	path  string
	items map[string]string
}

func NewFileStore(path string) (*FileStore, error) {
	store := &FileStore{
		path:  path,
		items: map[string]string{},
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *FileStore) Save(_ context.Context, key string, value string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ref := fmt.Sprintf("file://%s", key)
	s.items[ref] = value

	if err := s.persist(); err != nil {
		return "", err
	}

	return ref, nil
}

func (s *FileStore) Get(_ context.Context, ref string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.items[ref]
	if !ok {
		return "", fmt.Errorf("credential not found for ref %s", ref)
	}

	return value, nil
}

func (s *FileStore) Delete(_ context.Context, ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, ref)
	return s.persist()
}

func (s *FileStore) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create credential dir: %w", err)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read credential store: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &s.items); err != nil {
		return fmt.Errorf("decode credential store: %w", err)
	}

	return nil
}

func (s *FileStore) persist() error {
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credential store: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write credential store: %w", err)
	}

	return nil
}

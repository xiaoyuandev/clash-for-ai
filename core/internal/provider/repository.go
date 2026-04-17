package provider

import (
	"context"
	"errors"
	"sync"
)

var ErrProviderNotFound = errors.New("provider not found")

type Repository interface {
	List(ctx context.Context) ([]Provider, error)
	GetActive(ctx context.Context) (*Provider, error)
	Create(ctx context.Context, item Provider) (Provider, error)
	Activate(ctx context.Context, id string) (*Provider, error)
}

type InMemoryRepository struct {
	mu    sync.RWMutex
	items []Provider
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		items: []Provider{},
	}
}

func (r *InMemoryRepository) List(context.Context) ([]Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Provider, len(r.items))
	copy(items, r.items)

	return items, nil
}

func (r *InMemoryRepository) GetActive(context.Context) (*Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, item := range r.items {
		if item.Status.IsActive {
			provider := item
			return &provider, nil
		}
	}

	return nil, nil
}

func (r *InMemoryRepository) Create(_ context.Context, item Provider) (Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = append(r.items, item)

	return item, nil
}

func (r *InMemoryRepository) Activate(_ context.Context, id string) (*Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	index := -1
	for i := range r.items {
		r.items[i].Status.IsActive = false
		if r.items[i].ID == id {
			index = i
		}
	}

	if index < 0 {
		return nil, ErrProviderNotFound
	}

	r.items[index].Status.IsActive = true
	provider := r.items[index]
	return &provider, nil
}

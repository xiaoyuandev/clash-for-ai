package provider

import (
	"context"
	"errors"
	"sync"
)

var ErrProviderNotFound = errors.New("provider not found")
var ErrProviderNotEditable = errors.New("provider is not editable")
var ErrProviderNotDeletable = errors.New("provider is not deletable")

type Repository interface {
	List(ctx context.Context) ([]Provider, error)
	GetActive(ctx context.Context) (*Provider, error)
	GetByID(ctx context.Context, id string) (*Provider, error)
	ListSelectedModels(ctx context.Context, providerID string) ([]SelectedModel, error)
	ReplaceSelectedModels(ctx context.Context, providerID string, items []SelectedModel) error
	ListCodexModels(ctx context.Context, providerID string) ([]CodexModel, error)
	ReplaceCodexModels(ctx context.Context, providerID string, items []CodexModel) error
	Create(ctx context.Context, item Provider) (Provider, error)
	Update(ctx context.Context, item Provider) (Provider, error)
	Delete(ctx context.Context, id string) error
	Activate(ctx context.Context, id string) (*Provider, error)
}

type InMemoryRepository struct {
	mu           sync.RWMutex
	items        []Provider
	codexModels  map[string][]CodexModel
	selectedByID map[string][]SelectedModel
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		items:        []Provider{},
		codexModels:  map[string][]CodexModel{},
		selectedByID: map[string][]SelectedModel{},
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

func (r *InMemoryRepository) ListSelectedModels(_ context.Context, providerID string) ([]SelectedModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.selectedByID[providerID]
	result := make([]SelectedModel, len(items))
	copy(result, items)
	return result, nil
}

func (r *InMemoryRepository) ReplaceSelectedModels(_ context.Context, providerID string, items []SelectedModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	next := make([]SelectedModel, len(items))
	copy(next, items)
	r.selectedByID[providerID] = next
	return nil
}

func (r *InMemoryRepository) ListCodexModels(_ context.Context, providerID string) ([]CodexModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.codexModels[providerID]
	result := make([]CodexModel, len(items))
	copy(result, items)
	return result, nil
}

func (r *InMemoryRepository) ReplaceCodexModels(_ context.Context, providerID string, items []CodexModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	next := make([]CodexModel, len(items))
	copy(next, items)
	r.codexModels[providerID] = next
	return nil
}

func (r *InMemoryRepository) GetByID(_ context.Context, id string) (*Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, item := range r.items {
		if item.ID == id {
			provider := item
			return &provider, nil
		}
	}

	return nil, ErrProviderNotFound
}

func (r *InMemoryRepository) Update(_ context.Context, item Provider) (Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.items {
		if r.items[i].ID == item.ID {
			r.items[i] = item
			return item, nil
		}
	}

	return Provider{}, ErrProviderNotFound
}

func (r *InMemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.items {
		if r.items[i].ID == id {
			r.items = append(r.items[:i], r.items[i+1:]...)
			delete(r.selectedByID, id)
			delete(r.codexModels, id)
			return nil
		}
	}

	return ErrProviderNotFound
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

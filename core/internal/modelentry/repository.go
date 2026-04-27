package modelentry

import (
	"context"
	"errors"
)

var ErrEntryNotFound = errors.New("model entry not found")

type Repository interface {
	List(ctx context.Context) ([]Entry, error)
	GetByID(ctx context.Context, id string) (*Entry, error)
	Create(ctx context.Context, entry Entry) (Entry, error)
	Update(ctx context.Context, entry Entry) (Entry, error)
	Delete(ctx context.Context, id string) error
	ReplaceOrder(ctx context.Context, entries []Entry) error
}

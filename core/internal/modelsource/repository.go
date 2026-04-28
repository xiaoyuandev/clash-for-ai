package modelsource

import (
	"context"
	"errors"
)

var ErrSourceNotFound = errors.New("model source not found")

type Repository interface {
	List(ctx context.Context) ([]Source, error)
	GetByID(ctx context.Context, id string) (*Source, error)
	Create(ctx context.Context, source Source) (Source, error)
	Update(ctx context.Context, source Source) (Source, error)
	Delete(ctx context.Context, id string) error
	ReplaceOrder(ctx context.Context, sources []Source) error
}

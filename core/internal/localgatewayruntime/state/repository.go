package state

import "context"

type Repository interface {
	ListSelectedModels(ctx context.Context) ([]SelectedModel, error)
	ReplaceSelectedModels(ctx context.Context, items []SelectedModel) error
}

package image

import (
	"context"

	"github.com/kyleterry/jot/pkg/types"
)

// StoreService fetches, creates, and deletes image galleries
type StoreService interface {
	Stat(ctx context.Context, id string) (*types.GalleryFile, error)
	Get(ctx context.Context, id string) (*types.GalleryFile, error)
	Create(ctx context.Context, content *types.Images) (*types.GalleryFile, error)
	Delete(ctx context.Context, gf *types.GalleryFile) error
}

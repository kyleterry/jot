package image

import (
	"context"
	"io"

	"github.com/kyleterry/jot/pkg/types"
)

// StoreService fetches, creates, and deletes image galleries
type StoreService interface {
	Get(ctx context.Context, key string) (*types.GalleryFile, error)
	Create(ctx context.Context, content map[string]io.ReadCloser) (*types.GalleryFile, error)
	Delete(ctx context.Context, gf *types.GalleryFile) error
}

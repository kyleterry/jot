package text

import (
	"context"
	"io"

	"github.com/kyleterry/jot/pkg/types"
)

// StoreService fetches, stores, edits and deletes textual jots
type StoreService interface {
	Stat(ctx context.Context, key string) (*types.TextFile, error)
	Get(ctx context.Context, key string) (*types.TextFile, error)
	Create(ctx context.Context, content io.ReadCloser) (*types.TextFile, error)
	Update(ctx context.Context, jf *types.TextFile) error
	Delete(ctx context.Context, jf *types.TextFile) error
}

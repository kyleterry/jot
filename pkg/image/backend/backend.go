package backend

import (
	"context"
	"time"

	"github.com/kyleterry/jot/pkg/types"
)

type StatResponse struct {
	ModifiedDate time.Time
}

type Interface interface {
	Stat(ctx context.Context, key string) (*StatResponse, error)
	Get(ctx context.Context, key string) (*types.Images, error)
	Create(ctx context.Context, key string, images *types.Images) error
	Delete(ctx context.Context, key string) error
}

package backend

import (
	"context"
	"io"
	"time"
)

type StatResponse struct {
	ModifiedDate time.Time
}

type Interface interface {
	Stat(ctx context.Context, key string) (*StatResponse, error)
	Get(ctx context.Context, key string) (map[string]io.ReadCloser, error)
	Create(ctx context.Context, key string, images map[string]io.ReadCloser) error
	Delete(ctx context.Context, key string) error
}

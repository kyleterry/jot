package backend

import (
	"context"
	"io"
)

type Interface interface {
	Get(ctx context.Context, key string) (map[string]io.ReadCloser, error)
	Create(ctx context.Context, key string, images map[string]io.ReadCloser) error
	Delete(ctx context.Context, key string) error
}

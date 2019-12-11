package store

import (
	"io"
	"time"
)

type GetResponse struct {
	Content io.ReadCloser
}

type StatResponse struct {
	ModifiedDate time.Time
}

type Backend interface {
	Stat(key string) (*StatResponse, error)
	Get(key string) (*GetResponse, error)
	Put(key string, content io.ReadCloser) error
	Delete(key string) error
}

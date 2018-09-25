package store

import "io"

type Store interface {
	Get(key string) (io.ReadCloser, error)
	Put(key string, content io.ReadCloser) error
	Delete(key string) error
}

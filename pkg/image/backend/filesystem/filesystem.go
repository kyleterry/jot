package filesystem

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
)

type Config struct {
	Path                 string
	FilePermissions      os.FileMode
	DirectoryPermissions os.FileMode
}

type Backend struct {
	path                 string
	filePermissions      os.FileMode
	directoryPermissions os.FileMode
}

func (b *Backend) Get(ctx context.Context, key string) (map[string]io.ReadCloser, error) {
	images := map[string]io.ReadCloser{}

	path := filepath.Join(b.path, key)

	err := filepath.Walk(path, func(fp string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(fp)
			if err != nil {
				if f != nil {
					f.Close()
				}

				return err
			}

			images[info.Name()] = f
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return images, nil
}

func (b *Backend) Create(ctx context.Context, key string, images map[string]io.ReadCloser) error {
	defer func() {
		for _, c := range images {
			c.Close()
		}
	}()

	dir := filepath.Join(b.path, key)
	if err := os.Mkdir(dir, b.directoryPermissions); err != nil {
		return err
	}

	for fn, rc := range images {
		fp := filepath.Join(dir, fn)

		f, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, b.filePermissions)

		defer f.Close()

		if err != nil {
			return err
		}

		buf := bufio.NewReader(rc)

		if _, err := buf.WriteTo(f); err != nil {
			return err
		}
	}

	return nil
}

func (b *Backend) Delete(ctx context.Context, key string) error {
	return nil
}

func New(cfg *Config) (*Backend, error) {
	if err := os.MkdirAll(cfg.Path, cfg.DirectoryPermissions); err != nil {
		return nil, err
	}

	return &Backend{
		path:                 cfg.Path,
		filePermissions:      cfg.FilePermissions,
		directoryPermissions: cfg.DirectoryPermissions,
	}, nil
}

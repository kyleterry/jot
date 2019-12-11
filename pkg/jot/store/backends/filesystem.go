package backends

import (
	"bufio"
	"io"
	"os"
	"path/filepath"

	"github.com/kyleterry/jot/pkg/jot/errors"
	"github.com/kyleterry/jot/pkg/jot/store"
)

const DefaultPermissions = 0644

type FilesystemOptions struct {
	Path string
}

type Filesystem struct {
	path string
}

func (fs *Filesystem) Stat(key string) (*store.StatResponse, error) {
	path := filepath.Join(fs.path, key)

	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewNotFoundError(key).WithCause(err)
		}

		return nil, err
	}

	return &store.StatResponse{ModifiedDate: stat.ModTime()}, nil
}

func (fs *Filesystem) Get(key string) (*store.GetResponse, error) {
	path := filepath.Join(fs.path, key)

	f, err := os.Open(path)
	if err != nil {
		if f != nil {
			f.Close()
		}
		if os.IsNotExist(err) {
			return nil, errors.NewNotFoundError(key).WithCause(err)
		}

		return nil, err
	}

	return &store.GetResponse{Content: f}, nil
}

func (fs *Filesystem) Put(key string, content io.ReadCloser) error {
	path := filepath.Join(fs.path, key)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, DefaultPermissions)

	defer func() {
		f.Close()
		content.Close()
	}()

	if err != nil {
		return err
	}

	buf := bufio.NewReader(content)

	if _, err := buf.WriteTo(f); err != nil {
		return err
	}

	return nil
}

func (fs *Filesystem) Delete(key string) error {
	path := filepath.Join(fs.path, key)

	return os.Remove(path)
}

func NewFilesystem(opts FilesystemOptions) *Filesystem {
	return &Filesystem{
		path: opts.Path,
	}
}

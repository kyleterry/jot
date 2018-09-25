package backends

import (
	"bufio"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const DefaultPermissions = 0644

type FilesystemOptions struct {
	Path string
}

type Filesystem struct {
	path string
}

func (fs *Filesystem) Get(key string) (io.ReadCloser, error) {
	path := filepath.Join(fs.path, key)

	f, err := os.Open(path)
	if err != nil {
		if f != nil {
			f.Close()
		}

		return nil, errors.Wrap(err, "failed to get content from filesystem")
	}

	return f, nil
}

func (fs *Filesystem) Put(key string, content io.ReadCloser) error {
	path := filepath.Join(fs.path, key)

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, DefaultPermissions)

	defer func() {
		f.Close()
		content.Close()
	}()

	if err != nil {
		return errors.Wrap(err, "failed to open file for writing")
	}

	buf := bufio.NewReader(content)

	if _, err := buf.WriteTo(f); err != nil {
		return errors.Wrap(err, "failed to write to file")
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

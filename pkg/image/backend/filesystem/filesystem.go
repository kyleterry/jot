package filesystem

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kyleterry/jot/pkg/errors"
	"github.com/kyleterry/jot/pkg/image/backend"
)

const galleryFileName = "gallery"

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

func (b *Backend) Stat(ctx context.Context, key string) (*backend.StatResponse, error) {
	path := filepath.Join(b.path, key, galleryFileName)

	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewNotFoundError(key).WithCause(err)
		}

		return nil, err
	}

	return &backend.StatResponse{ModifiedDate: stat.ModTime()}, nil
}

func (b *Backend) Get(ctx context.Context, key string) (map[string]io.ReadCloser, error) {
	images := map[string]io.ReadCloser{}

	path := filepath.Join(b.path, key, galleryFileName)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	reader := bufio.NewReader(f)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, errors.NewUnknownError("malformed line from image gallery")
		}

		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, err
		}

		r := bytes.NewBuffer(decoded)
		images[parts[0]] = ioutil.NopCloser(r)
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

	fp := filepath.Join(dir, galleryFileName)

	galleryFile, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, b.filePermissions)
	if err != nil {
		return err
	}

	defer galleryFile.Close()

	for imageName, reader := range images {
		buf := &bytes.Buffer{}

		_, err := buf.ReadFrom(reader)
		if err != nil {
			return err
		}

		encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
		if _, err := fmt.Fprintf(galleryFile, "%s %s\n", imageName, encoded); err != nil {
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

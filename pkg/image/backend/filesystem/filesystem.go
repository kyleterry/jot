package filesystem

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/wire"
	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/errors"
	"github.com/kyleterry/jot/pkg/image/backend"
	"github.com/kyleterry/jot/pkg/types"
)

var ProviderSet = wire.NewSet(
	wire.Struct(new(Options), "*"),
	New,
)

var BoundProviderSet = wire.NewSet(
	ProviderSet,
	wire.Bind(new(backend.Interface), new(*Backend)),
)

const (
	filePermissions      = 0o640
	directoryPermissions = 0o740
	directoryName        = "img"
	galleryFileName      = "gallery"
)

type Options struct {
	StorageDir config.DataDir
}

type Backend struct {
	path string
}

func (b *Backend) Stat(ctx context.Context, id string) (*backend.StatResponse, error) {
	path := filepath.Join(b.path, id, galleryFileName)

	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewNotFoundError(id).WithCause(err)
		}

		return nil, err
	}

	return &backend.StatResponse{ModifiedDate: stat.ModTime()}, nil
}

func (b *Backend) Get(ctx context.Context, id string) (*types.Images, error) {
	images := &types.Images{}

	path := filepath.Join(b.path, id, galleryFileName)

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

		metadata := strings.Split(parts[0], ";")
		name := metadata[0]

		var contentType string
		if len(metadata) > 1 {
			contentType = metadata[1]
		}

		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, err
		}

		r := bytes.NewBuffer(decoded)
		images.Add(name, &types.ImageData{
			Name:        name,
			Content:     io.NopCloser(r),
			ContentType: contentType,
		})
	}

	return images, nil
}

func (b *Backend) Create(ctx context.Context, id string, images *types.Images) error {
	defer func() {
		for _, c := range images.Values {
			c.Content.Close()
		}
	}()

	dir := filepath.Join(b.path, id)
	if err := os.Mkdir(dir, directoryPermissions); err != nil {
		return err
	}

	fp := filepath.Join(dir, galleryFileName)

	galleryFile, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions)
	if err != nil {
		return err
	}

	defer galleryFile.Close()

	for _, imageName := range images.Keys {
		imageData := images.Values[imageName]
		buf := &bytes.Buffer{}

		_, err := buf.ReadFrom(imageData.Content)
		if err != nil {
			return err
		}

		imageName = url.QueryEscape(imageName)

		encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
		if _, err := fmt.Fprintf(galleryFile, "%s;%s %s\n", imageName, imageData.ContentType, encoded); err != nil {
			return err
		}
	}

	return nil
}

func (b *Backend) Delete(ctx context.Context, id string) error {
	panic("implement")
}

func New(opts *Options) (*Backend, error) {
	store := filepath.Join(directoryName, string(opts.StorageDir))
	if err := os.MkdirAll(store, directoryPermissions); err != nil {
		return nil, err
	}

	return &Backend{
		path: store,
	}, nil
}

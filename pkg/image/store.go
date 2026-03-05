package image

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/imageorient"
	"github.com/google/wire"
	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/errors"
	"github.com/kyleterry/jot/pkg/id"
	"github.com/kyleterry/jot/pkg/image/backend"
	"github.com/kyleterry/jot/pkg/types"
)

var ProviderSet = wire.NewSet(
	wire.Struct(new(Options), "*"),
	NewStore,
)

type Options struct {
	PasswordManager *auth.PasswordManager
	IDManager       *id.IDManager
}

type Store struct {
	pm             *auth.PasswordManager
	im             *id.IDManager
	storageBackend backend.Interface
}

func (s *Store) stat(ctx context.Context, key string) (*backend.StatResponse, error) {
	resp, err := s.storageBackend.Stat(ctx, key)
	if err != nil {
		if errors.IsStoreError(err) {
			return nil, err
		}

		return nil, errors.NewUnknownError("failed to get file from backend").WithCause(err)
	}

	return resp, nil
}

func (s *Store) Stat(ctx context.Context, key string) (*types.GalleryFile, error) {
	resp, err := s.stat(ctx, key)
	if err != nil {
		return nil, err
	}

	return &types.GalleryFile{ID: key, ObjectMeta: types.ObjectMeta{ModifiedDate: resp.ModifiedDate}}, nil
}

func (s *Store) Get(ctx context.Context, key string) (*types.GalleryFile, error) {
	statResp, err := s.stat(ctx, key)
	if err != nil {
		return nil, err
	}

	images, err := s.storageBackend.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	gallery := &types.GalleryFile{
		ID:         key,
		Images:     images,
		ObjectMeta: types.ObjectMeta{ModifiedDate: statResp.ModifiedDate},
	}

	return gallery, nil
}

func (s *Store) Create(ctx context.Context, images *types.Images) (*types.GalleryFile, error) {
	key, err := s.im.Generate()
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate id").WithCause(err)
	}

	password, err := s.pm.Generate(key)
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate password").WithCause(err)
	}

	if err := s.processImages(images); err != nil {
		return nil, fmt.Errorf("failed to process images: %w", err)
	}

	if err := s.storageBackend.Create(ctx, key, images); err != nil {
		return nil, err
	}

	g := types.GalleryFile{
		ID:       key,
		Password: password,
	}

	return &g, nil
}

func (s *Store) Delete(ctx context.Context, gf *types.GalleryFile) error {
	if err := s.storageBackend.Delete(ctx, gf.ID); err != nil {
		return errors.NewUnknownError("failed to delete gallery from backend").WithCause(err)
	}

	return nil
}

func (s *Store) processImages(images *types.Images) error {
	for _, imageName := range images.Keys {
		imageData := images.Values[imageName]

		raw, err := io.ReadAll(imageData.Content)
		imageData.Content.Close()
		if err != nil {
			return fmt.Errorf("failed to read image: %w", err)
		}

		_, format, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			return fmt.Errorf("failed to decode image config: %w", err)
		}

		var buf bytes.Buffer

		switch format {
		case "gif":
			// GIFs don't carry EXIF orientation data, so we bypass imageorient
			// and use gif.DecodeAll/EncodeAll to preserve animated GIF frames.
			g, err := gif.DecodeAll(bytes.NewReader(raw))
			if err != nil {
				return fmt.Errorf("failed to decode gif: %w", err)
			}
			if err := gif.EncodeAll(&buf, g); err != nil {
				return fmt.Errorf("failed to encode gif: %w", err)
			}
			imageData.ContentType = "image/gif"
		default:
			img, _, err := imageorient.Decode(bytes.NewReader(raw))
			if err != nil {
				return fmt.Errorf("failed to decode image: %w", err)
			}
			switch format {
			case "jpeg":
				if err := jpeg.Encode(&buf, img, nil); err != nil {
					return fmt.Errorf("failed to encode image: %w", err)
				}
				imageData.ContentType = "image/jpeg"
			case "png":
				if err := png.Encode(&buf, img); err != nil {
					return fmt.Errorf("failed to encode image: %w", err)
				}
				imageData.ContentType = "image/png"
			default:
				return errors.NewUnsupportedFormatError(format)
			}
		}

		imageData.Content = io.NopCloser(&buf)
		images.Values[imageName] = imageData
	}

	return nil
}

func NewStore(b backend.Interface, opts *Options) *Store {
	return &Store{
		pm:             opts.PasswordManager,
		im:             opts.IDManager,
		storageBackend: b,
	}
}

package image

import (
	"context"
	"io"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/image/backend"
	"github.com/kyleterry/jot/pkg/jot/errors"
	"github.com/kyleterry/jot/pkg/types"
	"github.com/teris-io/shortid"
)

type Store struct {
	cfg             *config.Config
	storageBackend  backend.Interface
	passwordManager auth.PasswordManager
}

func (s *Store) Get(ctx context.Context, key string) (*types.GalleryFile, error) {
	rawImages, err := s.storageBackend.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	images := map[string]*types.ImageData{}

	for name, content := range rawImages {
		images[name] = &types.ImageData{
			Name:    name,
			Content: content,
		}
	}

	gallery := &types.GalleryFile{
		Key:    key,
		Images: images,
	}

	return gallery, nil
}

func (s *Store) Create(ctx context.Context, images map[string]io.ReadCloser) (*types.GalleryFile, error) {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)
	if err != nil {
		return nil, err
	}

	key, err := sid.Generate()
	if err != nil {
		return nil, err
	}

	password, err := s.passwordManager.Generate(key)
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate password").WithCause(err)
	}

	if err := s.storageBackend.Create(ctx, key, images); err != nil {
		return nil, err
	}

	g := types.GalleryFile{
		Key:      key,
		Password: password,
	}

	return &g, nil
}

func (s *Store) Delete(ctx context.Context, gf *types.GalleryFile) error {
	return nil
}

func NewStore(cfg *config.Config, b backend.Interface, pm auth.PasswordManager) *Store {
	return &Store{
		cfg:             cfg,
		storageBackend:  b,
		passwordManager: pm,
	}
}

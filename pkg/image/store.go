package image

import (
	"context"
	"io"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/errors"
	"github.com/kyleterry/jot/pkg/id"
	"github.com/kyleterry/jot/pkg/image/backend"
	"github.com/kyleterry/jot/pkg/types"
)

type Services interface {
	PasswordManager() auth.PasswordManagerService
	IDManager() id.IDManagerService
}

type Store struct {
	services       Services
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

	return &types.GalleryFile{Key: key, ModifiedDate: resp.ModifiedDate}, nil
}

func (s *Store) Get(ctx context.Context, key string) (*types.GalleryFile, error) {
	statResp, err := s.stat(ctx, key)
	if err != nil {
		return nil, err
	}

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
		Key:          key,
		Images:       images,
		ModifiedDate: statResp.ModifiedDate,
	}

	return gallery, nil
}

func (s *Store) Create(ctx context.Context, images map[string]io.ReadCloser) (*types.GalleryFile, error) {
	key, err := s.services.IDManager().Generate()
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate id").WithCause(err)
	}

	password, err := s.services.PasswordManager().Generate(key)
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
	if err := s.storageBackend.Delete(ctx, gf.Key); err != nil {
		return errors.NewUnknownError("failed to delete gallery from backend").WithCause(err)
	}

	return nil
}

func NewStore(b backend.Interface, services Services) *Store {
	return &Store{
		services:       services,
		storageBackend: b,
	}
}

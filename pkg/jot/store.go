package jot

import (
	"context"
	"io"

	"github.com/google/wire"
	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/errors"
	"github.com/kyleterry/jot/pkg/id"
	"github.com/kyleterry/jot/pkg/jot/store"
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

// TextStore wraps a backend implementation and creates/checks passwords for a jot
type TextStore struct {
	pm      *auth.PasswordManager
	im      *id.IDManager
	backend store.Backend
}

func (s *TextStore) stat(key string) (*store.StatResponse, error) {
	resp, err := s.backend.Stat(key)
	if err != nil {
		if errors.IsStoreError(err) {
			return nil, err
		}

		return nil, errors.NewUnknownError("failed to get file from backend").WithCause(err)
	}

	return resp, nil
}

func (s *TextStore) getFile(key string) (*store.GetResponse, error) {
	resp, err := s.backend.Get(key)
	if err != nil {
		if errors.IsStoreError(err) {
			return nil, err
		}

		return nil, errors.NewUnknownError("failed to get file from backend").WithCause(err)
	}

	return resp, nil
}

func (s *TextStore) Stat(ctx context.Context, key string) (*types.TextFile, error) {
	resp, err := s.stat(key)
	if err != nil {
		return nil, err
	}

	return &types.TextFile{Key: key, ObjectMeta: types.ObjectMeta{ModifiedDate: resp.ModifiedDate}}, nil
}

func (s *TextStore) Get(ctx context.Context, key string) (*types.TextFile, error) {
	statResp, err := s.stat(key)
	if err != nil {
		return nil, err
	}

	resp, err := s.getFile(key)
	if err != nil {
		return nil, err
	}

	jotFile := &types.TextFile{
		Key:        key,
		Content:    resp.Content,
		ObjectMeta: types.ObjectMeta{ModifiedDate: statResp.ModifiedDate},
	}

	return jotFile, nil
}

func (s *TextStore) Create(ctx context.Context, content io.ReadCloser) (*types.TextFile, error) {
	key, err := s.im.Generate()
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate id").WithCause(err)
	}

	password, err := s.pm.Generate(key)
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate password").WithCause(err)
	}

	if err := s.backend.Put(key, content); err != nil {
		return nil, errors.NewUnknownError("failed to write file into backend").WithCause(err)
	}

	return &types.TextFile{
		Key:      key,
		Content:  content,
		Password: password,
	}, nil
}

func (s *TextStore) Update(ctx context.Context, jotFile *types.TextFile) error {
	if err := s.backend.Put(jotFile.Key, jotFile.Content); err != nil {
		return errors.NewUnknownError("failed to write file into backend").WithCause(err)
	}

	return nil
}

func (s *TextStore) Delete(ctx context.Context, jotFile *types.TextFile) error {
	if err := s.backend.Delete(jotFile.Key); err != nil {
		return errors.NewUnknownError("failed to delete file from backend").WithCause(err)
	}

	return nil
}

func NewStore(backend store.Backend, opts *Options) *TextStore {
	return &TextStore{
		pm:      opts.PasswordManager,
		im:      opts.IDManager,
		backend: backend,
	}
}

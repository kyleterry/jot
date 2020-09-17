package jot

import (
	"context"
	"io"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/jot/errors"
	"github.com/kyleterry/jot/pkg/jot/store"
	"github.com/kyleterry/jot/pkg/types"
	"github.com/teris-io/shortid"
)

// JotStore wraps a backend implementation and creates/checks passwords for a jot
type JotStore struct {
	passwordManager auth.PasswordManager
	backend         store.Backend
}

func (s *JotStore) stat(key string) (*store.StatResponse, error) {
	resp, err := s.backend.Stat(key)
	if err != nil {
		if errors.IsStoreError(err) {
			return nil, err
		}

		return nil, errors.NewUnknownError("failed to get file from backend").WithCause(err)
	}

	return resp, nil
}

func (s *JotStore) getFile(key string) (*store.GetResponse, error) {
	resp, err := s.backend.Get(key)
	if err != nil {
		if errors.IsStoreError(err) {
			return nil, err
		}

		return nil, errors.NewUnknownError("failed to get file from backend").WithCause(err)
	}

	return resp, nil
}

func (s *JotStore) Stat(ctx context.Context, key string) (*types.TextFile, error) {
	resp, err := s.stat(key)
	if err != nil {
		return nil, err
	}

	return &types.TextFile{Key: key, ModifiedDate: resp.ModifiedDate}, nil
}

func (s *JotStore) Get(ctx context.Context, key string) (*types.TextFile, error) {
	statResp, err := s.stat(key)
	if err != nil {
		return nil, err
	}

	resp, err := s.getFile(key)
	if err != nil {
		return nil, err
	}

	jotFile := &types.TextFile{
		Key:          key,
		Content:      resp.Content,
		ModifiedDate: statResp.ModifiedDate,
	}

	return jotFile, nil
}

func (s *JotStore) Create(ctx context.Context, content io.ReadCloser) (*types.TextFile, error) {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate shortid").WithCause(err)
	}

	key, err := sid.Generate()
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate shortid").WithCause(err)
	}

	password, err := s.passwordManager.Generate(key)
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

func (s *JotStore) Update(ctx context.Context, jotFile *types.TextFile) error {
	if err := s.backend.Put(jotFile.Key, jotFile.Content); err != nil {
		return errors.NewUnknownError("failed to write file into backend").WithCause(err)
	}

	return nil
}

func (s *JotStore) Delete(ctx context.Context, jotFile *types.TextFile) error {
	if err := s.backend.Delete(jotFile.Key); err != nil {
		return errors.NewUnknownError("failed to delete file from backend").WithCause(err)
	}

	return nil
}

func NewStore(cfg *config.Config, backend store.Backend, pm auth.PasswordManager) *JotStore {
	return &JotStore{
		passwordManager: pm,
		backend:         backend,
	}
}

package jot

import (
	"io"

	"github.com/kyleterry/jot/auth"
	"github.com/kyleterry/jot/config"
	"github.com/kyleterry/jot/jot/errors"
	"github.com/kyleterry/jot/jot/store"
	"github.com/kyleterry/jot/jot/store/backends"
	"github.com/teris-io/shortid"
)

// Version is the jot store application version
const Version = "0.1.1"

// JotStore wraps a backend implementation and creates/checks passwords for a jot
type JotStore struct {
	manager *auth.PasswordManager
	dataDir string
	backend store.Backend
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

func (s *JotStore) Stat(key string) (*JotFile, error) {
	resp, err := s.stat(key)
	if err != nil {
		return nil, err
	}

	return &JotFile{Key: key, ModifiedDate: resp.ModifiedDate}, nil
}

func (s *JotStore) GetFile(key string) (*JotFile, error) {
	statResp, err := s.stat(key)
	if err != nil {
		return nil, err
	}

	resp, err := s.getFile(key)
	if err != nil {
		return nil, err
	}

	jotFile := &JotFile{
		Key:          key,
		Content:      resp.Content,
		ModifiedDate: statResp.ModifiedDate,
	}

	return jotFile, nil
}

func (s *JotStore) CreateFile(content io.ReadCloser) (*JotFile, error) {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate shortid").WithCause(err)
	}

	key, err := sid.Generate()
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate shortid").WithCause(err)
	}

	password, err := s.manager.Generate(key)
	if err != nil {
		return nil, errors.NewUnknownError("failed to generate password").WithCause(err)
	}

	if err := s.backend.Put(key, content); err != nil {
		return nil, errors.NewUnknownError("failed to write file into backend").WithCause(err)
	}

	return &JotFile{
		Key:      key,
		Content:  content,
		Password: password,
	}, nil
}

func (s *JotStore) UpdateFile(etag string, suppliedPW string, jotFile *JotFile) error {
	if err := s.backend.Put(jotFile.Key, jotFile.Content); err != nil {
		return errors.NewUnknownError("failed to write file into backend").WithCause(err)
	}

	return nil
}

func (s *JotStore) DeleteFile(suppliedPW, key string) error {
	if err := s.backend.Delete(key); err != nil {
		return errors.NewUnknownError("failed to delete file from backend").WithCause(err)
	}

	return nil
}

func NewStore(cfg *config.Config, manager *auth.PasswordManager) (*JotStore, error) {
	backend := backends.NewFilesystem(backends.FilesystemOptions{
		Path: cfg.DataDir,
	})

	return &JotStore{
		manager: manager,
		dataDir: cfg.DataDir,
		backend: backend,
	}, nil
}

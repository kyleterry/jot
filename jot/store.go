package jot

import (
	"io"
	"io/ioutil"

	"github.com/cloudflare/gokey"
	"github.com/kyleterry/jot/config"
	"github.com/kyleterry/jot/jot/store"
	"github.com/kyleterry/jot/jot/store/backends"
	"github.com/pkg/errors"
	"github.com/teris-io/shortid"
)

const Version = "0.1.0"

func defaultSpec() *gokey.PasswordSpec {
	return &gokey.PasswordSpec{15, 3, 3, 1, 0, ""}
}

type JotStore struct {
	seed           []byte
	masterPassword string
	dataDir        string
	backend        store.Store
}

func (s *JotStore) CreateFile(content io.ReadCloser) (*JotFile, error) {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)
	if err != nil {
		return nil, err
	}

	key, err := sid.Generate()
	if err != nil {
		return nil, err
	}

	password, err := gokey.GetPass(s.masterPassword, key, s.seed, defaultSpec())
	if err != nil {
		return nil, err
	}

	if err := s.backend.Put(key, content); err != nil {
		return nil, err
	}

	return &JotFile{
		Key:      key,
		Content:  content,
		Password: password,
	}, err
}

func (s *JotStore) UpdateFile(suppliedPW string, jotFile *JotFile) error {
	password, err := gokey.GetPass(s.masterPassword, jotFile.Key, s.seed, defaultSpec())
	if err != nil {
		return err
	}

	if suppliedPW != password {
		return errors.New("invalid password")
	}

	if err := s.backend.Put(jotFile.Key, jotFile.Content); err != nil {
		return err
	}

	return nil
}

func (s *JotStore) DeleteFile(suppliedPW, key string) error {
	password, err := gokey.GetPass(s.masterPassword, key, s.seed, defaultSpec())
	if err != nil {
		return err
	}

	if suppliedPW != password {
		return errors.New("invalid password")
	}

	if err := s.backend.Delete(key); err != nil {
		return errors.Wrap(err, "could not delete jot")
	}

	return nil
}

func (s *JotStore) GetFile(key string) (*JotFile, error) {
	content, err := s.backend.Get(key)
	if err != nil {
		return nil, err
	}

	jotFile := &JotFile{
		Key:     key,
		Content: content,
	}

	return jotFile, nil
}

func NewStore(cfg *config.Config) (*JotStore, error) {
	seed, err := ioutil.ReadFile(cfg.SeedFile)
	if err != nil {
		return nil, err
	}

	backend := backends.NewFilesystem(backends.FilesystemOptions{
		Path: cfg.DataDir,
	})

	return &JotStore{
		seed:           seed,
		masterPassword: cfg.MasterPassword,
		dataDir:        cfg.DataDir,
		backend:        backend,
	}, nil
}

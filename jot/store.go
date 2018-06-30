package jot

import (
	"errors"
	"io/ioutil"
	"path/filepath"

	"github.com/cloudflare/gokey"
	"github.com/kyleterry/jot/config"
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
}

func (s *JotStore) CreateFile(content []byte) (*JotFile, error) {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)
	if err != nil {
		return nil, err
	}

	key, err := sid.Generate()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(s.dataDir, key)

	password, err := gokey.GetPass(s.masterPassword, key, s.seed, defaultSpec())
	if err != nil {
		return nil, err
	}

	jotFile := &JotFile{
		Key:      key,
		Content:  content,
		Password: password,
	}

	if err := s.writeFile(path, jotFile); err != nil {
		return nil, err
	}

	return &JotFile{
		Key:      key,
		Content:  content,
		Password: password,
	}, err
}

func (s *JotStore) UpdateFile(suppliedPW string, jotFile *JotFile) error {
	path := filepath.Join(s.dataDir, jotFile.Key)

	password, err := gokey.GetPass(s.masterPassword, jotFile.Key, s.seed, defaultSpec())
	if err != nil {
		return err
	}

	if suppliedPW != password {
		return errors.New("invalid password")
	}

	if err := s.writeFile(path, jotFile); err != nil {
		return err
	}

	return nil
}

func (s *JotStore) GetFile(key string) (*JotFile, error) {
	return s.loadFile(key)
}

func (s *JotStore) loadFile(key string) (*JotFile, error) {
	var err error

	path := filepath.Join(s.dataDir, key)

	jotFile := &JotFile{Key: key}

	jotFile.Content, err = ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return jotFile, nil
}

func (s *JotStore) writeFile(path string, jotFile *JotFile) error {
	return ioutil.WriteFile(path, jotFile.Content, 0644)
}

func NewStore(cfg *config.Config) (*JotStore, error) {
	seed, err := ioutil.ReadFile(cfg.SeedFile)
	if err != nil {
		return nil, err
	}

	return &JotStore{
		seed:           seed,
		masterPassword: cfg.MasterPassword,
		dataDir:        cfg.DataDir,
	}, nil
}

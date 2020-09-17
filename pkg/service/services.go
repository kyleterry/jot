package service

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/image"
	"github.com/kyleterry/jot/pkg/image/backend/filesystem"
	"github.com/kyleterry/jot/pkg/jot"
	"github.com/kyleterry/jot/pkg/jot/store/backends"
	"github.com/kyleterry/jot/pkg/text"
)

type Services interface {
	TextStore() text.StoreService
	ImageStore() image.StoreService
	PasswordManager() auth.PasswordManagerService
}

type DefaultServices struct {
	ts text.StoreService
	is image.StoreService
	pm auth.PasswordManagerService
}

func (s *DefaultServices) TextStore() text.StoreService {
	return s.ts
}

func (s *DefaultServices) ImageStore() image.StoreService {
	return s.is
}

func (s *DefaultServices) PasswordManager() auth.PasswordManagerService {
	return s.pm
}

func NewDefaultServices(cfg *config.Config) (*DefaultServices, error) {
	exists, err := auth.SeedFileExists(cfg.SeedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to setup services: %w", err)
	}

	if !exists {
		log.Printf("seedfile is missing; attempting to create one")
		if err := auth.MakeSeedFile(cfg); err != nil {
			return nil, fmt.Errorf("failed to setup services: %w", err)
		}

		log.Printf("created seedfile: %s", cfg.SeedFile)
	}

	seed, err := auth.LoadSeed(cfg.SeedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to setup services: %w", err)
	}

	pm := auth.NewPasswordManager(cfg.MasterPassword, seed)

	tsb, err := backends.NewFilesystem(backends.FilesystemOptions{
		Path:                 filepath.Join(cfg.DataDir, config.TextDirectory),
		FilePermissions:      config.FilePermissions,
		DirectoryPermissions: config.DirectoryPermissions,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup text storage backend: %w", err)
	}

	ts := jot.NewStore(cfg, tsb, pm)

	isb, err := filesystem.New(&filesystem.Config{
		Path:                 filepath.Join(cfg.DataDir, config.ImageDirectory),
		FilePermissions:      config.FilePermissions,
		DirectoryPermissions: config.DirectoryPermissions,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup image storage backend: %w", err)
	}

	is := image.NewStore(cfg, isb, pm)

	return &DefaultServices{
		ts: ts,
		is: is,
		pm: pm,
	}, nil
}

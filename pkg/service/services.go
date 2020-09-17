package service

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/id"
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
	IDManager() id.IDManagerService
}

type DefaultServices struct {
	ts  text.StoreService
	is  image.StoreService
	pm  auth.PasswordManagerService
	ids id.IDManagerService
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

func (s *DefaultServices) IDManager() id.IDManagerService {
	return s.ids
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

	services := &DefaultServices{}

	services.pm = auth.NewPasswordManager(cfg.MasterPassword, seed)

	idm, err := id.NewIDManager()
	if err != nil {
		return nil, fmt.Errorf("failed to setup services: %w", err)
	}
	services.ids = idm

	tsb, err := backends.NewFilesystem(backends.FilesystemOptions{
		Path:                 filepath.Join(cfg.DataDir, config.TextDirectory),
		FilePermissions:      config.FilePermissions,
		DirectoryPermissions: config.DirectoryPermissions,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup text storage backend: %w", err)
	}

	services.ts = jot.NewStore(tsb, services)

	isb, err := filesystem.New(&filesystem.Config{
		Path:                 filepath.Join(cfg.DataDir, config.ImageDirectory),
		FilePermissions:      config.FilePermissions,
		DirectoryPermissions: config.DirectoryPermissions,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup image storage backend: %w", err)
	}

	services.is = image.NewStore(isb, services)

	return services, nil
}

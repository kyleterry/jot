//go:build wireinject

package server

import (
	"github.com/google/wire"
	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/id"
	"github.com/kyleterry/jot/pkg/image"
	imagefs "github.com/kyleterry/jot/pkg/image/backend/filesystem"
	"github.com/kyleterry/jot/pkg/jot"
	textfs "github.com/kyleterry/jot/pkg/jot/store/backends"
	"github.com/kyleterry/jot/pkg/server"
	"github.com/kyleterry/jot/pkg/text"
)

func provideMasterPassword(cfg *config.Config) auth.MasterPassword {
	return cfg.MasterPassword
}

func provideSeedFileLocation(cfg *config.Config) auth.SeedFileLocation {
	return cfg.SeedFileLocation
}

func provideDataDir(cfg *config.Config) config.DataDir {
	return cfg.DataDir
}

func initServer() (*server.Server, error) {
	panic(wire.Build(
		config.ProviderSet,
		provideMasterPassword,
		provideSeedFileLocation,
		provideDataDir,
		auth.ProviderSet,
		wire.Bind(new(auth.PasswordManagerService), new(*auth.PasswordManager)),
		id.ProviderSet,
		textfs.BoundProviderSet,
		jot.ProviderSet,
		wire.Bind(new(text.StoreService), new(*jot.TextStore)),
		imagefs.BoundProviderSet,
		image.ProviderSet,
		wire.Bind(new(image.StoreService), new(*image.Store)),
		server.ProviderSet,
	))
}

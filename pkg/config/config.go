package config

import (
	"github.com/google/wire"
	"github.com/joeshaw/envdecode"
	"github.com/kyleterry/jot/pkg/auth"
)

var ProviderSet = wire.NewSet(
	New,
)

const (
	TextDirectoryName    = "txt"
	ImageDirectoryName   = "img"
	FilePermissions      = 0o640
	DirectoryPermissions = 0o740
)

// TODO: move this to the filesystem backend when img and txt are merged
type DataDir string

type Config struct {
	SeedFileLocation auth.SeedFileLocation `env:"JOT_SEED_FILE,required"`
	MasterPassword   auth.MasterPassword   `env:"JOT_MASTER_PASSWORD,required"`
	DataDir          DataDir               `env:"JOT_DATA_DIR,required"`
	BindAddr         string                `env:"JOT_BIND_ADDR,default=localhost:8095"`
	Host             string                `env:"JOT_HOST"`
}

func New() (*Config, error) {
	var cfg Config

	if err := envdecode.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

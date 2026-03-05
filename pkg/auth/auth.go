package auth

import (
	"fmt"
	"os"

	"github.com/cloudflare/gokey"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	DefaultSpec,
	NewSeedFile,
	ProvideAllSeedFiles,
	NewPasswordManager,
)

func DefaultSpec() *gokey.PasswordSpec {
	return &gokey.PasswordSpec{
		Length:         15,
		Upper:          3,
		Lower:          3,
		Digits:         1,
		Special:        0,
		AllowedSpecial: "",
	}
}

type PasswordManager struct {
	seedFiles []*SeedFile
}

func (p PasswordManager) Generate(key string) (string, error) {
	sf := p.seedFiles[0]

	password, err := gokey.GetPass(sf.password, key, sf.content, sf.spec)
	if err != nil {
		return "", err
	}

	return password, nil
}

func (p PasswordManager) IsMatch(key string, supplied string) (bool, error) {
	gen, err := p.Generate(key)
	if err != nil {
		return false, err
	}

	return supplied == gen, nil
}

func NewPasswordManager(seedFiles ...*SeedFile) *PasswordManager {
	return &PasswordManager{seedFiles: seedFiles}
}

type (
	MasterPassword   string
	SeedFileLocation string
)

type SeedFile struct {
	password string
	content  []byte
	spec     *gokey.PasswordSpec
}

func NewSeedFile(mp MasterPassword, loc SeedFileLocation, spec *gokey.PasswordSpec) (*SeedFile, error) {
	seedBytes, err := os.ReadFile(string(loc))
	if err != nil {
		return nil, fmt.Errorf("failed to load seed file: %w", err)
	}

	return &SeedFile{
		password: string(mp),
		content:  seedBytes,
		spec:     spec,
	}, nil
}

func ProvideAllSeedFiles(sf *SeedFile) []*SeedFile {
	return []*SeedFile{sf}
}

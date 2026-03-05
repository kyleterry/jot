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

// func SeedFileExists(f string) (bool, error) {
// 	if _, err := os.Stat(f); err != nil {
// 		if os.IsNotExist(err) {
// 			return false, nil
// 		}

// 		return false, err
// 	}

// 	return true, nil
// }

// func LoadSeed(f string) ([]byte, error) {
// 	seed, err := os.ReadFile(f)
// 	if err != nil {
// 		return nil, errors.NewUnknownError("failed to read seed file").WithCause(err)
// 	}

// 	return seed, nil
// }

// func MakeSeedFile(cfg *config.Config) error {
// 	b, err := gokey.GenerateEncryptedKeySeed(cfg.MasterPassword)
// 	if err != nil {
// 		return err
// 	}

// 	buf := bytes.NewBuffer(b)

// 	f, err := os.OpenFile(cfg.SeedFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
// 	if err != nil {
// 		return err
// 	}

// 	defer f.Close()

// 	if _, err := buf.WriteTo(f); err != nil {
// 		return err
// 	}

// 	return nil
// }

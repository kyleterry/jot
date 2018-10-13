package auth

import (
	"bytes"
	"io/ioutil"
	"os"

	"github.com/cloudflare/gokey"
	"github.com/kyleterry/jot/config"
	"github.com/kyleterry/jot/jot/errors"
)

func defaultSpec() *gokey.PasswordSpec {
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
	master string
	seed   []byte
}

func (p PasswordManager) Generate(key string) (string, error) {
	password, err := gokey.GetPass(p.master, key, p.seed, defaultSpec())
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

func NewPasswordManager(master string, seed []byte) PasswordManager {
	return PasswordManager{master, seed}
}

func SeedFileExists(f string) (bool, error) {
	if _, err := os.Stat(f); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func LoadSeed(f string) ([]byte, error) {
	seed, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, errors.NewUnknownError("failed to read seed file").WithCause(err)
	}

	return seed, nil
}

func MakeSeed(pass string) ([]byte, error) {
	return gokey.GenerateEncryptedKeySeed(pass)
}

func MakeSeedFile(cfg *config.Config) error {
	b, err := MakeSeed(cfg.MasterPassword)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(b)

	f, err := os.OpenFile(cfg.SeedFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		return err
	}

	if _, err := buf.WriteTo(f); err != nil {
		return err
	}

	return nil
}

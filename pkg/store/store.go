package store

import (
	"github.com/google/wire"
	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/errors"
	"github.com/kyleterry/jot/pkg/id"
)

var ProviderSet = wire.NewSet(
	wire.Struct(new(Options), "*"),
)

// Options holds the shared dependencies for all object stores.
type Options struct {
	PasswordManager *auth.PasswordManager
	IDManager       *id.IDManager
}

// NewIDAndPassword generates a unique key and its corresponding password.
func NewIDAndPassword(im *id.IDManager, pm *auth.PasswordManager) (key, password string, err error) {
	key, err = im.Generate()
	if err != nil {
		return "", "", errors.NewUnknownError("failed to generate id").WithCause(err)
	}

	password, err = pm.Generate(key)
	if err != nil {
		return "", "", errors.NewUnknownError("failed to generate password").WithCause(err)
	}

	return key, password, nil
}

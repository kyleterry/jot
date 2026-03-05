package id

import (
	"github.com/google/wire"
	"github.com/teris-io/shortid"
)

var ProviderSet = wire.NewSet(NewIDManager)

type IDManager struct {
	sid *shortid.Shortid
}

func (m *IDManager) Generate() (string, error) {
	return m.sid.Generate()
}

func NewIDManager() (*IDManager, error) {
	sid, err := shortid.New(0, shortid.DefaultABC, 2342)
	if err != nil {
		return nil, err
	}

	return &IDManager{
		sid: sid,
	}, nil
}

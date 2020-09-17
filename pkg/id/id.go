package id

import "github.com/teris-io/shortid"

type IDManagerService interface {
	Generate() (string, error)
}

type IDManager struct {
	sid *shortid.Shortid
}

func (m *IDManager) Generate() (string, error) {
	return m.sid.Generate()
}

func NewIDManager() (*IDManager, error) {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)
	if err != nil {
		return nil, err
	}

	return &IDManager{
		sid: sid,
	}, nil
}

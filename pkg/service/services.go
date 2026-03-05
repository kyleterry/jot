package service

import (
	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/image"
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


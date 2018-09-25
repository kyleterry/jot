package jot

import "io"

type JotFile struct {
	Key      string
	Content  io.ReadCloser
	Password string
}

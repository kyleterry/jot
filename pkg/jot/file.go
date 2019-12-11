package jot

import (
	"io"
	"time"
)

type JotFile struct {
	Key          string
	Content      io.ReadCloser
	Password     string
	ModifiedDate time.Time
}

func (j JotFile) ETag() string {
	return j.ModifiedDate.Format(time.RFC3339Nano)
}

func (j JotFile) ETagMatches(compare string) bool {
	return compare == j.ETag()
}

func (j JotFile) ShouldLoad(etag string) bool {
	return !j.ETagMatches(etag)
}

func (j JotFile) ShouldWrite(etag string) bool {
	return j.ETagMatches(etag)
}

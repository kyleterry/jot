package types

import (
	"io"
	"net/http"
	"time"
)

type Modifiable interface {
	ModifiedSince(timestamp string) bool
}

type Taggable interface {
	ETag() string
	ETagMatches(compare string) bool
	ShouldLoad(etag string) bool
	ShouldWrite(etag string) bool
}

type TextFile struct {
	Key          string
	Content      io.ReadCloser
	Password     string
	ModifiedDate time.Time
}

func (f TextFile) ETag() string {
	return f.ModifiedDate.Format(time.RFC3339Nano)
}

func (f TextFile) ETagMatches(compare string) bool {
	return compare == f.ETag()
}

func (f TextFile) ShouldLoad(etag string) bool {
	return !f.ETagMatches(etag)
}

func (f TextFile) ShouldWrite(etag string) bool {
	return f.ETagMatches(etag)
}

func (f TextFile) HasBeenModified(timestamp string) bool {
	modified := f.ModifiedDate.Format(http.TimeFormat)

	return timestamp == modified
}

type ImageData struct {
	Name        string
	Content     io.ReadCloser
	Description string
}

type GalleryFile struct {
	Key          string
	Images       map[string]*ImageData
	Password     string
	ModifiedDate time.Time
}

func (f GalleryFile) ETag() string {
	return f.ModifiedDate.Format(time.RFC3339Nano)
}

func (f GalleryFile) ETagMatches(compare string) bool {
	return compare == f.ETag()
}

func (f GalleryFile) ShouldLoad(etag string) bool {
	return !f.ETagMatches(etag)
}

func (f GalleryFile) ShouldWrite(etag string) bool {
	return f.ETagMatches(etag)
}

func (f GalleryFile) HasBeenModified(timestamp string) bool {
	modified := f.ModifiedDate.Format(http.TimeFormat)

	return timestamp == modified
}

func (f GalleryFile) Close() error {
	for _, rc := range f.Images {
		if err := rc.Content.Close(); err != nil {
			return err
		}
	}

	return nil
}

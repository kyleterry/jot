package types

import (
	"io"
	"time"
)

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

type ImageData struct {
	Name    string
	Content io.ReadCloser
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

func (f GalleryFile) Close() error {
	for _, rc := range f.Images {
		if err := rc.Content.Close(); err != nil {
			return err
		}
	}

	return nil
}

package types

import (
	"context"
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

type textFileKey struct{}

func WithTextFile(ctx context.Context, gf *TextFile) context.Context {
	return context.WithValue(ctx, textFileKey{}, gf)
}

func TextFileFromContext(ctx context.Context) *TextFile {
	return ctx.Value(textFileKey{}).(*TextFile)
}

type ImageData struct {
	Name        string
	Content     io.ReadCloser
	ContentType string
	Description string
}

type Images struct {
	Keys   []string
	Values map[string]*ImageData
}

func (i *Images) Add(key string, value *ImageData) {
	if i.Values == nil {
		i.Values = make(map[string]*ImageData)
	}

	i.Keys = append(i.Keys, key)
	i.Values[key] = value
}

type GalleryFile struct {
	ID           string
	Images       *Images
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
	for _, rc := range f.Images.Values {
		if err := rc.Content.Close(); err != nil {
			return err
		}
	}

	return nil
}

type galleryFileKey struct{}

func WithGalleryFile(ctx context.Context, gf *GalleryFile) context.Context {
	return context.WithValue(ctx, galleryFileKey{}, gf)
}

func GalleryFileFromContext(ctx context.Context) *GalleryFile {
	return ctx.Value(galleryFileKey{}).(*GalleryFile)
}

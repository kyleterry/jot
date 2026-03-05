package types

import (
	"context"
	"io"
	"net/http"
	"time"
)

// ObjectMeta holds modification time and provides ETag and cache-control
// methods shared by all stored objects.
type ObjectMeta struct {
	ModifiedDate time.Time
}

func (m ObjectMeta) ETag() string {
	return m.ModifiedDate.Format(time.RFC3339Nano)
}

func (m ObjectMeta) ETagMatches(compare string) bool {
	return compare == m.ETag()
}

func (m ObjectMeta) ShouldLoad(etag string) bool {
	return !m.ETagMatches(etag)
}

func (m ObjectMeta) ShouldWrite(etag string) bool {
	return m.ETagMatches(etag)
}

func (m ObjectMeta) HasBeenModified(timestamp string) bool {
	return timestamp == m.ModifiedDate.Format(http.TimeFormat)
}

type TextFile struct {
	Key      string
	Content  io.ReadCloser
	Password string
	ObjectMeta
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
	ID       string
	Images   *Images
	Password string
	ObjectMeta
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

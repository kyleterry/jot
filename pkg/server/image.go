package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/errors"
	"github.com/kyleterry/jot/pkg/image"
	"github.com/kyleterry/jot/pkg/types"
)

type imageServices interface {
	ImageStore() image.StoreService
}

type imageHandler struct {
	services      imageServices
	cfg           *config.Config
	getHandler    http.Handler
	postHandler   http.Handler
	deleteHandler http.Handler
}

func (h *imageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getHandler.ServeHTTP(w, r)
	case http.MethodPost, http.MethodPut:
		h.postHandler.ServeHTTP(w, r)
	case http.MethodDelete:
		h.deleteHandler.ServeHTTP(w, r)
	default:
		http.Error(w, "not implemented", http.StatusNotImplemented)
	}
}

func (h *imageHandler) post(w http.ResponseWriter, r *http.Request) {
	host := extractHost(h.cfg, r)

	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	images := map[string]io.ReadCloser{}

	images[handler.Filename] = file

	g, err := h.services.ImageStore().Create(r.Context(), images)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Jot-Password", g.Password)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("%s/img/%s\n", host, g.Key)))

	// r.Body = http.MaxBytesReader(w, r.Body, int64(MaxFileSize))
	// // err := r.ParseMultipartForm(int64(MaxFileSize))
	// // if err != nil {
	// // 	http.Error(w, err.Error(), http.StatusExpectationFailed)
	// // 	return
	// // }

	// mp, err := r.MultipartReader()
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusExpectationFailed)
	// 	return
	// }

	// // part, err := mp.NextPart()
	// // if err != nil {
	// // 	http.Error(w, err.Error(), http.StatusExpectationFailed)
	// // 	return
	// // }

	// f, err := mp.ReadForm(int64(MaxFileSize))
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusExpectationFailed)
	// 	return
	// }
}

func (h *imageHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	gallery := ctx.Value(CKImageGallery).(*types.GalleryFile)

	w.Header().Set("etag", gallery.ModifiedDate.Format(time.RFC3339Nano))

	defer gallery.Close()

	for _, image := range gallery.Images {
		w.Header().Set("content-disposition", fmt.Sprintf("filename=%s", image.Name))

		if _, err := io.Copy(w, image.Content); err != nil {
			log.Println("failed to write content to response")

			return
		}

		return
	}
}

func (h *imageHandler) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	gallery := ctx.Value(CKImageGallery).(*types.GalleryFile)

	if err := h.services.ImageStore().Delete(ctx, gallery); err != nil {
		WriteError(err, w)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// galleryPreloader instructs the store to do a stat on the object before loading
// the full content. This lets us check the If-Not-Match header browsers
// will pass in the presence of an ETag header for a resource.
func (h imageHandler) galleryPreloader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := r.Context().Value(CKObjectKey).(string)
		if !ok {
			WriteError(errors.NewInvalidKeyError(key), w)

			return
		}

		gallery, err := h.services.ImageStore().Stat(r.Context(), key)
		if err != nil {
			WriteError(err, w)

			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, CKImageGallery, gallery)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// checkPreconditions checks for If-None-Match in the case of GET and If-Match
// in the case of PUT and does a match against the Jot object's ETag (modified date).
// If GET and the tag doesn't match, then the content is loaded; otherwise it
// returns a 304. If PUT and the tag doesn't match, then a 412 is returned.
func (h imageHandler) checkPreconditions(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		gallery := ctx.Value(CKImageGallery).(*types.GalleryFile)

		switch r.Method {
		case http.MethodGet:
			precondition := r.Header.Get("If-None-Match")

			if precondition != "" {
				if !gallery.ShouldLoad(precondition) {
					w.WriteHeader(http.StatusNotModified)

					return
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// withGalleryLoaded is a middleware handler that loads the image gallery using a key derived
// from the http request URI and sets it in ctx.
func (h imageHandler) withGalleryLoaded(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		key, ok := ctx.Value(CKObjectKey).(string)
		if !ok {
			WriteError(errors.NewInvalidKeyError(key), w)

			return
		}

		gallery, err := h.services.ImageStore().Get(ctx, key)
		if err != nil {
			WriteError(err, w)

			return
		}

		ctx = context.WithValue(ctx, CKImageGallery, gallery)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newImageHandler(cfg *config.Config, s imageServices) *imageHandler {
	h := &imageHandler{
		cfg:      cfg,
		services: s,
	}

	objKey := NewMiddleware(keyRequired)
	galleryLoaded := NewMiddleware(
		(*h).galleryPreloader,
		(*h).checkPreconditions,
		(*h).withGalleryLoaded,
	)
	galleryLoaded = objKey.ExtendWith(galleryLoaded)

	h.getHandler = galleryLoaded.Wrap(http.HandlerFunc((*h).get))
	h.postHandler = http.HandlerFunc((*h).post)
	h.deleteHandler = galleryLoaded.Wrap(http.HandlerFunc((*h).delete))

	return h
}

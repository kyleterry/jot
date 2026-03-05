package server

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/errors"
	"github.com/kyleterry/jot/pkg/image"
	"github.com/kyleterry/jot/pkg/types"
)

type imageHandler struct {
	store           image.StoreService
	passwordManager auth.PasswordManagerService
	cfg             *config.Config
	getHandler      http.Handler
	postHandler     http.Handler
	deleteHandler   http.Handler
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

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusExpectationFailed)

		return
	}

	images := &types.Images{}

	imageFileHeaders := r.MultipartForm.File["images"]

	if len(imageFileHeaders) == 0 {
		http.Error(w, "no images found in request", http.StatusBadRequest)

		return
	}

	for _, header := range imageFileHeaders {
		file, err := header.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		images.Add(header.Filename, &types.ImageData{
			Name:    header.Filename,
			Content: file,
		})
	}

	g, err := h.store.Create(r.Context(), images)
	if err != nil {
		WriteError(err, w)

		return
	}

	imageURL, err := url.Parse(host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	imageURL = imageURL.JoinPath("img", g.ID)

	w.Header().Set("jot-password", g.Password)
	w.WriteHeader(http.StatusCreated)

	if _, err := fmt.Fprintf(w, "%s\n", imageURL.String()); err != nil {
		log.Println(fmt.Errorf("error while writing image url: %w", err))
	}
}

func (h *imageHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	gallery := types.GalleryFileFromContext(ctx)

	w.Header().Set("etag", gallery.ModifiedDate.Format(time.RFC3339Nano))
	w.Header().Set("last-modified", gallery.ModifiedDate.Format(http.TimeFormat))

	defer gallery.Close()

	_, tail := shiftPath(r.URL.Path)
	if tail == "" || tail == "/" {
		if err := galleryPage(gallery).Render(ctx, w); err != nil {
			log.Println(fmt.Errorf("error while rendering gallery page: %w", err))
		}

		return
	}

	tail = strings.TrimPrefix(tail, "/")

	image, ok := gallery.Images.Values[tail]
	if !ok {
		http.NotFound(w, r)

		return
	}

	w.Header().Set("content-disposition", fmt.Sprintf("filename=\"%s\"", image.Name))
	w.Header().Set("content-type", image.ContentType)

	buf := bytes.Buffer{}
	if _, err := buf.ReadFrom(image.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	seeker := bytes.NewReader(buf.Bytes())

	http.ServeContent(w, r, image.Name, gallery.ModifiedDate, seeker)
}

func (h *imageHandler) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	gallery := types.GalleryFileFromContext(ctx)

	if err := h.store.Delete(ctx, gallery); err != nil {
		WriteError(err, w)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// withGalleryPreloaded instructs the store to do a stat on the object before
// loading the full content. This lets us check the If-None-Match header
// browsers will pass in the presence of an ETag header for a resource.
func (h imageHandler) withGalleryPreloaded(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		key, ok := ObjectKeyFromContext(ctx)
		if !ok {
			WriteError(errors.NewInvalidKeyError(key), w)

			return
		}

		gallery, err := h.store.Stat(ctx, key)
		if err != nil {
			WriteError(err, w)

			return
		}

		ctx = WithTaggable(r.Context(), gallery)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// withGalleryLoaded is a middleware handler that loads the image gallery using
// a key derived from the http request URI and sets it in ctx.
func (h imageHandler) withGalleryLoaded(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		key, ok := ObjectKeyFromContext(ctx)
		if !ok {
			WriteError(errors.NewInvalidKeyError(key), w)

			return
		}

		gallery, err := h.store.Get(ctx, key)
		if err != nil {
			WriteError(err, w)

			return
		}

		ctx = types.WithGalleryFile(ctx, gallery)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func NewImageHandler(cfg *config.Config, store image.StoreService, pm auth.PasswordManagerService) *imageHandler {
	h := &imageHandler{
		cfg:             cfg,
		store:           store,
		passwordManager: pm,
	}

	authenticated := NewMiddleware(WithAuthenticationMiddleware(pm))
	keyRequired := NewMiddleware(WithKeyRequiredMiddleware)
	galleryLoaded := NewMiddleware(
		(*h).withGalleryPreloaded,
		WithPreconditionsMiddleware,
		(*h).withGalleryLoaded,
	)

	authenticated = keyRequired.ExtendWith(authenticated, galleryLoaded)
	keyRequired = keyRequired.ExtendWith(galleryLoaded)

	h.getHandler = keyRequired.Wrap(http.HandlerFunc((*h).get))
	h.postHandler = http.HandlerFunc((*h).post)
	h.deleteHandler = authenticated.Wrap(http.HandlerFunc((*h).delete))

	return h
}

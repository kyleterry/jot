package server

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
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

	writeCreatedResponse(w, r, h.cfg, "img", g.ID, g.Password)
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

func NewImageHandler(cfg *config.Config, store image.StoreService, pm auth.PasswordManagerService) *imageHandler {
	h := &imageHandler{
		cfg:             cfg,
		store:           store,
		passwordManager: pm,
	}

	authenticated := NewMiddleware(WithAuthenticationMiddleware(pm))
	keyRequired := NewMiddleware(WithKeyRequiredMiddleware)
	galleryLoaded := NewMiddleware(
		withPreloaded(func(ctx context.Context, key string) (*types.GalleryFile, error) {
			return h.store.Stat(ctx, key)
		}),
		WithPreconditionsMiddleware,
		withLoaded(func(ctx context.Context, key string) (*types.GalleryFile, error) {
			return h.store.Get(ctx, key)
		}, types.WithGalleryFile),
	)

	authenticated = keyRequired.ExtendWith(authenticated, galleryLoaded)
	keyRequired = keyRequired.ExtendWith(galleryLoaded)

	h.getHandler = keyRequired.Wrap(http.HandlerFunc((*h).get))
	h.postHandler = http.HandlerFunc((*h).post)
	h.deleteHandler = authenticated.Wrap(http.HandlerFunc((*h).delete))

	return h
}

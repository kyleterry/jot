package server

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/image"
)

type imageServices interface {
	ImageStore() image.StoreService
}

type imageHandler struct {
	services imageServices
	cfg      *config.Config
}

func (h *imageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPost, http.MethodPut:
		h.post(w, r)
	case http.MethodDelete:
		h.delete(w, r)
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
	key, _ := shiftPath(r.URL.Path)

	if key == "" {
		http.NotFound(w, r)

		return
	}

	ctx := r.Context()

	gallery, err := h.services.ImageStore().Get(ctx, key)
	if err != nil {
		WriteError(err, w)

		return
	}

	defer gallery.Close()

	for _, image := range gallery.Images {
		if _, err := io.Copy(w, image.Content); err != nil {
			log.Println("failed to write content to response")

			return
		}
	}
}

func (h *imageHandler) delete(w http.ResponseWriter, r *http.Request) {
}

func newImageHandler(cfg *config.Config, s imageServices) *imageHandler {
	return &imageHandler{
		cfg:      cfg,
		services: s,
	}
}

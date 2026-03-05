package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/errors"
	"github.com/kyleterry/jot/pkg/text"
	"github.com/kyleterry/jot/pkg/types"
)

// jotHandler handles GET, PUT, DELETE requests for a jot
type jotHandler struct {
	store           text.StoreService
	passwordManager auth.PasswordManagerService
	cfg             *config.Config
	getHandler      http.Handler
	postHandler     http.Handler
	putHandler      http.Handler
	deleteHandler   http.Handler
}

func (h jotHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handler http.Handler

	switch r.Method {
	case http.MethodGet:
		handler = h.getHandler
	case http.MethodPost:
		handler = h.postHandler
	case http.MethodPut:
		handler = h.putHandler
	case http.MethodDelete:
		handler = h.deleteHandler
	default:
		http.Error(w, "not implemented", http.StatusNotImplemented)

		return
	}

	handler.ServeHTTP(w, r)
}

func (h jotHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jotFile := types.TextFileFromContext(ctx)

	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.Header().Set("etag", jotFile.ModifiedDate.Format(time.RFC3339Nano))

	defer jotFile.Content.Close()

	seeker := jotFile.Content.(io.ReadSeeker)

	http.ServeContent(w, r, "", jotFile.ModifiedDate, seeker)
}

func (h jotHandler) post(w http.ResponseWriter, r *http.Request) {
	host := extractHost(h.cfg, r)

	jotFile, err := h.store.Create(r.Context(), r.Body)
	if err != nil {
		WriteError(err, w)

		return
	}

	jotURL, err := url.Parse(host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	jotURL = jotURL.JoinPath("txt", jotFile.Key)

	w.Header().Set("jot-password", jotFile.Password)
	w.WriteHeader(http.StatusCreated)

	if _, err := fmt.Fprintf(w, "%s\n", jotURL.String()); err != nil {
		log.Println(fmt.Errorf("error while writing jot url: %w", err))
	}
}

func (h jotHandler) put(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jotFile := types.TextFileFromContext(ctx)

	jotFile.Content = r.Body

	if err := h.store.Update(ctx, jotFile); err != nil {
		WriteError(err, w)

		return
	}

	http.Redirect(w, r, fmt.Sprintf("/%s", jotFile.Key), http.StatusSeeOther)
}

func (h jotHandler) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jotFile := types.TextFileFromContext(ctx)

	if err := h.store.Delete(ctx, jotFile); err != nil {
		WriteError(err, w)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// withJotPreloaded instructs the store to do a stat on the object before loading
// the full content. This lets us check the If-Not-Match header browsers
// will pass in the presence of an ETag header for a resource.
func (h jotHandler) withJotPreloaded(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		key, ok := ObjectKeyFromContext(ctx)
		if !ok {
			WriteError(errors.NewInvalidKeyError(key), w)

			return
		}

		jotFile, err := h.store.Stat(ctx, key)
		if err != nil {
			WriteError(err, w)

			return
		}

		ctx = WithTaggable(ctx, jotFile)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// withJotLoaded is a middleware handler that loads the jot using a key derived
// from the http request URI and sets it in ctx.
func (h jotHandler) withJotLoaded(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		key, ok := ObjectKeyFromContext(ctx)
		if !ok {
			WriteError(errors.NewInvalidKeyError(key), w)

			return
		}

		jotFile, err := h.store.Get(ctx, key)
		if err != nil {
			WriteError(err, w)

			return
		}

		ctx = types.WithTextFile(ctx, jotFile)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h jotHandler) checkModified(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			ctx := r.Context()
			jotFile := types.TextFileFromContext(ctx)

			msh := r.Header.Get("if-modified-since")

			if msh != "" {
				if !jotFile.HasBeenModified(msh) {
					w.WriteHeader(http.StatusNotModified)

					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// newJotHandler returns a new jotHandler setting the relevant middleware and creating
// a simple mux that switched on http method.
func NewJotHandler(cfg *config.Config, store text.StoreService, pm auth.PasswordManagerService) *jotHandler {
	h := &jotHandler{
		cfg:             cfg,
		store:           store,
		passwordManager: pm,
	}

	authenticated := NewMiddleware(WithAuthenticationMiddleware(h.passwordManager))
	keyRequired := NewMiddleware(WithKeyRequiredMiddleware)
	jotLoaded := NewMiddleware(
		(*h).withJotPreloaded,
		WithPreconditionsMiddleware,
		(*h).withJotLoaded,
	)

	authenticated = keyRequired.ExtendWith(authenticated, jotLoaded)
	keyRequired = keyRequired.ExtendWith(jotLoaded)

	h.getHandler = keyRequired.Wrap(http.HandlerFunc((*h).get))
	h.postHandler = http.HandlerFunc((*h).post)
	h.putHandler = authenticated.Wrap(http.HandlerFunc((*h).put))
	h.deleteHandler = authenticated.Wrap(http.HandlerFunc((*h).delete))

	return h
}

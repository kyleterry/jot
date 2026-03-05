package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
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
	jotFile, err := h.store.Create(r.Context(), r.Body)
	if err != nil {
		WriteError(err, w)

		return
	}

	writeCreatedResponse(w, r, h.cfg, "txt", jotFile.Key, jotFile.Password)
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

// NewJotHandler returns a new jotHandler setting the relevant middleware and creating
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
		withPreloaded(func(ctx context.Context, key string) (*types.TextFile, error) {
			return h.store.Stat(ctx, key)
		}),
		WithPreconditionsMiddleware,
		withLoaded(func(ctx context.Context, key string) (*types.TextFile, error) {
			return h.store.Get(ctx, key)
		}, types.WithTextFile),
	)

	authenticated = keyRequired.ExtendWith(authenticated, jotLoaded)
	keyRequired = keyRequired.ExtendWith(jotLoaded)

	h.getHandler = keyRequired.Wrap(http.HandlerFunc((*h).get))
	h.postHandler = http.HandlerFunc((*h).post)
	h.putHandler = authenticated.Wrap(http.HandlerFunc((*h).put))
	h.deleteHandler = authenticated.Wrap(http.HandlerFunc((*h).delete))

	return h
}

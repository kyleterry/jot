package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/errors"
	"github.com/kyleterry/jot/pkg/text"
	"github.com/kyleterry/jot/pkg/types"
)

type textServices interface {
	TextStore() text.StoreService
	PasswordManager() auth.PasswordManagerService
}

// jotHandler handles GET, PUT, DELETE requests for a jot
type jotHandler struct {
	services      textServices
	cfg           *config.Config
	getHandler    http.Handler
	postHandler   http.Handler
	putHandler    http.Handler
	deleteHandler http.Handler
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
	jotFile := ctx.Value(CKTextFile).(*types.TextFile)

	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.Header().Set("etag", jotFile.ModifiedDate.Format(time.RFC3339Nano))

	defer jotFile.Content.Close()

	if _, err := io.Copy(w, jotFile.Content); err != nil {
		log.Println("failed to write content to response")
		return
	}
}

func (h jotHandler) post(w http.ResponseWriter, r *http.Request) {
	host := extractHost(h.cfg, r)

	jotFile, err := h.services.TextStore().Create(r.Context(), r.Body)
	if err != nil {
		WriteError(err, w)

		return
	}

	w.Header().Set("Jot-Password", jotFile.Password)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("%s/txt/%s\n", host, jotFile.Key)))
}

func (h jotHandler) put(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jotFile := ctx.Value(CKTextFile).(*types.TextFile)

	jotFile.Content = r.Body

	if err := h.services.TextStore().Update(ctx, jotFile); err != nil {
		WriteError(err, w)

		return
	}

	http.Redirect(w, r, fmt.Sprintf("/%s", jotFile.Key), http.StatusSeeOther)
}

func (h jotHandler) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jotFile := ctx.Value(CKTextFile).(*types.TextFile)

	if err := h.services.TextStore().Delete(ctx, jotFile); err != nil {
		WriteError(err, w)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// jotPreloader instructs the store to do a stat on the object before loading
// the full content. This lets us check the If-Not-Match header browsers
// will pass in the presence of an ETag header for a resource.
func (h jotHandler) jotPreloader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := r.Context().Value(CKObjectKey).(string)
		if !ok {
			WriteError(errors.NewInvalidKeyError(key), w)

			return
		}

		jotFile, err := h.services.TextStore().Stat(r.Context(), key)
		if err != nil {
			WriteError(err, w)

			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, CKTextFile, jotFile)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// checkPreconditions checks for If-None-Match in the case of GET and If-Match
// in the case of PUT and does a match against the Jot object's ETag (modified date).
// If GET and the tag doesn't match, then the content is loaded; otherwise it
// returns a 304. If PUT and the tag doesn't match, then a 412 is returned.
func (h jotHandler) checkPreconditions(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		jotFile := ctx.Value(CKTextFile).(*types.TextFile)

		switch r.Method {
		case http.MethodGet:
			precondition := r.Header.Get("If-None-Match")

			if precondition != "" {
				if !jotFile.ShouldLoad(precondition) {
					w.WriteHeader(http.StatusNotModified)

					return
				}
			}
		case http.MethodPut:
			precondition := r.Header.Get("If-Match")

			if precondition != "" {
				if !jotFile.ShouldWrite(precondition) {
					WriteError(errors.NewETagMismatchError(), w)

					return
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// withJotLoaded is a middleware handler that loads the jot using a key derived
// from the http request URI and sets it in ctx.
func (h jotHandler) withJotLoaded(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		key, ok := ctx.Value(CKObjectKey).(string)
		if !ok {
			WriteError(errors.NewInvalidKeyError(key), w)

			return
		}

		jotFile, err := h.services.TextStore().Get(ctx, key)
		if err != nil {
			WriteError(err, w)

			return
		}

		ctx = context.WithValue(ctx, CKTextFile, jotFile)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authentication is a middleware handler that ensures the password is set
// in the request URI's query string, then checks to see if it's a valid password
// for the given path in the URI.
func (h jotHandler) authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := r.Context().Value(CKObjectKey).(string)
		if !ok {
			WriteError(errors.NewInvalidKeyError(key), w)

			return
		}

		supplied := r.URL.Query().Get("password")
		success, err := h.services.PasswordManager().IsMatch(key, supplied)
		if err != nil {
			err := errors.NewUnknownError("password manager failed").WithCause(err)
			WriteError(err, w)

			return
		}

		if !success {
			err := errors.NewInvalidPasswordError()
			WriteError(err, w)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// newJotHandler returns a new jotHandler setting the relevant middleware and creating
// a simple mux that switched on http method.
func newJotHandler(cfg *config.Config, services textServices) *jotHandler {
	h := &jotHandler{
		cfg:      cfg,
		services: services,
	}

	objKey := NewMiddleware(keyRequired)
	authenticated := NewMiddleware((*h).authentication)
	jotLoaded := NewMiddleware(
		(*h).jotPreloader,
		(*h).checkPreconditions,
		(*h).withJotLoaded,
	)
	authenticated = authenticated.ExtendWith(jotLoaded)
	authenticated = objKey.ExtendWith(authenticated)

	h.getHandler = objKey.Wrap(jotLoaded.Wrap(http.HandlerFunc((*h).get)))
	h.postHandler = http.HandlerFunc((*h).post)
	h.putHandler = authenticated.Wrap(http.HandlerFunc((*h).put))
	h.deleteHandler = authenticated.Wrap(http.HandlerFunc((*h).delete))

	return h
}

package server

import (
	"context"
	"net/http"

	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/errors"
)

// withPreloaded returns a MiddlewareFunc that calls stat to load object metadata
// (for ETag/precondition checks) and stores it in the context as a Taggable.
func withPreloaded[T Taggable](stat func(context.Context, string) (T, error)) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			key, ok := ObjectKeyFromContext(ctx)
			if !ok {
				WriteError(errors.NewInvalidKeyError(key), w)

				return
			}

			obj, err := stat(ctx, key)
			if err != nil {
				WriteError(err, w)

				return
			}

			ctx = WithTaggable(ctx, obj)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// withLoaded returns a MiddlewareFunc that calls get to fully load an object
// and stores it in the context using storeInCtx.
func withLoaded[T any](get func(context.Context, string) (T, error), storeInCtx func(context.Context, T) context.Context) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			key, ok := ObjectKeyFromContext(ctx)
			if !ok {
				WriteError(errors.NewInvalidKeyError(key), w)

				return
			}

			obj, err := get(ctx, key)
			if err != nil {
				WriteError(err, w)

				return
			}

			ctx = storeInCtx(ctx, obj)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// MiddlewareFunc is a type that allows the wrapping of an http.Handler in middleware.
type MiddlewareFunc func(http.Handler) http.Handler

// Middleware is a type that allows the wrapping of an http.Handler in middleware
// functions that will execute each other.
// See: https://en.wikipedia.org/wiki/Middleware
type Middleware struct {
	handlers []MiddlewareFunc
}

// Wrap takes a final http.Handler and wraps it with all the configured middleware
// handlers in the chain.
func (m Middleware) Wrap(h http.Handler) http.Handler {
	for i := range m.handlers {
		h = m.handlers[len(m.handlers)-1-i](h)
	}

	return h
}

// WithHandlers appends new middleware handlers to the current chain and
// returns a new Middleware.
func (m Middleware) WithHandlers(handlers ...MiddlewareFunc) Middleware {
	nmc := append([]MiddlewareFunc{}, m.handlers...)
	nmc = append(nmc, handlers...)

	return Middleware{nmc}
}

// ExtendWith takes other Middleware and extends the current one and returns a
// new Middleware.
func (m Middleware) ExtendWith(theirs ...Middleware) Middleware {
	nmc := append([]MiddlewareFunc{}, m.handlers...)

	for _, tmw := range theirs {
		nmc = append(nmc, tmw.handlers...)
	}

	return Middleware{nmc}
}

// NewMiddleware returns a new Middleware.
func NewMiddleware(handlers ...MiddlewareFunc) Middleware {
	return Middleware{handlers}
}

// WithAuthenticationMiddleware is a middleware handler that ensures a password
// has been provided via HTTP Basic Auth and then checks it against the
// password manager service using the provided object key in the URI. The
// username field is ignored.
func WithAuthenticationMiddleware(pm auth.PasswordManagerService) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, ok := ObjectKeyFromContext(r.Context())
			if !ok {
				WriteError(errors.NewInvalidKeyError(key), w)

				return
			}

			_, pw, ok := r.BasicAuth()
			if !ok || pw == "" {
				err := errors.NewInvalidPasswordError()
				WriteError(err, w)

				return
			}

			success, err := pm.IsMatch(key, pw)
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
}

// WithKeyRequiredMiddleware is used to ensure that an object key is present in
// the URI
func WithKeyRequiredMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, _ := shiftPath(r.URL.Path)

		if key == "" {
			http.NotFound(w, r)

			return
		}

		next.ServeHTTP(w, r.WithContext(WithObjectKey(r.Context(), key)))
	})
}

type objectKeyCtxKey struct{}

// WithObjectKey returns a copy of the parent context with the key set.
func WithObjectKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, objectKeyCtxKey{}, key)
}

// ObjectKeyFromContext returns the key from the context if it exists.
func ObjectKeyFromContext(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(objectKeyCtxKey{}).(string)
	return key, ok
}

// Taggable is an interface that defines methods for interacting with ETags.
// ETags are used for conditional requests in HTTP. Objects that implement this
// interface can be used in the WithPreconditionsMiddleware which will allow
// clients to check of they have been modified since they last cached the
// object.
type Taggable interface {
	ETag() string
	ETagMatches(string) bool
	ShouldLoad(string) bool
	ShouldWrite(string) bool
	HasBeenModified(string) bool
}

type taggableCtxKey struct{}

// WithTaggable returns a copy of the parent context with the Taggable object set.
func WithTaggable(ctx context.Context, t Taggable) context.Context {
	return context.WithValue(ctx, taggableCtxKey{}, t)
}

// TaggableFromContext returns the Taggable object from the context if it exists.
func TaggableFromContext(ctx context.Context) (Taggable, bool) {
	t, ok := ctx.Value(taggableCtxKey{}).(Taggable)
	return t, ok
}

// WithPreconditionsMiddleware checks for If-None-Match in the case of GET and If-Match
// in the case of PUT and does a match against the object's ETag (modified date).
// If GET and the tag doesn't match, then the content is loaded; otherwise it
// returns a 304. If PUT and the tag doesn't match, then a 412 is returned.
func WithPreconditionsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		to, ok := TaggableFromContext(ctx)
		if !ok {
			WriteError(errors.NewUnknownError("taggable object missing in context"), w)

			return
		}

		switch r.Method {
		case http.MethodGet, http.MethodHead:
			if r.Header.Get("if-none-match") != "" {
				precondition := r.Header.Get("if-none-match")

				if precondition != "" {
					if !to.ShouldLoad(precondition) {
						w.WriteHeader(http.StatusNotModified)

						return
					}
				}
			}
		case http.MethodPut:
			precondition := r.Header.Get("if-match")

			if precondition != "" {
				if !to.ShouldWrite(precondition) {
					WriteError(errors.NewETagMismatchError(), w)

					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

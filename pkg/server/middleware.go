package server

import "net/http"

type middlewareFunc func(http.Handler) http.Handler

// Middleware is a type that allows the wrapping of an http.Handler in middleware
// functions that will execute each other.
// See: https://en.wikipedia.org/wiki/Middleware
type Middleware struct {
	handlers []middlewareFunc
}

// Wrap takes a final http.Handler and wraps it with all the configured middleware
// handlers in the chain
func (m Middleware) Wrap(h http.Handler) http.Handler {
	for i := range m.handlers {
		h = m.handlers[len(m.handlers)-1-i](h)
	}

	return h
}

// WithHandlers appends new middleware handlers to the current chain and
// returns a new Middleware
func (m Middleware) WithHandlers(handlers ...middlewareFunc) Middleware {
	nmc := append([]middlewareFunc{}, m.handlers...)
	nmc = append(nmc, handlers...)

	return Middleware{nmc}
}

// ExtendWith takes another Middleware and extends the current one and
// returns a new Middleware
func (m Middleware) ExtendWith(theirs Middleware) Middleware {
	nmc := append([]middlewareFunc{}, m.handlers...)
	nmc = append(nmc, theirs.handlers...)

	return Middleware{nmc}
}

// NewMiddleware returns a new Middleware
func NewMiddleware(handlers ...middlewareFunc) Middleware {
	return Middleware{handlers}
}

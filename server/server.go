package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/kyleterry/jot/auth"
	"github.com/kyleterry/jot/config"
	"github.com/kyleterry/jot/jot"
	"github.com/kyleterry/jot/jot/errors"
)

// ContextKey prevents key collisions when using the global context
type ContextKey int

const (
	// ContextKeyJotFile is the key for context that holds the JotFile object loaded by middleware
	ContextKeyJotFile ContextKey = iota
)

const (
	//DefaultContentType is the default content type to use in responses that return the jot content
	DefaultContentType = "text/plain; charset=utf-8"
)

// Server listens to a port on an address as a HTTP server
// and uses gorilla/mux to route requests to HTTP handlers.
type Server struct {
	manager *auth.PasswordManager
	store   *jot.JotStore
	cfg     *config.Config
}

// New returns a new instance of a jot Server with
// the data from the seedFile loaded.
func New(cfg *config.Config, store *jot.JotStore, manager *auth.PasswordManager) *Server {
	return &Server{
		manager: manager,
		store:   store,
		cfg:     cfg,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var head string

	head, r.URL.Path = shiftPath(r.URL.Path)

	if head == "" {
		IndexHandler{s.store, s.cfg}.ServeHTTP(w, r)

		return
	}

	next := NewJotHandler(head, s.store, s.manager)
	next.ServeHTTP(w, r)
}

// Run starts an http listener on the configured bind address and returns a cancel
// function and an error channel.
func (s *Server) Run(ctx context.Context) (context.CancelFunc, chan error) {
	ctx, cancel := context.WithCancel(ctx)
	errch := make(chan error, 1)

	logging := handlers.LoggingHandler(os.Stdout, s)
	hsrv := &http.Server{Addr: s.cfg.BindAddr, Handler: logging}
	go func() {
		go s.run(hsrv, errch)

		<-ctx.Done()

		log.Println("shutting down")
		hsrv.Shutdown(ctx)
	}()

	scheme := "http"
	if hsrv.TLSConfig != nil {
		scheme = "https"
	}
	log.Printf("listening on: %s://%s", scheme, hsrv.Addr)

	return cancel, errch
}

func (s *Server) run(srv *http.Server, errch chan<- error) {
	err := srv.ListenAndServe()
	errch <- err
	close(errch)
}

var indexGetResponseTmpl = `Jot version %s

Usage: 
  Below examples use the curl command with -i set so we can see the headers.

  Creating a jot:
    Request:
      curl -i --data-binary @textfile.txt %s/
    Response:
      HTTP/1.1 201 Created
      Jot-Password: PE4VtqnNjrK3C07
      Date: Sat, 30 Jun 2018 19:09:03 GMT
      Content-Length: 32
      Content-Type: text/plain; charset=utf-8

      %s/LIU_JPnHp

  Getting a jot:
    Request:
      curl -i %s/LIU_JPnHp
	Response:
	  HTTP/1.1 200 OK
      Content-Type: text/plain; charset=utf-8
      Etag: 2018-06-30T19:09:03.735647737-07:00
      Date: Sat, 30 Jun 2018 19:10:13 GMT
      Content-Length: 38

	  here is my content from textfile.txt!

  Editing a jot:
    Request:
	  curl -i -H "If-Match: 2018-06-30T19:09:03.735647737-07:00" --data-binary @updated.txt %s/LIU_JPnHp?password=PE4VtqnNjrK3C07
    Response:
      HTTP/1.1 303 See Other
      Location: /LIU_JPnHp
      Date: Sat, 30 Jun 2018 19:14:26 GMT
      Content-Length: 0

Make note of the Jot-Password header as that's the password used to edit
your jot. ETag can be used in conjunction with If-None-Match and If-Match
for caching and collision prevention on PUT.

Source code: https://github.com/kyleterry/jot
`

// IndexHandler handles requests to the / endpoint
type IndexHandler struct {
	store *jot.JotStore
	cfg   *config.Config
}

func (h IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		host := extractHost(h.cfg, r)
		fmt.Fprintf(w, indexGetResponseTmpl, jot.Version, host, host, host)

		return
	} else if r.Method == http.MethodPost {
		jotFile, err := h.store.CreateFile(r.Body)
		if err != nil {
			WriteError(err, w)

			return
		}

		w.Header().Set("Jot-Password", jotFile.Password)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf("%s/%s", extractHost(h.cfg, r), jotFile.Key)))

		return
	}

	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// JotHandler handles GET, PUT, DELETE requests for a jot
type JotHandler struct {
	key     string
	store   *jot.JotStore
	manager *auth.PasswordManager

	mux map[string]http.Handler
}

func (h JotHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if endpoint, ok := h.mux[r.Method]; ok {
		endpoint.ServeHTTP(w, r)

		return
	}

	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h JotHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jotFile := ctx.Value(ContextKeyJotFile).(*jot.JotFile)

	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.Header().Set("etag", jotFile.ModifiedDate.Format(time.RFC3339Nano))

	defer jotFile.Content.Close()

	if _, err := io.Copy(w, jotFile.Content); err != nil {
		log.Println("failed to write content to response")
		return
	}
}

func (h JotHandler) put(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jotFile := ctx.Value(ContextKeyJotFile).(*jot.JotFile)

	pw := r.URL.Query().Get("password")

	jotFile.Content = r.Body

	if err := h.store.UpdateFile(r.Header.Get("if-match"), pw, jotFile); err != nil {
		WriteError(err, w)

		return
	}

	http.Redirect(w, r, fmt.Sprintf("/%s", jotFile.Key), http.StatusSeeOther)
}

func (h JotHandler) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jotFile := ctx.Value(ContextKeyJotFile).(*jot.JotFile)

	pw := r.URL.Query().Get("password")

	if err := h.store.DeleteFile(pw, jotFile.Key); err != nil {
		WriteError(err, w)

		return
	}

	w.WriteHeader(http.StatusNoContent)

	return
}

// jotPreloader instructs the store to do a stat on the object before loading
// the full content. This lets us check the If-Not-Match header browsers
// will pass in the presence of an ETag header for a resource.
func (h JotHandler) jotPreloader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jotFile, err := h.store.Stat(h.key)
		if err != nil {
			WriteError(err, w)

			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyJotFile, jotFile)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// checkPreconditions checks for If-None-Match in the case of GET and If-Match
// in the case of PUT and does a match against the Jot object's ETag (modified date).
// If GET and the tag doesn't match, then the content is loaded; otherwise it
// returns a 304. If PUT and the tag doesn't match, then a 412 is returned.
func (h JotHandler) checkPreconditions(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		jotFile := ctx.Value(ContextKeyJotFile).(*jot.JotFile)

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
func (h JotHandler) withJotLoaded(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jotFile, err := h.store.GetFile(h.key)
		if err != nil {
			WriteError(err, w)

			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyJotFile, jotFile)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authentication is a middleware handler that ensures the password is set
// in the request URI's query string, then checks to see if it's a valid password
// for the given path in the URI.
func (h JotHandler) authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		supplied := r.URL.Query().Get("password")
		success, err := h.manager.IsMatch(h.key, supplied)
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

// NewJotHandler returns a new JotHandler setting the relevant middleware and creating
// a simple mux that switched on http method.
func NewJotHandler(head string, store *jot.JotStore, manager *auth.PasswordManager) *JotHandler {
	h := &JotHandler{
		key:     head,
		store:   store,
		manager: manager,
	}

	authenticated := NewMiddleware((*h).authentication)
	jotLoaded := NewMiddleware(
		(*h).jotPreloader,
		(*h).checkPreconditions,
		(*h).withJotLoaded,
	)
	authenticated = authenticated.ExtendWith(jotLoaded)

	mux := map[string]http.Handler{
		http.MethodGet:    jotLoaded.Wrap(http.HandlerFunc((*h).get)),
		http.MethodPut:    authenticated.Wrap(http.HandlerFunc((*h).put)),
		http.MethodDelete: authenticated.Wrap(http.HandlerFunc((*h).delete)),
	}

	h.mux = mux

	return h
}

// shiftPath will take a path and pop off each entity at /, creating a head
// and tail. It's used to traverse paths and makes it useful to make branching
// decisions when handling an http request in ServeHTTP methods.
//
// TODO: we don't do much branching at all in jot, so we might want to get rid
// of this and just call r.URL.Path the key.
func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}

	return p[1:i], p[1:]
}

// extractHost checks to see if a Host is set in config and returns that, otherwise
// it returns the host generated by net/http.Request.
func extractHost(cfg *config.Config, r *http.Request) string {
	if cfg.Host != "" {
		return cfg.Host
	}

	return fmt.Sprintf("http://%s", r.Host)
}

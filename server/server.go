package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/kyleterry/jot/config"
	"github.com/kyleterry/jot/jot"
)

// Server listens to a port on an address as a HTTP server
// and uses gorilla/mux to route requests to HTTP handlers.
type Server struct {
	store *jot.JotStore
	cfg   *config.Config
}

// New returns a new instance of a jot Server with
// the data from the seedFile loaded.
func New(cfg *config.Config, store *jot.JotStore) *Server {
	return &Server{
		store: store,
		cfg:   cfg,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var head string

	head, r.URL.Path = shiftPath(r.URL.Path)

	if head == "" {
		IndexHandler{s.store, s.cfg}.ServeHTTP(w, r)

		return
	}

	JotHandler{head, s.store}.ServeHTTP(w, r)
}

func (s *Server) Run(ctx context.Context) (context.CancelFunc, chan error) {
	ctx, cancel := context.WithCancel(ctx)
	errch := make(chan error, 1)

	hsrv := &http.Server{Addr: s.cfg.BindAddr, Handler: s}
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
  Creating a jot:
    Request:
      curl -i --data-binary @textfile.txt %s/
    Response:
      HTTP/1.1 200 OK
      Jot-Password: PE4VtqnNjrK3C07
      Date: Sat, 30 Jun 2018 19:09:03 GMT
      Content-Length: 32
      Content-Type: text/plain; charset=utf-8

      %s/LIU_JPnHp

  Editing a jot:
    Request:
      curl -i --data-binary @updated.txt %s/LIU_JPnHp?password=PE4VtqnNjrK3C07
    Response:
      HTTP/1.1 303 See Other
      Location: /LIU_JPnHp
      Date: Sat, 30 Jun 2018 19:14:26 GMT
      Content-Length: 0

Make note of the Jot-Password header as that's the password used to edit
your jot.

Source code: https://github.com/kyleterry/jot
`

type IndexHandler struct {
	store *jot.JotStore
	cfg   *config.Config
}

func (h IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		host := extractHost(h.cfg, r)
		fmt.Fprintf(w, indexGetResponseTmpl, jot.Version, host, host, host)

		return
	} else if r.Method == "POST" {
		jotFile, err := h.store.CreateFile(r.Body)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Jot-Password", jotFile.Password)
		w.Write([]byte(fmt.Sprintf("%s/%s\n", extractHost(h.cfg, r), jotFile.Key)))

		return
	}

	http.Error(w, "Not found :(", http.StatusNotFound)
}

type JotHandler struct {
	key   string
	store *jot.JotStore
}

func (h JotHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	jotFile, err := h.store.GetFile(h.key)
	if err != nil {
		http.Error(w, "jot not found :(", http.StatusNotFound)

		return
	}

	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		defer jotFile.Content.Close()

		if _, err := io.Copy(w, jotFile.Content); err != nil {
			log.Println("failed to write content to response")
			return
		}

		return
	case "POST":
		pw := r.URL.Query().Get("password")

		jotFile.Content = r.Body

		if err := h.store.UpdateFile(pw, jotFile); err != nil {
			http.Error(w, "Nope", http.StatusForbidden)

			return
		}

		http.Redirect(w, r, fmt.Sprintf("/%s", jotFile.Key), http.StatusSeeOther)

		return
	case "DELETE":
		pw := r.URL.Query().Get("password")

		if err := h.store.DeleteFile(pw, jotFile.Key); err != nil {
			http.Error(w, "Nope", http.StatusForbidden)

			return
		}

		w.WriteHeader(http.StatusNoContent)

		return
	default:
		http.Error(w, "not implemented", http.StatusNotImplemented)

		return
	}
}

func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}

	return p[1:i], p[1:]
}

func extractHost(cfg *config.Config, r *http.Request) string {
	if cfg.Host != "" {
		return cfg.Host
	}

	return fmt.Sprintf("http://%s", r.Host)
}

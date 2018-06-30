package server

import (
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/kyleterry/jot/config"
)

// Server listens to a port on an address as a HTTP server
// and uses gorilla/mux to route requests to HTTP handlers.
type Server struct {
	seed           []byte
	masterPassword string
	dataDir        string
	bindAddr       string
}

// New returns a new instance of a jot Server with
// the data from the seedFile loaded or an error.
func New(cfg config.Config) (*Server, error) {
	seed, err := ioutil.ReadFile(cfg.SeedFile)
	if err != nil {
		return nil, err
	}

	return &Server{
		seed:           seed,
		masterPassword: cfg.MasterPassword,
		dataDir:        cfg.DataDir,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var head string

	head, r.URL.Path = shiftPath(r.URL.Path)

	if head == "" {
		Redirector{}.ServeHTTP(w, r)

		return
	}

	jotHandler, err := NewJotHandlerFromKey(head, s.seed, s.masterPassword, s.dataDir)
	if err != nil {
		http.Error(w, "Jot not found :(", http.StatusNotFound)

		return
	}

	jotHandler.ServeHTTP(w, r)
}

func (s *Server) Run() error {
	return http.ListenAndServe(s.bindAddr, s)
}

type Redirector struct {
}

type JotHandler struct {
}

func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}

	return p[1:i], p[1:]
}

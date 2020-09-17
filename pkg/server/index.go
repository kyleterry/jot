package server

import (
	"log"
	"net/http"

	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/jot"
	"github.com/kyleterry/jot/pkg/version"
)

// indexHandler handles requests to the / endpoint
type indexHandler struct {
	store *jot.JotStore
	cfg   *config.Config
}

func (h indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := extractHost(h.cfg, r)

	if r.Method != http.MethodGet {
		http.Error(w, "not implemented", http.StatusNotImplemented)

		return
	}

	ctx := IndexTemplateContext{
		Version: version.Version,
		Commit:  version.Commit,
		Host:    host,
	}
	if err := render(w, indexTemplate, ctx); err != nil {
		log.Println("err while rendering remplate: ", err)
	}
}

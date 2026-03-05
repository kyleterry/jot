package server

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/errors"
)

// WriteError takes an error and writes its text to the http response and sets
// the status code. If the err is an errors.StorageError, then the status code
// is extracted from the StatusCode field. Otherwise an
// http.StatusInternalServerError is used.
func WriteError(err error, w http.ResponseWriter) {
	if storeErr, ok := err.(*errors.StoreError); ok {
		for _, cause := range storeErr.Causes {
			log.Printf("[error cause] %s", cause)
		}

		http.Error(w, storeErr.Message, storeErr.StatusCode)

		return
	}

	log.Printf("[error cause] %s", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

// writeCreatedResponse builds the resource URL from the host, routePath, and key,
// sets the jot-password header, writes a 201 status, and writes the URL to the body.
func writeCreatedResponse(w http.ResponseWriter, r *http.Request, cfg *config.Config, routePath, key, password string) {
	host := extractHost(cfg, r)

	u, err := url.Parse(host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	u = u.JoinPath(routePath, key)

	w.Header().Set("jot-password", password)
	w.WriteHeader(http.StatusCreated)

	if _, err := fmt.Fprintf(w, "%s\n", u.String()); err != nil {
		log.Println(fmt.Errorf("error while writing response url: %w", err))
	}
}

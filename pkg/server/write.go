package server

import (
	"log"
	"net/http"

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

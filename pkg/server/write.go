package server

import (
	"log"
	"net/http"

	"github.com/kyleterry/jot/pkg/jot/errors"
)

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

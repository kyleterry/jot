package server

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
)

//go:embed favicon.ico
var favicon []byte

type faviconHandler struct{}

func (faviconHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")

	if _, err := w.Write(favicon); err != nil {
		log.Println(fmt.Errorf("failed to write favicon: %w", err))
	}
}

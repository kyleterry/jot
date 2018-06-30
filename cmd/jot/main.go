package main

import (
	"log"

	"github.com/joeshaw/envdecode"
	"github.com/kyleterry/jot/config"
	"github.com/kyleterry/jot/jot"
	"github.com/kyleterry/jot/server"
)

func main() {
	var cfg config.Config

	err := envdecode.Decode(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	store, err := jot.NewStore(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	srv, err := server.New(&cfg, store)
	if err != nil {
		log.Fatal(err)
	}

	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}

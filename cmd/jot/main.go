package main

import (
	"log"

	"github.com/joeshaw/envdecode"
	"github.com/kyleterry/jot/config"
	"github.com/kyleterry/jot/server"
)

func main() {
	var cfg config.Config
	err := envdecode.Decode(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	srv := server.New(cfg)
	if err := src.Run(); err != nil {
		log.Fatal(err)
	}
}

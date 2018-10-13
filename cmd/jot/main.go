package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joeshaw/envdecode"
	"github.com/kyleterry/jot/auth"
	"github.com/kyleterry/jot/config"
	"github.com/kyleterry/jot/jot"
	"github.com/kyleterry/jot/server"
)

func trap(cancel context.CancelFunc, errch chan error) int {
	sigch := make(chan os.Signal, 1)

	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)

	select {
	case sig := <-sigch:
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt:
			cancel()

			if err := <-errch; err != nil {
				log.Println(err)

				return 1
			}

			return 0

		}
	case err := <-errch:
		log.Println(err)

		return 1

	}

	return 0
}

func main() {
	var cfg config.Config

	err := envdecode.Decode(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	exists, err := auth.SeedFileExists(cfg.SeedFile)
	if err != nil {
		log.Fatal(err)
	}

	if !exists {
		log.Printf("seedfile is missing; attempting to create one")
		if err := auth.MakeSeedFile(&cfg); err != nil {
			log.Fatal(err)
		}

		log.Printf("created seedfile: %s", cfg.SeedFile)
	}

	seed, err := auth.LoadSeed(cfg.SeedFile)
	if err != nil {
		log.Fatal(err)
	}

	manager := auth.NewPasswordManager(cfg.MasterPassword, seed)

	store, err := jot.NewStore(&cfg, &manager)
	if err != nil {
		log.Fatal(err)
	}

	srv := server.New(&cfg, store, &manager)

	ctx := context.Background()
	cancel, errch := srv.Run(ctx)

	code := trap(cancel, errch)
	os.Exit(code)
}

package main

import (
	"log"

	"github.com/lipidbilayer/ddns/backend"
	"github.com/lipidbilayer/ddns/frontend"
	"github.com/lipidbilayer/ddns/shared"
	"golang.org/x/sync/errgroup"
)

var serviceConfig *shared.Config = &shared.Config{}

func init() {
	serviceConfig.Initialize()
}

func main() {
	serviceConfig.Validate()

	redis := shared.NewRedisBackend(serviceConfig)
	defer redis.Close()

	caddy := shared.NewCaddy(serviceConfig)

	var group errgroup.Group

	group.Go(func() error {
		lookup := backend.NewHostLookup(serviceConfig, redis, caddy)
		return backend.NewBackend(serviceConfig, lookup).Run()
	})

	group.Go(func() error {
		return frontend.NewFrontend(serviceConfig, redis, caddy).Run()
	})

	if err := group.Wait(); err != nil {
		log.Fatal(err)
	}
}

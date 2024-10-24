package main

import (
	"github.com/temporal-sa/temporal-entity-lifecycle-go/cmd/web/router"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/config"
	"log"
)

func main() {
	c := config.MustGetClient()
	defer c.Close()
	r, err := router.New(c)
	if err != nil {
		log.Fatalln("unable to initialize router", err)
	}
	err = r.Run("localhost:8081")
	if err != nil {
		log.Fatalln("unable to run router", err)
	}
}

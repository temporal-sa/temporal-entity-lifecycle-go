package main

import (
	"entity-demo/cmd/web/router"
	"entity-demo/config"
	"fmt"
	"log"
)

func main() {
	c := config.MustGetClient()
	defer c.Close()
	fmt.Println(c == nil)
	r, err := router.New(c)
	if err != nil {
		log.Fatalln("unable to initialize router", err)
	}
	err = r.Run("localhost:8081")
	if err != nil {
		log.Fatalln("unable to run router", err)
	}
}

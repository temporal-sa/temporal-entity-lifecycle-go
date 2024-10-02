package main

import (
	"github.com/temporal-sa/temporal-entity-lifecycle-go/config"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/constants"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/orchestrations"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/orchestrations/activity_handler"
	"go.temporal.io/sdk/worker"
	"log"
)

func main() {
	c := config.MustGetClient()
	defer c.Close()
	ah, err := activity_handler.New(c)
	if err != nil {
		log.Fatalln("Unable to initialize activity handler", err)
	}
	w := worker.New(c, constants.EntityTaskQueueName, worker.Options{})
	oh, err := orchestrations.New()
	if err != nil {
		log.Fatalln("unable to init orchestrations handler", err)
	}
	w.RegisterWorkflow(oh.Orchestration)
	w.RegisterActivity(ah.VerifyApprover)
	w.RegisterActivity(ah.SendNotifications)
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}

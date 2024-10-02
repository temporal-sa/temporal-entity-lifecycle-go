package main

import (
	"entity-demo/config"
	"entity-demo/constants"
	"entity-demo/orchestrations"
	"entity-demo/orchestrations/activity_handler"
	"go.temporal.io/sdk/worker"
	"log"
)

func main() {
	c := config.MustGetClient()
	defer c.Close()
	ah, err := activity_handler.New(activity_handler.WithClient(c))
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

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var err error
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)

	/*
		navigationStart -> redirectStart -> redirectEnd -> fetchStart -> domainLookupStart -> domainLookupEnd
		-> connectStart -> connectEnd -> requestStart -> responseStart -> responseEnd
		-> domLoading -> domInteractive -> domContentLoaded -> domComplete -> loadEventStart -> loadEventEnd
	*/

	c, err := NewConfig("config.yaml")
	if err != nil {
		log.Fatalln("Could not parse config file:", err)
	}

	StartJobs(c.Jobs, c.Selenium.URL)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		close(done)
	}()

	<-done

	fmt.Println("Exiting!")

}

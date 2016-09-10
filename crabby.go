package main

import (
	"flag"
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

	cfgFile := flag.String("config", "config.yaml", "Path to config file (default: ./config.yaml)")
	flag.Parse()

	c, err := NewConfig(cfgFile)
	if err != nil {
		log.Fatalln("Error reading config file.  Did you pass the -config flag?  Run with -h for help.\n", err)
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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	var err error
	var wg sync.WaitGroup

	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)

	cfgFile := flag.String("config", "config.yaml", "Path to config file (default: ./config.yaml)")
	flag.Parse()

	c, err := NewConfig(cfgFile)
	if err != nil {
		log.Fatalln("Error reading config file.  Did you pass the -config flag?  Run with -h for help.\n", err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	s, err := NewStorage(ctx, &wg, c)
	if err != nil {
		log.Fatalln(err)
	}

	StartJobs(ctx, &wg, c.Jobs, c.Selenium.URL, s)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func(cancel context.CancelFunc) {
		<-sigs
		cancel()
		close(done)
	}(cancel)

	<-done
	wg.Wait()

	fmt.Println("Exiting!")

}

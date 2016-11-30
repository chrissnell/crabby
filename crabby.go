package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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

	// If configured, start a goroutine to report internal metrics from the Go runtime
	if c.ReportInternalMetrics {
		go startInternalMetrics(ctx, &wg, s, c.InternalMetricsInterval)
	}

	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// We have to disable keep-alives to keep our server connection time
		// measurements accurate
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Transport: tr,
	}

	defer tr.CloseIdleConnections()

	StartJobs(ctx, &wg, c, s, client)

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

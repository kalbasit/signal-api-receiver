package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/kalbasit/signal-api-receiver/receiver"
	"github.com/kalbasit/signal-api-receiver/server"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	var (
		addr          string
		signalAPIURL  string
		signalAccount string
	)

	flag.StringVar(&addr, "addr", ":8105", "The address to listen and serve on")
	flag.StringVar(&signalAPIURL, "signal-api-url", "",
		"The URL of the Signal api including the scheme. e.g wss://signal-api.example.com")
	flag.StringVar(&signalAccount, "signal-account", "", "The account number for signal")

	flag.Parse()

	uri, err := url.Parse(signalAPIURL)
	if err != nil {
		log.Printf("error parsing the url %q: %s", signalAPIURL, err)

		return 1
	}

	if uri.Scheme == "" {
		log.Printf("the given url %q does not contain a scheme", uri)

		return 1
	}

	if uri.Host == "" {
		log.Printf("the given url %q does not contain a host", uri)

		return 1
	}

	uri.Path = fmt.Sprintf("/v1/receive/%s", signalAccount)
	log.Printf("the fully qualified URL for signal-api was computed as %q", uri.String())

	sarc, err := receiver.New(uri)
	if err != nil {
		log.Printf("error creating a new receiver: %s", err)

		return 1
	}

	srv := server.New(sarc)

	server := &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("Starting HTTP server on %s", addr)

	if err := server.ListenAndServe(); err != nil {
		log.Printf("error starting the server on %q: %s", addr, err)

		return 1
	}

	return 0
}

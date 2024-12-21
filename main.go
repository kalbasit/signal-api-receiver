package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/kalbasit/signal-api-receiver/receiver"
	"github.com/kalbasit/signal-api-receiver/server"
)

var (
	addr          string
	signalApiURL  string
	signalAccount string
)

func init() {
	flag.StringVar(&addr, "addr", ":8105", "The address to listen and serve on")
	flag.StringVar(&signalApiURL, "signal-api-url", "", "The URL of the Signal api including the scheme. e.g wss://signal-api.example.com")
	flag.StringVar(&signalAccount, "signal-account", "", "The account number for signal")
}

func main() {
	flag.Parse()

	uri, err := url.Parse(signalApiURL)
	if err != nil {
		log.Printf("error parsing the url %q: %s", signalApiURL, err)
		return
	}
	if uri.Scheme == "" {
		log.Printf("the given url %q does not contain a scheme", uri)
		return
	}
	if uri.Host == "" {
		log.Printf("the given url %q does not contain a host", uri)
		return
	}

	uri.Path = fmt.Sprintf("/v1/receive/%s", signalAccount)
	log.Printf("the fully qualified URL for signal-api was computed as %q", uri.String())

	sarc, err := receiver.New(uri)
	if err != nil {
		panic(err)
	}
	go sarc.ReceiveLoop()

	srv := server.New(sarc)

	log.Print("Starting HTTP server on :8105")

	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Printf("error starting the server on %q: %s", addr, err)
	}
}

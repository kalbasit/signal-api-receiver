package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/kalbasit/signal-api-receiver/receiver"
)

const usage = `
GET /receive/pop   => Return the oldest message
GET /receive/flush => Return all messages
`

// Server represent the HTTP server that exposes the pop/flush routes.
type Server struct {
	sarc       client
	repeatLast bool
	last       atomic.Pointer[receiver.Message]
}

type client interface {
	Connect() error
	ReceiveLoop() error
	Pop() *receiver.Message
	Flush() []receiver.Message
}

// New returns a new Server.
func New(sarc client, repeatLastMessage bool) *Server {
	s := &Server{sarc: sarc, repeatLast: repeatLastMessage}
	go s.start()

	return s
}

func (s *Server) start() {
	for {
		if err := s.sarc.ReceiveLoop(); err != nil {
			log.Printf("Error in the receive loop: %v", err)
		}
	Reconnect:
		if err := s.sarc.Connect(); err != nil {
			log.Printf("Error reconnecting: %v", err)
			time.Sleep(time.Second)

			goto Reconnect
		}
	}
}

// ServeHTTP implements the http.Handler interface
//
// /receive/pop
//
//	This returns status 200 and a receiver.Message or status 204 with no body
//
// /receive/flush
//
//	This returns status 200 and a list of receiver.Message
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET is the only allowed verb", http.StatusForbidden)

		return
	}

	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	//nolint:nestif
	if r.URL.Path == "/receive/pop" {
		msg := s.sarc.Pop()
		if s.repeatLast {
			if msg == nil {
				msg = s.last.Load()
			} else {
				s.last.Store(msg)
			}
		}

		if msg == nil {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(msg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	if r.URL.Path == "/receive/flush" {
		msgs := s.sarc.Flush()
		if s.repeatLast && len(msgs) > 0 {
			s.last.Store(&msgs[len(msgs)-1])
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(msgs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)

	notFoundMessage := []byte(fmt.Sprintf(
		"ERROR! GET %s is not supported. The supported paths are below:", r.URL.Path) + usage)

	if _, err := w.Write(notFoundMessage); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

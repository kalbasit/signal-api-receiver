package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/kalbasit/signal-api-receiver/receiver"
)

const usage = `
GET /receive/pop   => Return the oldest message
GET /receive/flush => Return all messages
`

// Server represent the HTTP server that exposes the pop/flush routes.
type Server struct {
	sarc client
}

type client interface {
	Pop() *receiver.Message
	Flush() []receiver.Message
}

// New returns a new Server.
func New(sarc client) *Server {
	return &Server{sarc: sarc}
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
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "GET is the only allowed verb")
		return
	}

	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.URL.Path == "/receive/pop" {
		msg := s.sarc.Pop()
		if msg == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(msg); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	if r.URL.Path == "/receive/flush" {
		msgs := s.sarc.Flush()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(msgs); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)

	notFoundMessage := []byte(fmt.Sprintf("ERROR! GET %s is not supported. The supported paths are below:", r.URL.Path) + usage)

	if _, err := w.Write(notFoundMessage); err != nil {
		log.Printf("error writing the body: %s", err)
	}
}

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/kalbasit/signal-api-receiver/pkg/receiver"
)

const (
	routeReceiveFlush = "/receive/flush"
	routeReceivePop   = "/receive/pop"

	contentType     = "Content-Type"
	contentTypeJSON = "application/json"
)

// Server represent the HTTP server that exposes the pop/flush routes.
type Server struct {
	logger zerolog.Logger

	router *chi.Mux

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
func New(ctx context.Context, sarc client, repeatLastMessage bool) *Server {
	s := &Server{
		logger:     *zerolog.Ctx(ctx),
		sarc:       sarc,
		repeatLast: repeatLastMessage,
	}

	s.createRouter()

	go s.start()

	return s
}

// ServeHTTP implements http.Handler and turns the Server type into a handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.router.ServeHTTP(w, r) }

func (s *Server) start() {
	log := s.logger.With().Str("func", "start").Logger()

	for {
		if err := s.sarc.ReceiveLoop(); err != nil {
			log.Error().Err(err).Msg("error in the receive loop")
		}
	Reconnect:
		if err := s.sarc.Connect(); err != nil {
			log.Error().Err(err).Msg("Error reconnecting: %v")
			time.Sleep(time.Second)

			goto Reconnect
		}
	}
}

func (s *Server) createRouter() {
	s.router = chi.NewRouter()

	s.router.Use(middleware.Heartbeat("/healthz"))
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(requestLogger(s.logger))
	s.router.Use(middleware.Recoverer)

	s.router.Get(routeReceiveFlush, s.receiveFlush)
	s.router.Get(routeReceivePop, s.receivePop)
}

func (s *Server) receivePop(w http.ResponseWriter, _ *http.Request) {
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

	w.Header().Set(contentType, contentTypeJSON)

	if err := json.NewEncoder(w).Encode(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) receiveFlush(w http.ResponseWriter, _ *http.Request) {
	msgs := s.sarc.Flush()
	if s.repeatLast && len(msgs) > 0 {
		s.last.Store(&msgs[len(msgs)-1])
	}

	w.Header().Set(contentType, contentTypeJSON)

	if err := json.NewEncoder(w).Encode(msgs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func requestLogger(logger zerolog.Logger) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			reqID := middleware.GetReqID(r.Context())

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				log := logger.With().
					Str("method", r.Method).
					Str("request-uri", r.RequestURI).
					Int("status", ww.Status()).
					Dur("elapsed", time.Since(startedAt)).
					Str("from", r.RemoteAddr).
					Str("reqID", reqID).
					Logger()

				switch r.Method {
				case http.MethodHead, http.MethodGet:
					log = log.With().Int("bytes", ww.BytesWritten()).Logger()
				case http.MethodPost, http.MethodPut, http.MethodPatch:
					log = log.With().Int64("bytes", r.ContentLength).Logger()
				}

				log.Info().Msg("handled request")
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}

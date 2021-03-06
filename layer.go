// Package layer implements a simple HTTP server middleware layer
// used internally by vinci to compose and trigger the middleware chain.
package layer

import (
	"gopkg.in/vinci-proxy/context.v0"
	"net/http"
)

const (
	// ErrorPhase defines error middleware phase idenfitier.
	ErrorPhase = "error"

	// RequestPhase defines the default middleware phase for request.
	RequestPhase = "request"
)

// FinalHandler stores the default http.Handler used as final middleware chain.
// You can customize this handler in order to reply with a default error response.
var FinalHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(502)
	w.Write([]byte("vinci: no route configured"))
})

// FinalErrorHandler stores the default http.Handler used as final middleware chain.
// You can customize this handler in order to reply with a default error response.
var FinalErrorHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(500)
	w.Write([]byte("vinci: internal server error"))
})

// Runnable represents the required interface for a runnable
type Runnable interface {
	Run(string, http.ResponseWriter, *http.Request, http.Handler)
}

// Pluggable represents a middleware pluggable interface implementing
// the required methods to plug in middleware handlers.
type Pluggable interface {
	// Use method is used to register a new middleware handler in the stack.
	Use(phase string, handler ...interface{})

	// UsePriority method is used to register a new middleware handler in a specific phase.
	UsePriority(string, Priority, ...interface{})

	// UseFinalHandler defines the middleware handler terminator
	UseFinalHandler(handler http.Handler)
}

// Middleware especifies the required interface that must be
// implemented by third-party middleware capable interfaces.
type Middleware interface {
	// Middleware embeds Runnable and Pluggable interfaces.
	Runnable
	Pluggable

	// Flush flushed the middleware handlers pool.
	Flush()
}

// Pool represents the phase-specific stack to store middleware functions.
type Pool map[string]*Stack

// Layer type represent an HTTP domain
// specific middleware layer with hieritance support.
type Layer struct {
	// stack stores the plugins registered in the current middleware instance.
	Pool Pool

	// finalHandler stores the final middleware chain handler.
	finalHandler http.Handler
}

// New creates a new middleware layer.
func New() *Layer {
	return &Layer{Pool: make(Pool), finalHandler: FinalHandler}
}

// Flush flushes the plugins stack.
func (s *Layer) Flush() {
	s.Pool = Pool{}
}

// Use registers a new request handler in the middleware stack.
func (s *Layer) Use(phase string, handler ...interface{}) {
	s.register(phase, Normal, handler...)
}

// UsePriority registers a new request handler in the middleware stack with the given priority.
func (s *Layer) UsePriority(phase string, priority Priority, handler ...interface{}) {
	s.register(phase, priority, handler...)
}

// UseFinalHandler uses a new http.Handler as final middleware call chain handler.
// This handler is tipically responsible of replying with a custom response
// or error (e.g: cannot route the request).
func (s *Layer) UseFinalHandler(fn http.Handler) {
	s.finalHandler = fn
}

func (s *Layer) register(phase string, priority Priority, handler ...interface{}) *Layer {
	if s.Pool[phase] == nil {
		s.Pool[phase] = &Stack{}
	}

	pool := s.Pool[phase]
	for _, h := range handler {
		// Vinci's plugin interface
		if mw, ok := h.(Plugin); ok {
			mw.Register(s)
			continue
		}

		// Otherwise infer function interface
		mw := AdaptFunc(h)
		if mw == nil {
			panic("vinci: unsupported middleware interface")
		}
		pool.Push(priority, mw)
	}

	return s
}

// Run triggers the middleware call chain for the given phase.
func (s *Layer) Run(phase string, w http.ResponseWriter, r *http.Request, h http.Handler) {
	defer func() {
		if phase == "error" {
			return
		}
		if re := recover(); re != nil {
			context.Set(r, "error", re)
			s.Run("error", w, r, FinalErrorHandler)
		}
	}()

	if h == nil {
		h = s.finalHandler
	}

	stack := s.Pool[phase]
	if stack == nil {
		if phase != "error" {
			h.ServeHTTP(w, r)
		}
		return
	}

	queue := stack.Join()
	for i := len(queue) - 1; i >= 0; i-- {
		h = queue[i](h)
	}

	h.ServeHTTP(w, r)
}

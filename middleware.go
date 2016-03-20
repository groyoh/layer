// Package layer implements a simple HTTP server middleware layer
// used internally by vinci to compose and trigger the middleware chain.
package layer

import (
	"gopkg.in/vinci-proxy/context.v0"
	"net/http"
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

// Handler represents an optional supported interface that could be implemented
// by middleware handlers.
type Handler interface {
	HandleHTTP(w http.ResponseWriter, r *http.Request, h http.Handler)
}

// HandlerFunc represents the required function interface for simple middleware handlers.
type HandlerFunc func(http.ResponseWriter, *http.Request)

// HandlerFuncNext represents a Negroni-like handler function notation.
type HandlerFuncNext func(w http.ResponseWriter, r *http.Request, h http.Handler)

// MiddlewareFunc represents the vinci's middleware capable interface.
type MiddlewareFunc func(h http.Handler) http.Handler

// Plugin represents the required interface.
type Plugin interface {
	// Register is designed to allow the plugin developers
	// to attach multiple middleware layers.
	Register(*Layer)
}

// Middleware especifies the required interface that must be
// implemented by middleware capable interfaces.
type Middleware interface {
	// Use method is used to register a new middleware handler in the stack.
	Use(handler interface{})

	// UsePhase method is used to register a new middleware handler in a specific phase.
	UsePhase(string, handler interface{})

	// UseFinalHandler defines the middleware handler terminator
	UseFinalHandler(handler http.Handler)
}

// Priority represents the middleware priority.
type Priority int

const (
	// Head priority defines the middleware handlers
	// in the head of the middleware stack.
	Head Priority = iota

	// Normal priority defines the middleware handlers
	// in the last middleware stack available.
	Normal

	// Tail priority defines the middleware handlers
	// in the tail of the middleware stack.
	Tail
)

// Stack stores the data to show.
type Stack struct {
	Head  []MiddlewareFunc
	Stack []MiddlewareFunc
	Tail  []MiddlewareFunc
}

// Push pushes a new middleware handler to the stack based on the given priority.
func (s *Stack) Push(order Priority, h MiddlewareFunc) {
	if order == Head {
		s.Head = append(s.Head, h)
	} else if order == Tail {
		s.Tail = append(s.Tail, h)
	} else {
		s.Stack = append(s.Stack, h)
	}
}

// Join joins the middleware functions into a unique slice.
func (s *Stack) Join() []MiddlewareFunc {
	return append(append(s.Head, s.Stack...), s.Tail...)
}

// Len returns the middleware stack length.
func (s *Stack) Len() int {
	return len(s.Stack) + len(s.Tail) + len(s.Head)
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

// Use registers a new request handler in the middleware stack.
func (s *Layer) Use(handler ...interface{}) {
	s.UsePhase("request", handler...)
}

// UseHead registers a new request handler in the middleware stack.
func (s *Layer) UseHead(handler ...interface{}) {
	s.push("request", Head, handler...)
}

// UseTail registers a new request handler in the middleware stack tail.
func (s *Layer) UseTail(handler ...interface{}) {
	s.push("request", Tail, handler...)
}

// UsePhase registers a new middleware handler in the current middleware stack.
func (s *Layer) UsePhase(phase string, handler ...interface{}) {
	s.push(phase, Normal, handler...)
}

// UsePhasePriority registers a new error handler in the current middleware stack.
func (s *Layer) UsePhasePriority(phase string, priority Priority, handler ...interface{}) {
	s.push(phase, priority, handler...)
}

// UseFinalHandler uses a new http.Handler as final middleware call chain handler.
// This handler is tipically responsible of replying with a custom response
// or error (e.g: cannot route the request).
func (s *Layer) UseFinalHandler(fn http.Handler) {
	s.finalHandler = fn
}

func (s *Layer) push(phase string, order Priority, handler ...interface{}) *Layer {
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
		pool.Push(order, mw)
	}

	return s
}

// Flush flushes the plugins stack.
func (s *Layer) Flush() {
	s.Pool = Pool{}
}

// SetAll sets the middleware pool overriding the existent one.
func (s *Layer) SetAll(stack Pool) {
	s.Pool = stack
}

// GetAll gets all the current middleware pool.
func (s *Layer) GetAll() Pool {
	return s.Pool
}

// GetAllByPhase gets all the current middleware stack by the given phase.
func (s *Layer) GetAllByPhase(phase string) *Stack {
	return s.Pool[phase]
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

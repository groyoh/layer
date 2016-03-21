package layer

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
	// memo stores the memorized pre-computed merged stack for better performance.
	memo []MiddlewareFunc

	// Head stores the head priority handlers.
	Head []MiddlewareFunc

	// Stack stores the middleware normal priority handlers.
	Stack []MiddlewareFunc

	// Tail stores the middleware tail priority handlers.
	Tail []MiddlewareFunc
}

// Push pushes a new middleware handler to the stack based on the given priority.
func (s *Stack) Push(order Priority, h MiddlewareFunc) {
	s.memo = nil // flush the memoized stack
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
	if s.memo != nil {
		return s.memo
	}
	s.memo = append(append(s.Head, s.Stack...), s.Tail...)
	return s.memo
}

// Len returns the middleware stack length.
func (s *Stack) Len() int {
	return len(s.Stack) + len(s.Tail) + len(s.Head)
}

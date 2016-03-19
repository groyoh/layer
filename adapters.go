package layer

import "net/http"

func adapt(h interface{}) MiddlewareFunc {
	// Vinci/Alice interface
	if mw, ok := h.(func(h http.Handler) http.Handler); ok {
		return MiddlewareFunc(mw)
	}

	// Negroni like interface
	if mw, ok := h.(func(w http.ResponseWriter, r *http.Request, h http.Handler)); ok {
		return adaptHandlerFuncNext(mw)
	}

	// Standard net/http function handler interface
	if mw, ok := h.(func(http.ResponseWriter, *http.Request)); ok {
		return adaptHandlerFunc(mw)
	}

	// Standard net/http handler
	if mw, ok := h.(http.Handler); ok {
		return adaptNativeHandler(mw)
	}

	// Vinci's built-in handler
	if mw, ok := h.(Handler); ok {
		return adaptHandler(mw)
	}

	return nil
}

func adaptHandlerFunc(fn HandlerFunc) MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fn(w, r)
		})
	}
}

func adaptHandlerFuncNext(fn HandlerFuncNext) MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fn(w, r, h)
		})
	}
}

func adaptHandler(fn Handler) MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fn.HandleHTTP(w, r, h)
		})
	}
}

func adaptNativeHandler(fn http.Handler) MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return fn
	}
}

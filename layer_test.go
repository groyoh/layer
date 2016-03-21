package layer

import (
	"github.com/nbio/st"
	"gopkg.in/vinci-proxy/utils.v0"
	"net/http"
	"testing"
)

func TestMiddleware(t *testing.T) {
	mw := New()

	mw.Use(RequestPhase, func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("foo", "bar")
			h.ServeHTTP(w, r)
		})
	})

	st.Expect(t, mw.Pool["request"].Len(), 1)

	w := utils.NewWriterStub()
	req := &http.Request{}
	mw.Run("request", w, req, nil)

	st.Expect(t, w.Header().Get("foo"), "bar")
}

func TestSimpleMiddlewareCallChain(t *testing.T) {
	mw := New()

	calls := 0
	fn := func(w http.ResponseWriter, r *http.Request, h http.Handler) {
		calls++
		h.ServeHTTP(w, r)
	}
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	})

	mw.Use(RequestPhase, fn)
	mw.Use(RequestPhase, fn)
	mw.Use(RequestPhase, fn)

	wrt := utils.NewWriterStub()
	req := &http.Request{}

	mw.Run("request", wrt, req, final)
	st.Expect(t, calls, 4)
}

func BenchmarkLayerRun(b *testing.B) {
	w := utils.NewWriterStub()
	req := &http.Request{}

	mw := New()
	for i := 0; i < 100; i++ {
		mw.Use(RequestPhase, func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("foo", string(i))
				h.ServeHTTP(w, r)
			})
		})
	}

	for n := 0; n < b.N; n++ {
		mw.Run(RequestPhase, w, req, http.HandlerFunc(nil))
	}
}

func BenchmarkStackLayers(b *testing.B) {
	w := utils.NewWriterStub()
	req := &http.Request{}

	handler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("foo", "bar")
			h.ServeHTTP(w, r)
		})
	}

	mw := New()
	for i := 0; i < 100; i++ {
		mw.UsePriority(RequestPhase, Head, handler)
		mw.UsePriority(RequestPhase, Normal, handler)
		mw.UsePriority(RequestPhase, Tail, handler)
	}

	for n := 0; n < b.N; n++ {
		mw.Run(RequestPhase, w, req, http.HandlerFunc(nil))
	}
}

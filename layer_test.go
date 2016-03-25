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

func TestNoHandlerRegistered(t *testing.T) {
	mw := New()

	st.Expect(t, mw.Pool["request"], (*Stack)(nil))

	w := utils.NewWriterStub()
	req := &http.Request{}
	mw.Run("request", w, req, nil)

	st.Expect(t, w.Code, 502)
	st.Expect(t, w.Body, []byte("vinci: no route configured"))
}

func TestFinalErrorHandling(t *testing.T) {
	mw := New()

	mw.Use("request", func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("something went wrong")
		})
	})

	st.Expect(t, mw.Pool["request"].Len(), 1)

	w := utils.NewWriterStub()
	req := &http.Request{}
	mw.Run("request", w, req, nil)

	st.Expect(t, w.Code, 500)
	st.Expect(t, w.Body, []byte("vinci: internal server error"))
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

func TestFlush(t *testing.T) {
	mw := New()

	mw.Use("request", func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	})
	mw.Flush()
	st.Expect(t, mw.Pool, Pool{})
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

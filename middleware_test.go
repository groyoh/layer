package layer

import (
	"github.com/nbio/st"
	"gopkg.in/vinci-proxy/utils.v0"
	"net/http"
	"testing"
)

func TestMiddleware(t *testing.T) {
	mw := New()

	mw.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("foo", "bar")
			h.ServeHTTP(w, r)
		})
	})

	st.Expect(t, mw.GetAll()["request"].Len(), 1)

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

	mw.Use(fn)
	mw.Use(fn)
	mw.Use(fn)

	wrt := utils.NewWriterStub()
	req := &http.Request{}

	mw.Run("request", wrt, req, final)
	st.Expect(t, calls, 4)
}

type responseStub struct {
	status int
	body   []byte
	header http.Header
}

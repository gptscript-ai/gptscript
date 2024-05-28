package sdkserver

import (
	"net/http"
	"runtime/debug"

	"github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
)

type middleware func(http.Handler) http.Handler

func apply(h http.Handler, m ...func(http.Handler) http.Handler) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

func contentType(contentTypes ...string) middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, ct := range contentTypes {
				w.Header().Add("Content-Type", ct)
			}
			h.ServeHTTP(w, r)
		})
	}
}

func logRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := context.GetLogger(r.Context())

		defer func() {
			if err := recover(); err != nil {
				l.Fields("stack", string(debug.Stack())).Errorf("Panic: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"stderr": "encountered an unexpected error"}`))
			}
		}()

		l.Infof("Handling request: method %s, path %s", r.Method, r.URL.Path)
		h.ServeHTTP(w, r)
		l.Infof("Handled request: method %s, path %s", r.Method, r.URL.Path)
	})
}

func addRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithNewRequestID(r.Context())))
	})
}

func addLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(
			w,
			r.WithContext(context.WithLogger(
				r.Context(),
				mvl.NewWithID(context.GetRequestID(r.Context())),
			)),
		)
	})
}

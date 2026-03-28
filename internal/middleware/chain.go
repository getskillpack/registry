package middleware

import "net/http"

// Chain applies wrappers outer-to-inner: Chain(f,g,h)(mux) => f(g(h(mux))).
func Chain(h http.Handler, wrappers ...func(http.Handler) http.Handler) http.Handler {
	for i := len(wrappers) - 1; i >= 0; i-- {
		h = wrappers[i](h)
	}
	return h
}

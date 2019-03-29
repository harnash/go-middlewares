package middlewares

import (
	"net/http"
)

type Middleware func(handler http.Handler) http.Handler

func Use(h http.Handler, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h)
	}
	return h
}

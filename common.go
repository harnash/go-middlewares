package middlewares

import (
	"net/http"
)

//Middleware is a convenient shortcut for http.Handler wrappers
type Middleware func(handler http.Handler) http.Handler

//Use will apply sets of middlewares to a given http.Handler
func Use(h http.Handler, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h)
	}
	return h
}

func UseFunc(h http.HandlerFunc, middlewares ...Middleware) http.Handler {
	var res http.Handler
	for _, middleware := range middlewares {
		res = middleware(h)
	}

	return res
}

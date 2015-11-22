package main

import (
	"net/http"
	"os"
)

type AuthHandler struct {
	username     string
	password     string
	authenticate bool
	next         http.Handler
}

func (handler AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	username, password, ok := r.BasicAuth()

	if handler.authenticate {
		if ok && username == handler.username && password == handler.password {
			handler.next.ServeHTTP(w, r)
		} else {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"apparatchik\"")
			w.WriteHeader(401)
		}
	} else {
		handler.next.ServeHTTP(w, r)
	}

}

func NewAuthHandler(next http.Handler) http.Handler {
	username := os.Getenv("AUTH_USERNAME")
	password := os.Getenv("AUTH_PASSWORD")
	authenticate := username != "" && password != ""
	return AuthHandler{username, password, authenticate, next}

}

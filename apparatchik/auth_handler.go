package main

import (
	"net/http"
	"os"
)

type AuthHandler struct {
	username     string
	password     string
	authenticate bool
}

func (handler AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	username, password, ok := r.BasicAuth()

	if handler.authenticate {
		if ok && username == handler.username && password == handler.password {
			next.ServeHTTP(w, r)
		} else {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"apparatchik\"")
			w.WriteHeader(401)
		}
	} else {
		next.ServeHTTP(w, r)
	}

}

func NewAuthHandler() AuthHandler {
	username := os.Getenv("AUTH_USERNAME")
	password := os.Getenv("AUTH_PASSWORD")
	authenticate := username != "" && password != ""
	return AuthHandler{username, password, authenticate}

}

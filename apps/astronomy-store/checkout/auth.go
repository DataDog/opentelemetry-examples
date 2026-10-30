package main

import (
	"crypto/subtle"
	"net/http"
)

const (
	basicAuthUsername = "frontend.astronomy-store"
	basicAuthPassword = "changeme"
)

// requireBasicAuth rejects requests without valid HTTP Basic credentials with 401 Unauthorized.
func requireBasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(username), []byte(basicAuthUsername)) != 1 ||
			subtle.ConstantTimeCompare([]byte(password), []byte(basicAuthPassword)) != 1 {
			logger.WarnContext(r.Context(), "checkout authentication failed", "usr.login", username)
			w.Header().Set("WWW-Authenticate", `Basic realm="checkout", charset="UTF-8"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

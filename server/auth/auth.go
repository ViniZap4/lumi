// server/auth/auth.go
package auth

import (
	"net/http"
	"os"
)

func Middleware(next http.HandlerFunc) http.HandlerFunc {
	password := os.Getenv("LUMI_PASSWORD")
	if password == "" {
		password = "dev"
	}

	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Lumi-Token")
		if token != password {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

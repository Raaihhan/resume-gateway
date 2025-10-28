package middleware

import (
	"net/http"
	"strings"
)

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

func CORSWith(cfg CORSConfig) func(http.Handler) http.Handler {
	methods := strings.Join(defaultIfEmpty(cfg.AllowedMethods, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}), ", ")
	headers := strings.Join(defaultIfEmpty(cfg.AllowedHeaders, []string{"Content-Type", "Authorization", "X-Request-Id"}), ", ")
	origins := cfg.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := "*"
			if len(origins) == 1 {
				origin = origins[0]
			} else if o := r.Header.Get("Origin"); o != "" && (contains(origins, "*") || contains(origins, o)) {
				origin = o
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func defaultIfEmpty(s []string, fb []string) []string {
	if len(s) == 0 {
		return fb
	}
	return s
}
func contains(a []string, v string) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

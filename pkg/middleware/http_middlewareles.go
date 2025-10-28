package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type ctxKey int

const (
	reqIDKey ctxKey = iota
)

// ====== HTTP chain helper ======
func ChainHTTP(mw ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(mw) - 1; i >= 0; i-- {
			final = mw[i](final)
		}
		return final
	}
}

// ====== Request ID (FIX: pakai context.WithValue standar) ======
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if id == "" {
				id = genID()
			}
			ctx := context.WithValue(r.Context(), reqIDKey, id)
			r = r.WithContext(ctx)
			w.Header().Set("X-Request-Id", id)
			next.ServeHTTP(w, r)
		})
	}
}

func GetReqIDFrom(r *http.Request) string {
	if v, _ := r.Context().Value(reqIDKey).(string); v != "" {
		return v
	}
	return ""
}

func genID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// ====== Recover from panic (HTTP) ======
func Recover() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("panic: %v", rec)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// ====== Logging (HTTP) ======
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func Logging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w}
			start := time.Now()
			next.ServeHTTP(sw, r)
			dur := time.Since(start)

			ip := clientIP(r)
			reqID := GetReqIDFrom(r)
			log.Printf("http method=%s path=%s status=%d bytes=%d dur=%s ip=%s req_id=%s",
				r.Method, r.URL.Path, sw.status, sw.bytes, dur, ip, reqID)
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return xr
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ====== API Key (HTTP) – optional ======
func APIKeyHTTP(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if strings.TrimSpace(expected) == "" {
			return next // open
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok := false
			if k := r.Header.Get("X-API-Key"); k != "" && k == expected {
				ok = true
			}
			if !ok {
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "ApiKey ") && strings.TrimPrefix(auth, "ApiKey ") == expected {
					ok = true
				} else if strings.HasPrefix(auth, "Bearer ") && strings.TrimPrefix(auth, "Bearer ") == expected {
					ok = true
				}
			}
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, `{"error":"unauthorized"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

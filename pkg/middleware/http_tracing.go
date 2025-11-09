// pkg/middleware/http_tracing.go
package middleware

import (
	"net/http"

	"github.com/opentracing/opentracing-go"
)

func TracingHTTP() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tracer := opentracing.GlobalTracer()
			var span opentracing.Span

			// extract from incoming headers
			wireContext, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
			if err == nil {
				span = tracer.StartSpan(r.URL.Path, opentracing.ChildOf(wireContext))
			} else {
				span = tracer.StartSpan(r.URL.Path)
			}
			defer span.Finish()

			ctx := opentracing.ContextWithSpan(r.Context(), span)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

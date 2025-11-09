// pkg/middleware/grpc_tracing.go
package middleware

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func GRPCTracingUnary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		var parentCtx context.Context = ctx

		// extract span context from metadata
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			carrier := metadataReaderWriter{md}
			if spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier); err == nil {
				span := opentracing.StartSpan(info.FullMethod, opentracing.ChildOf(spanCtx))
				defer span.Finish()
				parentCtx = opentracing.ContextWithSpan(ctx, span)
			} else {
				span := opentracing.StartSpan(info.FullMethod)
				defer span.Finish()
				parentCtx = opentracing.ContextWithSpan(ctx, span)
			}
		} else {
			span := opentracing.StartSpan(info.FullMethod)
			defer span.Finish()
			parentCtx = opentracing.ContextWithSpan(ctx, span)
		}

		return handler(parentCtx, req)
	}
}

// metadataReaderWriter adapts grpc metadata to opentracing TextMapReader/TextMapWriter
type metadataReaderWriter struct {
	metadata.MD
}

func (w metadataReaderWriter) Set(key, val string) {
	key = canonicalize(key)
	w.MD[key] = append(w.MD[key], val)
}

func (w metadataReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vals := range w.MD {
		for _, v := range vals {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func canonicalize(k string) string {
	// grpc metadata keys are lowercase already
	return k
}

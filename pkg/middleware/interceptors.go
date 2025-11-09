package middleware

import (
	"context"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Pakai di server: grpc.NewServer(WithUnaryServerInterceptors())

func WithUnaryServerInterceptors() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(
		GRPCTracingUnary(),
		loggingUnary(),
		recoverUnary(),
		timeoutUnary(5*time.Second),
		// apiKeyUnary("") // aktifkan kalau mau pakai API key di gRPC
	)
}

func loggingUnary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Printf("grpc method=%s dur=%s err=%v", info.FullMethod, time.Since(start), err)
		return resp, err
	}
}

func recoverUnary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("grpc panic: %v", r)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func timeoutUnary(d time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		c, cancel := context.WithTimeout(ctx, d)
		defer cancel()
		done := make(chan struct{})
		var (
			resp interface{}
			err  error
		)
		go func() {
			resp, err = handler(c, req)
			close(done)
		}()
		select {
		case <-done:
			return resp, err
		case <-c.Done():
			if c.Err() == context.DeadlineExceeded {
				return nil, status.Errorf(codes.DeadlineExceeded, "deadline exceeded")
			}
			return nil, status.Errorf(codes.Canceled, "canceled")
		}
	}
}

// ====== API Key (gRPC) – optional ======
func apiKeyUnary(expected string) grpc.UnaryServerInterceptor {
	expected = strings.TrimSpace(expected)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// jika expected kosong → lewati (open)
		if expected == "" {
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		var ok bool

		// x-api-key
		if keys := md.Get("x-api-key"); len(keys) > 0 && keys[0] == expected {
			ok = true
		}
		// authorization: ApiKey <key> / Bearer <key>
		if !ok {
			if auths := md.Get("authorization"); len(auths) > 0 {
				auth := auths[0]
				if strings.HasPrefix(auth, "ApiKey ") && strings.TrimPrefix(auth, "ApiKey ") == expected {
					ok = true
				} else if strings.HasPrefix(auth, "Bearer ") && strings.TrimPrefix(auth, "Bearer ") == expected {
					ok = true
				}
			}
		}
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
		}
		return handler(ctx, req)
	}
}

package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	_ "strconv"
	"syscall"
	"time"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	_ "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	resumev1 "github.com/Raaihhan/resume-gateway/gen/go/proto/resume/v1"
	"github.com/Raaihhan/resume-gateway/internal/resume"
	"github.com/Raaihhan/resume-gateway/pkg/config"
	mw "github.com/Raaihhan/resume-gateway/pkg/middleware"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config.yaml")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("loaded config from: %s", configPath)

	// === Repo & Service
	repo, err := resume.NewMySQLRepo(cfg.Database.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()
	svc := resume.NewService(repo)

	// === gRPC
	lis, err := net.Listen("tcp", cfg.Server.GRPCAddr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer(mw.WithUnaryServerInterceptors())
	resumev1.RegisterResumeGatewayServiceServer(grpcServer, svc)
	reflection.Register(grpcServer)
	go func() {
		log.Printf("gRPC listening on %s", cfg.Server.GRPCAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	// === REST (manual)
	mux := http.NewServeMux()
	mo := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}
	uo := protojson.UnmarshalOptions{DiscardUnknown: true}

	// GET /v1/profile
	mux.HandleFunc("/v1/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		resp, err := svc.GetProfile(r.Context(), &resumev1.GetProfileRequest{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, &mo, resp)
	})

	// GET /v1/projects?page_number=&page_size=&search_query=
	// (tetap backward compatible: ?page= & ?q= juga diterima)
	mux.HandleFunc("/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		req := &resumev1.ListProjectsRequest{
			PageNumber:  firstNonEmpty(q.Get("page_number"), q.Get("page")),
			PageSize:    q.Get("page_size"),
			SearchQuery: firstNonEmpty(q.Get("search_query"), q.Get("q")),
		}
		resp, err := svc.ListProjects(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, &mo, resp)
	})

	// POST /v1/contact {name,email,message}
	mux.HandleFunc("/v1/contact", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var req resumev1.CreateContactMessageRequest
		if err := uo.Unmarshal(b, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err := svc.CreateContactMessage(r.Context(), &req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, &mo, resp)
	})

	// chain middleware
	h := mw.ChainHTTP(
		mw.RequestID(),
		mw.Recover(),
		mw.Logging(),
		mw.APIKeyHTTP(cfg.Server.APIKey), // kosong = off
		mw.CORSWith(mw.CORSConfig{
			AllowedOrigins: cfg.CORS.AllowedOrigins,
			AllowedMethods: cfg.CORS.AllowedMethods,
			AllowedHeaders: cfg.CORS.AllowedHeaders,
		}),
	)(mux)

	httpSrv := &http.Server{Addr: cfg.Server.HTTPAddr, Handler: h}
	go func() {
		log.Printf("HTTP (REST) listening on %s", cfg.Server.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, mo *protojson.MarshalOptions, m proto.Message) {
	w.Header().Set("Content-Type", "application/json")
	b, err := mo.Marshal(m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

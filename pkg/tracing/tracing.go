// pkg/tracing/tracing.go
package tracing

import (
	"fmt"
	"io"
	"log"

	"github.com/opentracing/opentracing-go"
	jconfig "github.com/uber/jaeger-client-go/config"
)

type Config struct {
	Enabled     bool
	ServiceName string
	AgentHost   string
	AgentPort   int
	Sampler     float64
}

func Init(cfg Config) (opentracing.Tracer, io.Closer, error) {
	if !cfg.Enabled {
		tr := opentracing.NoopTracer{}
		opentracing.SetGlobalTracer(tr)
		return tr, io.NopCloser(nil), nil
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = "resume-gateway"
	}
	if cfg.AgentHost == "" {
		cfg.AgentHost = "127.0.0.1"
	}
	if cfg.AgentPort == 0 {
		cfg.AgentPort = 6831
	}
	if cfg.Sampler == 0 {
		cfg.Sampler = 1
	}

	jcfg := jconfig.Configuration{
		ServiceName: cfg.ServiceName,
		Sampler: &jconfig.SamplerConfig{
			Type:  "const",
			Param: cfg.Sampler,
		},
		Reporter: &jconfig.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: fmt.Sprintf("%s:%d", cfg.AgentHost, cfg.AgentPort),
			CollectorEndpoint:  "http://localhost:14268/api/traces",
		},
	}

	tracer, closer, err := jcfg.NewTracer()
	if err != nil {
		return nil, nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	log.Printf("jaeger tracing enabled -> %s:%d", cfg.AgentHost, cfg.AgentPort)
	return tracer, closer, nil
}

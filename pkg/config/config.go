// pkg/config/config.go
package config

import (
	"fmt"
	"net/url"
)

// Root config struct sesuai config.yaml
type Config struct {
	Env      string   `yaml:"env"`
	Server   Server   `yaml:"server"`
	CORS     CORS     `yaml:"cors"`
	Database Database `yaml:"database"`
	Log      Log      `yaml:"log"`
	Tracing  Tracing  `yaml:"tracing"`
}

type Server struct {
	HTTPAddr string `yaml:"http_addr"` // contoh: ":8080"
	GRPCAddr string `yaml:"grpc_addr"` // contoh: ":9090"
	APIKey   string `yaml:"api_key"`   // opsional ("" = off)
}

type CORS struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

type Database struct {
	Driver   string            `yaml:"driver"` // "mysql"
	Host     string            `yaml:"host"`   // "127.0.0.1"
	Port     int               `yaml:"port"`   // 3306
	User     string            `yaml:"user"`   // "resume_app"
	Password string            `yaml:"password"`
	Name     string            `yaml:"name"`   // "resume"
	Params   map[string]string `yaml:"params"` // parseTime, charset, loc, dll.
}

type Log struct {
	Level string `yaml:"level"` // "info" | "debug" | ...
}

type Tracing struct {
	Enabled     bool    `yaml:"enabled"`
	ServiceName string  `yaml:"service_name"`
	AgentHost   string  `yaml:"agent_host"`
	AgentPort   int     `yaml:"agent_port"`
	Sampler     float64 `yaml:"sampler"`
}

// DSN membangun connection string MySQL dari field YAML.
// Hasil contoh: user:pass@tcp(127.0.0.1:3306)/resume?charset=utf8mb4&loc=Asia%2FJakarta&parseTime=true
func (d Database) DSN() string {
	if d.Driver == "" || d.Driver == "mysql" {
		q := url.Values{}
		for k, v := range d.Params {
			q.Set(k, v)
		}
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
			d.User, d.Password, d.Host, d.Port, d.Name, q.Encode())
	}
	// tambahkan driver lain bila diperlukan
	return ""
}

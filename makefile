# ============ Service Vars ============
SERVICE_NAME   ?= resume-gateway
APP_NAME       ?= resume-api
CMD_DIR        := ./cmd/api
PROTO_DIR      := api
BIN_DIR        := bin

# ============ Config Paths ============
CONFIG_DIR      := ./pkg/config
CONFIG_EXAMPLE  := $(CONFIG_DIR)/config_example.yaml
CONFIG_FILE     := $(CONFIG_DIR)/config.yaml

# ============ DB Seed Helper ============
DB_USER ?= root
DB_PASS ?= root
DB_HOST ?= 127.0.0.1
DB_PORT ?= 3306

.PHONY: help config proto tidy build run run-m1 run-windows test fmt vet clean \
        docker-build run-docker up down logs db-seed env

help:
	@echo "Targets:"
	@echo "  config   - copy $(CONFIG_EXAMPLE) -> $(CONFIG_FILE) (sekali saja)"
	@echo "  proto    - generate kode dari .proto (buf)"
	@echo "  run      - go run $(CMD_DIR) -config $(CONFIG_FILE)"
	@echo "  build    - build binary ke $(BIN_DIR)/$(APP_NAME)"
	@echo "  up|down  - docker compose (MySQL + API)"
	@echo "  logs     - tail logs service API (compose)"
	@echo "  db-seed  - apply migrations/001_init.sql ke MySQL lokal"
	@echo "  fmt|vet|test|clean|docker-build|run-docker|env"

# ---- Config (copy example -> real) ----
config:
	@mkdir -p $(CONFIG_DIR)
	@test -f $(CONFIG_EXAMPLE) || (echo "!! Missing $(CONFIG_EXAMPLE)"; exit 1)
	@test -f $(CONFIG_FILE) || cp $(CONFIG_EXAMPLE) $(CONFIG_FILE) && echo "OK: $(CONFIG_FILE) created"

# ---- Protobuf (BUF) ----
proto:
	cd $(PROTO_DIR) && buf generate

tidy:
	go mod tidy

# ---- Build & Run ----
build: proto tidy
	@mkdir -p $(BIN_DIR)
	go build -buildvcs=false -v -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)

run: proto
	@[ -f $(CONFIG_FILE) ] || (echo ">> $(CONFIG_FILE) missing. Run: make config"; exit 1)
	@echo "using config: $(CONFIG_FILE)"
	go run -buildvcs=false -v ./cmd/api -config $(CONFIG_FILE)

# variasi opsional
run-m1: proto
	@[ -f $(CONFIG_FILE) ] || (echo ">> $(CONFIG_FILE) missing. Run: make config"; exit 1)
	go run -tags "osusergo netgo" -v $(CMD_DIR) -config $(CONFIG_FILE)

run-windows: proto
	@[ -f $(CONFIG_FILE) ] || (echo ">> $(CONFIG_FILE) missing. Run: make config"; exit 1)
	go run -tags "osusergo netgo" $(CMD_DIR) -config $(CONFIG_FILE)

test: tidy
	go test -v ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)

# ---- Docker (image service) ----
docker-build:
	docker build -t $(SERVICE_NAME):dev .

# Run container standalone (tanpa compose), mount config.yaml
run-docker: docker-build
	@[ -f $(CONFIG_FILE) ] || (echo ">> $(CONFIG_FILE) missing. Run: make config"; exit 1)
	docker run --rm -it \
	  -e CONFIG_PATH=/app/config.yaml \
	  -v $(PWD)/config/config.yaml:/app/config.yaml:ro \
	  -p 8080:8080 -p 9090:9090 \
	  $(SERVICE_NAME):dev

# ---- Docker Compose (MySQL + API) ----
up:
	docker compose up --build -d

logs:
	docker compose logs -f api

down:
	docker compose down

# ---- Seed DB lokal (butuh mysql client) ----
db-seed:
	mysql -u$(DB_USER) -p$(DB_PASS) -h $(DB_HOST) -P $(DB_PORT) < migrations/001_init.sql

# ---- Debug Env ----
env:
	@echo SERVICE_NAME=$(SERVICE_NAME)
	@echo APP_NAME=$(APP_NAME)
	@echo CONFIG_FILE=$(CONFIG_FILE)

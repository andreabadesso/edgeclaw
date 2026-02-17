BINARY_NAME=edgeclaw
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GOFLAGS=-trimpath

.PHONY: all build clean test lint run docker

all: build

build:
	go build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/edgeclaw

build-arm64:
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-arm64 ./cmd/edgeclaw

build-riscv64:
	GOOS=linux GOARCH=riscv64 go build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-riscv64 ./cmd/edgeclaw

clean:
	rm -rf bin/

test:
	go test -race -count=1 ./...

lint:
	go vet ./...

run: build
	./bin/$(BINARY_NAME) -config configs/config.json

docker:
	docker build -t $(BINARY_NAME):$(VERSION) .

compose-up:
	docker compose -f deploy/docker-compose.yml up -d

compose-down:
	docker compose -f deploy/docker-compose.yml down

workspace-init:
	@echo "Initializing PicoClaw workspace with EdgeClaw templates..."
	mkdir -p ~/.picoclaw/workspace/skills/iot-monitor
	cp workspace/HEARTBEAT.md ~/.picoclaw/workspace/HEARTBEAT.md
	cp workspace/IDENTITY.md ~/.picoclaw/workspace/IDENTITY.md
	cp workspace/SOUL.md ~/.picoclaw/workspace/SOUL.md
	cp workspace/AGENTS.md ~/.picoclaw/workspace/AGENTS.md
	cp workspace/skills/iot-monitor/SKILL.md ~/.picoclaw/workspace/skills/iot-monitor/SKILL.md
	@echo "Workspace initialized."

config-init:
	@echo "Installing EdgeClaw config..."
	mkdir -p ~/.picoclaw
	cp configs/config.example.json ~/.picoclaw/config.json
	@echo "Edit ~/.picoclaw/config.json with your provider keys and database credentials."

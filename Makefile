BINARY := ctx
VERSION := $(shell grep -oE 'var Version = "[^"]*"' pkg/ctx/version.go | head -1 | sed -E 's/.*"([^"]*)".*/\1/')
export GOCACHE := $(CURDIR)/.gocache

.PHONY: build tidy test install clean smoke

tidy:
	go mod tidy

build: tidy
	go build -o bin/$(BINARY) ./cmd/ctx

install: build
	go install ./cmd/ctx

# Override version for release builds: make build VERSION=0.1.0
release-build:
	go build -ldflags "-X github.com/donmclean/ctx/pkg/ctx.Version=$(VERSION)" -o bin/$(BINARY) ./cmd/ctx

test:
	go test ./...

# Smoke-test ctx init against a throwaway repo.
smoke: build
	@tmp=$$(mktemp -d); \
	git init -q "$$tmp/proj"; \
	./bin/$(BINARY) --version; \
	./bin/$(BINARY) init "$$tmp/proj"; \
	echo "--- files ---"; find "$$tmp/proj/.ctx" -type f | sort; \
	echo "--- version stamp ---"; cat "$$tmp/proj/.ctx/.ctx-version"; \
	echo "--- exclude ---"; tail -3 "$$tmp/proj/.git/info/exclude"; \
	echo "--- init-substituted placeholders left? (should be none) ---"; grep -rl '{{PROJECT}}\|{{DATE}}' "$$tmp/proj/.ctx" || echo "(none — good)"; \
	echo "--- idempotency (re-run refuses) ---"; ./bin/$(BINARY) init "$$tmp/proj" 2>&1 | tail -1; \
	echo "--- custom folder ---"; rm -rf "$$tmp/proj/.ctx"; ./bin/$(BINARY) init "$$tmp/proj" --folder .agent 2>&1 | tail -1; \
	rm -rf "$$tmp"

clean:
	rm -rf bin .gocache
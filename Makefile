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
	go build -ldflags "-X github.com/neuvybe/ctx/pkg/ctx.Version=$(VERSION)" -o bin/$(BINARY) ./cmd/ctx

test:
	go test -race ./...

# Smoke-test ctx init/update/doctor against a throwaway repo.
smoke: build
	@tmp=$$(mktemp -d); \
	git init -q "$$tmp/proj"; \
	echo "--- ctx --version ---"; ./bin/$(BINARY) --version; \
	echo "--- ctx init ---"; ./bin/$(BINARY) init "$$tmp/proj" | head -1; \
	echo "--- files ---"; find "$$tmp/proj/.ctx" -type f | sort; \
	echo "--- version stamp ---"; cat "$$tmp/proj/.ctx/.ctx-version"; \
	echo "--- exclude ---"; tail -3 "$$tmp/proj/.git/info/exclude"; \
	echo "--- init-substituted placeholders left? (should be none) ---"; grep -rl '{{PROJECT}}\|{{DATE}}' "$$tmp/proj/.ctx" || echo "(none — good)"; \
	echo "--- idempotency (re-run refuses) ---"; ./bin/$(BINARY) init "$$tmp/proj" 2>&1 | tail -1; \
	echo "--- ctx doctor (healthy) ---"; ./bin/$(BINARY) doctor "$$tmp/proj" 2>&1 | tail -3; \
	echo "--- set user fill + corrupt a managed line ---"; \
	R="$$tmp/proj/.ctx/README.md"; \
	perl -pi -e 's/\{\{OWNER_INSTRUCTIONS_PATH\}\}/OWNERPATH-XYZ/' "$$R"; \
	perl -pi -e 's/It is gitignored via/CORRUPTED-MANAGED HERE/' "$$R"; \
	echo "--- ctx update ---"; ./bin/$(BINARY) update "$$tmp/proj" 2>&1; \
	echo "user fill preserved: $$(grep -c OWNERPATH-XYZ "$$R") (want 1)"; \
	echo "corruption gone: $$(grep -c CORRUPTED-MANAGED "$$R") (want 0)"; \
	echo "managed restored: $$(grep -c 'It is gitignored via' "$$R") (want 1)"; \
	echo "--- ctx doctor (healthy, post-update) ---"; ./bin/$(BINARY) doctor "$$tmp/proj" >/dev/null 2>&1; echo "doctor exit=$$? (0=healthy)"; \
	echo "--- doctor catches unbalanced markers ---"; \
	perl -pi -e 's/<!-- ctx:managed end -->// if !$$done' "$$R"; \
	./bin/$(BINARY) doctor "$$tmp/proj" >/dev/null 2>&1; echo "doctor exit=$$? (non-zero expected)"; \
	rm -rf "$$tmp"

clean:
	rm -rf bin .gocache
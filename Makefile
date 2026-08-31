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

# Smoke-test team/local init plus update/doctor against throwaway repos.
smoke: build
	@set -eu; \
	ctx_smoke_tmp=$$(mktemp -d); \
	trap 'rm -rf "$$ctx_smoke_tmp"' EXIT; \
	git init -q "$$ctx_smoke_tmp/team"; \
	./bin/$(BINARY) init "$$ctx_smoke_tmp/team" >"$$ctx_smoke_tmp/team-init.out" 2>"$$ctx_smoke_tmp/team-init.err"; \
	test -s "$$ctx_smoke_tmp/team-init.out"; \
	test ! -s "$$ctx_smoke_tmp/team-init.err"; \
	test -f "$$ctx_smoke_tmp/team/.ctx/config.json"; \
	test -f "$$ctx_smoke_tmp/team/.ctx/local/CONTINUE.md"; \
	git -C "$$ctx_smoke_tmp/team" check-ignore -q .ctx/local/CONTINUE.md; \
	if git -C "$$ctx_smoke_tmp/team" check-ignore -q .ctx/README.md; then exit 1; fi; \
	./bin/$(BINARY) doctor "$$ctx_smoke_tmp/team" >"$$ctx_smoke_tmp/team-doctor.out" 2>"$$ctx_smoke_tmp/team-doctor.err"; \
	test -s "$$ctx_smoke_tmp/team-doctor.out"; \
	test ! -s "$$ctx_smoke_tmp/team-doctor.err"; \
	rm "$$ctx_smoke_tmp/team/.ctx/local/CONTINUE.md"; \
	rmdir "$$ctx_smoke_tmp/team/.ctx/local"; \
	./bin/$(BINARY) init "$$ctx_smoke_tmp/team" >/dev/null; \
	test -f "$$ctx_smoke_tmp/team/.ctx/local/CONTINUE.md"; \
	ctx_smoke_readme="$$ctx_smoke_tmp/team/.ctx/README.md"; \
	perl -pi -e 's/\{\{OWNER_INSTRUCTIONS_PATH\}\}/OWNERPATH-XYZ/' "$$ctx_smoke_readme"; \
	perl -pi -e 's/Team mode leaves the durable files visible to Git/CORRUPTED-MANAGED/' "$$ctx_smoke_readme"; \
	./bin/$(BINARY) update "$$ctx_smoke_tmp/team"; \
	grep -q OWNERPATH-XYZ "$$ctx_smoke_readme"; \
	if grep -q CORRUPTED-MANAGED "$$ctx_smoke_readme"; then exit 1; fi; \
	./bin/$(BINARY) doctor "$$ctx_smoke_tmp/team" >/dev/null; \
	perl -0pi -e 's/<!-- ctx:managed end -->//' "$$ctx_smoke_readme"; \
	if ./bin/$(BINARY) doctor "$$ctx_smoke_tmp/team" >/dev/null 2>&1; then exit 1; fi; \
	git init -q "$$ctx_smoke_tmp/local"; \
	./bin/$(BINARY) init "$$ctx_smoke_tmp/local" --mode local >"$$ctx_smoke_tmp/local-init.out" 2>"$$ctx_smoke_tmp/local-init.err"; \
	test -s "$$ctx_smoke_tmp/local-init.out"; \
	test ! -s "$$ctx_smoke_tmp/local-init.err"; \
	git -C "$$ctx_smoke_tmp/local" check-ignore -q .ctx/README.md; \
	./bin/$(BINARY) doctor "$$ctx_smoke_tmp/local" >/dev/null; \
	echo "smoke: team + local mode checks passed"

clean:
	rm -rf bin .gocache

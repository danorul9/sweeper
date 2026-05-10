.PHONY: all build test vet clean install lint

BINARY  = sweeper
OUTPUT  = ./bin/$(BINARY)
VERSION = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -ldflags="-s -w -X main.version=$(VERSION)"
GOFLAGS = -trimpath

all: test build

build:
	@mkdir -p bin
	go build $(GOFLAGS) $(LDFLAGS) -o $(OUTPUT) ./cmd/sweeper/
	@echo "Built $(OUTPUT) ($(VERSION))"

install: build
	cp $(OUTPUT) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

test:
	go test ./... -v -count=1

vet:
	go vet ./...

lint:
	@which staticcheck 2>/dev/null || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

clean:
	rm -rf bin/

run:
	go run ./cmd/sweeper/ $(ARGS)

# Cross-compile for macOS (Apple Silicon + Intel)
dist: clean
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY)-arm64 ./cmd/sweeper/
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY)-amd64 ./cmd/sweeper/
	lipo -create -output bin/$(BINARY) bin/$(BINARY)-arm64 bin/$(BINARY)-amd64
	rm bin/$(BINARY)-arm64 bin/$(BINARY)-amd64
	@echo "Built universal binary: bin/$(BINARY)"

# Goreleaser snapshot build (local testing)
snapshot:
	@goreleaser release --snapshot --clean 2>/dev/null || \
		echo "goreleaser not installed. Run: brew install goreleaser"

# Full release (requires GITHUB_TOKEN)
release:
	@goreleaser release --clean 2>/dev/null || \
		echo "goreleaser not installed. Run: brew install goreleaser"

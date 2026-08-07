.PHONY: build test vet fmt fmt-check clean run-docker run-gvisor help

BIN := bin/harness
PKG := ./harness

## build: compile the harness binary to bin/harness
build:
	go build -o $(BIN) $(PKG)

## test: run unit tests (capture layer; no Docker required)
test:
	go test $(PKG) -v

## vet: run go vet
vet:
	go vet ./...

## fmt: format all Go sources
fmt:
	gofmt -w harness/

## fmt-check: fail if any Go source is unformatted
fmt-check:
	@out=$$(gofmt -l harness/); if [ -n "$$out" ]; then echo "unformatted:"; echo "$$out"; exit 1; fi

## clean: remove build artifacts
clean:
	rm -rf bin

## run-docker: example run (requires a running Docker daemon + strace in image)
run-docker: build
	$(BIN) run --backend=docker -- whoami

## run-gvisor: example run (requires Docker + runsc registered on a Linux host)
run-gvisor: build
	$(BIN) run --backend=gvisor -- cat /etc/passwd

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'

MAKEFLAGS+=--silent

.DEFAULT_GOAL=help

AUTHOR=m1x0n
NAME=curly
VERSION=latest
LICENCE=MIT

OS=linux
ARCH=amd64

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

SRC=curly.go
BIN=$(ROOT_DIR)/bin
DIST=$(ROOT_DIR)/dist
LDFLAGS="-w -s"

.PHONY: help ## Shows this help
help:
	echo "List of available targets:"
	@printf "%-19s %s\n" "Target" "Description"
	@printf "%-19s %s\n" "------" "-----------"
	@grep '^.PHONY: .* ##' Makefile | sed 's/\.PHONY: \(.*\) ## \(.*\)/\1	\2/' | expand -t20

.PHONY: build ## Builds binary
build:
	mkdir -p $(BIN) && \
	GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags=$(FLAGS) -a -x -v -o $(BIN)/$(NAME) $(SRC)

.PHONY: download ## Download dependencies
download:
	go mod download && go mod verify

.PHONY: test ## Runs tests
test:
	go test -v ./...

.PHONY: run ## Runs application without build
run:
	go run curly.go

.PHONY: clean ## Clean up
clean:
	rm -rf $(BIN) $(DIST)

.PHONY: release ## Test release (goreleaser)
release:
	goreleaser release --snapshot --rm-dist
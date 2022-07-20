MAKEFLAGS+=--silent

.DEFAULT_GOAL=help

AUTHOR=m1x0n
NAME=curly
LICENCE=MIT

OS=linux
ARCH=amd64

SRC=main.go
BIN=./bin

# Does not build w/o CGO_ENABLED=0

.PHONY: help ## Shows this help
help:
	echo "List of available targets:"
	@printf "%-19s %s\n" "Target" "Description"
	@printf "%-19s %s\n" "------" "-----------"
	@grep '^.PHONY: .* ##' Makefile | sed 's/\.PHONY: \(.*\) ## \(.*\)/\1	\2/' | expand -t20

.PHONY: build ## Builds binary
build:
	mkdir -p ./bin && \
	GOOS=$(OS) GOARCH=$(ARCH) go build -a -o $(BIN)/$(NAME) $(SRC)

.PHONY: download ## Download dependencies
download:
	go mod download

.PHONY: test ## Runs tests
test:
	go test -v ./...

.PHONY: run ## Runs application without build
run:
	go run main.go

.PHONY: clean ## Clean up bin
clean:
	ls $(BIN)
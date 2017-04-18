SHELL := /bin/bash

# The name of the executable (default is current directory name)
TARGET := $(shell echo $${PWD\#\#*/})
.DEFAULT_GOAL: $(TARGET)

# These will be provided to the target
BUILD := "`cat VERSION|tr -d '\n'`-`date +%Y%m%d-%H%M%S`+`git rev-parse --short HEAD`"

# Use linker flags to provide version/build settings to the target
LDFLAGS=-ldflags "-X=main.build=$(BUILD)"

# go source files, ignore vendor directory
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: all build clean install uninstall fmt simplify check run test

all: install

$(TARGET): $(SRC)
	@go build $(LDFLAGS) -o $(TARGET)

build: $(TARGET)
	@true

clean:
	@rm -f $(TARGET)

install:
	@go install $(LDFLAGS)

uninstall: clean
	@rm $$(which ${TARGET})

fmt:
	@gofmt -l -w $(SRC)

simplify:
	@gofmt -s -l -w $(SRC)

check:
	@test -z $(shell gofmt -l main.go | tee /dev/stderr) || echo "[WARN] Fix formatting issues with 'make fmt'"
	@for d in $$(go list ./... | grep -v /vendor/); do golint $${d}; done
	@go tool vet ${SRC}

test:
	./test/run-integration-tests.sh

run: install
	@$(TARGET)

vendor-deps:
	@echo ">> Fetching dependencies"
	go get github.com/rancher/trash

vendor: vendor-deps
	rm -r vendor/
	${GOPATH}/bin/trash -u
	${GOPATH}/bin/trash

BINARY_NAME := fabricmon
VERSION := 0.1
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)

# Use linker flags to provide version/build settings to the target
BUILDINFO = -X github.com/dswarbrick/fabricmon/version.Version=$(VERSION)
BUILDINFO += -X github.com/dswarbrick/fabricmon/version.Branch=$(BRANCH)
BUILDINFO += -X github.com/dswarbrick/fabricmon/version.Revision=$(REVISION)
BUILDINFO += -X github.com/dswarbrick/fabricmon/version.BuildUser=$(shell whoami)@$(shell hostname)
BUILDINFO += -X github.com/dswarbrick/fabricmon/version.BuildDate=$(shell date --utc +%Y%m%d-%T)
LDFLAGS := -ldflags "$(BUILDINFO)"

.PHONY:
	all build clean test install fmt check run

all: check install

build:
	@go build $(LDFLAGS) -o $(BINARY_NAME) -v

clean:
	@go clean && rm -f $(BINARY_NAME)

test:
	@go test -v .

install:
	@go install $(LDFLAGS)

fmt:
	@go fmt ./...

check: fmt
	@go vet

run:
	@go run $(LDFLAGS) $(filter-out *_test.go, $(wildcard *.go))

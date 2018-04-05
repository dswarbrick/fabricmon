BINARY_NAME := fabricmon

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
	@go fmt

check: fmt
	@go vet

run:
	@go run $(LDFLAGS) $(filter-out *_test.go, $(wildcard *.go))

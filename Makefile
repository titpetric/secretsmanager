.PHONY: all

export CGO_ENABLED=0

all:
	go fmt ./...
	go install ./cmd/...

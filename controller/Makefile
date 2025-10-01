SHELL := /bin/bash

.PHONY: run build tidy

run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/controller ./cmd/server

tidy:
	go mod tidy

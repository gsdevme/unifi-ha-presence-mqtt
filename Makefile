.PHONY: all
default: all;

fmt:
	go fmt ./...

tests:
	go test ./...

build:
	goreleaser build --snapshot --skip-validate --rm-dist --single-target
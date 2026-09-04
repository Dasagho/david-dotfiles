.PHONY: build test vet fmt install

build:
	go build -buildvcs=false ./cmd/dotfiles

test:
	go test ./...

vet:
	go vet -buildvcs=false ./...

fmt:
	gofmt -w cmd internal

install:
	./bootstrap.sh

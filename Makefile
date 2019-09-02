.PHONY: test deps docs
.EXPORT_ALL_VARIABLES:

GO111MODULE ?= on
LOCALS      := $(shell find . -type f -name '*.go')
GEESE       += linux-amd64
GEESE       += linux-arm
GEESE       += linux-arm64
GEESE       += darwin-amd64
GEESE       += freebsd-amd64

all: fmt deps test build docs

deps:
	go get ./...
	-go mod tidy

fmt:
	go generate -x ./...
	gofmt -w $(LOCALS)
	go vet ./...

test:
	go test -count=1 ./...

$(GEESE):
	GOOS=$(firstword $(subst -, ,$(@))) GOARCH=$(lastword $(subst -, ,$(@))) go build --ldflags '-extldflags "-static"' -installsuffix cgo -ldflags '-s' -o bin/mdnstool-$(@) ./

build: fmt $(GEESE)

release:
	hub release create $(ls -1 bin/*-*-* | xargs -I{} echo "-a {}") $(VERSION)
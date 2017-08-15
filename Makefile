.PHONY: clean clean-all test

NAME        := go-feature
IMPORT_PATH := github.com/spreadshirt/$(NAME)
VERSION     := $(shell git describe --tags --always|sed 's/^v//g')
GO_FLAGS    :=
SOURCES     := $(wildcard *.go examples/*.go)
GOPATH      := $(shell pwd)/.go
TEST_OPTS   :=

all: example-hello

$(GOPATH):
	scripts/bootstrap

example-hello: $(SOURCES) $(GOPATH)
	go install $(GO_FLAGS) $(IMPORT_PATH)/examples/hello
	@cp $(GOPATH)/bin/hello example-hello

test-all: test

test: $(GOPATH)
	@go test $(GO_FLAGS) $(TEST_OPTS) $(IMPORT_PATH)

fmt:
	@gofmt -w $(NAME).go
	@goimports -w $(NAME).go

check: vet lint

vet:
	go vet $(IMPORT_PATH)

lint:
	golint $(IMPORT_PATH)

clean:
	rm -f example-hello

clean-go:
	rm -rf .go vendor

clean-all: clean clean-go

PREFIX  ?= /usr/local
BINDIR  ?= $(PREFIX)/bin
BINARY  := lustre
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

LDFLAGS := -s -w -X main.Version=$(VERSION)
GOBIN := $(shell go env GOPATH)/bin

.PHONY: build install uninstall clean
.PHONY: lint fmt check

fmt:
	gofmt -w .
	npx @biomejs/biome format --write template.html static/

lint:
	go vet ./...
	$(GOBIN)/staticcheck ./...
	npx @biomejs/biome check template.html static/

check: fmt lint

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)

PREFIX  ?= /usr/local
BINDIR  ?= $(PREFIX)/bin
BINARY  := lustre
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

LDFLAGS := -s -w -X main.Version=$(VERSION)
GOBIN := $(shell go env GOPATH)/bin

.PHONY: build install uninstall clean
.PHONY: lint fmt check test

fmt:
	gofmt -w src/
	npx @biomejs/biome format --write src/template.html src/static/

lint:
	cd src && go vet ./...
	cd src && $(GOBIN)/staticcheck ./...
	npx @biomejs/biome check src/template.html src/static/

test:
	cd src && go test -v ./...

check: fmt lint test

build:
	cd src && go build -ldflags "$(LDFLAGS)" -o ../$(BINARY) .

install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)

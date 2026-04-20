BINARY  := vn
MODULE  := github.com/marad/vinote
CMD     := ./cmd/vn
PREFIX  ?= /usr/local
BINDIR  := $(PREFIX)/bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install uninstall test clean

build:
	go build -ldflags '$(LDFLAGS)' -o $(BINARY) $(CMD)

install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

test:
	go test ./...

clean:
	rm -f $(BINARY)

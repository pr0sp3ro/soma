PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

.PHONY: build install

build:
	go build -o soma ./cmd/soma

install: build
	install -Dm755 soma "$(DESTDIR)$(BINDIR)/soma"

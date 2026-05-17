BIN := ask
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BIN) .

install:
	go install $(LDFLAGS) .

clean:
	rm -f $(BIN)

.PHONY: build install clean

BINARY := docktop
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint run install clean tidy
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/docktop
test:
	go test -race ./...
lint:
	golangci-lint run ./...
run:
	go run ./cmd/docktop
install: build
	install -Dm755 $(BINARY) $(DESTDIR)/usr/local/bin/$(BINARY)
clean:
	rm -f $(BINARY)
tidy:
	go mod tidy

#!/usr/bin/env sh
set -eu
version="${VERSION:-dev}"
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$version" -o docktop ./cmd/docktop

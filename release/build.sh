#!/bin/bash

set -euo pipefail

die() {
	echo "$*" >&2
	exit 1
}

try() {
	"$@" || die "failed: $*"
}

[ -n "${1-}" ] || die "specify tag (v0.0.x)"

cleanup() {
	rm -rf gamearc-windows gamearc-linux gamearc-windows.zip gamearc-linux.tar.gz go.mod go.sum
}

cleanup
mkdir -p gamearc-windows gamearc-linux
go mod init pkg
go get github.com/kaey/gamearc@"$1"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gamearc-linux github.com/kaey/gamearc/cmd/...
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o gamearc-windows github.com/kaey/gamearc/cmd/...
bsdtar caf gamearc-windows.zip gamearc-windows
bsdtar caf gamearc-linux.tar.gz gamearc-linux
gh release upload --clobber "$1" gamearc-windows.zip gamearc-linux.tar.gz
cleanup

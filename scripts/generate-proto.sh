#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
# Requires protoc, protoc-gen-go v1.36.5 and protoc-gen-go-grpc v1.5.1 on PATH.
modulePath=$(sed -n 's/^module //p' go.mod)
protoc -I proto --go_out=. --go_opt=module="$modulePath" --go-grpc_out=. --go-grpc_opt=module="$modulePath" proto/griyo.proto

#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT_DIR}/api/pb"
PROTO_FILE_REL="api/pb/vote.proto"
GOBIN_DIR="$(go env GOBIN)"
if [[ -z "${GOBIN_DIR}" ]]; then
  GOBIN_DIR="$(go env GOPATH)/bin"
fi
PROTOC_GEN_GO="${GOBIN_DIR}/protoc-gen-go"
PROTOC_GEN_GO_GRPC="${GOBIN_DIR}/protoc-gen-go-grpc"

if ! command -v protoc >/dev/null 2>&1; then
  echo "error: protoc not found; please install Protocol Buffers compiler." >&2
  exit 1
fi

if [[ ! -x "${PROTOC_GEN_GO}" ]]; then
  echo "error: protoc-gen-go not found at ${PROTOC_GEN_GO}; run: go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.30.0" >&2
  exit 1
fi

if [[ ! -x "${PROTOC_GEN_GO_GRPC}" ]]; then
  echo "error: protoc-gen-go-grpc not found at ${PROTOC_GEN_GO_GRPC}; run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0" >&2
  exit 1
fi

protoc \
  --proto_path="${ROOT_DIR}" \
  --plugin=protoc-gen-go="${PROTOC_GEN_GO}" \
  --plugin=protoc-gen-go-grpc="${PROTOC_GEN_GO_GRPC}" \
  --go_out="${ROOT_DIR}" \
  --go_opt=paths=source_relative \
  --go-grpc_out="${ROOT_DIR}" \
  --go-grpc_opt=paths=source_relative \
  "${PROTO_FILE_REL}"

echo "generated: api/pb/vote.pb.go api/pb/vote_grpc.pb.go"

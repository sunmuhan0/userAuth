#!/bin/bash

# 生成 protobuf Go 代码
# 在 auth-client/proto/ 目录下执行: bash gen.sh

set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PROJECT_DIR=$(cd "$SCRIPT_DIR/.." && pwd)

echo "[gen] generating protobuf code..."
echo "[gen] proto dir: $SCRIPT_DIR"
echo "[gen] output dir: $PROJECT_DIR/auth"

# 清除旧的生成文件
rm -f "$PROJECT_DIR/auth/auth.pb.go" "$PROJECT_DIR/auth/auth_grpc.pb.go"

# 生成
protoc \
  --go_out="$PROJECT_DIR" --go_opt=module=ttuser/auth-client \
  --go-grpc_out="$PROJECT_DIR" --go-grpc_opt=module=ttuser/auth-client \
  -I "$SCRIPT_DIR" \
  "$SCRIPT_DIR/auth.proto"

echo "[gen] done. generated files:"
ls -la "$PROJECT_DIR/auth/"*.pb.go

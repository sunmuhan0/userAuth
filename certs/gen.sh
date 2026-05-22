#!/bin/bash

# 生成自签名 TLS 证书（开发用）
# 在 certs/ 目录下执行: bash gen.sh

set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"

echo "[certs] generating CA..."
openssl req -x509 -newkey rsa:4096 -days 3650 -nodes \
  -keyout ca-key.pem -out ca.pem \
  -subj "/C=CN/ST=Shanghai/L=Shanghai/O=TTUser/CN=TTUser CA" \
  2>/dev/null

echo "[certs] generating server certificate..."
# 生成服务端私钥
openssl genrsa -out server-key.pem 4096 2>/dev/null

# 生成 CSR
openssl req -new -key server-key.pem -out server.csr \
  -subj "/C=CN/ST=Shanghai/L=Shanghai/O=TTUser/CN=localhost" \
  2>/dev/null

# 创建扩展文件（支持 SAN）
cat > server-ext.cnf <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
IP.2 = 0.0.0.0
EOF

# 用 CA 签发服务端证书
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem \
  -CAcreateserial -out server.pem -days 3650 \
  -extfile server-ext.cnf \
  2>/dev/null

# 清理中间文件
rm -f server.csr server-ext.cnf ca-key.pem ca.srl

echo "[certs] done. files:"
ls -la *.pem

echo ""
echo "  ca.pem         - CA certificate (client uses this)"
echo "  server.pem     - Server certificate"
echo "  server-key.pem - Server private key"

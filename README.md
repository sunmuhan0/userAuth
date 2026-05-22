# 用户认证服务

基于 Go + Gin + gRPC + MySQL + inji(DI) 的微服务架构鉴权系统。

## 项目结构

```
ttuser/
├── auth-client/              # gRPC 客户端 SDK（proto + 生成代码 + client 封装）
│   ├── proto/auth.proto      # 服务定义
│   ├── proto/gen.sh          # proto 代码生成脚本
│   ├── auth/                 # 生成的 pb/grpc 代码
│   └── client/client.go      # AuthClient（Start/Close，inji 自动注册）
│
├── auth-server/              # 认证 gRPC 服务端
│   ├── cmd/server/main.go    # 入口（inji.Reg → sp.Init → Run）
│   ├── internal/
│   │   ├── dao/              # 数据访问层（UserDAO + TokenDAO）
│   │   ├── model/            # 用户模型
│   │   └── service/          # 业务逻辑（注册/登录/注销/续签/验证）
│   ├── pkg/token/            # JWT 工具（access 2h + refresh 7d）
│   ├── server/               # gRPC server 实现
│   └── sp/                   # ServiceProvider（inji 依赖聚合）
│
├── proc/                     # HTTP 网关（Gin）
│   ├── cmd/server/main.go    # 入口
│   ├── filter/               # 鉴权过滤器（按路由分组）
│   ├── internal/
│   │   ├── handler/          # HTTP handler
│   │   └── manager/          # AuthManager（内嵌 AuthClient，inject 注入）
│   ├── router/               # 路由配置
│   ├── sp/                   # ServiceProvider
│   └── e2e/                  # E2E 测试
│
├── data-store/               # MySQL 引擎库（被 auth-server 引用）
│   ├── engine/
│   │   ├── interface.go      # IMysqlClient 接口定义
│   │   ├── mysql.go          # BaseMysqlClient 通用实现
│   │   └── proc.go           # ProcMysqlClient（DSN 配置）
│   └── ddl/                  # 数据库 DDL
│       ├── 001_create_users.sql
│       └── 002_create_token_blacklist.sql
│
└── Makefile                  # 构建/测试/proto生成
```

## 技术栈

| 组件 | 技术 |
|------|------|
| HTTP 网关 | Gin |
| 服务间通信 | gRPC + Protobuf |
| 依赖注入 | github.com/teou/inji |
| 认证方案 | JWT（access_token 2h + refresh_token 7d，轮转续签） |
| 数据存储 | MySQL |
| 密码加密 | bcrypt |

## 快速开始

### 1. 环境准备

- Go 1.18+
- MySQL 8.0+
- protoc + protoc-gen-go + protoc-gen-go-grpc

### 2. 初始化数据库

```bash
mysql -u root -p123456 -e "CREATE DATABASE IF NOT EXISTS ttuser"
mysql -u root -p123456 ttuser < data-store/ddl/001_create_users.sql
mysql -u root -p123456 ttuser < data-store/ddl/002_create_token_blacklist.sql
```

### 3. 启动服务

```bash
# 终端1：启动 auth-server (gRPC :9090)
cd auth-server && go run ./cmd/server/

# 终端2：启动 proc (HTTP :8080)
cd proc && go run ./cmd/server/
```

### 4. 运行测试

```bash
make test         # 运行所有测试（单元 + e2e）
make unit-test    # 只跑单元测试
make e2e-test     # 自动编译→清数据→启动服务→跑 e2e→停服务
make build        # 编译到 bin/
make proto        # 重新生成 proto 代码
make clean        # 清理编译产物
```

## API 接口

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/api/v1/register` | 注册 | 否 |
| POST | `/api/v1/login` | 登录 | 否 |
| POST | `/api/v1/refresh` | 续签 token | 否 |
| POST | `/api/v1/logout` | 注销 | 是 |
| GET | `/api/v1/user/info` | 获取用户信息 | 是 |
| PUT | `/api/v1/user/info` | 更新用户信息 | 是 |

## curl 测试

### 注册

```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "123456",
    "nickname": "测试用户",
    "email": "test@example.com"
  }'
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid-xxx",
    "username": "testuser",
    "nickname": "测试用户",
    "email": "test@example.com"
  }
}
```

### 登录

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "123456"
  }'
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "eyJhbGciOi...",
    "expires_at": 1779440478,
    "user": {
      "id": "uuid-xxx",
      "username": "testuser",
      "nickname": "测试用户",
      "email": "test@example.com",
      "avatar": ""
    }
  }
}
```

### 获取用户信息

```bash
curl http://localhost:8080/api/v1/user/info \
  -H "Authorization: Bearer <access_token>"
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid-xxx",
    "username": "testuser",
    "nickname": "测试用户",
    "email": "test@example.com",
    "avatar": "",
    "created_at": 1779441441,
    "updated_at": 1779441441
  }
}
```

### 更新用户信息

```bash
curl -X PUT http://localhost:8080/api/v1/user/info \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "nickname": "新昵称",
    "email": "new@example.com",
    "avatar": "https://example.com/avatar.png"
  }'
```

### 续签 Token

```bash
curl -X POST http://localhost:8080/api/v1/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<refresh_token>"
  }'
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOi...(新)",
    "refresh_token": "eyJhbGciOi...(新)",
    "expires_at": 1779447678
  }
}
```

> 注意：续签后旧的 refresh_token 立即失效（轮转模式）。

### 注销

```bash
curl -X POST http://localhost:8080/api/v1/logout \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<refresh_token>"
  }'
```

响应：
```json
{
  "code": 0,
  "message": "logout success"
}
```

> 注销后 access_token 和 refresh_token 同时失效。

### 错误响应示例

```bash
# 无 token 访问需要鉴权的接口
curl http://localhost:8080/api/v1/user/info
# {"code":401,"message":"authorization header is required"}

# 注销后再访问
curl http://localhost:8080/api/v1/user/info -H "Authorization: Bearer <已注销的token>"
# {"code":401,"message":"invalid or expired token: token has been revoked"}

# 重复注册
curl -X POST http://localhost:8080/api/v1/register -H "Content-Type: application/json" -d '{"username":"testuser","password":"123"}'
# {"code":409,"message":"rpc error: code = AlreadyExists desc = username already exists"}

# 旧 refresh_token 续签（已轮转作废）
curl -X POST http://localhost:8080/api/v1/refresh -H "Content-Type: application/json" -d '{"refresh_token":"<旧token>"}'
# {"code":401,"message":"refresh failed: rpc error: code = Unauthenticated desc = token has been revoked"}
```

## 完整 curl 测试脚本

```bash
#!/bin/bash
BASE_URL="http://localhost:8080/api/v1"

echo "=== 1. 注册 ==="
curl -s -X POST $BASE_URL/register \
  -H "Content-Type: application/json" \
  -d '{"username":"demo","password":"123456","nickname":"Demo","email":"demo@test.com"}'
echo ""

echo "=== 2. 登录 ==="
LOGIN=$(curl -s -X POST $BASE_URL/login \
  -H "Content-Type: application/json" \
  -d '{"username":"demo","password":"123456"}')
echo "$LOGIN" | python3 -m json.tool

ACCESS_TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
REFRESH_TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['refresh_token'])")

echo "=== 3. 获取用户信息 ==="
curl -s $BASE_URL/user/info -H "Authorization: Bearer $ACCESS_TOKEN" | python3 -m json.tool
echo ""

echo "=== 4. 更新用户信息 ==="
curl -s -X PUT $BASE_URL/user/info \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"nickname":"Demo改名","email":"new@test.com"}' | python3 -m json.tool
echo ""

echo "=== 5. 续签 Token ==="
REFRESH_RESP=$(curl -s -X POST $BASE_URL/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
echo "$REFRESH_RESP" | python3 -m json.tool
NEW_AT=$(echo "$REFRESH_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
NEW_RT=$(echo "$REFRESH_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['refresh_token'])")

echo "=== 6. 旧 refresh_token 再次使用（应失败） ==="
curl -s -X POST $BASE_URL/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | python3 -m json.tool
echo ""

echo "=== 7. 注销 ==="
curl -s -X POST $BASE_URL/logout \
  -H "Authorization: Bearer $NEW_AT" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$NEW_RT\"}" | python3 -m json.tool
echo ""

echo "=== 8. 注销后访问（应失败） ==="
curl -s $BASE_URL/user/info -H "Authorization: Bearer $NEW_AT" | python3 -m json.tool
echo ""
```

## 架构设计

### 依赖注入（inji）

所有组件通过 inji 的 `inject` tag 自动装配。ServiceProvider 中字段顺序即创建顺序：

```
auth-server ServiceProvider:
  ProcMysql(Start()建连) → UserDAO(inject engine) → TokenDAO(inject engine)
  → TokenMgr(Start()加载配置) → AuthService(inject DAO+TokenMgr)
  → GRPCServer(inject authService)

proc ServiceProvider:
  AuthManager → AuthClient(Start()建连 gRPC)
```

### 认证流程

```
注册: POST /register → proc → gRPC → auth-server → bcrypt加密 → MySQL
登录: POST /login → proc → gRPC → auth-server → 验密 → 签发 access+refresh token
请求: GET /user/info → proc filter → gRPC ValidateToken → auth-server → 检查黑名单+解析JWT
续签: POST /refresh → proc → gRPC → auth-server → 验证refresh → 旧token加黑名单 → 签发新pair
注销: POST /logout → proc filter → gRPC → auth-server → access+refresh 加入黑名单
```

### Token 策略

| Token | 有效期 | 用途 |
|-------|--------|------|
| access_token | 2 小时 | 请求鉴权（Bearer header） |
| refresh_token | 7 天 | 续签（获取新 token 对） |

- 续签采用**轮转模式**：每次续签旧 refresh_token 立即作废
- 注销同时废弃 access_token 和 refresh_token
- 黑名单存储在 MySQL `token_blacklist` 表（存 SHA256 hash）

## 配置说明

当前所有配置写死在代码中（带 `TODO: 后续从配置中心获取` 注释），后续接入配置中心只需修改对应位置：

| 配置项 | 当前默认值 | 位置 |
|--------|-----------|------|
| MySQL DSN | `root:123456@tcp(localhost:3306)/ttuser` | `data-store/engine/proc.go` |
| JWT Secret | `my-secret-key-for-ttuser-2024` | `auth-server/pkg/token/jwt.go` |
| Access Token 有效期 | 2h | `auth-server/pkg/token/jwt.go` |
| Refresh Token 有效期 | 7d | `auth-server/pkg/token/jwt.go` |
| auth-server gRPC 端口 | 9090 | `auth-server/server/grpc_server.go` |
| auth-client gRPC 地址 | `localhost:9090` | `auth-client/client/client.go` |
| proc HTTP 端口 | 8080 | `proc/cmd/server/main.go` (env: HTTP_PORT) |

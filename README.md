# 用户认证服务

基于 Go + gRPC + MySQL + RocketMQ + inji(DI) + 配置中心 的微服务架构鉴权系统。

## 项目结构

```
ttuser/
├── auth-client/              # gRPC 客户端 SDK（proto + 生成代码 + client 封装）
│   ├── proto/auth.proto      # 服务定义
│   ├── proto/gen.sh          # proto 代码生成脚本
│   ├── auth/                 # 生成的 pb/grpc 代码
│   └── client/client.go      # AuthClient（TLS 证书从配置中心加载）
│
├── auth-server/              # 认证 gRPC 服务端
│   ├── cmd/server/main.go    # 入口（CLI 参数 -name -port -env）
│   ├── internal/
│   │   ├── dao/              # 数据访问层
│   │   ├── model/            # 用户模型
│   │   └── service/          # 业务逻辑
│   ├── pkg/token/            # JWT 工具
│   ├── server/               # gRPC server（TLS 证书从配置中心加载）
│   └── sp/                   # ServiceProvider
│
├── event-producer/           # RocketMQ 通用生产者库
│   └── producer/
│       ├── interface.go      # IRmqPublisher 接口
│       ├── config.go         # RMQConfig（从配置中心加载）
│       └── publisher.go      # EventRMQPublisher
│
├── async-handler/            # 异步消息处理服务
│   ├── cmd/server/main.go    # 入口（CLI 参数 -name -port -env）
│   ├── pkg/router/           # 消息路由框架
│   ├── biz/
│   │   ├── register/         # 路由注册
│   │   └── actions/          # 业务 action
│   ├── server/server.go      # ConsumerServer
│   ├── internal/sms/         # 短信发送（配置从配置中心加载）
│   └── sp/                   # ServiceProvider
│
├── config-server/            # 配置中心服务（文件系统存储，HTTP :7963）
│   ├── config-center/        # 配置文件目录（随代码提交）
│   │   ├── base/             # 保底配置（所有环境共用）
│   │   │   ├── auth-server/
│   │   │   ├── proc/
│   │   │   ├── event-producer/
│   │   │   └── async-handler/
│   │   ├── prod/             # 生产环境覆盖
│   │   ├── staging/
│   │   └── preview/
│   ├── cmd/server/main.go
│   ├── internal/service/     # ConfigService（文件系统读取 + base/env 合并）
│   └── server/               # HTTP API (Gin, :7963)
│
├── config-client/            # 配置中心客户端 SDK
│   └── client/client.go      # FetchConfigs + LoadFile
│
├── pkg/                      # 公共库
│   ├── trace/                # 全链路 trace_id
│   ├── log/                  # zap 日志封装（日志轮转）
│   ├── crypto/               # AES-256-GCM 加解密
│   ├── metrics/              # Prometheus 指标
│   ├── nacos/                # Nacos 服务注册发现封装
│   └── http/                 # HTTP 服务端/客户端封装（内置日志、trace、metrics）
│
├── proc/                     # HTTP 网关（Gin）
│   ├── cmd/server/main.go    # 入口（CLI 参数 -name -port -env）
│   ├── filter/               # 鉴权过滤器 + metrics 中间件
│   ├── internal/
│   │   ├── handler/          # HTTP handler
│   │   └── manager/          # AuthManager
│   ├── router/               # 路由配置
│   └── sp/                   # ServiceProvider
│
├── data-store/               # MySQL 引擎库
│   ├── engine/
│   │   ├── interface.go      # IMysqlClient 接口
│   │   ├── mysql.go          # BaseMysqlClient
│   │   └── proc.go           # ProcMysqlClient（DSN 从配置中心加载）
│   └── ddl/                  # 数据库 DDL
│
├── dev.ps1                   # Windows 一键启动脚本
├── Makefile                  # 构建 & 开发（Linux/WSL）
└── deploy/                   # Docker Compose 部署配置
    └── loki/                 # Loki + Prometheus + Grafana
```

## 技术栈

| 组件 | 技术 |
|------|------|
| HTTP 网关 | Gin |
| 服务间通信 | gRPC + Protobuf（TLS） |
| 消息队列 | Apache RocketMQ |
| 配置中心 | 文件系统存储 + HTTP API |
| 链路追踪 | 全链路 trace_id |
| 日志 | zap + lumberjack（轮转） |
| 依赖注入 | github.com/teou/inji |
| 认证方案 | JWT（access 2h + refresh 7d） |
| 监控 | Prometheus |
| 配置/密码加密 | AES-256-GCM / bcrypt |
| 服务注册发现 | Nacos（nacos-sdk-go/v2） |

## pkg/http 公共 HTTP 封装

### 服务端

封装 Gin，内置 Trace / AccessLog / Metrics 中间件 + `/metrics` 端点，支持优雅关闭。

```go
srv := http.New(http.ServerConfig{Name: "myapp", Port: 8080})
srv.Engine().GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
srv.Engine().POST("/api/v1/users", handler)
srv.Start()
http.GracefulStop(srv, 10*time.Second)
```

每条请求自动记录日志（path、method、耗时、状态码、请求/响应体、trace_id），按状态码分级：≥500→Error，≥400→Warn，其余→Info。

### 客户端

支持 GET / POST / PUT / DELETE / PATCH，每次请求自动记录日志。建议通过 `Service` 字段标记来源服务，便于 Prometheus 区分指标。

```go
cli := http.NewClient(http.ClientOption{Service: "proc", Timeout: 10 * time.Second})

type SMSReq struct {
    Phone    string `json:"phone"`
    Template string `json:"template"`
    Params   string `json:"params"`
}

resp, err := cli.Post(ctx, "https://sms.api/send",
    SMSReq{Phone: "138xxxx", Template: "verify", Params: `{"code":"1234"}`},
    http.WithBearerToken("sk-xxx"),
)
if err != nil { return err }

var result struct { Code string `json:"code"` }
resp.JSON(&result)
```

## Prometheus 指标

服务端和客户端统一在 `/metrics` 端点暴露，所有指标均带 `service` 标签区分来源。

### 服务端指标

| 指标 | 类型 | Labels | 说明 |
|------|------|--------|------|
| `ttuser_http_requests_total` | Counter | `service`, `method`, `path`, `status` | HTTP 请求总数 |
| `ttuser_http_request_duration_seconds` | Histogram | `service`, `method`, `path` | 请求耗时分布 |
| `ttuser_http_requests_active` | Gauge | - | 当前活跃请求数 |

示例：
```
ttuser_http_requests_total{service="proc",method="POST",path="/api/v1/login",status="200"} 1024
```

### 客户端指标

| 指标 | 类型 | Labels | 说明 |
|------|------|--------|------|
| `ttuser_http_client_requests_total` | Counter | `service`, `method`, `host`, `status` | 第三方 API 调用总数 |
| `ttuser_http_client_request_duration_seconds` | Histogram | `service`, `method`, `host` | 第三方 API 调用耗时 |
| `ttuser_http_client_requests_active` | Gauge | - | 当前活跃客户端请求数 |

示例（调用短信服务）：
```
ttuser_http_client_requests_total{service="proc",method="POST",host="sms.api.aliyun.com",status="200"} 42
```

### gRPC 指标

| 指标 | 类型 | Labels | 说明 |
|------|------|--------|------|
| `ttuser_grpc_calls_total` | Counter | `method`, `status` | gRPC 调用总数 |
| `ttuser_grpc_call_duration_seconds` | Histogram | `method` | gRPC 调用耗时 |

## 服务注册发现（Nacos）

### 架构

```
auth-server (gRPC)
  ├── 启动时 → Nacos.RegisterInstance("auth-server", ip, port)
  ├── 运行中 → SDK 自动发送心跳（5s）
  └── 关闭时 → Nacos.DeregisterInstance("auth-server")

auth-client (proc 内)
  ├── 启动时 → Nacos.SelectOneHealthyInstance("auth-server")
  ├── 成功   → 使用 Nacos 返回的 ip:port 建立 gRPC 连接
  └── 失败   → 降级到 auth-client.json 中的静态 addr
```

- **注册**：`auth-server` 启动时以临时实例（ephemeral）注册到 Nacos，权重 10
- **发现**：`auth-client` 优先通过 `SelectOneHealthyInstance` 获取服务端地址，Nacos 不可用时降级到配置中心中的静态地址
- **心跳**：Nacos SDK 内置 5s 心跳，实例宕机自动摘除
- **配置**：Nacos 服务器地址存储在 `nacos.json` 中，走现有配置中心流程

### 支持的配置文件

| 服务 | 文件名 | 内容 |
|------|--------|------|
| auth-server | nacos.json | server_addr, server_port, namespace_id |
| proc | nacos.json | server_addr, server_port, namespace_id |

## 配置中心

### 架构

配置文件按环境存储在 `config-center/{env}/{service}/*.json`，base 目录保底，各环境同名文件覆盖 base。

### API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/config/files?env=prod&service=auth-server` | 获取合并后的配置文件列表 |

认证：`Authorization: Bearer ttuser-config-token-2024`

### 启动流程

```
CLI 参数 -name -port -env
   → inji 注册 serverName/serverPort/env
     → log.Init(nil)
       → FetchConfigs() 下载到 ./config/{serviceName}/
         → sp.Init()（各组件从本地文件读取配置）
           → Nacos 注册（auth-server）/ 发现（auth-client）
```

- 配置获取失败直接报错，**无降级默认值**
- 服务名通过 `inject:"serverName"` 字段注入
- 证书以 PEM 字符串存储在 JSON 中，`tls.X509KeyPair`/`x509.CertPool` 加载

### 支持的配置文件

| 服务 | 文件名 | 内容 |
|------|--------|------|
| auth-server | mysql.json | DSN |
| auth-server | jwt.json | secret, access_expire, refresh_expire |
| auth-server | certs.json | cert, key（PEM） |
| auth-server | nacos.json | Nacos 服务器地址（server_addr, server_port） |
| proc | auth-client.json | addr, ca_cert |
| proc | nacos.json | Nacos 服务器地址（同上，供 auth-client 发现用） |
| event-producer | rocketmq.json | name_server, group_name |
| async-handler | rocketmq.json | name_server, group_name |
| async-handler | sms.json | api_key, api_secret, sign_name, template |

## CLI 参数

```bash
./auth-server -name=auth-server -port=9090 -env=prod
./proc -name=proc -port=8080 -env=staging
./config-server -name=config-server -port=7963 -env=prod
./async-handler -name=async-handler -port=0 -env=preview
```

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `MYSQL_MAX_OPEN_CONNS` | 最大连接数 | 100 |
| `MYSQL_MAX_IDLE_CONNS` | 最大空闲连接 | 10 |
| `MYSQL_CONN_MAX_LIFETIME` | 连接最大存活（秒） | 3600 |
| `NACOS_DISABLE` | 设为 `true` 跳过 Nacos 注册/发现 | 空 |

## 本地开发

### Docker 依赖

本地开发需要以下 Docker 容器：

```bash
# MySQL
docker run -d --name mysql -e MYSQL_ROOT_PASSWORD=123456 -e MYSQL_DATABASE=ttuser \
  -p 3306:3306 mysql:8.0 \
  --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci

# Nacos
docker run -d --name nacos -p 8848:8848 -p 9848:9848 -e MODE=standalone \
  nacos/nacos-server:v2.3.2
```

### 初始化数据库

```bash
docker exec -i mysql mysql -u root -p123456 ttuser < data-store/ddl/001_create_users.sql
docker exec -i mysql mysql -u root -p123456 ttuser < data-store/ddl/002_create_token_blacklist.sql
```

### 启动服务

**Windows（dev.ps1）：**

```powershell
.\dev.ps1                     # 默认 staging 环境，auth-server :9091
.\dev.ps1 -AuthPort 9090      # 指定端口
```

**Linux/WSL（Makefile）：**

```bash
make dev        # 构建并启动（staging 环境）
make stop-dev   # 停止所有服务
```

> 说明：
> - Docker Desktop for Windows 的 wslrelay 占用 9090 端口，故开发环境 auth-server 默认使用 9091
> - 每个服务额外开 `port+100` 的 HTTP 端口暴露 `/metrics` 和 `/healthz`
> - 日志统一输出到 `./log/` 目录
> - 配置从配置中心下载到 `./config/{serviceName}/` 目录

### 生成 TLS 证书

如果证书过期或需要重新生成，在项目根目录执行：

```bash
docker run --rm -v "$(pwd)/config-server/config-center:/config" golang:1.22 \
  go run /config/generate_cert.go
```

或者使用以下 OpenSSL 脚本后，将 PEM 内容写入对应环境的 `certs.json` 和 `auth-client.json`（注意 JSON 中的换行符需转义为 `\n`）：

```bash
cat > gen_cert.sh <<'SCRIPT'
#!/bin/bash
set -e
openssl req -x509 -newkey rsa:4096 -days 3650 -nodes \
  -keyout ca-key.pem -out ca.pem \
  -subj "/C=CN/ST=Shanghai/L=Shanghai/O=TTUser/CN=TTUser CA"
openssl genrsa -out server-key.pem 4096
openssl req -new -key server-key.pem -out server.csr \
  -subj "/C=CN/ST=Shanghai/L=Shanghai/O=TTUser/CN=localhost"
cat > server-ext.cnf <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=digitalSignature,nonRepudiation,keyEncipherment,dataEncipherment
subjectAltName=@alt_names
[alt_names]
DNS.1=localhost
IP.1=127.0.0.1
IP.2=0.0.0.0
EOF
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem \
  -CAcreateserial -out server.pem -days 3650 -extfile server-ext.cnf
rm -f server.csr server-ext.cnf ca-key.pem ca.srl
echo "CA cert:"
cat ca.pem
echo "Server cert:"
cat server.pem
echo "Server key:"
cat server-key.pem
SCRIPT
bash gen_cert.sh
```

### 手动启动（不依赖脚本）

```bash
# 配置中心
cd config-server && go run ./cmd/server/ \
  -name=config-server -port=7963 -env=staging \
  --config-dir=config-server/config-center

# auth-server（启动后自动注册到 Nacos）
cd auth-server && go run ./cmd/server/ \
  -name=auth-server -port=9091 -env=staging

# proc（从 Nacos 发现 auth-server，降级到 auth-client.json）
cd proc && go run ./cmd/server/ \
  -name=proc -port=8080 -env=staging

# async-handler
cd async-handler && go run ./cmd/server/ \
  -name=async-handler -port=0 -env=prod
```

### 测试

```bash
cd auth-server && go test ./...
cd config-client && go test ./client/ -v -count=1
cd event-producer && go test ./producer/ -v -count=1
cd pkg && go test ./trace/ ./crypto/ ./metrics/ -v -count=1

# e2e 测试（需要先启动 MySQL + config-server + auth-server + proc）
# cd proc && go test ./e2e/ -v -count=1 -run "TestE2E" -timeout=30s
```

## API

| 方法 | 路径 | 鉴权 |
|------|------|------|
| POST | `/api/v1/register` | 否 |
| POST | `/api/v1/login` | 否 |
| POST | `/api/v1/refresh` | 否 |
| POST | `/api/v1/logout` | 是 |
| GET | `/api/v1/user/info` | 是 |
| PUT | `/api/v1/user/info` | 是 |
| GET | `/metrics` | 否 |

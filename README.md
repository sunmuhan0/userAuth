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

支持 GET / POST / PUT / DELETE / PATCH，每次请求自动记录日志。

```go
cli := http.NewClient()

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
       → FetchConfigs() 下载到 /home/work/config/{serviceName}/
         → sp.Init()（各组件从本地文件读取配置）
           → Nacos 注册（auth-server）/ 发现（auth-client）
```

- 配置获取失败直接报错，**无降级默认值**
- 服务名通过 `inji.Find("serverName")` 注入，各包不硬编码
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

## 快速开始

### 1. 环境准备

Go 1.21+, MySQL 8.0+, RocketMQ 4.x+, OpenSSL, Nacos 2.x

### 2. 初始化数据库

```bash
mysql -u root -p123456 -e "CREATE DATABASE IF NOT EXISTS ttuser"
mysql -u root -p123456 ttuser < data-store/ddl/001_create_users.sql
mysql -u root -p123456 ttuser < data-store/ddl/002_create_token_blacklist.sql
```

### 3. 生成 TLS 证书

在项目根目录执行：

```bash
#!/bin/bash
set -e
cd "$(dirname "$0")"

# 生成 CA
openssl req -x509 -newkey rsa:4096 -days 3650 -nodes \
  -keyout ca-key.pem -out ca.pem \
  -subj "/C=CN/ST=Shanghai/L=Shanghai/O=TTUser/CN=TTUser CA"

# 生成服务端私钥
openssl genrsa -out server-key.pem 4096

# 生成 CSR
openssl req -new -key server-key.pem -out server.csr \
  -subj "/C=CN/ST=Shanghai/L=Shanghai/O=TTUser/CN=localhost"

# 创建扩展文件（支持 SAN）
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

# 用 CA 签发服务端证书
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem \
  -CAcreateserial -out server.pem -days 3650 -extfile server-ext.cnf

# 清理中间文件
rm -f server.csr server-ext.cnf ca-key.pem ca.srl

# 读取 PEM 内容，更新 config-center 各环境 auth-server/certs.json
echo "ca.pem:"
cat ca.pem
echo ""
echo "server.pem:"
cat server.pem
echo ""
echo "server-key.pem:"
cat server-key.pem
```

### 4. 启动

```bash
# 配置中心
cd config-server && go run ./cmd/server/ -name=config-server -port=7963 -env=prod

# auth-server
cd auth-server && go run ./cmd/server/ -name=auth-server -port=9090 -env=prod

# proc
cd proc && go run ./cmd/server/ -name=proc -port=8080 -env=prod

# async-handler
cd async-handler && go run ./cmd/server/ -name=async-handler -port=0 -env=prod
```

### 5. 测试

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

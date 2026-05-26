# 用户认证服务

基于 Go + gRPC + MySQL + RocketMQ + inji(DI) 的微服务架构鉴权系统。

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
│   │   └── service/          # 业务逻辑（注册/登录/注销/续签/验证 + 事件发布）
│   ├── pkg/token/            # JWT 工具（access 2h + refresh 7d）
│   ├── server/               # gRPC server 实现
│   └── sp/                   # ServiceProvider（inji 依赖聚合）
│
├── event-producer/           # RocketMQ 通用生产者库
│   └── producer/
│       ├── interface.go      # IRmqPublisher 接口定义
│       ├── config.go         # RMQConfig（NameServer + GroupName）
│       └── publisher.go      # EventRMQPublisher 实现（Start/Publish/Close）
│
├── async-handler/             # 短信消费服务（RocketMQ PushConsumer）
│   ├── cmd/server/main.go    # 入口（inji.Reg → sp.Init）
│   ├── server/server.go      # RMQConfig + SMSConsumerServer（多topic订阅，按topic路由handler）
│   ├── internal/
│   │   ├── handler/          # SMSHandler（实现IMessageHandler）
│   │   └── sms/              # Sender + Config（短信发送，当前模拟）
│   └── sp/                   # ServiceProvider（注册handler到server）
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
│
└── Makefile                  # 构建/测试/proto生成
```

## 技术栈

| 组件 | 技术 |
|------|------|
| HTTP 网关 | Gin |
| 服务间通信 | gRPC + Protobuf |
| 消息队列 | Apache RocketMQ |
| 依赖注入 | github.com/teou/inji |
| 认证方案 | JWT（access_token 2h + refresh_token 7d，轮转续签） |
| 数据存储 | MySQL |
| 密码加密 | bcrypt |

## 消息队列架构

### 整体流程

```
用户注册 → auth-server → RocketMQ(topic=UserTopic, tag=registered, key=userID)
                                ↓
                         async-handler → 发送注册短信
```

### 生产端（event-producer）

通用 RocketMQ 生产者库，任何服务引用后注入 `IRmqPublisher` 即可发消息：

```go
// 接口定义
type IRmqPublisher interface {
    Publish(topic string, tag string, key string, payload interface{}) error
}

// 业务调用示例（auth-server）
s.EventPublisher.Publish("UserTopic", "registered", userID, payload)
```

- 配置（NameServer、GroupName）在 `producer/config.go` 的 `Start()` 中初始化
- 通过 implmap 注册，业务方 inji 注入即可使用，无需手动构造

### 消费端（async-handler）

基于 RocketMQ PushConsumer，支持多 topic 订阅，按 topic 路由到对应 handler：

```go
// 订阅配置
Subscriptions: []Subscription{
    {Topic: "UserTopic", Tag: "registered", HandlerName: "userRegisteredHandler"},
    // 新增订阅只需加一行
}

// handler 注册（ServiceProvider.Start()中）
p.Server.RegisterHandler("userRegisteredHandler", p.SMSHandler)
```

- 每个 Subscription 指定 topic + tag + handler 名称
- `SMSConsumerServer.Start()` 时按配置订阅，消息到达后路由到对应 handler
- handler 实现 `IMessageHandler` 接口：`Handle(body []byte) error`

### 扩展方式

| 场景 | 操作 |
|------|------|
| auth-server 发新事件 | 定义新 payload，调用 `Publish("Topic", "tag", key, payload)` |
| 其他服务也要发消息 | 引用 `event-producer`，inji 注入 `IRmqPublisher`，直接调用 |
| async-handler 订阅新 topic | config 加 Subscription + 写新 handler + SP 注册 |

## 快速开始

### 1. 环境准备

- Go 1.18+
- MySQL 8.0+
- Apache RocketMQ 4.x+（NameServer 默认 127.0.0.1:9876）
- protoc + protoc-gen-go + protoc-gen-go-grpc

### 2. 初始化数据库

```bash
mysql -u root -p123456 -e "CREATE DATABASE IF NOT EXISTS ttuser"
mysql -u root -p123456 ttuser < data-store/ddl/001_create_users.sql
mysql -u root -p123456 ttuser < data-store/ddl/002_create_token_blacklist.sql
```

### 3. 启动 RocketMQ

```bash
# 启动 NameServer
nohup sh mqnamesrv &

# 启动 Broker
nohup sh mqbroker -n 127.0.0.1:9876 &
```

### 4. 启动服务

```bash
# 终端1：启动 auth-server (gRPC :9090)
cd auth-server && go run ./cmd/server/

# 终端2：启动 proc (HTTP :8080)
cd proc && go run ./cmd/server/

# 终端3：启动 async-handler
cd async-handler && go run ./cmd/server/
```

### 5. 运行测试

```bash
make test         # 运行所有测试（单元 + e2e）
make unit-test    # 只跑单元测试
make e2e-test     # 自动编译→清数据→启动服务→跑 e2e→停服务
make build        # 编译到 bin/（含 async-handler）
make proto        # 重新生成 proto 代码
make clean        # 清理编译产物
```

## API 接口

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/api/v1/register` | 注册（触发短信通知） | 否 |
| POST | `/api/v1/login` | 登录 | 否 |
| POST | `/api/v1/refresh` | 续签 token | 否 |
| POST | `/api/v1/logout` | 注销 | 是 |
| GET | `/api/v1/user/info` | 获取用户信息 | 是 |
| PUT | `/api/v1/user/info` | 更新用户信息 | 是 |

## 架构设计

### 依赖注入（inji）

所有组件通过 inji 的 `inject` tag 自动装配。ServiceProvider 只声明外部需要访问的顶层组件，中间依赖由 inji 自动递归创建：

```
auth-server ServiceProvider:
  GRPCServer → AuthService → UserDAO/TokenDAO/TokenMgr/EventPublisher
                                                        ↓
                                              RMQConfig → EventRMQPublisher(Start()连接RocketMQ)

async-handler ServiceProvider:
  SMSHandler + SMSConsumerServer
  Start()中注册 handler → Server
  Server.Start()订阅RocketMQ → 消息到达 → 路由到handler → 发短信
```

### 认证流程

```
注册: POST /register → proc → gRPC → auth-server → bcrypt加密 → MySQL → 发RocketMQ事件 → async-handler发短信
登录: POST /login → proc → gRPC → auth-server → 验密 → 签发 access+refresh token
请求: GET /user/info → proc filter → gRPC ValidateToken → 检查黑名单+解析JWT
续签: POST /refresh → proc → gRPC → auth-server → 旧token加黑名单 → 签发新pair
注销: POST /logout → proc filter → gRPC → access+refresh 加入黑名单
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

当前所有配置写死在代码中（带 `TODO: 后续从配置中心获取` 注释），后续接入配置中心只需修改对应 `Start()` 方法：

| 配置项 | 当前默认值 | 位置 |
|--------|-----------|------|
| MySQL DSN | `root:123456@tcp(localhost:3306)/ttuser` | `data-store/engine/proc.go` |
| JWT Secret | `my-secret-key-for-ttuser-2024` | `auth-server/pkg/token/jwt.go` |
| Access Token 有效期 | 2h | `auth-server/pkg/token/jwt.go` |
| Refresh Token 有效期 | 7d | `auth-server/pkg/token/jwt.go` |
| auth-server gRPC 端口 | 9090 | `auth-server/server/grpc_server.go` |
| auth-client gRPC 地址 | `localhost:9090` | `auth-client/client/client.go` |
| proc HTTP 端口 | 8080 | `proc/cmd/server/main.go` (env: HTTP_PORT) |
| RocketMQ NameServer | `127.0.0.1:9876` | `event-producer/producer/config.go` |
| RocketMQ Producer Group | `ttuser-producer-group` | `event-producer/producer/config.go` |
| RocketMQ Consumer Group | `async-handler-group` | `async-handler/server/server.go` |
| 短信签名 | `TT用户平台` | `async-handler/internal/sms/config.go` |

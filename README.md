# 用户认证服务

基于 Go + gRPC + MySQL + RocketMQ + inji(DI) + 配置中心 的微服务架构鉴权系统。

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
├── async-handler/            # 异步消息处理服务（RocketMQ PushConsumer）
│   ├── cmd/server/main.go    # 入口（inji.Reg → sp.Init）
│   ├── pkg/router/           # 通用消息路由框架（Engine + TopicGroup + WrapHandleFunc）
│   ├── biz/
│   │   ├── register/         # 路由注册（统一配置topic/tag/handler映射）
│   │   └── actions/          # 业务action函数（强类型，自动反序列化）
│   ├── server/server.go      # ConsumerServer（基于router.Engine启动PushConsumer）
│   ├── internal/sms/         # 短信发送（Sender + Config）
│   └── sp/                   # ServiceProvider
│
├── config-server/            # 配置中心服务（MySQL存储 + HTTP API + AES加密）
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── dao/              # MySQL CRUD
│   │   ├── model/            # Config模型（service/key/value/encrypted/version）
│   │   └── service/          # 配置管理（Get/Set/SetEncrypted）
│   ├── server/               # HTTP API (Gin, :7963)
│   └── sp/
│
├── config-client/            # 配置中心客户端SDK
│   └── client/client.go      # HTTP客户端（启动拉取 + 定时刷新 + 自动解密）
│
├── pkg/                      # 公共库（所有服务共享）
│   ├── trace/                # 全链路trace_id（生成/context存取/常量）
│   ├── log/                  # zap日志封装（自动带trace_id, 支持Loki推送）
│   └── crypto/               # AES-256-GCM加解密（密钥从环境变量获取）
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
| 配置中心 | config-server（MySQL + HTTP API + AES-256-GCM加密） |
| 链路追踪 | 全链路 trace_id（32-hex, HTTP→gRPC→MQ→Consumer） |
| 日志框架 | zap（JSON结构化, 支持Loki推送） |
| 依赖注入 | github.com/teou/inji |
| 认证方案 | JWT（access_token 2h + refresh_token 7d，轮转续签） |
| 数据存储 | MySQL |
| 密码加密 | bcrypt |
| 配置加密 | AES-256-GCM（密钥通过环境变量注入） |
| 日志管理 | Grafana Loki + Promtail |

## 配置中心

### 架构

```
config-server (MySQL存储, HTTP :7963)
      ↑ 管理员写入配置（敏感配置加密存储）
      ↓ 各服务启动时拉取
┌─────────────┬──────────────┬───────────────┐
│ auth-server │ async-handler│ event-producer│
│ (config-client SDK)                        │
└─────────────┴──────────────┴───────────────┘
```

### API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/config?service=xxx&key=xxx` | 获取单个配置 |
| GET | `/configs?service=xxx` | 获取服务所有配置 |
| POST | `/config` | 写入配置（明文） |
| POST | `/config/encrypted` | 写入配置（加密存储） |

认证：`Authorization: Bearer ttuser-config-token-2024`

### 敏感配置加密

```bash
# 设置加密密钥（生产环境由运维注入，不进代码）
export CONFIG_ENCRYPT_KEY="your-32-byte-production-key!!!!"

# 写入敏感配置（自动加密存储）
curl -X POST http://localhost:7963/config/encrypted \
  -H "Authorization: Bearer ttuser-config-token-2024" \
  -H "Content-Type: application/json" \
  -d '{"service":"auth-server","key":"mysql","value":"{\"dsn\":\"root:123456@tcp(localhost:3306)/ttuser\"}"}'
```

客户端获取时根据 `encrypted` 字段自动解密，对业务透明。

### 配置降级

各服务获取配置的策略：
1. 优先从配置中心获取
2. 配置中心不可用时，降级使用代码中的默认值
3. 保证服务可独立启动（不强依赖配置中心）

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
    Publish(ctx context.Context, topic string, tag string, payload interface{}) error
}

// 业务调用示例（auth-server），trace_id自动从ctx提取写入message key
s.EventPublisher.Publish(ctx, "UserTopic", "registered", payload)
```

- 配置从配置中心获取（降级用默认值）
- trace_id 自动从 ctx 提取，写入 RocketMQ message key（便于控制台按 trace_id 检索）
- 通过 implmap 注册，业务方 inji 注入即可使用，无需手动构造

### 消费端（async-handler）

基于 RocketMQ PushConsumer + router engine，支持多 topic 订阅，按 tag 路由到对应 handler：

```go
// biz/register/register.go — 统一注册路由
func InitRouter(engine *router.Engine) error {
    userGroup := engine.NewTopicGroup("sms-consumer-group", "UserTopic")
    h, _ := router.WrapHandleFunc(actions.UserRegistered)
    userGroup.Handle("registered", h)
    return nil
}

// biz/actions/user_registered.go — 强类型action函数
func UserRegistered(ctx context.Context, req *UserRegisteredReq) error {
    log.Info(ctx, "user registered", "user_id", req.UserID)
    return sms.GetSender().SendRegistrationSMS(req.Phone, req.Username)
}
```

- `router.WrapHandleFunc` 自动将 `func(ctx, *T) error` 包装为 handler（反射反序列化 JSON → typed struct）
- 消息到达时从 message key 提取 trace_id，构建 ctx 传入 handler
- 新增业务只需：写 action 函数 + register 加一行

### 扩展方式

| 场景 | 操作 |
|------|------|
| auth-server 发新事件 | 定义新 payload，调用 `Publish(ctx, "Topic", "tag", payload)` |
| 其他服务也要发消息 | 引用 `event-producer`，inji 注入 `IRmqPublisher`，直接调用 |
| async-handler 订阅新 topic | 写 action 函数 + `register.go` 加一行路由注册 |

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
mysql -u root -p123456 ttuser < data-store/ddl/003_create_configs.sql
mysql -u root -p123456 ttuser < data-store/ddl/004_insert_configs.sql
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
# 终端1：启动配置中心 (:7963)
cd config-server && go run ./cmd/server/

# 终端2：启动 auth-server (gRPC :9090)
cd auth-server && go run ./cmd/server/

# 终端3：启动 proc (HTTP :8080)
cd proc && go run ./cmd/server/

# 终端4：启动 async-handler
cd async-handler && go run ./cmd/server/
```

或者一键部署：
```bash
make deploy    # 编译+安装+启动所有服务
make status    # 查看服务状态
make stop      # 停止所有服务
```

### 5. 运行测试

```bash
make build        # 编译所有服务到 bin/
make test         # 运行所有测试（单元 + e2e）
make unit-test    # 只跑单元测试
make e2e-test     # 自动编译→清数据→启动服务→跑 e2e→停服务
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
auth-server:
  GRPCServer → AuthService → DAO/TokenMgr/EventPublisher → RMQConfig(配置中心)

async-handler:
  SMSSender + ConsumerServer → router.Engine → actions → SMSSender

config-server:
  HTTPServer → ConfigService → ConfigDAO → MySQL
```

### 全链路 trace_id

```
[HTTP] X-Trace-Id header (TraceFilter 生成/提取)
  → [gRPC Client Interceptor] ctx → metadata
    → [gRPC Server Interceptor] metadata → ctx
      → [Publish] ctx → RocketMQ message key
        → [Consumer] message key → ctx → action handler
          → log.Info(ctx, ...) 自动带 trace_id
```

### 日志

- 框架：zap（JSON结构化）
- 输出：stdout + 文件（`/home/work/log/{服务名}_{端口}/{日期}.log`）
- 可选：Loki 直推（开发环境）
- 生产：Promtail 采集文件 → Loki → Grafana

Access log 示例：
```json
{"level":"info","ts":"2026-05-26T10:00:00","msg":"access_log","trace_id":"0af765...","data":{"method":"POST","path":"/api/v1/register","cost":"45ms","uid":"","httpStatus":200,"req":{...},"ret":{...}}}
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

配置优先从配置中心获取（`config-server :7963`），不可用时降级到代码默认值：

| service | key | 内容 | 是否加密 |
|---------|-----|------|---------|
| auth-server | jwt | secret, access_expire, refresh_expire | 建议加密 |
| auth-server | grpc | port, cert, cert_key | 否 |
| auth-server | mysql | dsn | 加密 |
| event-producer | rocketmq | name_server, group_name | 否 |
| async-handler | rocketmq | name_server, consumer_group | 否 |
| async-handler | sms | api_key, api_secret, sign_name, template | 加密 |
| proc | server | port | 否 |
| proc | auth-client | addr, ca_cert | 否 |

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `CONFIG_ENCRYPT_KEY` | AES加密密钥（32字节） | `ttuser-config-secret-key-2024!!`（仅开发） |

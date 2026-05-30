# Loki + Prometheus + Promtail + Grafana 可观测性栈

## 架构

```
                               ┌──────────────────────┐
Go 服务 ──写入日志──> /home/work/log/{service}/{日期}.log
                               │                     │
                               ▼                     ▼
                          Promtail (Docker)    Prometheus (Docker)
                               │                     │
                               ▼                     ▼
                          Loki (Docker)        Prometheus (Docker)
                               │                     │
                               └────────┬────────────┘
                                        ▼
                                   Grafana (Docker) :3000
```

- **Prometheus** :9090 — 指标采集，通过 `host.docker.internal` 抓取本地 Go 服务 metrics
- **Loki** :3100 — 日志存储与查询
- **Promtail** — 日志采集器，从文件 tail 并推送到 Loki
- **Grafana** :3000 — 可视化面板，数据源已预配（admin/admin）

## 启动

```bash
cd deploy/loki
docker compose up -d
```

## 目录结构

| 路径 | 说明 |
|------|------|
| `./logs/{service}/{日期}.log` | Go 服务日志文件（Docker 挂载点） |
| `./docker-compose.yml` | 容器编排 |
| `./prometheus.yml` | Prometheus 采集配置 |
| `./promtail-config.yml` | Promtail 配置，日志解析规则 |
| `./loki-config.yml` | Loki 配置 |
| `./grafana-datasource.yml` | Grafana 数据源预配（Loki + Prometheus） |

## Prometheus 采集目标

| job_name | 目标地址 | 说明 |
|----------|---------|------|
| `prometheus` | `localhost:9090` | Prometheus 自身 |
| `proc` | `host.docker.internal:8080` | HTTP 网关 |
| `auth-server` | `host.docker.internal:9090` | 认证服务 |
| `config-server` | `host.docker.internal:7963` | 配置中心 |

所有指标已带 `service` 标签区分来源。

## Windows 路径映射说明

### 问题背景

Docker Desktop for Windows（WSL2 模式）对 Windows 路径的 bind mount 有限制：
- 不能直接从 `I:` 以外的驱动器挂载
- Docker Compose v5 重启不会更新卷挂载（需 `down` 后重新 `up -d`）

### 解决方案

| 角色 | 实际路径（Windows） | 类型 |
|------|-------------------|------|
| Go 写入 | `/home/work/log/...` → `C:\home\work\log\...` | Windows 文件系统 |
| 重定向 | `C:\home\work\log` → `I:\...\deploy\loki\logs` | NTFS Junction |
| Promtail 挂载 | `./logs` → `/home/work/log`（容器内） | Docker bind mount |

Junction 确保 Go 服务（运行在 Windows 上）写入的日志自动落入项目目录，Promtail 容器通过相对路径 `./logs` 挂载同一目录。

## 关键端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Grafana | 3000 | admin/admin |
| Prometheus | 9090 | Web UI + API |
| Loki | 3100 | HTTP API |
| Promtail | 9080 | 管理接口（容器内） |

## Prometheus 指标

### 服务端（pkg/http/server）

| 指标 | 类型 | 标签 | 说明 |
|------|------|------|------|
| `http_server_requests_total` | Counter | `service, method, path, status` | 请求总数 |
| `http_server_request_duration_seconds` | Histogram | `service, method, path` | 请求耗时 |
| `http_server_requests_active` | Gauge | `service` | 活跃请求数 |

### 客户端（pkg/http/client）

| 指标 | 类型 | 标签 | 说明 |
|------|------|------|------|
| `http_client_requests_total` | Counter | `service, method, host, status` | 请求总数 |
| `http_client_request_duration_seconds` | Histogram | `service, method, host` | 请求耗时 |
| `http_client_requests_active` | Gauge | `service` | 活跃请求数 |

### gRPC（go-grpc-prometheus 标准指标）

| 指标 | 类型 | 标签 | 说明 |
|------|------|------|------|
| `grpc_server_started_total` | Counter | `grpc_type, grpc_service, grpc_method` | 请求开始数 |
| `grpc_server_handled_total` | Counter | `grpc_type, grpc_service, grpc_method, grpc_code` | 请求完成数 |
| `grpc_server_handling_seconds` | Histogram | `grpc_type, grpc_service, grpc_method` | 请求耗时分布 |
| `grpc_server_msg_received_total` | Counter | `grpc_type, grpc_service, grpc_method` | 接收消息数 |
| `grpc_server_msg_sent_total` | Counter | `grpc_type, grpc_service, grpc_method` | 发送消息数 |

这些是 `go-grpc-prometheus` 库自带的标准指标，etcd、CockroachDB 等大厂通用。

### 自定义 gRPC 指标（带 service 标签）

| 指标 | 类型 | 标签 | 说明 |
|------|------|------|------|
| `ttuser_grpc_calls_total` | Counter | `service, method, status` | 请求总数 |
| `ttuser_grpc_call_duration_seconds` | Histogram | `service, method` | 请求耗时 |

## 日志字段说明

| 标签 | 来源 | 值示例 |
|------|------|--------|
| `job` | 静态配置 | `ttuser` |
| `env` | 静态配置 | `dev` |
| `service` | 从文件路径标签 | `proc` |
| `level` | JSON 日志字段 | `info` |
| `trace_id` | JSON 日志字段 | `abc123` |
| `msg` | JSON 日志字段 | `request completed` |
| `path` | 从 `data.path` 解析 | `/api/user` |
| `method` | 从 `data.method` 解析 | `GET` |

## 日志格式

```json
{"level":"info","trace_id":"xxx","msg":"request completed","data":"{\"path\":\"/api/user\",\"method\":\"GET\",\"status\":200,\"latency\":\"5ms\"}"}
```

## 常用查询

### Grafana Loki Explore

```logql
# 查所有日志
{job="ttuser"}

# 按服务查
{service="proc"}

# 按级别筛
{service="proc"} |= "level\":\"error"

# 按 trace_id 查
{trace_id="xxx"}
```

### Grafana Prometheus Explore / Dashboard

```promql
# 服务端请求速率（rps）
rate(http_server_requests_total{service="proc"}[5m])

# 请求耗时 p99
histogram_quantile(0.99, rate(http_server_request_duration_seconds_bucket{service="proc"}[5m]))

# 活跃请求数
http_server_requests_active{service="proc"}
```

## 常见问题

### 修改配置后容器不生效

```bash
docker compose down && docker compose up -d
```

`docker compose restart` **不会**更新卷挂载和网络配置。

### Promtail 看不到日志文件

```bash
docker compose exec promtail ls -la /home/work/log/{service}/
```

### 告警 "No labels found."

说明 Loki 中没有数据。检查：
1. Go 服务是否运行并写入日志
2. Junction `C:\home\work\log` 是否存在且指向 `deploy/loki/logs`
3. Promtail 日志：`docker compose logs promtail`

### Prometheus 抓取目标 down

在 http://localhost:9090/targets 检查各 job 状态。服务未运行时对应 target 会显示 `DOWN`。

## 参考链接

- https://grafana.com/docs/loki/latest/
- https://grafana.com/docs/loki/latest/clients/promtail/
- https://prometheus.io/docs/prometheus/latest/
- https://grafana.com/docs/grafana/latest/

.PHONY: build build-auth-server build-proc build-async-handler build-config-server test unit-test e2e-test clean proto install deploy stop status vet fmt tidy dev stop-dev docker-up docker-down

# ==================== 配置 ====================
BIN_DIR := ./bin
LOG_DIR := ./log
INSTALL_DIR := ./ttuser

# 服务配置
AUTH_SERVER_PORT := 9090
PROC_PORT := 8080
CONFIG_SERVER_PORT := 7963
DEV_AUTH_PORT := 9091

BUILD_FLAGS := -ldflags="-s -w" -trimpath

# ==================== Docker 依赖 ====================
docker-up:
	@echo "[docker] starting MySQL..."
	@docker rm -f mysql 2>/dev/null; docker run -d --name mysql -e MYSQL_ROOT_PASSWORD=123456 -e MYSQL_DATABASE=ttuser -p 3306:3306 mysql:8.0 --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci >/dev/null
	@sleep 8
	@echo "[docker] starting Nacos..."
	@docker rm -f nacos 2>/dev/null; docker run -d --name nacos -p 8848:8848 -p 9848:9848 -e MODE=standalone nacos/nacos-server:v2.3.2 >/dev/null
	@sleep 10
	@echo "[docker] MySQL + Nacos ready"

docker-down:
	@echo "[docker] stopping..."
	@-docker rm -f mysql nacos 2>/dev/null
	@echo "[docker] done"

# ==================== 编译 ====================
build: build-auth-server build-proc build-async-handler build-config-server

build-auth-server:
	@echo "[build] auth-server..."
	@cd auth-server && go build $(BUILD_FLAGS) -o ../$(BIN_DIR)/auth-server ./cmd/server/

build-proc:
	@echo "[build] proc..."
	@cd proc && go build $(BUILD_FLAGS) -o ../$(BIN_DIR)/proc ./cmd/server/

build-async-handler:
	@echo "[build] async-handler..."
	@cd async-handler && go build $(BUILD_FLAGS) -o ../$(BIN_DIR)/async-handler ./cmd/server/

build-config-server:
	@echo "[build] config-server..."
	@cd config-server && go build $(BUILD_FLAGS) -o ../$(BIN_DIR)/config-server ./cmd/server/

# ==================== 代码质量 ====================
vet:
	@echo "[vet] running go vet on all modules..."
	@cd auth-server && go vet ./...
	@cd proc && go vet ./...
	@cd config-server && go vet ./...
	@cd async-handler && go vet ./...
	@cd config-client && go vet ./...
	@cd data-store && go vet ./...
	@cd event-producer && go vet ./...
	@cd pkg && go vet ./...

fmt:
	@echo "[fmt] checking go formatting..."
	@cd auth-server && gofmt -l . || true

tidy:
	@echo "[tidy] running go mod tidy on all modules..."
	@cd auth-server && go mod tidy
	@cd proc && go mod tidy
	@cd config-server && go mod tidy
	@cd async-handler && go mod tidy
	@cd config-client && go mod tidy
	@cd data-store && go mod tidy
	@cd event-producer && go mod tidy
	@cd pkg && go mod tidy

# ==================== 一键安装 ====================
install: build
	@echo "[install] installing ttuser services..."
	@mkdir -p $(INSTALL_DIR)/bin $(LOG_DIR)/auth-server_$(AUTH_SERVER_PORT) $(LOG_DIR)/proc_$(PROC_PORT) $(LOG_DIR)/async-handler_0
	@cp $(BIN_DIR)/* $(INSTALL_DIR)/bin/
	@echo "[install] done. binaries at $(INSTALL_DIR)/bin/"
	@echo "[install] logs at $(LOG_DIR)/"

# ==================== 一键部署（启动所有服务） ====================
deploy: install
	@echo "[deploy] starting all services..."
	@echo "[deploy] starting config-server on :$(CONFIG_SERVER_PORT)..."
	@cd $(INSTALL_DIR) && nohup ./bin/config-server -name=config-server -port=$(CONFIG_SERVER_PORT) -env=prod > $(LOG_DIR)/config-server_$(CONFIG_SERVER_PORT)/stdout.log 2>&1 & echo $$! > /tmp/ttuser_config_server.pid
	@sleep 2
	@echo "[deploy] starting auth-server on :$(AUTH_SERVER_PORT)..."
	@cd $(INSTALL_DIR) && nohup ./bin/auth-server -name=auth-server -port=$(AUTH_SERVER_PORT) -env=prod > $(LOG_DIR)/auth-server_$(AUTH_SERVER_PORT)/stdout.log 2>&1 & echo $$! > /tmp/ttuser_auth_server.pid
	@sleep 2
	@echo "[deploy] starting proc on :$(PROC_PORT)..."
	@cd $(INSTALL_DIR) && nohup ./bin/proc -name=proc -port=$(PROC_PORT) -env=prod > $(LOG_DIR)/proc_$(PROC_PORT)/stdout.log 2>&1 & echo $$! > /tmp/ttuser_proc.pid
	@sleep 1
	@echo "[deploy] starting async-handler..."
	@cd $(INSTALL_DIR) && nohup ./bin/async-handler -name=async-handler -port=0 -env=prod > $(LOG_DIR)/async-handler_0/stdout.log 2>&1 & echo $$! > /tmp/ttuser_async_handler.pid
	@sleep 1
	@echo "[deploy] all services started."
	@echo "  config-server PID=$$(cat /tmp/ttuser_config_server.pid)"
	@echo "  auth-server   PID=$$(cat /tmp/ttuser_auth_server.pid)"
	@echo "  proc          PID=$$(cat /tmp/ttuser_proc.pid)"
	@echo "  async-handler PID=$$(cat /tmp/ttuser_async_handler.pid)"

# ==================== 停止所有服务 ====================
stop:
	@echo "[stop] stopping all services..."
	@-kill $$(cat /tmp/ttuser_config_server.pid) 2>/dev/null && echo "  config-server stopped" || echo "  config-server not running"
	@-kill $$(cat /tmp/ttuser_auth_server.pid) 2>/dev/null && echo "  auth-server stopped" || echo "  auth-server not running"
	@-kill $$(cat /tmp/ttuser_proc.pid) 2>/dev/null && echo "  proc stopped" || echo "  proc not running"
	@-kill $$(cat /tmp/ttuser_async_handler.pid) 2>/dev/null && echo "  async-handler stopped" || echo "  async-handler not running"
	@rm -f /tmp/ttuser_*.pid
	@echo "[stop] done."

# ==================== 查看服务状态 ====================
status:
	@echo "[status] checking services..."
	@for pid_file in /tmp/ttuser_config_server.pid /tmp/ttuser_auth_server.pid /tmp/ttuser_proc.pid /tmp/ttuser_async_handler.pid; do \
		name=$$(basename $$pid_file .pid | sed 's/ttuser_//'); \
		if [ -f $$pid_file ] && kill -0 $$(cat $$pid_file) 2>/dev/null; then \
			echo "  $$name RUNNING (PID=$$(cat $$pid_file))"; \
		else \
			echo "  $$name STOPPED"; \
		fi; \
	done

# ==================== 测试 ====================
test: unit-test e2e-test

unit-test:
	@echo "[test] running unit tests..."
	@cd auth-server && go test ./... -count=1
	@cd config-client && go test ./... -count=1
	@cd event-producer && go test ./... -count=1
	@cd pkg && go test ./... -count=1

e2e-test: build
	@echo "[e2e] preparing database..."
	@echo "[e2e] make sure mysql credentials are configured in ~/.my.cnf"
	@mysql ttuser -e "TRUNCATE TABLE users; TRUNCATE TABLE token_blacklist;" 2>/dev/null || true
	@echo "[e2e] starting config-server..."
	@cd config-server && nohup ../$(BIN_DIR)/config-server -name=config-server -port=7963 -env=test --config-dir=config-server/config-center > /dev/null 2>&1 & echo $$! > /tmp/e2e_config_server.pid
	@sleep 1
	@echo "[e2e] starting auth-server..."
	@cd auth-server && nohup ../$(BIN_DIR)/auth-server -name=auth-server -port=9090 -env=test > /dev/null 2>&1 & echo $$! > /tmp/e2e_auth_server.pid
	@sleep 2
	@echo "[e2e] starting proc..."
	@cd proc && nohup ../$(BIN_DIR)/proc -name=proc -port=8080 -env=test > /dev/null 2>&1 & echo $$! > /tmp/e2e_proc.pid
	@sleep 2
	@echo "[e2e] running e2e tests..."
	@cd proc && go test ./e2e/ -v -count=1 -run "TestE2E" -timeout=30s; \
		EXIT_CODE=$$?; \
		kill $$(cat /tmp/e2e_config_server.pid) 2>/dev/null; \
		kill $$(cat /tmp/e2e_proc.pid) 2>/dev/null; \
		kill $$(cat /tmp/e2e_auth_server.pid) 2>/dev/null; \
		rm -f /tmp/e2e_config_server.pid /tmp/e2e_proc.pid /tmp/e2e_auth_server.pid; \
		rm -rf ./config; \
		exit $$EXIT_CODE

# ==================== 开发环境启动（Linux/Mac/WSL） ====================
dev: build
	@echo "[dev] starting development environment..."
	@mkdir -p $(LOG_DIR)
	@echo "[dev] starting config-server on :$(CONFIG_SERVER_PORT)..."
	@SERVICE_NAME=config-server SERVICE_PORT=$(CONFIG_SERVER_PORT) ENV=staging \
		nohup $(BIN_DIR)/config-server -name=config-server -port=$(CONFIG_SERVER_PORT) -env=staging -config-dir=config-server/config-center > $(LOG_DIR)/config-server.log 2>&1 & echo $$! > /tmp/ttuser_dev_config_server.pid
	@sleep 3
	@echo "[dev] starting auth-server on :$(DEV_AUTH_PORT)..."
	@SERVICE_NAME=auth-server SERVICE_PORT=$(DEV_AUTH_PORT) ENV=staging \
		nohup $(BIN_DIR)/auth-server -name=auth-server -port=$(DEV_AUTH_PORT) -env=staging > $(LOG_DIR)/auth-server.log 2>&1 & echo $$! > /tmp/ttuser_dev_auth_server.pid
	@sleep 4
	@echo "[dev] starting proc on :$(PROC_PORT)..."
	@SERVICE_NAME=proc SERVICE_PORT=$(PROC_PORT) ENV=staging \
		nohup $(BIN_DIR)/proc -name=proc -port=$(PROC_PORT) -env=staging > $(LOG_DIR)/proc.log 2>&1 & echo $$! > /tmp/ttuser_dev_proc.pid
	@sleep 2
	@echo "[dev] all services started."
	@echo "  config-server  http://127.0.0.1:$(CONFIG_SERVER_PORT)"
	@echo "  auth-server    grpc://127.0.0.1:$(DEV_AUTH_PORT)  metrics http://127.0.0.1:$(shell echo $$(($(DEV_AUTH_PORT)+100)))"
	@echo "  proc           http://127.0.0.1:$(PROC_PORT)"
	@echo "  logs: $(LOG_DIR)/"
	@echo "  Stop with: make stop-dev"

stop-dev:
	@echo "[stop-dev] stopping..."
	@-kill $$(cat /tmp/ttuser_dev_config_server.pid) 2>/dev/null && echo "  config-server stopped" || true
	@-kill $$(cat /tmp/ttuser_dev_auth_server.pid) 2>/dev/null && echo "  auth-server stopped" || true
	@-kill $$(cat /tmp/ttuser_dev_proc.pid) 2>/dev/null && echo "  proc stopped" || true
	@rm -f /tmp/ttuser_dev_*.pid
	@rm -rf ./config
	@echo "[stop-dev] done."

# ==================== 工具 ====================
proto:
	@echo "[proto] generating auth-client proto..."
	@cd auth-client/proto && bash gen.sh

clean:
	@rm -rf $(BIN_DIR) ./config
	@echo "[clean] done"

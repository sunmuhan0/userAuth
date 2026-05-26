.PHONY: build build-auth-server build-proc build-async-handler build-config-server test unit-test e2e-test clean proto install deploy stop status

# ==================== 配置 ====================
BIN_DIR := ./bin
LOG_DIR := /home/work/log
INSTALL_DIR := /home/work/ttuser

# 服务配置
AUTH_SERVER_PORT := 9090
PROC_PORT := 8080

# ==================== 编译 ====================
build: build-auth-server build-proc build-async-handler build-config-server

build-auth-server:
	@echo "[build] auth-server..."
	@cd auth-server && go build -o ../$(BIN_DIR)/auth-server ./cmd/server/

build-proc:
	@echo "[build] proc..."
	@cd proc && go build -o ../$(BIN_DIR)/proc ./cmd/server/

build-async-handler:
	@echo "[build] async-handler..."
	@cd async-handler && go build -o ../$(BIN_DIR)/async-handler ./cmd/server/

build-config-server:
	@echo "[build] config-server..."
	@cd config-server && go build -o ../$(BIN_DIR)/config-server ./cmd/server/

# ==================== 一键安装 ====================
install: build
	@echo "[install] installing ttuser services..."
	@mkdir -p $(INSTALL_DIR)/bin
	@mkdir -p $(INSTALL_DIR)/certs
	@mkdir -p $(LOG_DIR)/auth-server_$(AUTH_SERVER_PORT)
	@mkdir -p $(LOG_DIR)/proc_$(PROC_PORT)
	@mkdir -p $(LOG_DIR)/async-handler_0
	@cp $(BIN_DIR)/* $(INSTALL_DIR)/bin/
	@cp -r certs/* $(INSTALL_DIR)/certs/ 2>/dev/null || true
	@echo "[install] done. binaries at $(INSTALL_DIR)/bin/"
	@echo "[install] logs at $(LOG_DIR)/"

# ==================== 一键部署（启动所有服务） ====================
deploy: install
	@echo "[deploy] starting all services..."
	@echo "[deploy] starting auth-server on :$(AUTH_SERVER_PORT)..."
	@cd $(INSTALL_DIR) && nohup ./bin/auth-server > $(LOG_DIR)/auth-server_$(AUTH_SERVER_PORT)/stdout.log 2>&1 & echo $$! > /tmp/ttuser_auth_server.pid
	@sleep 2
	@echo "[deploy] starting proc on :$(PROC_PORT)..."
	@cd $(INSTALL_DIR) && nohup ./bin/proc > $(LOG_DIR)/proc_$(PROC_PORT)/stdout.log 2>&1 & echo $$! > /tmp/ttuser_proc.pid
	@sleep 1
	@echo "[deploy] starting async-handler..."
	@cd $(INSTALL_DIR) && nohup ./bin/async-handler > $(LOG_DIR)/async-handler_0/stdout.log 2>&1 & echo $$! > /tmp/ttuser_async_handler.pid
	@sleep 1
	@echo "[deploy] all services started."
	@echo "  auth-server  PID=$$(cat /tmp/ttuser_auth_server.pid)"
	@echo "  proc         PID=$$(cat /tmp/ttuser_proc.pid)"
	@echo "  async-handler PID=$$(cat /tmp/ttuser_async_handler.pid)"

# ==================== 停止所有服务 ====================
stop:
	@echo "[stop] stopping all services..."
	@kill $$(cat /tmp/ttuser_auth_server.pid) 2>/dev/null && echo "  auth-server stopped" || echo "  auth-server not running"
	@kill $$(cat /tmp/ttuser_proc.pid) 2>/dev/null && echo "  proc stopped" || echo "  proc not running"
	@kill $$(cat /tmp/ttuser_async_handler.pid) 2>/dev/null && echo "  async-handler stopped" || echo "  async-handler not running"
	@rm -f /tmp/ttuser_*.pid
	@echo "[stop] done."

# ==================== 查看服务状态 ====================
status:
	@echo "[status] checking services..."
	@if [ -f /tmp/ttuser_auth_server.pid ] && kill -0 $$(cat /tmp/ttuser_auth_server.pid) 2>/dev/null; then \
		echo "  auth-server    RUNNING (PID=$$(cat /tmp/ttuser_auth_server.pid))"; \
	else \
		echo "  auth-server    STOPPED"; \
	fi
	@if [ -f /tmp/ttuser_proc.pid ] && kill -0 $$(cat /tmp/ttuser_proc.pid) 2>/dev/null; then \
		echo "  proc           RUNNING (PID=$$(cat /tmp/ttuser_proc.pid))"; \
	else \
		echo "  proc           STOPPED"; \
	fi
	@if [ -f /tmp/ttuser_async_handler.pid ] && kill -0 $$(cat /tmp/ttuser_async_handler.pid) 2>/dev/null; then \
		echo "  async-handler  RUNNING (PID=$$(cat /tmp/ttuser_async_handler.pid))"; \
	else \
		echo "  async-handler  STOPPED"; \
	fi

# ==================== 测试 ====================
test: unit-test e2e-test

unit-test:
	@echo "[test] running unit tests..."
	@cd auth-server && go test ./pkg/token/ ./internal/service/ -v -count=1

e2e-test: build
	@echo "[e2e] preparing database..."
	@mysql -u root -p123456 ttuser -e "TRUNCATE TABLE users; TRUNCATE TABLE token_blacklist;" 2>/dev/null || true
	@echo "[e2e] starting auth-server..."
	@cd auth-server && ../$(BIN_DIR)/auth-server > /dev/null 2>&1 & echo $$! > /tmp/e2e_auth_server.pid
	@sleep 2
	@echo "[e2e] starting proc..."
	@cd proc && ../$(BIN_DIR)/proc > /dev/null 2>&1 & echo $$! > /tmp/e2e_proc.pid
	@sleep 2
	@echo "[e2e] running e2e tests..."
	@cd proc && go test ./e2e/ -v -count=1 -run "TestE2E" -timeout=30s; \
		EXIT_CODE=$$?; \
		kill $$(cat /tmp/e2e_proc.pid) 2>/dev/null; \
		kill $$(cat /tmp/e2e_auth_server.pid) 2>/dev/null; \
		rm -f /tmp/e2e_proc.pid /tmp/e2e_auth_server.pid; \
		exit $$EXIT_CODE

# ==================== 工具 ====================
proto:
	@echo "[proto] generating auth-client proto..."
	@cd auth-client/proto && bash gen.sh

clean:
	@rm -rf $(BIN_DIR)
	@echo "[clean] done"

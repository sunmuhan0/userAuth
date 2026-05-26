.PHONY: build build-auth-server build-proc build-async-handler test unit-test e2e-test clean proto

# 输出目录
BIN_DIR := ./bin

# 编译所有服务
build: build-auth-server build-proc build-async-handler

build-auth-server:
	@echo "[build] auth-server..."
	@cd auth-server && go build -o ../$(BIN_DIR)/auth-server ./cmd/server/

build-proc:
	@echo "[build] proc..."
	@cd proc && go build -o ../$(BIN_DIR)/proc ./cmd/server/

build-async-handler:
	@echo "[build] async-handler..."
	@cd async-handler && go build -o ../$(BIN_DIR)/async-handler ./cmd/server/

# 运行所有测试
test: unit-test e2e-test

# 单元测试
unit-test:
	@echo "[test] running unit tests..."
	@cd auth-server && go test ./pkg/token/ ./internal/service/ -v -count=1

# E2E 测试（自动启动/停止服务）
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

# 生成 proto 代码
proto:
	@echo "[proto] generating auth-client proto..."
	@cd auth-client/proto && bash gen.sh

# 清理编译产物
clean:
	@rm -rf $(BIN_DIR)
	@echo "[clean] done"

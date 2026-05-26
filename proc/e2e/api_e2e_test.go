package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

const (
	procBaseURL = "http://localhost:8080"
)

var (
	configServerProcess *os.Process
	authServerProcess   *os.Process
	procProcess         *os.Process
)

func TestMain(m *testing.M) {
	// 清空测试数据
	exec.Command("mysql", "-u", "root", "-p123456", "ttuser", "-e",
		"TRUNCATE TABLE users; TRUNCATE TABLE token_blacklist;").Run()

	// 编译
	exec.Command("go", "build", "-o", "/tmp/e2e_config_server", "./cmd/server/").
		Run()
	exec.Command("go", "build", "-o", "/tmp/e2e_auth_server", "./cmd/server/").
		Run()
	exec.Command("go", "build", "-o", "/tmp/e2e_proc", "./cmd/server/").
		Run()

	// 启动 config-server (e2e 目录在 proc/e2e，向上两级是项目根目录)
	cmd0 := exec.Command("/tmp/e2e_config_server",
		"-name=config-server", "-port=7963", "-env=test")
	cmd0.Dir = "../../config-server"
	cmd0.Start()
	configServerProcess = cmd0.Process
	time.Sleep(1 * time.Second)

	// 启动 auth-server
	cmd1 := exec.Command("/tmp/e2e_auth_server",
		"-name=auth-server", "-port=9090", "-env=test")
	cmd1.Dir = "../../auth-server"
	cmd1.Start()
	authServerProcess = cmd1.Process
	time.Sleep(2 * time.Second)

	// 启动 proc
	cmd2 := exec.Command("/tmp/e2e_proc",
		"-name=proc", "-port=8080", "-env=test")
	cmd2.Dir = ".."
	cmd2.Start()
	procProcess = cmd2.Process
	time.Sleep(2 * time.Second)

	// 等待就绪
	waitForReady(procBaseURL + "/api/v1/login")

	code := m.Run()

	// 清理
	if procProcess != nil {
		procProcess.Kill()
	}
	if authServerProcess != nil {
		authServerProcess.Kill()
	}
	if configServerProcess != nil {
		configServerProcess.Kill()
	}

	os.Exit(code)
}

func TestE2E_Register(t *testing.T) {
	resp := post(t, "/api/v1/register", map[string]string{
		"username": "e2e_user",
		"password": "testpass123",
		"nickname": "E2E测试用户",
		"email":    "e2e@test.com",
	})
	assertCode(t, resp, 0)

	data := resp["data"].(map[string]interface{})
	if data["username"] != "e2e_user" {
		t.Fatalf("expected username=e2e_user, got %v", data["username"])
	}
}

func TestE2E_RegisterDuplicate(t *testing.T) {
	resp := post(t, "/api/v1/register", map[string]string{
		"username": "e2e_user",
		"password": "testpass123",
	})
	assertCode(t, resp, 409)
}

func TestE2E_Login(t *testing.T) {
	resp := post(t, "/api/v1/login", map[string]string{
		"username": "e2e_user",
		"password": "testpass123",
	})
	assertCode(t, resp, 0)

	data := resp["data"].(map[string]interface{})
	if data["access_token"] == nil || data["access_token"] == "" {
		t.Fatal("access_token should not be empty")
	}
	if data["refresh_token"] == nil || data["refresh_token"] == "" {
		t.Fatal("refresh_token should not be empty")
	}
}

func TestE2E_LoginWrongPassword(t *testing.T) {
	resp := post(t, "/api/v1/login", map[string]string{
		"username": "e2e_user",
		"password": "wrongpass",
	})
	assertCode(t, resp, 401)
}

func TestE2E_GetUserInfo(t *testing.T) {
	accessToken := login(t)
	resp := getWithAuth(t, "/api/v1/user/info", accessToken)
	assertCode(t, resp, 0)

	data := resp["data"].(map[string]interface{})
	if data["username"] != "e2e_user" {
		t.Fatalf("expected username=e2e_user, got %v", data["username"])
	}
}

func TestE2E_UpdateUserInfo(t *testing.T) {
	accessToken := login(t)
	resp := putWithAuth(t, "/api/v1/user/info", accessToken, map[string]string{
		"nickname": "更新后的名字",
		"email":    "updated@test.com",
	})
	assertCode(t, resp, 0)

	data := resp["data"].(map[string]interface{})
	if data["nickname"] != "更新后的名字" {
		t.Fatalf("expected updated nickname, got %v", data["nickname"])
	}
}

func TestE2E_RefreshToken(t *testing.T) {
	_, refreshToken := loginFull(t)

	resp := post(t, "/api/v1/refresh", map[string]string{
		"refresh_token": refreshToken,
	})
	assertCode(t, resp, 0)

	data := resp["data"].(map[string]interface{})
	if data["access_token"] == nil || data["access_token"] == "" {
		t.Fatal("new access_token should not be empty")
	}

	// 旧 refresh token 再次使用应失败（轮转）
	resp2 := post(t, "/api/v1/refresh", map[string]string{
		"refresh_token": refreshToken,
	})
	assertCode(t, resp2, 401)
}

func TestE2E_Logout(t *testing.T) {
	accessToken, refreshToken := loginFull(t)

	resp := postWithAuth(t, "/api/v1/logout", accessToken, map[string]string{
		"refresh_token": refreshToken,
	})
	assertCode(t, resp, 0)

	// access token 应失效
	resp2 := getWithAuth(t, "/api/v1/user/info", accessToken)
	assertCode(t, resp2, 401)

	// refresh token 也应失效
	resp3 := post(t, "/api/v1/refresh", map[string]string{
		"refresh_token": refreshToken,
	})
	assertCode(t, resp3, 401)
}

func TestE2E_NoToken(t *testing.T) {
	resp := get(t, "/api/v1/user/info")
	assertCode(t, resp, 401)
}

// ============ 辅助函数 ============

func login(t *testing.T) string {
	at, _ := loginFull(t)
	return at
}

func loginFull(t *testing.T) (string, string) {
	resp := post(t, "/api/v1/login", map[string]string{
		"username": "e2e_user",
		"password": "testpass123",
	})
	data := resp["data"].(map[string]interface{})
	return data["access_token"].(string), data["refresh_token"].(string)
}

func post(t *testing.T, path string, body interface{}) map[string]interface{} {
	return doRequest(t, "POST", path, "", body)
}

func postWithAuth(t *testing.T, path, token string, body interface{}) map[string]interface{} {
	return doRequest(t, "POST", path, token, body)
}

func get(t *testing.T, path string) map[string]interface{} {
	return doRequest(t, "GET", path, "", nil)
}

func getWithAuth(t *testing.T, path, token string) map[string]interface{} {
	return doRequest(t, "GET", path, token, nil)
}

func putWithAuth(t *testing.T, path, token string, body interface{}) map[string]interface{} {
	return doRequest(t, "PUT", path, token, body)
}

func doRequest(t *testing.T, method, path, token string, body interface{}) map[string]interface{} {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, procBaseURL+path, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request to %s failed: %v", path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v, body: %s", err, string(respBody))
	}
	return result
}

func assertCode(t *testing.T, resp map[string]interface{}, expectedCode int) {
	t.Helper()
	code := int(resp["code"].(float64))
	if code != expectedCode {
		t.Fatalf("expected code=%d, got code=%d, message=%v", expectedCode, code, resp["message"])
	}
}

func waitForReady(url string) {
	for i := 0; i < 15; i++ {
		resp, err := http.Post(url, "application/json", bytes.NewBufferString(`{}`))
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

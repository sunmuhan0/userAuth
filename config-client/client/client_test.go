package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Env = "prod"
	cfg.ServiceName = "auth-server"
	c := New(cfg)

	if c.serviceName != "auth-server" {
		t.Fatalf("expected serviceName=auth-server, got %s", c.serviceName)
	}
	if c.env != "prod" {
		t.Fatalf("expected env=prod, got %s", c.env)
	}
	if c.endpoint != "http://127.0.0.1:7963" {
		t.Fatalf("expected endpoint=http://127.0.0.1:7963, got %s", c.endpoint)
	}
}

func TestFetchConfigsWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config/files" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"files": []map[string]interface{}{
						{"name": "mysql.json", "content": `{"dsn":"root:@tcp(localhost:3306)/test"}`},
					},
				},
			})
		}
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.Endpoint = server.URL
	cfg.Env = "prod"
	cfg.ServiceName = "auth-server"
	cfg.Token = "test-token"

	c := New(cfg)
	if err := c.FetchConfigs(); err != nil {
		t.Fatalf("FetchConfigs failed: %v", err)
	}

	writtenFile := filepath.Join("/home/work/config", "auth-server", "mysql.json")
	if _, err := os.Stat(writtenFile); os.IsNotExist(err) {
		t.Fatalf("expected file %s to exist", writtenFile)
	}
	data, err := os.ReadFile(writtenFile)
	if err != nil {
		t.Fatalf("read %s failed: %v", writtenFile, err)
	}
	var result struct {
		DSN string `json:"dsn"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if result.DSN != "root:@tcp(localhost:3306)/test" {
		t.Fatalf("expected DSN=root:@tcp(localhost:3306)/test, got %s", result.DSN)
	}

	os.RemoveAll("/home/work/config")
}

func TestFetchConfigsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.Endpoint = server.URL
	cfg.Env = "prod"
	cfg.ServiceName = "auth-server"
	cfg.Token = "test-token"

	c := New(cfg)
	if err := c.FetchConfigs(); err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestFetchConfigsEmptyEnv(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ServiceName = "auth-server"
	c := New(cfg)
	if err := c.FetchConfigs(); err == nil {
		t.Fatal("expected error for empty env")
	}
}

func TestLoadFile(t *testing.T) {
	configDir := filepath.Join("/home/work/config", "auth-server")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Skipf("skipping: cannot create %s: %v", configDir, err)
	}
	defer os.RemoveAll("/home/work/config")

	content := `{"dsn":"root:@tcp(localhost:3306)/test"}`
	if err := os.WriteFile(filepath.Join(configDir, "mysql.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var result struct {
		DSN string `json:"dsn"`
	}
	if err := LoadFile("auth-server", "mysql.json", &result); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if result.DSN != "root:@tcp(localhost:3306)/test" {
		t.Fatalf("expected DSN=root:@tcp(localhost:3306)/test, got %s", result.DSN)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	var result struct{}
	if err := LoadFile("non-existent", "no-such-file.json", &result); err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

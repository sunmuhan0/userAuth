package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ttuser/auth-server/internal/dao"
	"ttuser/auth-server/pkg/token"
	"ttuser/data-store/engine"
)

// mockMysqlClient 模拟 IMysqlClient（最小实现，仅供 DAO 测试使用）
// 实际上我们直接构造 DAO 并注入内存数据来测试 service 层

// 构造测试用的 AuthServiceImpl
func newTestService() *AuthServiceImpl {
	// 用真实的 JWTManager（不依赖外部）
	tokenMgr := &token.JWTManager{
		Secret:            "test-secret",
		AccessExpireTime:  2 * time.Hour,
		RefreshExpireTime: 7 * 24 * time.Hour,
	}

	// 使用内存 mock DAO
	userDAO := &dao.UserDAO{Mysql: newMockMysql()}
	tokenDAO := &dao.TokenDAO{Mysql: newMockMysql()}

	return &AuthServiceImpl{
		UserDAO:  userDAO,
		TokenDAO: tokenDAO,
		TokenMgr: tokenMgr,
	}
}

func TestRegisterAndLogin(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	// 注册
	user, err := svc.Register(ctx, "testuser", "password123", "Test", "test@test.com")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if user.Username != "testuser" {
		t.Fatalf("expected username=testuser, got %s", user.Username)
	}

	// 登录
	accessToken, refreshToken, expiresAt, loginUser, err := svc.Login(ctx, "testuser", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if accessToken == "" || refreshToken == "" {
		t.Fatal("tokens should not be empty")
	}
	if expiresAt <= time.Now().Unix() {
		t.Fatal("expiresAt should be in the future")
	}
	if loginUser.Username != "testuser" {
		t.Fatalf("expected username=testuser, got %s", loginUser.Username)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	// 先注册
	_, _ = svc.Register(ctx, "testuser", "password123", "", "")

	// 错误密码登录
	_, _, _, _, err := svc.Login(ctx, "testuser", "wrongpass")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginNonexistentUser(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _, _, _, err := svc.Login(ctx, "nouser", "pass")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _ = svc.Register(ctx, "dup", "pass", "", "")
	_, err := svc.Register(ctx, "dup", "pass", "", "")
	if err != ErrUserAlreadyExists {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestValidateToken(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _ = svc.Register(ctx, "user1", "pass", "", "")
	accessToken, _, _, _, _ := svc.Login(ctx, "user1", "pass")

	claims, err := svc.ValidateToken(ctx, accessToken)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.Username != "user1" {
		t.Fatalf("expected username=user1, got %s", claims.Username)
	}
}

func TestLogoutInvalidatesToken(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _ = svc.Register(ctx, "user1", "pass", "", "")
	accessToken, refreshToken, _, _, _ := svc.Login(ctx, "user1", "pass")

	// 注销
	err := svc.Logout(ctx, accessToken, refreshToken)
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	// access token 应该无效
	_, err = svc.ValidateToken(ctx, accessToken)
	if err != ErrTokenBlacklisted {
		t.Fatalf("expected ErrTokenBlacklisted, got %v", err)
	}
}

func TestRefreshToken(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _ = svc.Register(ctx, "user1", "pass", "", "")
	_, refreshToken, _, _, _ := svc.Login(ctx, "user1", "pass")

	// 续签
	newAccess, newRefresh, expiresAt, err := svc.RefreshToken(ctx, refreshToken)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Fatal("new tokens should not be empty")
	}
	if expiresAt <= time.Now().Unix() {
		t.Fatal("expiresAt should be in the future")
	}

	// 旧 refresh token 应该失效（轮转）
	_, _, _, err = svc.RefreshToken(ctx, refreshToken)
	if err != ErrTokenBlacklisted {
		t.Fatalf("expected ErrTokenBlacklisted for old refresh token, got %v", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	user, _ := svc.Register(ctx, "info_user", "pass", "Nick", "nick@test.com")

	got, err := svc.GetUserInfo(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}
	if got.Nickname != "Nick" {
		t.Fatalf("expected nickname=Nick, got %s", got.Nickname)
	}
}

func TestUpdateUserInfo(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	user, _ := svc.Register(ctx, "upd_user", "pass", "Old", "old@test.com")

	updated, err := svc.UpdateUserInfo(ctx, user.ID, "New", "new@test.com", "")
	if err != nil {
		t.Fatalf("UpdateUserInfo failed: %v", err)
	}
	if updated.Nickname != "New" {
		t.Fatalf("expected nickname=New, got %s", updated.Nickname)
	}
	if updated.Email != "new@test.com" {
		t.Fatalf("expected email=new@test.com, got %s", updated.Email)
	}
}

// ============ 内存 Mock IMysqlClient ============

// mockMysql 简单的内存 mock，实现 IMysqlClient 中 service 测试需要的方法
type mockMysql struct {
	users           map[string]*dao.UserRecord // key: id
	usersByUsername map[string]string          // username -> id
	tokenBlacklist  map[string]bool
}

func newMockMysql() *mockMysql {
	return &mockMysql{
		users:           make(map[string]*dao.UserRecord),
		usersByUsername: make(map[string]string),
		tokenBlacklist:  make(map[string]bool),
	}
}

func (m *mockMysql) Start() error { return nil }
func (m *mockMysql) Close()       {}

func (m *mockMysql) Add(tableName string, d interface{}, ondupUpdate bool) error {
	switch tableName {
	case "users":
		r := d.(*dao.UserRecord)
		if _, exists := m.usersByUsername[r.Username]; exists {
			return fmt.Errorf("Error 1062: Duplicate entry")
		}
		m.users[r.ID] = r
		m.usersByUsername[r.Username] = r.ID
	}
	return nil
}

func (m *mockMysql) Query(dataType interface{}, query string, args ...interface{}) (interface{}, error) {
	// 简单解析：根据 args[0] 查找
	if len(args) == 0 {
		return nil, nil
	}
	key := fmt.Sprintf("%v", args[0])

	// 判断是按 id 查还是按 username 查
	for _, u := range m.users {
		if u.ID == key || u.Username == key {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockMysql) GetCount(query string, args ...interface{}) (int64, error) {
	if len(args) == 0 {
		return 0, nil
	}
	key := fmt.Sprintf("%v", args[0])
	if m.tokenBlacklist[key] {
		return 1, nil
	}
	return 0, nil
}

func (m *mockMysql) ExecContext(ctx context.Context, sqlStr string, args ...interface{}) (int64, error) {
	// token blacklist insert
	if len(args) >= 1 {
		key := fmt.Sprintf("%v", args[0])
		m.tokenBlacklist[key] = true
	}
	return 1, nil
}

func (m *mockMysql) Update(tableName string, d interface{}, primaryKeys map[string]interface{}, fieldsToUpdate []string) error {
	if tableName == "users" {
		r := d.(*dao.UserRecord)
		id := ""
		for _, v := range primaryKeys {
			id = fmt.Sprintf("%v", v)
		}
		if existing, ok := m.users[id]; ok {
			for _, f := range fieldsToUpdate {
				switch f {
				case "nickname":
					existing.Nickname = r.Nickname
				case "email":
					existing.Email = r.Email
				case "avatar":
					existing.Avatar = r.Avatar
				}
			}
		}
	}
	return nil
}

// 以下方法 service 测试中不会用到，空实现
func (m *mockMysql) QueryList(dataType interface{}, query string, args ...interface{}) ([]interface{}, error) {
	return nil, nil
}
func (m *mockMysql) Delete(tableName string, condition map[string]interface{}) (int64, error) {
	return 0, nil
}
func (m *mockMysql) Execute(sqlStr string, args ...interface{}) (int64, error) { return 0, nil }
func (m *mockMysql) ExecTransaction(transactionExec engine.TransactionExec) (int64, error) {
	return 0, nil
}
func (m *mockMysql) InsertOrUpdateOnDup(tableName string, d interface{}, updateFields []string, ignoreFields ...string) (int64, error) {
	return 0, nil
}
func (m *mockMysql) AddAndRetLastId(tableName string, d interface{}, ignoreFields ...string) (int64, error) {
	return 0, nil
}

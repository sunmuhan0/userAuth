package dao

import (
	"context"
	"fmt"
	"testing"

	"ttuser/data-store/engine"
)

// mockDAOClient 模拟IMysqlClient用于DAO测试
type mockDAOClient struct {
	users           map[string]*UserRecord
	usersByUsername map[string]string
	tokenBlacklist  map[string]bool
}

func newMockDAOClient() *mockDAOClient {
	return &mockDAOClient{
		users:           make(map[string]*UserRecord),
		usersByUsername: make(map[string]string),
		tokenBlacklist:  make(map[string]bool),
	}
}

func (m *mockDAOClient) Start() error { return nil }
func (m *mockDAOClient) Close()       {}

func (m *mockDAOClient) Add(tableName string, d interface{}, ondupUpdate bool) error {
	switch tableName {
	case "users":
		r := d.(*UserRecord)
		if _, exists := m.usersByUsername[r.Username]; exists {
			return fmt.Errorf("Error 1062: Duplicate entry '%s' for key 'username'", r.Username)
		}
		if r.ID == "" {
			r.ID = fmt.Sprintf("mock-id-%s", r.Username)
		}
		m.users[r.ID] = r
		m.usersByUsername[r.Username] = r.ID
	}
	return nil
}

func (m *mockDAOClient) Query(dataType interface{}, query string, args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, nil
	}
	key := fmt.Sprintf("%v", args[0])
	for _, u := range m.users {
		if u.ID == key || u.Username == key {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockDAOClient) Update(tableName string, d interface{}, primaryKeys map[string]interface{}, fieldsToUpdate []string) error {
	if tableName == "users" {
		r := d.(*UserRecord)
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

func (m *mockDAOClient) Delete(tableName string, condition map[string]interface{}) (int64, error) {
	if tableName == "users" {
		id := ""
		for _, v := range condition {
			id = fmt.Sprintf("%v", v)
		}
		if _, ok := m.users[id]; ok {
			delete(m.users, id)
			// Also clean up usersByUsername
			for username, uid := range m.usersByUsername {
				if uid == id {
					delete(m.usersByUsername, username)
					break
				}
			}
			return 1, nil
		}
		return 0, nil
	}
	return 0, nil
}

func (m *mockDAOClient) GetCount(query string, args ...interface{}) (int64, error) {
	if len(args) == 0 {
		return 0, nil
	}
	key := fmt.Sprintf("%v", args[0])
	if m.tokenBlacklist[key] {
		return 1, nil
	}
	return 0, nil
}

func (m *mockDAOClient) ExecContext(ctx context.Context, sqlStr string, args ...interface{}) (int64, error) {
	if len(args) >= 1 {
		key := fmt.Sprintf("%v", args[0])
		m.tokenBlacklist[key] = true
	}
	return 1, nil
}

// Unused methods
func (m *mockDAOClient) QueryList(dataType interface{}, query string, args ...interface{}) ([]interface{}, error) {
	return nil, nil
}
func (m *mockDAOClient) Execute(sqlStr string, args ...interface{}) (int64, error) { return 0, nil }
func (m *mockDAOClient) ExecTransaction(transactionExec engine.TransactionExec) (int64, error) {
	return 0, nil
}
func (m *mockDAOClient) InsertOrUpdateOnDup(tableName string, d interface{}, updateFields []string, ignoreFields ...string) (int64, error) {
	return 0, nil
}
func (m *mockDAOClient) AddAndRetLastId(tableName string, d interface{}, ignoreFields ...string) (int64, error) {
	return 0, nil
}

// ============ UserDAO Tests ============

func TestUserDAOCreate(t *testing.T) {
	mock := newMockDAOClient()
	dao := &UserDAO{Mysql: mock}

	user, err := dao.Create(context.Background(), "testuser", "hashedpass", "Test", "test@test.com")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if user.Username != "testuser" {
		t.Fatalf("expected username=testuser, got %s", user.Username)
	}
	if user.Password != "hashedpass" {
		t.Fatalf("expected password=hashedpass, got %s", user.Password)
	}
}

func TestUserDAOCreateDuplicate(t *testing.T) {
	mock := newMockDAOClient()
	dao := &UserDAO{Mysql: mock}

	_, err := dao.Create(context.Background(), "dupuser", "pass1", "", "")
	if err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	_, err = dao.Create(context.Background(), "dupuser", "pass2", "", "")
	if err != ErrUserAlreadyExists {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestUserDAOGetByID(t *testing.T) {
	mock := newMockDAOClient()
	dao := &UserDAO{Mysql: mock}

	created, _ := dao.Create(context.Background(), "getbyid", "pass", "Nick", "nick@test.com")
	found, err := dao.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if found.Username != "getbyid" {
		t.Fatalf("expected username=getbyid, got %s", found.Username)
	}
}

func TestUserDAOGetByIDNotFound(t *testing.T) {
	mock := newMockDAOClient()
	dao := &UserDAO{Mysql: mock}

	_, err := dao.GetByID(context.Background(), "nonexistent-id")
	if err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserDAOGetByUsername(t *testing.T) {
	mock := newMockDAOClient()
	dao := &UserDAO{Mysql: mock}

	dao.Create(context.Background(), "findme", "pass", "Find", "find@test.com")
	found, err := dao.GetByUsername(context.Background(), "findme")
	if err != nil {
		t.Fatalf("GetByUsername failed: %v", err)
	}
	if found.Nickname != "Find" {
		t.Fatalf("expected nickname=Find, got %s", found.Nickname)
	}
}

func TestUserDAOGetByUsernameNotFound(t *testing.T) {
	mock := newMockDAOClient()
	dao := &UserDAO{Mysql: mock}

	_, err := dao.GetByUsername(context.Background(), "nobody")
	if err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserDAOUpdate(t *testing.T) {
	mock := newMockDAOClient()
	dao := &UserDAO{Mysql: mock}

	created, _ := dao.Create(context.Background(), "updateuser", "pass", "Old", "old@test.com")

	updated, err := dao.Update(context.Background(), created.ID, "New", "new@test.com", "avatar-url")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Nickname != "New" {
		t.Fatalf("expected nickname=New, got %s", updated.Nickname)
	}
	if updated.Email != "new@test.com" {
		t.Fatalf("expected email=new@test.com, got %s", updated.Email)
	}
}

func TestUserDAODelete(t *testing.T) {
	mock := newMockDAOClient()
	dao := &UserDAO{Mysql: mock}

	created, _ := dao.Create(context.Background(), "deleteuser", "pass", "", "")
	if err := dao.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := dao.GetByID(context.Background(), created.ID)
	if err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound after delete, got %v", err)
	}
}

func TestUserDAODeleteNotFound(t *testing.T) {
	mock := newMockDAOClient()
	dao := &UserDAO{Mysql: mock}

	err := dao.Delete(context.Background(), "nonexistent")
	if err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

// ============ TokenDAO Tests ============

func TestTokenDAOAddAndExists(t *testing.T) {
	mock := newMockDAOClient()
	dao := &TokenDAO{Mysql: mock}

	ctx := context.Background()
	token := "test-jwt-token"

	// Initially should not exist
	exists, err := dao.Exists(ctx, token)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Fatal("expected token to not exist initially")
	}

	// Add to blacklist
	if err := dao.Add(ctx, token, 9999999999); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Should exist now
	exists, err = dao.Exists(ctx, token)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Fatal("expected token to exist after adding")
	}
}

func TestTokenDAOAddDuplicate(t *testing.T) {
	mock := newMockDAOClient()
	dao := &TokenDAO{Mysql: mock}

	ctx := context.Background()
	token := "duplicate-token"

	// First add should succeed
	if err := dao.Add(ctx, token, 9999999999); err != nil {
		t.Fatalf("first Add failed: %v", err)
	}

	// Second add (duplicate) should also succeed (INSERT IGNORE)
	if err := dao.Add(ctx, token, 9999999999); err != nil {
		t.Fatalf("second Add (duplicate) failed: %v", err)
	}
}

func TestTokenDAOHashConsistency(t *testing.T) {
	token := "consistent-token"
	hash1 := hashToken(token)
	hash2 := hashToken(token)

	if hash1 != hash2 {
		t.Fatal("hashToken should be deterministic")
	}
	if hash1 == "" {
		t.Fatal("hash should not be empty")
	}
}

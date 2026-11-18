package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/database"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type failingUserRepo struct {
	UserRepository
}

func (f *failingUserRepo) Create(user *database.User) error {
	return errors.New("db create error")
}

func (f *failingUserRepo) Update(user *database.User) error {
	return errors.New("db update error")
}

func TestAuthRepositoryServiceAndHandler(t *testing.T) {
	db, err := database.ConnectTest()
	if err != nil {
		t.Fatal(err)
	}

	repo := NewUserRepository(db)
	svc := NewAuthService(repo, "test-jwt-secret")
	handler := NewAuthHandler(svc)

	// 1. Service Register & duplicate checks
	res, err := svc.Register(RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password123",
		Role:     "admin",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if res.User.Role != "member" || res.Token == "" {
		t.Fatalf("unexpected register response: %+v", res)
	}

	if _, err := svc.Register(RegisterRequest{Username: "alice", Email: "other@example.com", Password: "password123"}); err == nil {
		t.Fatal("expected duplicate username error")
	}
	if _, err := svc.Register(RegisterRequest{Username: "bob", Email: "alice@example.com", Password: "password123"}); err == nil {
		t.Fatal("expected duplicate email error")
	}
	// Password > 72 bytes error
	if _, err := svc.Register(RegisterRequest{Username: "longp", Email: "longp@example.com", Password: strings.Repeat("a", 80)}); err == nil {
		t.Fatal("expected bcrypt password too long error")
	}
	// Repo Create error
	failSvc := NewAuthService(&failingUserRepo{repo}, "secret")
	if _, err := failSvc.Register(RegisterRequest{Username: "newu", Email: "newu@example.com", Password: "password123"}); err == nil {
		t.Fatal("expected repo create error")
	}

	// 2. Service Login
	loginRes, err := svc.Login(LoginRequest{Username: "alice", Password: "password123"})
	if err != nil || loginRes.Token == "" {
		t.Fatalf("login failed: %v", err)
	}
	if _, err := svc.Login(LoginRequest{Username: "alice", Password: "wrongpassword"}); err == nil {
		t.Fatal("expected invalid password error")
	}
	if _, err := svc.Login(LoginRequest{Username: "nonexistent", Password: "password123"}); err == nil {
		t.Fatal("expected non-existent user error")
	}

	// 3. Service GetProfile & UpdateProfile
	prof, err := svc.GetProfile(res.User.ID)
	if err != nil || prof.Username != "alice" {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if _, err := svc.GetProfile(9999); err == nil {
		t.Fatal("expected error for non-existent user ID")
	}

	updatedProf, err := svc.UpdateProfile(res.User.ID, UpdateProfileRequest{
		Email:    "alice_new@example.com",
		Password: "newpassword123",
	})
	if err != nil || updatedProf.Email != "alice_new@example.com" {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if _, err := svc.UpdateProfile(res.User.ID, UpdateProfileRequest{Password: strings.Repeat("a", 80)}); err == nil {
		t.Fatal("expected bcrypt error on long password in UpdateProfile")
	}
	if _, err := failSvc.UpdateProfile(res.User.ID, UpdateProfileRequest{Email: "fail@example.com"}); err == nil {
		t.Fatal("expected repo update error")
	}
	if _, err := svc.UpdateProfile(9999, UpdateProfileRequest{Email: "x@example.com"}); err == nil {
		t.Fatal("expected error updating non-existent user")
	}

	allUsers, err := repo.FindAll()
	if err != nil || len(allUsers) != 1 {
		t.Fatalf("FindAll failed: %v, len=%d", err, len(allUsers))
	}

	// 4. HTTP Handler tests
	r := gin.New()
	r.POST("/register", handler.Register)
	r.POST("/login", handler.Login)
	r.POST("/logout", handler.Logout)
	r.GET("/profile-unauth", handler.GetProfile)
	r.PUT("/profile-unauth", handler.UpdateProfile)
	r.GET("/profile", func(c *gin.Context) {
		c.Set("userID", res.User.ID)
		handler.GetProfile(c)
	})
	r.GET("/profile-notfound", func(c *gin.Context) {
		c.Set("userID", uint(9999))
		handler.GetProfile(c)
	})
	r.PUT("/profile", func(c *gin.Context) {
		c.Set("userID", res.User.ID)
		handler.UpdateProfile(c)
	})
	r.PUT("/profile-notfound", func(c *gin.Context) {
		c.Set("userID", uint(9999))
		handler.UpdateProfile(c)
	})

	// Register HTTP
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("{bad")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	body, _ := json.Marshal(RegisterRequest{Username: "charlie", Email: "charlie@example.com", Password: "password123"})
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on duplicate register, got %d", w.Code)
	}

	// Login HTTP
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("{bad")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	loginBody, _ := json.Marshal(LoginRequest{Username: "charlie", Password: "password123"})
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	badLogin, _ := json.Marshal(LoginRequest{Username: "charlie", Password: "wrong"})
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(badLogin)))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	// Logout HTTP
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/logout", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on logout, got %d", w.Code)
	}

	// GetProfile HTTP
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/profile-unauth", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/profile", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/profile-notfound", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	// UpdateProfile HTTP
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/profile-unauth", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/profile", bytes.NewBufferString("{bad")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	updBody, _ := json.Marshal(UpdateProfileRequest{Email: "alice_updated@example.com"})
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/profile", bytes.NewReader(updBody)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/profile-notfound", bytes.NewReader(updBody)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

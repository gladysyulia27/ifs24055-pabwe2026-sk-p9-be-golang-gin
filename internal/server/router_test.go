package server

import (
	"app/internal/config"
	"app/internal/database"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetupRouter(t *testing.T) {
	db, err := database.ConnectTest()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		AppEnv:               "production",
		AppPort:              "8081",
		AppKey:               "secret-key",
		OpenAIEndpoint:       "https://example.com",
		OpenAIKey:            "key",
		OpenAIModel:          "gpt-test",
		SearchEngineEndpoint: "https://example.com/search",
	}

	r := SetupRouter(cfg, db)

	// 1. GET /swagger redirect
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/swagger", nil))
	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301 redirect on /swagger, got %d", w.Code)
	}

	// 2. GET /swagger/api.json with X-Forwarded headers
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swagger/api.json", nil)
	req.Header.Set("X-Forwarded-Host", "blog.delcom.org")
	req.Header.Set("X-Forwarded-Proto", "https")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /swagger/api.json, got %d", w.Code)
	}

	// 3. GET /swagger/index.html (ginSwagger handler)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /swagger/index.html, got %d", w.Code)
	}

	// 4. GET /api.json without forwarded headers
	w = httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "http://localhost:8081/api.json", nil)
	r.ServeHTTP(w, req2)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /api.json, got %d", w.Code)
	}

	// 5. GET /api/public/categories
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/categories", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /api/public/categories, got %d", w.Code)
	}

	// 6. NoRoute 404
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unknown-endpoint", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on unknown route, got %d", w.Code)
	}
}

package post

import (
	"app/internal/database"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestReadingTime(t *testing.T) {
	if ReadingTime("") != 1 {
		t.Fatal("empty article must have a one-minute minimum")
	}
	words := make([]string, 401)
	for i := range words {
		words[i] = "word"
	}
	if ReadingTime(join(words)) != 3 {
		t.Fatal("reading time must use 200 words per minute")
	}
}

func join(words []string) string {
	out := ""
	for i, w := range words {
		if i > 0 {
			out += " "
		}
		out += w
	}
	return out
}

func TestArticleRepositoryAndHandlers(t *testing.T) {
	db, err := database.ConnectTest()
	if err != nil {
		t.Fatal(err)
	}
	user := database.User{Username: "writer", Email: "writer@example.test", Password: "hashed", Role: "member"}
	otherUser := database.User{Username: "other", Email: "other@example.test", Password: "hashed", Role: "member"}
	if err = db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&otherUser).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	in := Input{Title: "Hello, Clean Architecture!", Category: "Teknologi", Excerpt: "Summary", Content: "# Article", IsPublished: true}
	p, err := repo.Create(user.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if p.Slug != "hello-clean-architecture" || !p.IsPublished || p.PublishedAt == nil {
		t.Fatalf("unexpected created article: %+v", p)
	}
	p2, err := repo.Create(user.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if p2.Slug == p.Slug {
		t.Fatal("slug must be unique")
	}
	// Test symbol-only title fallback to "artikel" and empty category fallback to "Umum"
	pSym, err := repo.Create(user.ID, Input{Title: "!!!", Category: "", Content: "Body"})
	if err != nil || pSym.Slug != "artikel" || pSym.Category != "Umum" {
		t.Fatalf("expected fallback slug 'artikel' and category 'Umum', got %+v (%v)", pSym, err)
	}

	mine, err := repo.ListMine(user.ID)
	if err != nil || len(mine) != 3 {
		t.Fatalf("list mine: %v, %d", err, len(mine))
	}
	got, err := repo.GetMine(user.ID, p.ID)
	if err != nil || got.ID != p.ID {
		t.Fatalf("get mine: %v", err)
	}
	// Forbidden when another user accesses
	if _, err := repo.GetMine(otherUser.ID, p.ID); err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	updated, err := repo.Update(user.ID, p.ID, UpdateInput{
		Title:          strPtr("Updated title"),
		Category:       strPtr(""),
		Excerpt:        strPtr("New excerpt"),
		Content:        strPtr("Updated body"),
		CoverImagePath: strPtr("/img.png"),
		IsPublished:    boolPtr(false),
	})
	if err != nil || updated.Category != "Umum" || updated.IsPublished {
		t.Fatalf("update failed: %+v, %v", updated, err)
	}
	// Re-publish via Update
	updated, err = repo.Update(user.ID, p.ID, UpdateInput{
		Category:    strPtr("Opini"),
		IsPublished: boolPtr(true),
	})
	if err != nil || !updated.IsPublished || updated.PublishedAt == nil {
		t.Fatalf("re-publish via update failed: %+v, %v", updated, err)
	}
	// Invalid UpdateInput (empty title)
	if _, err := repo.Update(user.ID, p.ID, UpdateInput{Title: strPtr("   ")}); err != ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	stats, err := repo.Stats(user.ID)
	if err != nil || stats.Total != 3 || stats.Published != 2 || stats.Draft != 1 || stats.TotalArticles != 3 || stats.TotalSelesai != 2 || stats.BelumSelesai != 1 {
		t.Fatalf("stats mismatch: %+v, %v", stats, err)
	}
	pub, err := repo.Public("", "Semua")
	if err != nil || len(pub) != 2 {
		t.Fatalf("public list mismatch: %v", err)
	}
	pubFiltered, err := repo.Public("Updated", "Opini")
	if err != nil || len(pubFiltered) != 1 {
		t.Fatalf("public filtered list error: %v, len=%d", err, len(pubFiltered))
	}

	pubDetail, err := repo.PublicBySlug(p2.Slug)
	if err != nil || pubDetail.ID != p2.ID {
		t.Fatalf("public detail mismatch: %v", err)
	}
	toggled, err := repo.Toggle(user.ID, p.ID)
	if err != nil || toggled.IsPublished {
		t.Fatalf("toggle to draft failed: %v", err)
	}
	toggled, err = repo.Toggle(user.ID, p.ID)
	if err != nil || !toggled.IsPublished {
		t.Fatalf("toggle to published failed: %v", err)
	}
	users, err := repo.AdminUsers()
	if err != nil || len(users) != 2 {
		t.Fatalf("admin users failed: %v", err)
	}
	all, err := repo.AdminPosts()
	if err != nil || len(all) != 3 {
		t.Fatalf("admin posts failed: %v", err)
	}
	byUser, err := repo.AdminUserPosts(user.ID)
	if err != nil || len(byUser) != 3 {
		t.Fatalf("admin user posts failed: %v", err)
	}
	if err = repo.Delete(user.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	if err = repo.Delete(user.ID, p.ID); err == nil {
		t.Fatal("deleting already deleted article should return error")
	}

	// HTTP Handler tests
	h := NewHandler(repo)
	currentUID := user.ID
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", currentUID)
		c.Next()
	})
	r.GET("/public/posts", h.PublicList)
	r.GET("/public/posts/:slug", h.PublicDetail)
	r.GET("/posts", h.List)
	r.GET("/posts/stats", h.Stats)
	r.POST("/posts", h.Create)
	r.GET("/posts/:id", h.Detail)
	r.PUT("/posts/:id", h.Update)
	r.PATCH("/posts/:id/toggle-publish", h.Toggle)
	r.DELETE("/posts/:id", h.Delete)
	r.GET("/admin/users", h.AdminUsers)
	r.GET("/admin/posts", h.AdminPosts)
	r.GET("/admin/users/:id/posts", h.AdminUserPosts)

	// PublicList & PublicDetail
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/public/posts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/public/posts/"+p2.Slug, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/public/posts/not-found", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	// List & Stats
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts/stats", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Create (bad JSON & valid)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString("{bad")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	createBody, _ := json.Marshal(Input{Title: "From Handler", Category: "Teknologi", Content: "Isi artikel", IsPublished: true})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(createBody)))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	idStr := strconv.Itoa(int(p2.ID))

	// Detail (invalid ID, not found, forbidden, valid)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts/abc", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts/9999", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	currentUID = otherUser.ID
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts/"+idStr, nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	currentUID = user.ID
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts/"+idStr, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Update (invalid ID, bad JSON, empty body 400, not found 404, forbidden 403, valid 200)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/posts/abc", bytes.NewReader(createBody)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/posts/"+idStr, bytes.NewBufferString("{bad")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/posts/"+idStr, bytes.NewBufferString("{}")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty update body, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/posts/9999", bytes.NewReader(createBody)))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	currentUID = otherUser.ID
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/posts/"+idStr, bytes.NewReader(createBody)))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	currentUID = user.ID
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/posts/"+idStr, bytes.NewReader(createBody)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Toggle (invalid ID, not found, valid)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/posts/abc/toggle-publish", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/posts/9999/toggle-publish", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/posts/"+idStr+"/toggle-publish", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Admin endpoints
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/posts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users/abc/posts", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users/1/posts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Delete (invalid ID, not found, valid)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/posts/abc", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/posts/9999", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/posts/"+idStr, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	queryCount := 0
	_ = db.Callback().Query().Before("gorm:query").Register("test:fail_on_count", func(tx *gorm.DB) {
		queryCount++
		if queryCount >= 3 {
			_ = tx.AddError(gorm.ErrInvalidDB)
		}
	})
	_, _ = repo.Update(user.ID, pSym.ID, UpdateInput{Title: strPtr("Trigger Count Fail")})
	_ = db.Callback().Query().Remove("test:fail_on_count")

	sqlDB, _ := db.DB()
	_ = sqlDB.Close()

	// All DB error paths on closed DB:
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/public/posts", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts/stats", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(createBody)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/posts", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users/1/posts", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

package post

import (
	"app/internal/database"
	"testing"
)

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
func TestArticleRepositoryLifecycle(t *testing.T) {
	db, err := database.ConnectTest()
	if err != nil {
		t.Fatal(err)
	}
	user := database.User{Username: "writer", Email: "writer@example.test", Password: "hashed", Role: "member"}
	if err = db.Create(&user).Error; err != nil {
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
	mine, err := repo.ListMine(user.ID)
	if err != nil || len(mine) != 2 {
		t.Fatalf("list mine: %v, %d", err, len(mine))
	}
	got, err := repo.GetMine(user.ID, p.ID)
	if err != nil || got.ID != p.ID {
		t.Fatalf("get mine: %v", err)
	}
	updated, err := repo.Update(user.ID, p.ID, Input{Title: "Updated title", Category: "Opini", Content: "Updated body"})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.IsPublished || updated.PublishedAt == nil {
		t.Fatal("content update must preserve existing publication state")
	}
	stats, err := repo.Stats(user.ID)
	if err != nil || stats.Total != 2 || stats.Published != 2 || stats.Draft != 0 {
		t.Fatalf("stats mismatch: %+v, %v", stats, err)
	}
	pub, err := repo.Public("", "Semua")
	if err != nil || len(pub) != 2 {
		t.Fatalf("public list mismatch: %v", err)
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
	if err != nil || len(users) != 1 {
		t.Fatalf("admin users failed: %v", err)
	}
	all, err := repo.AdminPosts()
	if err != nil || len(all) != 2 {
		t.Fatalf("admin posts failed: %v", err)
	}
	byUser, err := repo.AdminUserPosts(user.ID)
	if err != nil || len(byUser) != 2 {
		t.Fatalf("admin user posts failed: %v", err)
	}
	if err = repo.Delete(user.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetMine(user.ID, p.ID); err == nil {
		t.Fatal("deleted article should not be found")
	}
}

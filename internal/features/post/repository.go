package post

import (
	"app/internal/database"
	"errors"
	"gorm.io/gorm"
)

var (
	ErrForbidden    = errors.New("forbidden: article belongs to another user")
	ErrInvalidInput = errors.New("invalid article input")
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db} }

func toPublic(p database.Post) PublicPost {
	out := PublicPost{
		ID:             p.ID,
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		IsPublished:    p.IsPublished,
		PublishedAt:    p.PublishedAt,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
		ReadingTime:    ReadingTime(p.Content),
	}
	if p.User != nil {
		out.Author = &PublicAuthor{
			ID:        p.User.ID,
			Username:  p.User.Username,
			Email:     p.User.Email,
			Role:      p.User.Role,
			CreatedAt: p.User.CreatedAt,
			UpdatedAt: p.User.UpdatedAt,
		}
	}
	return out
}

func (r *Repository) ListMine(uid uint) ([]database.Post, error) {
	var p []database.Post
	e := r.db.Where("user_id = ?", uid).Order("updated_at desc").Find(&p).Error
	return p, e
}

func (r *Repository) GetMine(uid, id uint) (database.Post, error) {
	var p database.Post
	if e := r.db.Preload("User").First(&p, id).Error; e != nil {
		return p, e
	}
	if p.UserID != uid {
		return p, ErrForbidden
	}
	return p, nil
}

func (r *Repository) Delete(uid, id uint) error {
	p, e := r.GetMine(uid, id)
	if e != nil {
		return e
	}
	return r.db.Delete(&p).Error
}

func (r *Repository) Public(q, cat string) ([]PublicPost, error) {
	var p []database.Post
	db := r.db.Preload("User").Where("is_published = true")
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("LOWER(title) LIKE LOWER(?) OR LOWER(excerpt) LIKE LOWER(?) OR LOWER(content) LIKE LOWER(?)", like, like, like)
	}
	if cat != "" && cat != "Semua" {
		db = db.Where("category = ?", cat)
	}
	e := db.Order("published_at desc").Find(&p).Error
	out := make([]PublicPost, 0, len(p))
	for _, item := range p {
		out = append(out, toPublic(item))
	}
	return out, e
}

func (r *Repository) PublicBySlug(slug string) (PublicPost, error) {
	var p database.Post
	e := r.db.Preload("User").Where("slug = ? AND is_published = true", slug).First(&p).Error
	return toPublic(p), e
}

func (r *Repository) AdminUsers() ([]map[string]interface{}, error) {
	var users []database.User
	e := r.db.Order("id").Find(&users).Error
	if e != nil {
		return nil, e
	}
	out := []map[string]interface{}{}
	for _, u := range users {
		var total, pub int64
		r.db.Model(&database.Post{}).Where("user_id = ?", u.ID).Count(&total)
		r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = true", u.ID).Count(&pub)
		out = append(out, map[string]interface{}{"id": u.ID, "username": u.Username, "email": u.Email, "role": u.Role, "total_posts": total, "published_posts": pub, "created_at": u.CreatedAt})
	}
	return out, nil
}

func (r *Repository) AdminPosts() ([]database.Post, error) {
	var p []database.Post
	e := r.db.Preload("User").Order("created_at desc").Find(&p).Error
	return p, e
}

func (r *Repository) AdminUserPosts(uid uint) ([]database.Post, error) {
	var p []database.Post
	e := r.db.Preload("User").Where("user_id = ?", uid).Order("created_at desc").Find(&p).Error
	return p, e
}

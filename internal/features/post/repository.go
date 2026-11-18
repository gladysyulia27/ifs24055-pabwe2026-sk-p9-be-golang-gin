package post

import (
	"app/internal/database"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db} }
func (r *Repository) ListMine(uid uint) ([]database.Post, error) {
	var p []database.Post
	e := r.db.Where("user_id = ?", uid).Order("updated_at desc").Find(&p).Error
	return p, e
}
func (r *Repository) GetMine(uid, id uint) (database.Post, error) {
	var p database.Post
	e := r.db.Preload("User").Where("user_id = ? AND id = ?", uid, id).First(&p).Error
	return p, e
}
func (r *Repository) Delete(uid, id uint) error {
	res := r.db.Where("user_id = ? AND id = ?", uid, id).Delete(&database.Post{})
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return res.Error
}
func (r *Repository) Public(q, cat string) ([]database.Post, error) {
	var p []database.Post
	db := r.db.Preload("User").Where("is_published = true")
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("title ILIKE ? OR excerpt ILIKE ? OR content ILIKE ?", like, like, like)
	}
	if cat != "" && cat != "Semua" {
		db = db.Where("category = ?", cat)
	}
	e := db.Order("published_at desc").Find(&p).Error
	return p, e
}
func (r *Repository) PublicBySlug(slug string) (database.Post, error) {
	var p database.Post
	e := r.db.Preload("User").Where("slug = ? AND is_published = true", slug).First(&p).Error
	return p, e
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

package post

import (
	"app/internal/database"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	dash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func ReadingTime(s string) int {
	n := len(strings.Fields(regexp.MustCompile("[#>*_`~\\[\\]()]").ReplaceAllString(s, " ")))
	v := (n + 199) / 200
	if v < 1 {
		return 1
	}
	return v
}

func (r *Repository) uniqueSlug(title string, exclude uint) (string, error) {
	base := slugify(title)
	if base == "" {
		base = "artikel"
	}
	slug := base
	for i := 1; ; i++ {
		var n int64
		q := r.db.Model(&database.Post{}).Where("slug = ?", slug)
		if exclude > 0 {
			q = q.Where("id <> ?", exclude)
		}
		if e := q.Count(&n).Error; e != nil {
			return "", e
		}
		if n == 0 {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i+1)
	}
}

func (r *Repository) Create(uid uint, in Input) (database.Post, error) {
	slug, e := r.uniqueSlug(in.Title, 0)
	if e != nil {
		return database.Post{}, e
	}
	cat := strings.TrimSpace(in.Category)
	if cat == "" {
		cat = "Umum"
	}
	p := database.Post{UserID: uid, Title: in.Title, Slug: slug, Category: cat, Excerpt: in.Excerpt, Content: in.Content, CoverImagePath: in.CoverImagePath, IsPublished: in.IsPublished}
	if p.IsPublished {
		now := time.Now()
		p.PublishedAt = &now
	}
	e = r.db.Create(&p).Error
	return p, e
}

func (r *Repository) Update(uid, id uint, in UpdateInput) (database.Post, error) {
	p, e := r.GetMine(uid, id)
	if e != nil {
		return p, e
	}
	if (in.Title == nil && in.Category == nil && in.Excerpt == nil && in.Content == nil && in.CoverImagePath == nil && in.IsPublished == nil) ||
		(in.Title != nil && strings.TrimSpace(*in.Title) == "") ||
		(in.Content != nil && strings.TrimSpace(*in.Content) == "") {
		return p, ErrInvalidInput
	}
	if in.Title != nil {
		slug, err := r.uniqueSlug(*in.Title, id)
		if err != nil {
			return p, err
		}
		p.Title = *in.Title
		p.Slug = slug
	}
	if in.Category != nil {
		cat := strings.TrimSpace(*in.Category)
		if cat == "" {
			cat = "Umum"
		}
		p.Category = cat
	}
	if in.Excerpt != nil {
		p.Excerpt = *in.Excerpt
	}
	if in.Content != nil {
		p.Content = *in.Content
	}
	if in.CoverImagePath != nil {
		p.CoverImagePath = *in.CoverImagePath
	}
	if in.IsPublished != nil {
		p.IsPublished = *in.IsPublished
		if p.IsPublished && p.PublishedAt == nil {
			now := time.Now()
			p.PublishedAt = &now
		} else if !p.IsPublished {
			p.PublishedAt = nil
		}
	}
	e = r.db.Save(&p).Error
	return p, e
}

func (r *Repository) Toggle(uid, id uint) (database.Post, error) {
	p, e := r.GetMine(uid, id)
	if e != nil {
		return p, e
	}
	p.IsPublished = !p.IsPublished
	if p.IsPublished {
		now := time.Now()
		p.PublishedAt = &now
	} else {
		p.PublishedAt = nil
	}
	e = r.db.Save(&p).Error
	return p, e
}

func (r *Repository) Stats(uid uint) (Stats, error) {
	var s Stats
	e := r.db.Model(&database.Post{}).Where("user_id = ?", uid).Count(&s.Total).Error
	if e == nil {
		e = r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = true", uid).Count(&s.Published).Error
	}
	s.Draft = s.Total - s.Published
	s.TotalArticles = s.Total
	s.TotalSelesai = s.Published
	s.BelumSelesai = s.Draft
	return s, e
}

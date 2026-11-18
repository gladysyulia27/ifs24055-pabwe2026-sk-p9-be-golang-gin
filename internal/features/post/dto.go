package post

import "time"

type Input struct {
	Title          string `json:"title" binding:"required,max=255"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content" binding:"required"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    bool   `json:"is_published"`
}

type UpdateInput struct {
	Title          *string `json:"title"`
	Category       *string `json:"category"`
	Excerpt        *string `json:"excerpt"`
	Content        *string `json:"content"`
	CoverImagePath *string `json:"cover_image_path"`
	IsPublished    *bool   `json:"is_published"`
}

type Stats struct {
	Total         int64 `json:"total"`
	Published     int64 `json:"published"`
	Draft         int64 `json:"draft"`
	TotalArticles int64 `json:"total_articles"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
}

type PublicAuthor struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PublicPost struct {
	ID             uint          `json:"id"`
	Title          string        `json:"title"`
	Slug           string        `json:"slug"`
	Category       string        `json:"category"`
	Excerpt        string        `json:"excerpt"`
	Content        string        `json:"content"`
	CoverImagePath string        `json:"cover_image_path"`
	IsPublished    bool          `json:"is_published"`
	PublishedAt    *time.Time    `json:"published_at"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	ReadingTime    int           `json:"reading_time"`
	Author         *PublicAuthor `json:"author,omitempty"`
}

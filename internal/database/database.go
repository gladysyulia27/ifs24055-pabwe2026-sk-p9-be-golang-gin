package database

import (
	"app/internal/config"
	"fmt"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;not null" json:"username"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	Role      string    `gorm:"not null;default:member" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type Post struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	Title          string     `gorm:"size:255;not null" json:"title"`
	Slug           string     `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Category       string     `gorm:"size:100;not null" json:"category"`
	Excerpt        string     `gorm:"type:text" json:"excerpt"`
	Content        string     `gorm:"type:text" json:"content"`
	CoverImagePath string     `gorm:"type:text" json:"cover_image_path"`
	IsPublished    bool       `gorm:"default:false;index" json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	User           *User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"author,omitempty"`
}

func Connect(cfg *config.Config) (*gorm.DB, error) {
	var d gorm.Dialector
	if cfg.DBConnection == "sqlite" {
		d = sqlite.Open(cfg.DBDatabase)
	} else {
		d = postgres.Open(cfg.DSN())
	}
	db, e := gorm.Open(d, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		return nil, fmt.Errorf("database connection: %w", e)
	}
	return db, nil
}
func ConnectTest() (*gorm.DB, error) {
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e == nil {
		e = Migrate(db)
	}
	return db, e
}
func Migrate(db *gorm.DB) error { return db.AutoMigrate(&User{}, &Post{}) }
func Seed(db *gorm.DB, cfg *config.Config) error {
	var u User
	if e := db.Where("username = ?", cfg.AdminUsername).First(&u).Error; e == nil {
		return nil
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if e != nil {
		return e
	}
	return db.Create(&User{Username: cfg.AdminUsername, Email: cfg.AdminUsername + "@delcom.org", Password: string(hash), Role: "admin"}).Error
}

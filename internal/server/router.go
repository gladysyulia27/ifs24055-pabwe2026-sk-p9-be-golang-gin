package server

import (
	"app/docs"
	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
	"net/http"
	"time"
)

func SetupRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS(), middleware.RateLimiter(120, time.Minute))
	authH := auth.NewAuthHandler(auth.NewAuthService(auth.NewUserRepository(db), cfg.AppKey))
	posts := post.NewHandler(post.NewRepository(db))
	blogAI := ai.NewBlogHandler(cfg.OpenAIEndpoint, cfg.OpenAIKey, cfg.OpenAIModel, cfg.SearchEngineEndpoint)
	getSwagger := func(c *gin.Context) []byte {
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = c.Request.Host
		}
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		return docs.GetSwaggerJSON(host, scheme)
	}
	r.GET("/swagger", func(c *gin.Context) { c.Redirect(301, "/swagger/index.html") })
	sw := ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/api.json"))
	r.GET("/swagger/*any", func(c *gin.Context) {
		p := c.Param("any")
		if p == "/api.json" || p == "/doc.json" {
			c.Data(200, "application/json; charset=utf-8", getSwagger(c))
			return
		}
		sw(c)
	})
	r.GET("/api.json", func(c *gin.Context) { c.Data(200, "application/json; charset=utf-8", getSwagger(c)) })
	api := r.Group("/api")
	api.GET("/public/categories", func(c *gin.Context) {
		response.Success(c, 200, "Categories retrieved", []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"})
	})
	api.GET("/public/posts", posts.PublicList)
	api.GET("/public/posts/:slug", posts.PublicDetail)
	a := api.Group("/auth")
	a.POST("/register", authH.Register)
	a.POST("/login", authH.Login)
	a.POST("/logout", authH.Logout)
	ap := a.Group("")
	ap.Use(middleware.Auth(cfg.AppKey))
	ap.GET("/profile", authH.GetProfile)
	ap.PUT("/profile", authH.UpdateProfile)
	p := api.Group("")
	p.Use(middleware.Auth(cfg.AppKey))
	p.GET("/posts", posts.List)
	p.GET("/posts/stats", posts.Stats)
	p.POST("/posts", posts.Create)
	p.GET("/posts/:id", posts.Detail)
	p.PUT("/posts/:id", posts.Update)
	p.PATCH("/posts/:id/toggle-publish", posts.Toggle)
	p.DELETE("/posts/:id", posts.Delete)
	p.POST("/posts/ai-generate", blogAI.Generate)
	adm := p.Group("/admin")
	adm.Use(middleware.RequireRole("admin"))
	adm.GET("/users", posts.AdminUsers)
	adm.GET("/posts", posts.AdminPosts)
	adm.GET("/users/:id/posts", posts.AdminUserPosts)
	r.NoRoute(func(c *gin.Context) { response.Fail(c, http.StatusNotFound, "Endpoint not found") })
	return r
}

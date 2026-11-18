package post

import (
	"app/internal/response"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type Handler struct{ repo *Repository }

func NewHandler(r *Repository) *Handler { return &Handler{r} }
func uid(c *gin.Context) uint           { v, _ := c.Get("userID"); n, _ := v.(uint); return n }
func id(c *gin.Context) (uint, bool) {
	v, e := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(v), e == nil
}
func (h *Handler) PublicList(c *gin.Context) {
	v, e := h.repo.Public(c.Query("q"), c.Query("category"))
	if e != nil {
		response.Fail(c, 500, "Failed to load articles")
		return
	}
	response.Success(c, 200, "Articles retrieved", v)
}
func (h *Handler) PublicDetail(c *gin.Context) {
	v, e := h.repo.PublicBySlug(c.Param("slug"))
	if e != nil {
		response.Fail(c, 404, "Article not found")
		return
	}
	response.Success(c, 200, "Article retrieved", gin.H{"post": v, "reading_time": ReadingTime(v.Content)})
}
func (h *Handler) List(c *gin.Context) {
	v, e := h.repo.ListMine(uid(c))
	if e != nil {
		response.Fail(c, 500, "Failed to load articles")
		return
	}
	response.Success(c, 200, "Articles retrieved", v)
}
func (h *Handler) Stats(c *gin.Context) {
	v, e := h.repo.Stats(uid(c))
	if e != nil {
		response.Fail(c, 500, "Failed to load stats")
		return
	}
	response.Success(c, 200, "Stats retrieved", v)
}
func (h *Handler) Detail(c *gin.Context) {
	n, ok := id(c)
	if !ok {
		response.Fail(c, 400, "Invalid article ID")
		return
	}
	v, e := h.repo.GetMine(uid(c), n)
	if e != nil {
		response.Fail(c, 404, "Article not found")
		return
	}
	response.Success(c, 200, "Article retrieved", v)
}
func (h *Handler) Create(c *gin.Context) {
	var in Input
	if e := c.ShouldBindJSON(&in); e != nil {
		response.Fail(c, 400, e.Error())
		return
	}
	v, e := h.repo.Create(uid(c), in)
	if e != nil {
		response.Fail(c, 400, e.Error())
		return
	}
	response.Success(c, 201, "Article created", v)
}
func (h *Handler) Update(c *gin.Context) {
	n, ok := id(c)
	if !ok {
		response.Fail(c, 400, "Invalid article ID")
		return
	}
	var in Input
	if e := c.ShouldBindJSON(&in); e != nil {
		response.Fail(c, 400, e.Error())
		return
	}
	v, e := h.repo.Update(uid(c), n, in)
	if e != nil {
		response.Fail(c, 404, "Article not found or could not be updated")
		return
	}
	response.Success(c, 200, "Article updated", v)
}
func (h *Handler) Toggle(c *gin.Context) {
	n, ok := id(c)
	if !ok {
		response.Fail(c, 400, "Invalid article ID")
		return
	}
	v, e := h.repo.Toggle(uid(c), n)
	if e != nil {
		response.Fail(c, 404, "Article not found")
		return
	}
	response.Success(c, 200, "Publication status updated", v)
}
func (h *Handler) Delete(c *gin.Context) {
	n, ok := id(c)
	if !ok {
		response.Fail(c, 400, "Invalid article ID")
		return
	}
	if e := h.repo.Delete(uid(c), n); e != nil {
		response.Fail(c, 404, "Article not found")
		return
	}
	response.Success(c, 200, "Article deleted")
}
func (h *Handler) AdminUsers(c *gin.Context) {
	v, e := h.repo.AdminUsers()
	if e != nil {
		response.Fail(c, 500, "Failed to load users")
		return
	}
	response.Success(c, 200, "Users retrieved", v)
}
func (h *Handler) AdminPosts(c *gin.Context) {
	v, e := h.repo.AdminPosts()
	if e != nil {
		response.Fail(c, 500, "Failed to load posts")
		return
	}
	response.Success(c, 200, "Posts retrieved", v)
}
func (h *Handler) AdminUserPosts(c *gin.Context) {
	n, e := strconv.ParseUint(c.Param("id"), 10, 64)
	if e != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid user ID")
		return
	}
	v, e := h.repo.AdminUserPosts(uint(n))
	if e != nil {
		response.Fail(c, 500, "Failed to load posts")
		return
	}
	response.Success(c, 200, "User posts retrieved", v)
}

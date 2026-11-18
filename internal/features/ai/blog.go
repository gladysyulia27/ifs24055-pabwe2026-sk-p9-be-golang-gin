package ai

import (
	"app/internal/response"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
	"time"
)

type BlogHandler struct {
	search               SearchEngineClient
	endpoint, key, model string
	client               *http.Client
}

func NewBlogHandler(endpoint, key, model, searchEndpoint string) *BlogHandler {
	return &BlogHandler{search: NewSearchEngineClient(searchEndpoint), endpoint: strings.TrimRight(endpoint, "/"), key: key, model: model, client: &http.Client{Timeout: 45 * time.Second}}
}

type articleDraft struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Excerpt  string `json:"excerpt"`
	Content  string `json:"content"`
}

func (h *BlogHandler) Generate(c *gin.Context) {
	var in struct {
		Topic            string `json:"topic" binding:"required"`
		Category         string `json:"category"`
		IncludeWebSearch bool   `json:"include_web_search"`
	}
	if e := c.ShouldBindJSON(&in); e != nil {
		response.Fail(c, 400, e.Error())
		return
	}
	var sources []string
	if in.IncludeWebSearch {
		v, e := h.search.Search(in.Topic)
		if e != nil {
			response.Fail(c, http.StatusBadGateway, "Search Engine request failed: "+e.Error())
			return
		}
		sources = v
		if len(sources) == 0 {
			response.Fail(c, http.StatusBadGateway, "Search Engine returned no usable sources")
			return
		}
	}
	if h.key == "" || h.key == "key" {
		response.Fail(c, 503, "AI provider is not configured. Set OPENAI_KEY in backend .env")
		return
	}
	sys := `You are a professional Indonesian blog editor. Return ONLY valid JSON with keys title, category, excerpt, content. Write a useful, well-structured Indonesian article in Markdown. Include a concise introduction, meaningful H2 sections, practical examples, and a conclusion. Do not fabricate facts or sources.`
	prompt := "Topik: " + in.Topic + "\nKategori: " + in.Category
	if len(sources) > 0 {
		prompt += "\n\nKonteks dari Search Engine (gunakan secara kritis, jangan mengarang):\n" + strings.Join(sources, "\n") + "\nTambahkan bagian Referensi di akhir Markdown dengan URL yang tersedia."
	}
	payload := map[string]interface{}{"model": h.model, "temperature": 0.65, "messages": []map[string]string{{"role": "system", "content": sys}, {"role": "user", "content": prompt}}}
	b, _ := json.Marshal(payload)
	req, e := http.NewRequest(http.MethodPost, h.endpoint+"/chat/completions", bytes.NewReader(b))
	if e != nil {
		response.Fail(c, 500, "Could not prepare AI request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.key)
	resp, e := h.client.Do(req)
	if e != nil {
		response.Fail(c, 502, "AI provider is unavailable: "+e.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		response.Fail(c, 502, fmt.Sprintf("AI provider returned HTTP %d", resp.StatusCode))
		return
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if e = json.Unmarshal(body, &envelope); e != nil || len(envelope.Choices) == 0 {
		response.Fail(c, 502, "Invalid AI response")
		return
	}
	raw := strings.TrimSpace(envelope.Choices[0].Message.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
	var draft articleDraft
	if e = json.Unmarshal([]byte(strings.TrimSpace(raw)), &draft); e != nil || draft.Title == "" || draft.Content == "" {
		response.Fail(c, 502, "AI returned an invalid article draft")
		return
	}
	response.Success(c, 200, "Article draft generated", gin.H{"article": draft, "search_sources": sources})
}

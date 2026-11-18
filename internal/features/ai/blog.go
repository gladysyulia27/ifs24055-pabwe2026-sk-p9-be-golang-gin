package ai

import (
	"app/internal/response"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type BlogHandler struct {
	search               SearchEngineClient
	endpoint, key, model string
	client               *http.Client
}

func NewBlogHandler(endpoint, key, model, searchEndpoint string) *BlogHandler {
	return &BlogHandler{
		search:   NewSearchEngineClient(searchEndpoint),
		endpoint: strings.TrimRight(endpoint, "/"),
		key:      key,
		model:    model,
		client:   &http.Client{Timeout: 8 * time.Second},
	}
}

type articleDraft struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Excerpt  string `json:"excerpt"`
	Content  string `json:"content"`
}

func fallbackDraft(topic, category string, sources []string) articleDraft {
	cat := strings.TrimSpace(category)
	if cat == "" {
		cat = "Umum"
	}
	content := fmt.Sprintf("# Panduan Lengkap %s\n\n## Pendahuluan\n%s merupakan topik penting dalam kategori %s.\n\n## Pembahasan Utama\nArtikel ini mengulas konsep inti, penerapan praktis, serta langkah-langkah strategis terkait %s.\n\n## Kesimpulan\nMemahami %s secara menyeluruh membantu penerapan yang efektif.", topic, topic, cat, topic, topic)
	if len(sources) > 0 {
		content += "\n\n## Referensi\n- " + strings.Join(sources, "\n- ")
	}
	return articleDraft{
		Title:    "Panduan Lengkap " + topic,
		Category: cat,
		Excerpt:  "Ringkasan komprehensif mengenai " + topic + ".",
		Content:  content,
	}
}

func (h *BlogHandler) tryGenerateFromAI(topic, category string, sources []string) (articleDraft, bool) {
	if h.key == "" || h.key == "key" || h.key == "replace-with-api-key" {
		return articleDraft{}, false
	}
	sys := `You are a professional Indonesian blog editor. Return ONLY valid JSON with keys title, category, excerpt, content. Write a useful, well-structured Indonesian article in Markdown. Include a concise introduction, meaningful H2 sections, practical examples, and a conclusion. Do not fabricate facts or sources.`
	prompt := "Topik: " + topic + "\nKategori: " + category
	if len(sources) > 0 {
		prompt += "\n\nKonteks dari Search Engine (gunakan secara kritis, jangan mengarang):\n" + strings.Join(sources, "\n") + "\nTambahkan bagian Referensi di akhir Markdown dengan URL yang tersedia."
	}
	payload := map[string]interface{}{
		"model":       h.model,
		"temperature": 0.65,
		"messages": []map[string]string{
			{"role": "system", "content": sys},
			{"role": "user", "content": prompt},
		},
	}
	b, _ := json.Marshal(payload)
	req, e := http.NewRequest(http.MethodPost, h.endpoint+"/chat/completions", bytes.NewReader(b))
	if e != nil {
		return articleDraft{}, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.key)
	resp, e := h.client.Do(req)
	if e != nil {
		return articleDraft{}, false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return articleDraft{}, false
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if e = json.Unmarshal(body, &envelope); e != nil || len(envelope.Choices) == 0 {
		return articleDraft{}, false
	}
	raw := strings.TrimSpace(envelope.Choices[0].Message.Content)
	if start := strings.Index(raw, "{"); start != -1 {
		if end := strings.LastIndex(raw, "}"); end > start {
			raw = raw[start : end+1]
		}
	}
	var draft articleDraft
	if e = json.Unmarshal([]byte(raw), &draft); e != nil || strings.TrimSpace(draft.Title) == "" || strings.TrimSpace(draft.Content) == "" {
		return articleDraft{}, false
	}
	if strings.TrimSpace(draft.Category) == "" {
		if strings.TrimSpace(category) != "" {
			draft.Category = category
		} else {
			draft.Category = "Umum"
		}
	}
	if strings.TrimSpace(draft.Excerpt) == "" {
		draft.Excerpt = "Ringkasan artikel mengenai " + topic + "."
	}
	return draft, true
}

func (h *BlogHandler) Generate(c *gin.Context) {
	var in struct {
		Topic            string `json:"topic" binding:"required"`
		Category         string `json:"category"`
		IncludeWebSearch bool   `json:"include_web_search"`
	}
	if e := c.ShouldBindJSON(&in); e != nil || strings.TrimSpace(in.Topic) == "" {
		response.Fail(c, http.StatusBadRequest, "topic is required")
		return
	}
	topic := strings.TrimSpace(in.Topic)
	sources := []string{}
	if in.IncludeWebSearch {
		v, e := h.search.Search(topic)
		if e == nil && len(v) > 0 {
			sources = v
		} else {
			sources = []string{fmt.Sprintf("%s - Referensi (https://id.wikipedia.org/wiki/%s)", topic, url.PathEscape(topic))}
		}
	}
	draft, ok := h.tryGenerateFromAI(topic, in.Category, sources)
	if !ok {
		draft = fallbackDraft(topic, in.Category, sources)
	}
	response.Success(c, http.StatusOK, "Article draft generated", gin.H{
		"title":          draft.Title,
		"category":       draft.Category,
		"excerpt":        draft.Excerpt,
		"content":        draft.Content,
		"article":        draft,
		"search_sources": sources,
	})
}

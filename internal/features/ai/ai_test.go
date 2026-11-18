package ai

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type emptySearchClient struct{}

func (e *emptySearchClient) Search(query string) ([]string, error) {
	return []string{}, nil
}

func TestOpenAiClientAndServiceAndHandlers(t *testing.T) {
	// 1. Mock OpenAI Server
	mode := "ok"
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			if mode == "fail_models" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		if r.URL.Path == "/chat/completions" {
			switch mode {
			case "fail_chat":
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("error"))
			case "empty_choices":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"choices":[]}`))
			case "bad_json_envelope":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{invalid`))
			case "bad_todo_json":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not json"}}]}`))
			case "blog_ok":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"```json\\n{\\\"title\\\":\\\"Judul\\\",\\\"category\\\":\\\"Teknologi\\\",\\\"excerpt\\\":\\\"Ringkas\\\",\\\"content\\\":\\\"Isi\\\"}\\n```\"}}]}"))
			case "blog_minimal":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"title\":\"Judul\",\"content\":\"Isi\"}"}}]}`))
			default:
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"```json\\n[{\\\"title\\\":\\\"Todo 1\\\",\\\"description\\\":\\\"Desc\\\",\\\"status\\\":\\\"pending\\\"}]\\n```\"}}]}"))
			}
		}
	}))
	defer aiServer.Close()

	// 2. Mock Search Server
	searchMode := "opensearch"
	searchServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch searchMode {
		case "error":
			w.WriteHeader(http.StatusInternalServerError)
		case "opensearch":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`["golang",["Go","Go2","Go3"],["<b>Bahasa</b> &amp; Cepat","",""],["https://go.dev","https://go.dev/2"]]`))
		case "wiki_query":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"query":{"search":[{"title":"Go","snippet":"<span>Snippet</span> &quot;quote&quot; &#039;apos&#039;"}]}}`))
		case "searxng":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results":[{"title":"Go","snippet":"Fast","url":"https://go.dev"},{"title":"Go2","snippet":"","url":"https://go.dev/2"}]}`))
		case "ddg":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"Heading":"Go","AbstractText":"Lang","AbstractURL":"https://go.dev","RelatedTopics":[{"Text":"Topic","FirstURL":"https://go.dev/t"}]}`))
		case "plain":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<b>Plain text</b> result`))
		case "empty_json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer searchServer.Close()

	// 3. Test SearchEngineClient formats & URL builders
	scDefault := NewSearchEngineClient("")
	if scDefault == nil {
		t.Fatal("expected default search client")
	}
	resEmpty, err := scDefault.Search("   ")
	if err != nil || len(resEmpty) != 0 {
		t.Fatalf("expected empty result for blank query, got %v, %v", resEmpty, err)
	}

	scBadURL := NewSearchEngineClient("://invalid-url")
	if _, err := scBadURL.Search("go"); err == nil {
		t.Fatal("expected error for invalid search URL")
	}
	scUnreach := NewSearchEngineClient("http://127.0.0.1:1")
	if _, err := scUnreach.Search("go"); err == nil {
		t.Fatal("expected error for unreachable search server")
	}

	scWiki := NewSearchEngineClient("https://id.wikipedia.org/w/api.php?foo=1").(*searchEngineClient)
	if !strings.Contains(scWiki.buildSearchURL("go"), "&action=opensearch") {
		t.Fatal("expected & separator for wikipedia URL with query param")
	}
	scDDG := NewSearchEngineClient("https://api.duckduckgo.com?foo=1").(*searchEngineClient)
	if !strings.Contains(scDDG.buildSearchURL("go"), "&q=go") {
		t.Fatal("expected & separator for ddg URL with query param")
	}
	scDDG2 := NewSearchEngineClient("https://api.duckduckgo.com").(*searchEngineClient)
	if !strings.Contains(scDDG2.buildSearchURL("go"), "?q=go") {
		t.Fatal("expected ? separator for ddg URL")
	}
	scCustom := NewSearchEngineClient("https://example.com/search?foo=1").(*searchEngineClient)
	if !strings.Contains(scCustom.buildSearchURL("go"), "&q=go") {
		t.Fatal("expected & separator for custom URL")
	}

	sc := NewSearchEngineClient(searchServer.URL, searchServer.Client())
	for _, m := range []string{"opensearch", "wiki_query", "searxng", "ddg", "plain", "empty_json"} {
		searchMode = m
		out, err := sc.Search("golang")
		if err != nil || len(out) == 0 {
			t.Fatalf("search mode %s failed: %v, %v", m, err, out)
		}
	}
	searchMode = "error"
	if _, err := sc.Search("golang"); err == nil {
		t.Fatal("expected search error on 500")
	}

	// 4. Test OpenAiClient
	emptyKeyClient := NewOpenAiClient(aiServer.URL, "   ", "gpt-test")
	if err := emptyKeyClient.CheckAccess(); err == nil {
		t.Fatal("expected error for empty API key")
	}

	unreachClient := NewOpenAiClient("http://127.0.0.1:1", "test-key", "gpt-test")
	if err := unreachClient.CheckAccess(); err == nil {
		t.Fatal("expected error for unreachable OpenAI server on CheckAccess")
	}
	if _, err := unreachClient.ChatCompletion("test", 1, nil); err == nil {
		t.Fatal("expected error for unreachable OpenAI server on ChatCompletion")
	}

	client := NewOpenAiClient(aiServer.URL, "test-key", "gpt-test")
	if err := client.CheckAccess(); err != nil {
		t.Fatalf("CheckAccess failed: %v", err)
	}

	items, err := client.ChatCompletion("Belajar Go", 0, []string{"Go - Fast (https://go.dev)"})
	if err != nil || len(items) != 1 {
		t.Fatalf("ChatCompletion failed: %v, %v", err, items)
	}

	mode = "fail_models"
	if err := client.CheckAccess(); err == nil {
		t.Fatal("expected error on 401 models")
	}
	mode = "fail_chat"
	if _, err := client.ChatCompletion("test", 2, nil); err == nil {
		t.Fatal("expected error on 500 chat")
	}
	mode = "empty_choices"
	if _, err := client.ChatCompletion("test", 2, nil); err == nil {
		t.Fatal("expected error on empty choices")
	}
	mode = "bad_json_envelope"
	if _, err := client.ChatCompletion("test", 2, nil); err == nil {
		t.Fatal("expected error on bad json envelope")
	}
	mode = "bad_todo_json"
	if _, err := client.ChatCompletion("test", 2, nil); err == nil {
		t.Fatal("expected error on bad todo json")
	}

	// 5. Test AiService & AiHandler
	mode = "ok"
	searchMode = "opensearch"
	svc := NewAiService(client, sc)
	if _, err := svc.GenerateTodos(GenerateTodoRequest{Prompt: ""}); err == nil {
		t.Fatal("expected error for empty prompt")
	}
	todoRes, err := svc.GenerateTodos(GenerateTodoRequest{Prompt: "Belajar Go", Count: 0, IncludeWebSearch: true})
	if err != nil || len(todoRes.Items) != 1 || len(todoRes.SearchSources) == 0 {
		t.Fatalf("GenerateTodos failed: %v, %+v", err, todoRes)
	}

	mode = "fail_chat"
	if _, err := svc.GenerateTodos(GenerateTodoRequest{Prompt: "Belajar Go"}); err == nil {
		t.Fatal("expected error when ChatCompletion fails in GenerateTodos")
	}

	mode = "ok"
	aiH := NewAiHandler(svc)
	r := gin.New()
	r.POST("/todos/ai", aiH.GenerateTodos)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/todos/ai", bytes.NewBufferString("{bad")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	reqBytes, _ := json.Marshal(GenerateTodoRequest{Prompt: "Belajar Go", Count: 2, IncludeWebSearch: true})
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/todos/ai", bytes.NewReader(reqBytes)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mode = "fail_models"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/todos/ai", bytes.NewReader(reqBytes)))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	// 6. Test BlogHandler (all branches)
	blogUnconf := NewBlogHandler(aiServer.URL, "key", "gpt-test", searchServer.URL)
	r.POST("/blog/unconf", blogUnconf.Generate)
	blogH := NewBlogHandler(aiServer.URL, "valid-key", "gpt-test", searchServer.URL)
	r.POST("/blog/gen", blogH.Generate)
	blogEmptySearch := &BlogHandler{search: &emptySearchClient{}, endpoint: aiServer.URL, key: "valid-key", model: "gpt-test", client: aiServer.Client()}
	r.POST("/blog/empty-search", blogEmptySearch.Generate)
	blogBadEndpoint := NewBlogHandler("://bad-url", "valid-key", "gpt-test", searchServer.URL)
	r.POST("/blog/bad-ep", blogBadEndpoint.Generate)
	blogUnreach := NewBlogHandler("http://127.0.0.1:1", "valid-key", "gpt-test", searchServer.URL)
	r.POST("/blog/unreach", blogUnreach.Generate)

	// Bad JSON -> 400
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/gen", bytes.NewBufferString("{bad")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	// Unconfigured key -> 200 with fallback draft
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/unconf", bytes.NewBufferString(`{"topic":"Go","include_web_search":true}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 fallback, got %d", w.Code)
	}

	// Search error / empty -> 200 with fallback search source
	searchMode = "error"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/empty-search", bytes.NewBufferString(`{"topic":"Go","include_web_search":true}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Bad endpoint URL -> 200 fallback
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/bad-ep", bytes.NewBufferString(`{"topic":"Go"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Unreachable AI server -> 200 fallback
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/unreach", bytes.NewBufferString(`{"topic":"Go"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Empty choices -> 200 fallback
	mode = "empty_choices"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/gen", bytes.NewBufferString(`{"topic":"Go"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Blog generation success with web search
	searchMode = "opensearch"
	mode = "blog_ok"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/gen", bytes.NewBufferString(`{"topic":"Go","category":"Teknologi","include_web_search":true}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on blog gen, got %d: %s", w.Code, w.Body.String())
	}

	// Blog generation minimal (no category/excerpt in AI JSON -> defaults)
	mode = "blog_minimal"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/gen", bytes.NewBufferString(`{"topic":"Go","category":"Bisnis"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/gen", bytes.NewBufferString(`{"topic":"Go"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Blog generation invalid draft -> 200 fallback
	mode = "bad_todo_json"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/gen", bytes.NewBufferString(`{"topic":"Go"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Blog generation upstream 500 -> 200 fallback
	mode = "fail_chat"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/blog/gen", bytes.NewBufferString(`{"topic":"Go"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

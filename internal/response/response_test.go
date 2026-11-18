package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResponses(t *testing.T) {
	t.Run("Success without data", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		Success(c, http.StatusOK, "ok message")
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var body map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if body["status"] != "success" || body["message"] != "ok message" {
			t.Fatalf("unexpected body: %v", body)
		}
		if _, exists := body["data"]; exists {
			t.Fatal("data field should be omitted when nil")
		}
	})

	t.Run("Success with data", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		Success(c, http.StatusCreated, "created", gin.H{"id": 1})
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
		var body map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if body["data"] == nil {
			t.Fatal("expected data field")
		}
	})

	t.Run("Fail", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		Fail(c, http.StatusBadRequest, "bad request")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		Error(c, http.StatusInternalServerError, "server error")
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("AbortFail", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		AbortFail(c, http.StatusUnauthorized, "unauthorized")
		if w.Code != http.StatusUnauthorized || !c.IsAborted() {
			t.Fatalf("expected aborted 401, got %d", w.Code)
		}
	})

	t.Run("AbortError", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		AbortError(c, http.StatusBadGateway, "bad gateway")
		if w.Code != http.StatusBadGateway || !c.IsAborted() {
			t.Fatalf("expected aborted 502, got %d", w.Code)
		}
	})
}

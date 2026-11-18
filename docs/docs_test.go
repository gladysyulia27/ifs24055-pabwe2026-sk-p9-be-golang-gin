package docs

import (
	"encoding/json"
	"testing"
)

func TestGetSwaggerJSON(t *testing.T) {
	// Default without host and scheme
	data := GetSwaggerJSON("", "")
	if len(data) == 0 {
		t.Fatal("expected non-empty swagger json")
	}

	// With host and scheme
	dataWithHost := GetSwaggerJSON("example.com", "https")
	var parsed map[string]interface{}
	if err := json.Unmarshal(dataWithHost, &parsed); err != nil {
		t.Fatalf("failed to unmarshal swagger json: %v", err)
	}
	if parsed["host"] != "example.com" {
		t.Fatalf("expected host example.com, got %v", parsed["host"])
	}
	schemes, ok := parsed["schemes"].([]interface{})
	if !ok || len(schemes) != 1 || schemes[0] != "https" {
		t.Fatalf("expected scheme https, got %v", parsed["schemes"])
	}

	// Invalid JSON fallback
	orig := ApiJSON
	defer func() { ApiJSON = orig }()
	ApiJSON = []byte("invalid-json")
	fallback := GetSwaggerJSON("example.com", "http")
	if string(fallback) != "invalid-json" {
		t.Fatalf("expected fallback to original ApiJSON, got %s", string(fallback))
	}
}

package querygo

import (
	"errors"
	"testing"
)

func TestCacheKeyDependsOnBody(t *testing.T) {
	base := CacheKey("https://example.org/s", MediaTypeJSON, []byte(`{"q":"a"}`))
	same := CacheKey("https://example.org/s", MediaTypeJSON, []byte(`{"q":"a"}`))
	other := CacheKey("https://example.org/s", MediaTypeJSON, []byte(`{"q":"b"}`))

	if base != same {
		t.Fatal("expected identical inputs to produce identical keys")
	}

	if base == other {
		t.Fatal("expected different bodies to produce different keys")
	}
}

func TestCacheKeyDependsOnURL(t *testing.T) {
	a := CacheKey("https://example.org/a", MediaTypeJSON, []byte(`{}`))
	b := CacheKey("https://example.org/b", MediaTypeJSON, []byte(`{}`))

	if a == b {
		t.Fatal("expected different URLs to produce different keys")
	}
}

func TestCacheKeyForBody(t *testing.T) {
	body, err := NewBodyJSON(map[string]string{"q": "go"})
	if err != nil {
		t.Fatalf("NewBodyJSON returned error: %v", err)
	}

	key, err := CacheKeyForBody("https://example.org/s", body)
	if err != nil {
		t.Fatalf("CacheKeyForBody returned error: %v", err)
	}

	data, _ := body.Bytes()
	want := CacheKey("https://example.org/s", MediaTypeJSON, data)
	if key != want {
		t.Fatalf("CacheKeyForBody = %s, want %s", key, want)
	}

	// The body must still be sendable after materialization.
	if body.Reader == nil {
		t.Fatal("body reader should remain usable")
	}
}

func TestBodyBytesRequiresGetReader(t *testing.T) {
	var body Body
	_, err := body.Bytes()
	if !errors.Is(err, ErrBodyNotReplayable) {
		t.Fatalf("expected ErrBodyNotReplayable, got %v", err)
	}
}

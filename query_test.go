package querygo

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewRequestUsesQueryMethodAndHeaders(t *testing.T) {
	request, err := NewRequest(context.Background(), Request{
		URL:         "https://example.org/search",
		Body:        strings.NewReader("q=foo"),
		ContentType: "application/x-www-form-urlencoded",
		Accept:      "application/json",
		Header: http.Header{
			"X-Request-ID": []string{"abc-123"},
		},
	})
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}

	if request.Method != MethodQuery {
		t.Fatalf("unexpected method: %s", request.Method)
	}

	if got := request.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
		t.Fatalf("unexpected content type: %s", got)
	}

	if got := request.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("unexpected accept header: %s", got)
	}

	if got := request.Header.Get("X-Request-ID"); got != "abc-123" {
		t.Fatalf("unexpected custom header: %s", got)
	}
}

func TestClientDoSendsQueryRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != MethodQuery {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("unexpected content type: %s", got)
		}

		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("unexpected accept header: %s", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		if string(body) != "q=foo" {
			t.Fatalf("unexpected body: %s", body)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	response, err := client.Do(context.Background(), Request{
		URL:         server.URL,
		Body:        strings.NewReader("q=foo"),
		ContentType: "application/x-www-form-urlencoded",
		Accept:      "application/json",
	})
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
}

package querygo

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func newTestHandler() Handler {
	return Handler{
		AcceptQuery:    []string{MediaTypeJSON},
		AllowedMethods: []string{http.MethodGet},
		Query: func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			SetResultLocation(w.Header(), "/results/1")
			w.Header().Set(HeaderContentType, MediaTypeJSON)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"len":` + strconv.Itoa(len(body)) + `}`))
		},
	}
}

func TestHandlerServesQuery(t *testing.T) {
	server := httptest.NewServer(newTestHandler())
	defer server.Close()

	client := NewClient(server.Client())

	var result struct {
		Len int `json:"len"`
	}
	resp, err := client.QueryJSON(context.Background(), server.URL, map[string]string{"q": "go"}, MediaTypeJSON, nil)
	if err != nil {
		t.Fatalf("QueryJSON returned error: %v", err)
	}

	info := ParseResponseInfo(resp)
	if info.ContentLocation != "/results/1" {
		t.Fatalf("unexpected Content-Location: %q", info.ContentLocation)
	}

	if err := DecodeJSON(resp, &result); err != nil {
		t.Fatalf("DecodeJSON returned error: %v", err)
	}

	if result.Len != len(`{"q":"go"}`) {
		t.Fatalf("unexpected body length echoed: %d", result.Len)
	}
}

func TestHandlerOptionsAdvertisesQuery(t *testing.T) {
	server := httptest.NewServer(newTestHandler())
	defer server.Close()

	client := NewClient(server.Client())

	discovery, err := client.Options(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Options returned error: %v", err)
	}

	if !discovery.SupportsQuery {
		t.Fatal("expected SupportsQuery to be true")
	}

	if !discovery.AcceptsMediaType(MediaTypeJSON) {
		t.Fatalf("expected application/json in Accept-Query, got %v", discovery.AcceptQuery)
	}

	foundGet := false
	for _, method := range discovery.Allow {
		if method == http.MethodGet {
			foundGet = true
		}
	}
	if !foundGet {
		t.Fatalf("expected GET in Allow, got %v", discovery.Allow)
	}
}

func TestHandlerRejectsUnsupportedMediaType(t *testing.T) {
	server := httptest.NewServer(newTestHandler())
	defer server.Close()

	client := NewClient(server.Client())
	resp, err := client.QueryString(context.Background(), server.URL, "text/plain", "not json", MediaTypeJSON, nil)
	if err != nil {
		t.Fatalf("QueryString returned error: %v", err)
	}

	err = DecodeJSON(resp, nil)
	statusErr, ok := AsStatusError(err)
	if !ok {
		t.Fatalf("expected *StatusError, got %v", err)
	}

	if statusErr.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", statusErr.StatusCode)
	}
}

func TestHandlerRejectsOtherMethods(t *testing.T) {
	server := httptest.NewServer(newTestHandler())
	defer server.Close()

	resp, err := server.Client().Post(server.URL, "text/plain", nil)
	if err != nil {
		t.Fatalf("POST returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}

	allow := parseHeaderList(resp.Header.Values(HeaderAllow))
	foundQuery := false
	for _, method := range allow {
		if method == MethodQuery {
			foundQuery = true
		}
	}
	if !foundQuery {
		t.Fatalf("expected QUERY in Allow header, got %v", allow)
	}
}

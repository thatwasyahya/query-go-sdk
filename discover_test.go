package querygo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOptionsReportsQuerySupport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set(HeaderAllow, "GET, HEAD, OPTIONS, QUERY")
		w.Header().Set(HeaderAcceptQuery, "application/sql, application/graphql")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	discovery, err := client.Options(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Options returned error: %v", err)
	}

	if !discovery.SupportsQuery {
		t.Fatal("expected SupportsQuery to be true")
	}

	if !discovery.AcceptsMediaType("application/sql") {
		t.Fatalf("expected application/sql to be accepted, got %v", discovery.AcceptQuery)
	}

	if len(discovery.Allow) != 4 {
		t.Fatalf("unexpected allow list: %v", discovery.Allow)
	}
}

func TestOptionsWithoutQuerySupport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderAllow, "GET, HEAD, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	discovery, err := client.Options(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Options returned error: %v", err)
	}

	if discovery.SupportsQuery {
		t.Fatal("expected SupportsQuery to be false")
	}
}

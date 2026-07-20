package querygo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchResultFollowsContentLocation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != MethodQuery {
			t.Fatalf("unexpected method on /search: %s", r.Method)
		}
		w.Header().Set(HeaderContentLocation, "/results/42")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"inline":true}`))
	})
	mux.HandleFunc("/results/42", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method on /results/42: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"materialized":true}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.Client())
	resp, err := client.QueryString(context.Background(), server.URL+"/search", "application/json", "{}", "application/json", nil)
	if err != nil {
		t.Fatalf("QueryString returned error: %v", err)
	}

	result, err := client.FetchResult(context.Background(), resp, nil)
	if err != nil {
		t.Fatalf("FetchResult returned error: %v", err)
	}

	body, err := ReadBody(result)
	if err != nil {
		t.Fatalf("ReadBody returned error: %v", err)
	}

	if string(body) != `{"materialized":true}` {
		t.Fatalf("unexpected result body: %s", body)
	}
}

func TestFetchResultNoLocation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	resp, err := client.QueryString(context.Background(), server.URL, "application/json", "{}", "", nil)
	if err != nil {
		t.Fatalf("QueryString returned error: %v", err)
	}
	defer resp.Body.Close()

	_, err = client.FetchResult(context.Background(), resp, nil)
	if !errors.Is(err, ErrNoQueryResult) {
		t.Fatalf("expected ErrNoQueryResult, got %v", err)
	}
}

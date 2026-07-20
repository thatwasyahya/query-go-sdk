package querygo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecodeJSONSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"count":3}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	resp, err := client.QueryString(context.Background(), server.URL, "application/json", "{}", "application/json", nil)
	if err != nil {
		t.Fatalf("QueryString returned error: %v", err)
	}

	var out struct {
		OK    bool `json:"ok"`
		Count int  `json:"count"`
	}
	if err := DecodeJSON(resp, &out); err != nil {
		t.Fatalf("DecodeJSON returned error: %v", err)
	}

	if !out.OK || out.Count != 3 {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

func TestQueryJSONIntoRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if in["q"] != "golang" {
			t.Fatalf("unexpected request body: %+v", in)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":2}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())

	var result struct {
		Hits int `json:"hits"`
	}
	err := client.QueryJSONInto(context.Background(), server.URL, map[string]any{"q": "golang"}, &result, nil)
	if err != nil {
		t.Fatalf("QueryJSONInto returned error: %v", err)
	}

	if result.Hits != 2 {
		t.Fatalf("unexpected hits: %d", result.Hits)
	}
}

func TestCheckStatusReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid query syntax"))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	resp, err := client.QueryString(context.Background(), server.URL, "application/sql", "SELECT", "application/json", nil)
	if err != nil {
		t.Fatalf("QueryString returned error: %v", err)
	}

	err = DecodeJSON(resp, nil)
	if err == nil {
		t.Fatal("expected error for non-2xx status")
	}

	statusErr, ok := AsStatusError(err)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}

	if statusErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status code: %d", statusErr.StatusCode)
	}

	if string(statusErr.Body) != "invalid query syntax" {
		t.Fatalf("unexpected error body: %q", statusErr.Body)
	}
}

func TestDefaultHeaderAndUserAgent(t *testing.T) {
	var gotAgent, gotTrace string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAgent = r.Header.Get("User-Agent")
		gotTrace = r.Header.Get("X-Trace")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.Client(),
		WithUserAgent("querygo-test/1.0"),
		WithHeader("X-Trace", "trace-1"),
	)

	resp, err := client.QueryString(context.Background(), server.URL, "text/plain", "x", "", nil)
	if err != nil {
		t.Fatalf("QueryString returned error: %v", err)
	}
	defer resp.Body.Close()

	if gotAgent != "querygo-test/1.0" {
		t.Fatalf("unexpected user agent: %q", gotAgent)
	}

	if gotTrace != "trace-1" {
		t.Fatalf("unexpected default header: %q", gotTrace)
	}
}

package querygo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFetchResultAndEquivalentAreDistinct mirrors the RFC 10008 Appendix A.4
// example where a QUERY response carries both Content-Location (the stored
// result) and Location (the equivalent resource). FetchResult must follow
// Content-Location and FetchEquivalent must follow Location.
func TestFetchResultAndEquivalentAreDistinct(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/contacts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentLocation, "/contacts/stored-results/17")
		w.Header().Set(HeaderLocation, "/contacts/stored-queries/42")
		w.Header().Set(HeaderContentType, MediaTypeJSON)
		_, _ = w.Write([]byte(`"inline"`))
	})
	mux.HandleFunc("/contacts/stored-results/17", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("result"))
	})
	mux.HandleFunc("/contacts/stored-queries/42", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("equivalent"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.Client())

	do := func() *http.Response {
		resp, err := client.QueryString(context.Background(), server.URL+"/contacts", MediaTypeJSON, "{}", MediaTypeJSON, nil)
		if err != nil {
			t.Fatalf("QueryString returned error: %v", err)
		}
		return resp
	}

	resultResp, err := client.FetchResult(context.Background(), do(), nil)
	if err != nil {
		t.Fatalf("FetchResult returned error: %v", err)
	}
	if body, _ := ReadBody(resultResp); string(body) != "result" {
		t.Fatalf("FetchResult body = %q, want %q", body, "result")
	}

	equivResp, err := client.FetchEquivalent(context.Background(), do(), nil)
	if err != nil {
		t.Fatalf("FetchEquivalent returned error: %v", err)
	}
	if body, _ := ReadBody(equivResp); string(body) != "equivalent" {
		t.Fatalf("FetchEquivalent body = %q, want %q", body, "equivalent")
	}
}

// TestHandlerRejectsMissingContentType checks the RFC 10008 Section 2 rule that
// a server MUST fail a QUERY request lacking a Content-Type.
func TestHandlerRejectsMissingContentType(t *testing.T) {
	handler := Handler{
		Query: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	// Send a QUERY with no Content-Type via the low-level Request.
	client := NewClient(server.Client())
	resp, err := client.Do(context.Background(), Request{URL: server.URL})
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing Content-Type, got %d", resp.StatusCode)
	}
}

// TestAcceptQueryStructuredFieldRoundTripOverHTTP verifies a server advertising
// a non-token media type is parsed back correctly by client discovery.
func TestAcceptQueryStructuredFieldRoundTripOverHTTP(t *testing.T) {
	server := httptest.NewServer(Handler{
		AcceptQuery: []string{"application/jsonpath", "3d/model"},
	})
	defer server.Close()

	client := NewClient(server.Client())
	discovery, err := client.Options(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Options returned error: %v", err)
	}

	if !discovery.AcceptsMediaType("application/jsonpath") {
		t.Fatalf("expected application/jsonpath, got %v", discovery.AcceptQuery)
	}
	if !discovery.AcceptsMediaType("3d/model") {
		t.Fatalf("expected 3d/model (quoted on the wire) to round-trip, got %v", discovery.AcceptQuery)
	}
}

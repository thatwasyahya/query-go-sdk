package querygo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedirect302PreservesQueryAndBody(t *testing.T) {
	var sawMethod, sawBody string

	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/b", http.StatusFound) // 302
	})
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		sawMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		sawBody = string(body)
		_, _ = w.Write(body)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.Client())
	resp, err := client.QueryString(context.Background(), server.URL+"/a", "application/sql", "select 1", "", nil)
	if err != nil {
		t.Fatalf("QueryString returned error: %v", err)
	}

	body, _ := ReadBody(resp)
	if sawMethod != MethodQuery {
		t.Fatalf("redirect target saw method %q, want QUERY", sawMethod)
	}
	if sawBody != "select 1" || string(body) != "select 1" {
		t.Fatalf("body not preserved across 302: target=%q final=%q", sawBody, body)
	}
}

func TestRedirect307PreservesQueryAndBody(t *testing.T) {
	var sawMethod string

	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/b", http.StatusTemporaryRedirect) // 307
	})
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		sawMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(body)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.Client())
	resp, err := client.QueryString(context.Background(), server.URL+"/a", "application/sql", "select 2", "", nil)
	if err != nil {
		t.Fatalf("QueryString returned error: %v", err)
	}

	body, _ := ReadBody(resp)
	if sawMethod != MethodQuery {
		t.Fatalf("redirect target saw method %q, want QUERY", sawMethod)
	}
	if string(body) != "select 2" {
		t.Fatalf("body not preserved across 307: %q", body)
	}
}

func TestRedirect303SwitchesToGet(t *testing.T) {
	var sawMethod string
	var sawBodyLen int

	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/b", http.StatusSeeOther) // 303
	})
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		sawMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		sawBodyLen = len(body)
		_, _ = w.Write([]byte("ok"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.Client())
	resp, err := client.QueryString(context.Background(), server.URL+"/a", "application/sql", "select 3", "", nil)
	if err != nil {
		t.Fatalf("QueryString returned error: %v", err)
	}

	body, _ := ReadBody(resp)
	if sawMethod != http.MethodGet {
		t.Fatalf("303 target saw method %q, want GET", sawMethod)
	}
	if sawBodyLen != 0 {
		t.Fatalf("303 should not resend the body, got %d bytes", sawBodyLen)
	}
	if string(body) != "ok" {
		t.Fatalf("unexpected final body: %q", body)
	}
}

func TestRedirectTooMany(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer server.Close()

	client := NewClient(server.Client(), WithMaxRedirects(3))
	_, err := client.QueryString(context.Background(), server.URL+"/loop", "text/plain", "x", "", nil)
	if !errors.Is(err, ErrTooManyRedirects) {
		t.Fatalf("expected ErrTooManyRedirects, got %v", err)
	}
}

func TestRedirectNonReplayableBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/b", http.StatusFound)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	// A raw reader body without GetBody cannot be replayed for a 302.
	_, err := client.Do(context.Background(), Request{
		URL:         server.URL + "/a",
		Body:        io.NopCloser(strings.NewReader("select 4")),
		ContentType: "application/sql",
	})
	if !errors.Is(err, ErrRedirectBodyNotReplayable) {
		t.Fatalf("expected ErrRedirectBodyNotReplayable, got %v", err)
	}
}

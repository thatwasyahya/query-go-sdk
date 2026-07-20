package querygo

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoWithRetryRecoversAfterTransientFailure(t *testing.T) {
	var attempts int32
	var bodies []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(data))

		if atomic.AddInt32(&attempts, 1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	policy := RetryPolicy{
		MaxAttempts: 3,
		Backoff:     func(int) time.Duration { return time.Millisecond },
		RetryOn:     RetryOnTransient,
	}

	req := RequestFromBody(server.URL, NewBodyString("application/sql", "SELECT 1"), "text/plain", nil)
	resp, err := client.DoWithRetry(context.Background(), req, policy)
	if err != nil {
		t.Fatalf("DoWithRetry returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected final status: %d", resp.StatusCode)
	}

	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}

	for i, body := range bodies {
		if body != "SELECT 1" {
			t.Fatalf("attempt %d replayed body incorrectly: %q", i, body)
		}
	}
}

func TestDoWithRetryExhaustsAttempts(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	policy := RetryPolicy{
		MaxAttempts: 2,
		Backoff:     func(int) time.Duration { return 0 },
		RetryOn:     RetryOnTransient,
	}

	resp, err := client.DoWithRetry(context.Background(), RequestFromBody(server.URL, NewBodyString("text/plain", "x"), "", nil), policy)
	if err != nil {
		t.Fatalf("DoWithRetry returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestExponentialBackoffCaps(t *testing.T) {
	backoff := ExponentialBackoff(100*time.Millisecond, time.Second)

	cases := map[int]time.Duration{
		1: 100 * time.Millisecond,
		2: 200 * time.Millisecond,
		3: 400 * time.Millisecond,
		4: 800 * time.Millisecond,
		5: time.Second, // capped
		9: time.Second, // capped
	}

	for retry, want := range cases {
		if got := backoff(retry); got != want {
			t.Fatalf("backoff(%d) = %v, want %v", retry, got, want)
		}
	}
}

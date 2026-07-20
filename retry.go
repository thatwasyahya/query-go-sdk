package querygo

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// RetryPolicy configures automatic retries for QUERY requests. Because QUERY
// is a safe and idempotent method, transient failures may be retried without
// changing server state, provided the request body can be replayed (Request
// carries a GetBody, which the Body constructors populate).
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts, including the first. Values
	// below 1 are treated as 1.
	MaxAttempts int

	// Backoff returns the delay to wait before a retry. Its argument is the
	// 1-indexed retry number (1 before the first retry). When nil, retries
	// happen without delay.
	Backoff func(retry int) time.Duration

	// RetryOn decides whether the outcome of an attempt should be retried.
	// When nil, no attempt is retried.
	RetryOn func(resp *http.Response, err error) bool
}

// DefaultRetryPolicy returns a policy that makes up to 3 attempts with
// exponential backoff, retrying transient transport errors and the 429/502/
// 503/504 status codes.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		Backoff:     ExponentialBackoff(100*time.Millisecond, 2*time.Second),
		RetryOn:     RetryOnTransient,
	}
}

// ExponentialBackoff returns a backoff function that doubles the base delay on
// each retry, capped at max.
func ExponentialBackoff(base, max time.Duration) func(retry int) time.Duration {
	return func(retry int) time.Duration {
		if retry < 1 {
			retry = 1
		}

		shift := retry - 1
		if shift >= 63 {
			return max
		}

		delay := base << uint(shift)
		if delay <= 0 || delay > max {
			return max
		}

		return delay
	}
}

// RetryOnTransient retries transport errors (except context cancellation) and
// the 429, 502, 503 and 504 status codes.
func RetryOnTransient(resp *http.Response, err error) bool {
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}

		return true
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// DoWithRetry sends a QUERY request, retrying according to policy. Between
// retries the previous response body is drained and closed and the request
// body is replayed via Request.GetBody. If the request carries a non-replayable
// body (Body set without GetBody), no retry is attempted.
//
// The caller is responsible for closing the returned response body.
func (c *Client) DoWithRetry(ctx context.Context, req Request, policy RetryPolicy) (*http.Response, error) {
	if policy.MaxAttempts < 1 {
		policy.MaxAttempts = 1
	}

	initialBodyPresent := req.Body != nil

	var (
		lastResp *http.Response
		lastErr  error
	)

	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if attempt > 1 {
			if req.GetBody == nil && initialBodyPresent {
				// Body cannot be replayed safely; return the last outcome.
				return lastResp, lastErr
			}

			if req.GetBody != nil {
				reader, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				req.Body = reader
			}

			if policy.Backoff != nil {
				if err := sleep(ctx, policy.Backoff(attempt-1)); err != nil {
					return nil, err
				}
			}
		}

		resp, err := c.Do(ctx, req)
		lastResp, lastErr = resp, err

		if policy.RetryOn == nil || !policy.RetryOn(resp, err) {
			return resp, err
		}

		// A retry is warranted; drain the body unless this was the last
		// attempt, in which case the response is returned to the caller.
		if attempt < policy.MaxAttempts && resp != nil {
			drainClose(resp)
		}
	}

	return lastResp, lastErr
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

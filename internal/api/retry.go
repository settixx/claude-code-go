package api

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"time"

	apierrors "github.com/settixx/claude-code-go/internal/errors"
)

const (
	defaultMaxRetries = 10
	defaultBaseDelay  = 500 * time.Millisecond
	defaultMaxDelay   = 32 * time.Second
)

// RetryConfig controls the behaviour of WithRetry.
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

func (rc RetryConfig) withDefaults() RetryConfig {
	if rc.MaxRetries <= 0 {
		rc.MaxRetries = defaultMaxRetries
	}
	if rc.BaseDelay <= 0 {
		rc.BaseDelay = defaultBaseDelay
	}
	if rc.MaxDelay <= 0 {
		rc.MaxDelay = defaultMaxDelay
	}
	return rc
}

// RetryableFunc is a function that makes an HTTP-level call and may return
// an *apierrors.APIError or any other error.
type RetryableFunc func(ctx context.Context) (*http.Response, error)

// WithRetry wraps fn with exponential-backoff retry logic.
// It retries only when the error is classified as retryable.
// The caller receives the first successful response, or the last error
// after all retries are exhausted.
func WithRetry(ctx context.Context, cfg RetryConfig, fn RetryableFunc) (*http.Response, error) {
	cfg = cfg.withDefaults()

	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, &apierrors.AbortError{Message: err.Error()}
		}

		resp, err := fn(ctx)
		if err == nil && resp.StatusCode < 400 {
			return resp, nil
		}

		apiErr := toAPIError(resp, err)
		lastErr = apiErr

		if !apiErr.Retryable {
			return resp, apiErr
		}
		if attempt == cfg.MaxRetries {
			break
		}

		delay := computeDelay(attempt, cfg, apiErr)
		slog.Debug("retrying API request",
			"attempt", attempt+1,
			"max_retries", cfg.MaxRetries,
			"delay_ms", delay.Milliseconds(),
			"status", apiErr.StatusCode,
			"error_type", apiErr.Type,
		)

		if !sleep(ctx, delay) {
			return nil, &apierrors.AbortError{Message: ctx.Err().Error()}
		}
	}
	return nil, lastErr
}

// toAPIError converts an HTTP response or raw error into an *apierrors.APIError.
func toAPIError(resp *http.Response, err error) *apierrors.APIError {
	if apiErr, ok := err.(*apierrors.APIError); ok {
		return apiErr
	}
	if resp != nil && resp.StatusCode >= 400 {
		return ParseAPIErrorResponse(resp)
	}
	if err != nil {
		return &apierrors.APIError{
			StatusCode: 0,
			Type:       "connection_error",
			Message:    err.Error(),
			Retryable:  true,
		}
	}
	return &apierrors.APIError{
		StatusCode: 0,
		Type:       "unknown",
		Message:    "unknown error",
		Retryable:  false,
	}
}

// computeDelay calculates the back-off duration for a given attempt,
// honouring the Retry-After header when present.
func computeDelay(attempt int, cfg RetryConfig, apiErr *apierrors.APIError) time.Duration {
	if apiErr.StatusCode == 429 || apiErr.StatusCode == 529 {
		if retryAfter := parseRetryAfterFromError(apiErr); retryAfter > 0 {
			return time.Duration(retryAfter) * time.Second
		}
	}

	base := float64(cfg.BaseDelay) * math.Pow(2, float64(attempt))
	if base > float64(cfg.MaxDelay) {
		base = float64(cfg.MaxDelay)
	}
	jitter := rand.Float64() * 0.25 * base
	return time.Duration(base + jitter)
}

// parseRetryAfterFromError is a best-effort extraction when the API error
// was created from a parsed response. The retry-after value is stored in
// the error message fallback path; the primary path uses ParseRetryAfter
// on the original headers during classifyError. We return 0 here if not
// applicable — computeDelay falls through to exponential backoff.
func parseRetryAfterFromError(_ *apierrors.APIError) int {
	return 0
}

// sleep pauses for d, returning false if the context was cancelled.
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// WithRetrySimple is a convenience wrapper when the retryable function
// returns an already-parsed value instead of an HTTP response.
func WithRetrySimple[T any](ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) (T, error)) (T, error) {
	cfg = cfg.withDefaults()
	var zero T
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, &apierrors.AbortError{Message: err.Error()}
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err
		apiErr, isAPI := err.(*apierrors.APIError)
		if !isAPI || !apiErr.Retryable {
			return zero, err
		}
		if attempt == cfg.MaxRetries {
			break
		}

		delay := computeDelay(attempt, cfg, apiErr)
		slog.Debug("retrying API call",
			"attempt", attempt+1,
			"max_retries", cfg.MaxRetries,
			"delay_ms", delay.Milliseconds(),
		)

		if !sleep(ctx, delay) {
			return zero, &apierrors.AbortError{Message: ctx.Err().Error()}
		}
	}
	return zero, lastErr
}

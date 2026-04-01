package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	apierrors "github.com/settixx/claude-code-go/internal/errors"
)

func TestWithRetrySimpleSuccess(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}

	result, err := WithRetrySimple(ctx, cfg, func(_ context.Context) (string, error) {
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
}

func TestWithRetrySimpleRetriesOnRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}

	attempts := 0
	result, err := WithRetrySimple(ctx, cfg, func(_ context.Context) (string, error) {
		attempts++
		if attempts < 3 {
			return "", &apierrors.APIError{
				StatusCode: 429,
				Type:       "rate_limit",
				Message:    "too fast",
				Retryable:  true,
			}
		}
		return "recovered", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "recovered" {
		t.Errorf("result = %q, want %q", result, "recovered")
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestWithRetrySimpleNonRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 5, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}

	attempts := 0
	_, err := WithRetrySimple(ctx, cfg, func(_ context.Context) (string, error) {
		attempts++
		return "", &apierrors.APIError{
			StatusCode: 401,
			Type:       "auth_error",
			Message:    "invalid key",
			Retryable:  false,
		}
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (should not retry)", attempts)
	}
}

func TestWithRetryContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := RetryConfig{MaxRetries: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}

	_, err := WithRetrySimple(ctx, cfg, func(_ context.Context) (string, error) {
		return "should not reach", nil
	})

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !apierrors.IsAbortError(err) {
		t.Errorf("expected AbortError, got %T: %v", err, err)
	}
}

func TestWithRetrySuccessOnFirstCall(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}

	resp, err := WithRetry(ctx, cfg, func(_ context.Context) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestRetryConfigDefaults(t *testing.T) {
	cfg := RetryConfig{}
	filled := cfg.withDefaults()

	if filled.MaxRetries != defaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", filled.MaxRetries, defaultMaxRetries)
	}
	if filled.BaseDelay != defaultBaseDelay {
		t.Errorf("BaseDelay = %v, want %v", filled.BaseDelay, defaultBaseDelay)
	}
	if filled.MaxDelay != defaultMaxDelay {
		t.Errorf("MaxDelay = %v, want %v", filled.MaxDelay, defaultMaxDelay)
	}
}

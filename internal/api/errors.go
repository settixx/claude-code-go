package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	apierrors "github.com/settixx/claude-code-go/internal/errors"
)

// apiErrorBody is the JSON envelope returned by the Anthropic API on error.
type apiErrorBody struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// RateLimitInfo holds rate-limit metadata extracted from response headers.
type RateLimitInfo struct {
	RequestsRemaining int
	TokensRemaining   int
	ResetAt           time.Time
	RetryAfterSeconds int
}

// ParseAPIErrorResponse reads an HTTP response body and returns a structured APIError.
func ParseAPIErrorResponse(resp *http.Response) *apierrors.APIError {
	if resp == nil {
		return &apierrors.APIError{
			StatusCode: 0,
			Type:       "connection_error",
			Message:    "nil HTTP response",
			Retryable:  true,
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &apierrors.APIError{
			StatusCode: resp.StatusCode,
			Type:       "read_error",
			Message:    fmt.Sprintf("failed to read error body: %v", err),
			Retryable:  classifyRetryable(resp.StatusCode),
		}
	}

	return classifyError(resp.StatusCode, body, resp.Header)
}

// classifyError builds an APIError from a status code and response body,
// setting the Retryable flag based on the error type.
func classifyError(statusCode int, body []byte, headers http.Header) *apierrors.APIError {
	retryAfter := ParseRetryAfter(headers)

	var parsed apiErrorBody
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Type != "" {
		apiErr := &apierrors.APIError{
			StatusCode: statusCode,
			Type:       parsed.Error.Type,
			Message:    parsed.Error.Message,
			Retryable:  classifyRetryable(statusCode),
		}
		if retryAfter > 0 {
			storeRetryAfter(apiErr, retryAfter)
		}
		return apiErr
	}

	apiErr := &apierrors.APIError{
		StatusCode: statusCode,
		Type:       "unknown",
		Message:    truncateBody(body, 512),
		Retryable:  classifyRetryable(statusCode),
	}
	if retryAfter > 0 {
		storeRetryAfter(apiErr, retryAfter)
	}
	return apiErr
}

// classifyRetryable determines whether a request with the given status code
// should be retried. 429 (rate limit), 529 (overloaded), 408, 409, and 5xx
// are retryable; 401/403 auth errors are not.
func classifyRetryable(statusCode int) bool {
	switch statusCode {
	case 429, 529:
		return true
	case 408, 409:
		return true
	case 401, 403, 400, 404:
		return false
	default:
		return statusCode >= 500
	}
}

// ExtractRateLimitInfo parses rate-limit headers from an API response.
func ExtractRateLimitInfo(headers http.Header) *RateLimitInfo {
	info := &RateLimitInfo{}
	hasData := false

	if v := headers.Get("anthropic-ratelimit-requests-remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			info.RequestsRemaining = n
			hasData = true
		}
	}
	if v := headers.Get("anthropic-ratelimit-tokens-remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			info.TokensRemaining = n
			hasData = true
		}
	}
	if v := headers.Get("anthropic-ratelimit-tokens-reset"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			info.ResetAt = t
			hasData = true
		}
	}
	if v := headers.Get("retry-after"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			info.RetryAfterSeconds = secs
			hasData = true
		}
	}

	if !hasData {
		return nil
	}
	return info
}

// ParseRetryAfter extracts the Retry-After header value in seconds.
// Returns 0 if the header is missing or unparseable.
func ParseRetryAfter(headers http.Header) int {
	v := headers.Get("retry-after")
	if v == "" {
		return 0
	}
	secs, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return secs
}

// IsOverloadedError checks whether an error message indicates a 529 overloaded state,
// which the SDK may surface without a proper status code during streaming.
func IsOverloadedError(message string) bool {
	return strings.Contains(message, `"type":"overloaded_error"`)
}

// IsPromptTooLongError checks whether an error indicates the prompt exceeded the context limit.
func IsPromptTooLongError(message string) bool {
	return strings.Contains(strings.ToLower(message), "prompt is too long")
}

func truncateBody(body []byte, maxLen int) string {
	if len(body) <= maxLen {
		return string(body)
	}
	return string(body[:maxLen]) + "..."
}

// retryAfterStore maps APIError pointers to their Retry-After seconds.
// This avoids modifying the shared APIError struct in internal/errors.
var retryAfterStore sync.Map

func storeRetryAfter(apiErr *apierrors.APIError, seconds int) {
	retryAfterStore.Store(apiErr, seconds)
}

func loadRetryAfter(apiErr *apierrors.APIError) int {
	if v, ok := retryAfterStore.LoadAndDelete(apiErr); ok {
		return v.(int)
	}
	return 0
}

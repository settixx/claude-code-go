package notify

import "fmt"

// RateLimitInfo holds parsed rate limit details for display.
type RateLimitInfo struct {
	LimitType    string // "requests", "tokens", "daily"
	RetryAfterS  int
	CurrentUsage string // e.g. "85/100"
}

// FormatRateLimitMessage creates a user-friendly rate limit message.
func FormatRateLimitMessage(info RateLimitInfo) string {
	msg := fmt.Sprintf("⏳ Rate limited (%s)", info.LimitType)
	if info.CurrentUsage != "" {
		msg += fmt.Sprintf(" — usage: %s", info.CurrentUsage)
	}
	if info.RetryAfterS <= 0 {
		return msg
	}
	if info.RetryAfterS < 60 {
		msg += fmt.Sprintf(". Retrying in %ds...", info.RetryAfterS)
	} else {
		mins := info.RetryAfterS / 60
		secs := info.RetryAfterS % 60
		msg += fmt.Sprintf(". Retrying in %dm%ds...", mins, secs)
	}
	return msg
}

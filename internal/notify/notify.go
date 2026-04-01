package notify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Notifier sends desktop/terminal notifications after long-running operations.
type Notifier struct {
	enabled bool
}

// New creates a Notifier. Notifications are enabled by default.
func New() *Notifier {
	return &Notifier{enabled: true}
}

// SetEnabled toggles notifications on/off.
func (n *Notifier) SetEnabled(v bool) { n.enabled = v }

// Notify sends a notification through available channels.
// It tries multiple methods in order of preference.
func (n *Notifier) Notify(title, body string) {
	if !n.enabled {
		return
	}

	fmt.Fprint(os.Stderr, "\a")

	if isITerm2() {
		sendITermNotification(title, body)
		return
	}

	sendSystemNotification(title, body)
}

// NotifyCompletion is a convenience for "task complete" notifications.
func (n *Notifier) NotifyCompletion(taskDescription string) {
	n.Notify("Ti Code", fmt.Sprintf("Completed: %s", taskDescription))
}

// NotifyRateLimit sends a friendly rate limit notification.
func (n *Notifier) NotifyRateLimit(limitType string, retryAfterSecs int) {
	body := fmt.Sprintf("Rate limited (%s). Retrying in %ds...", limitType, retryAfterSecs)
	n.Notify("Ti Code — Rate Limited", body)
}

func isITerm2() bool {
	return os.Getenv("TERM_PROGRAM") == "iTerm.app"
}

func sendITermNotification(_, body string) {
	fmt.Fprintf(os.Stderr, "\033]9;%s\a", body)
}

func sendSystemNotification(title, body string) {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		_ = exec.Command("osascript", "-e", script).Run()
	case "linux":
		_ = exec.Command("notify-send", title, body).Run()
	}
}

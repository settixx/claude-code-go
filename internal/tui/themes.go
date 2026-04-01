package tui

import (
	"os"
	"strings"
	"sync"
)

// Theme defines a named color scheme with semantic color mappings.
// Each color name maps to an ANSI escape sequence.
type Theme struct {
	// Name is the human-readable theme identifier.
	Name string
	// Colors maps semantic color names (e.g. "error", "success") to ANSI codes.
	Colors map[string]string
}

// Color returns the ANSI escape code for the given semantic name,
// falling back to AnsiReset if the name is not defined.
func (t *Theme) Color(name string) string {
	if c, ok := t.Colors[name]; ok {
		return c
	}
	return AnsiReset
}

// Colorize wraps s with the named semantic color and a reset suffix.
func (t *Theme) Colorize(name, s string) string {
	return t.Color(name) + s + AnsiReset
}

var (
	activeTheme *Theme
	themeMu     sync.RWMutex
)

// ActiveTheme returns the currently active theme (never nil).
func ActiveTheme() *Theme {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if activeTheme != nil {
		return activeTheme
	}
	return defaultTheme()
}

// ApplyTheme sets the global active theme.
func ApplyTheme(theme *Theme) {
	if theme == nil {
		return
	}
	themeMu.Lock()
	defer themeMu.Unlock()
	activeTheme = theme
}

// GetTheme returns a built-in theme by name.
// Returns the default theme if the name is unrecognized.
func GetTheme(name string) *Theme {
	switch strings.ToLower(name) {
	case "dark":
		return darkTheme()
	case "light":
		return lightTheme()
	case "solarized":
		return solarizedTheme()
	default:
		return defaultTheme()
	}
}

// DetectSystemTheme checks the COLORFGBG environment variable to infer
// whether the terminal has a light or dark background.
// Returns "light" or "dark". Defaults to "dark" when detection fails.
func DetectSystemTheme() string {
	val := os.Getenv("COLORFGBG")
	if val == "" {
		return "dark"
	}
	parts := strings.Split(val, ";")
	if len(parts) < 2 {
		return "dark"
	}
	bg := strings.TrimSpace(parts[len(parts)-1])
	if bgIsLight(bg) {
		return "light"
	}
	return "dark"
}

func bgIsLight(bg string) bool {
	lightValues := map[string]bool{
		"7": true, "15": true, "white": true,
	}
	return lightValues[strings.ToLower(bg)]
}

func defaultTheme() *Theme {
	return &Theme{
		Name: "default",
		Colors: map[string]string{
			"primary":    ansiFgCyan,
			"secondary":  ansiFgBlue,
			"success":    ansiFgGreen,
			"warning":    ansiFgYellow,
			"error":      ansiFgRed,
			"info":       ansiFgBlue,
			"muted":      AnsiDim,
			"accent":     ansiFgMagenta,
			"text":       AnsiReset,
			"bold":       AnsiBold,
			"heading":    AnsiBold + AnsiUnderline,
			"prompt":     ansiFgCyan,
			"tooluse":    ansiFgYellow,
			"toolresult": AnsiDim,
			"addition":   ansiFgGreen,
			"deletion":   ansiFgRed,
			"context":    AnsiDim,
		},
	}
}

func darkTheme() *Theme {
	return &Theme{
		Name: "dark",
		Colors: map[string]string{
			"primary":    "\033[38;5;75m",
			"secondary":  "\033[38;5;111m",
			"success":    "\033[38;5;114m",
			"warning":    "\033[38;5;221m",
			"error":      "\033[38;5;203m",
			"info":       "\033[38;5;75m",
			"muted":      "\033[38;5;243m",
			"accent":     "\033[38;5;176m",
			"text":       "\033[38;5;252m",
			"bold":       AnsiBold,
			"heading":    AnsiBold + "\033[38;5;75m",
			"prompt":     "\033[38;5;75m",
			"tooluse":    "\033[38;5;221m",
			"toolresult": "\033[38;5;243m",
			"addition":   "\033[38;5;114m",
			"deletion":   "\033[38;5;203m",
			"context":    "\033[38;5;243m",
		},
	}
}

func lightTheme() *Theme {
	return &Theme{
		Name: "light",
		Colors: map[string]string{
			"primary":    "\033[38;5;25m",
			"secondary":  "\033[38;5;30m",
			"success":    "\033[38;5;28m",
			"warning":    "\033[38;5;130m",
			"error":      "\033[38;5;124m",
			"info":       "\033[38;5;25m",
			"muted":      "\033[38;5;245m",
			"accent":     "\033[38;5;90m",
			"text":       "\033[38;5;235m",
			"bold":       AnsiBold,
			"heading":    AnsiBold + "\033[38;5;25m",
			"prompt":     "\033[38;5;25m",
			"tooluse":    "\033[38;5;130m",
			"toolresult": "\033[38;5;245m",
			"addition":   "\033[38;5;28m",
			"deletion":   "\033[38;5;124m",
			"context":    "\033[38;5;245m",
		},
	}
}

func solarizedTheme() *Theme {
	return &Theme{
		Name: "solarized",
		Colors: map[string]string{
			"primary":    "\033[38;5;33m",  // blue
			"secondary":  "\033[38;5;37m",  // cyan
			"success":    "\033[38;5;64m",  // green
			"warning":    "\033[38;5;136m", // yellow
			"error":      "\033[38;5;160m", // red
			"info":       "\033[38;5;33m",  // blue
			"muted":      "\033[38;5;246m", // base0
			"accent":     "\033[38;5;125m", // magenta
			"text":       "\033[38;5;244m", // base00
			"bold":       AnsiBold,
			"heading":    AnsiBold + "\033[38;5;33m",
			"prompt":     "\033[38;5;37m",
			"tooluse":    "\033[38;5;136m",
			"toolresult": "\033[38;5;246m",
			"addition":   "\033[38;5;64m",
			"deletion":   "\033[38;5;160m",
			"context":    "\033[38;5;246m",
		},
	}
}

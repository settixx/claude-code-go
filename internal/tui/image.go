package tui

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// CanDisplayImages reports whether the terminal supports inline image display.
// Currently detects iTerm2 and Kitty via TERM_PROGRAM environment variable.
func CanDisplayImages() bool {
	term := os.Getenv("TERM_PROGRAM")
	return term == "iTerm.app" || term == "WezTerm" || isKittyTerminal()
}

func isKittyTerminal() bool {
	return os.Getenv("TERM_PROGRAM") == "kitty" || os.Getenv("TERM") == "xterm-kitty"
}

// DetectImageProtocol returns the best image protocol for the current terminal.
// Returns "iterm2", "kitty", or "fallback".
func DetectImageProtocol() string {
	term := os.Getenv("TERM_PROGRAM")
	switch {
	case term == "iTerm.app" || term == "WezTerm":
		return "iterm2"
	case isKittyTerminal():
		return "kitty"
	default:
		return "fallback"
	}
}

// RenderImageiTerm2 encodes image data using the iTerm2 inline image protocol (OSC 1337).
// The width parameter sets the display column width. Pass 0 for auto sizing.
func RenderImageiTerm2(data []byte, width int) string {
	encoded := base64.StdEncoding.EncodeToString(data)

	widthSpec := "auto"
	if width > 0 {
		widthSpec = fmt.Sprintf("%d", width)
	}

	var b strings.Builder
	b.WriteString("\033]1337;File=inline=1")
	b.WriteString(fmt.Sprintf(";width=%s", widthSpec))
	b.WriteString(fmt.Sprintf(";size=%d", len(data)))
	b.WriteString(":")
	b.WriteString(encoded)
	b.WriteString("\a")
	return b.String()
}

// RenderImageKitty encodes image data using the Kitty graphics protocol.
// Data is sent as a base64 payload with chunked transmission for large images.
// The width parameter sets the display column width. Pass 0 for auto sizing.
func RenderImageKitty(data []byte, width int) string {
	encoded := base64.StdEncoding.EncodeToString(data)

	colSpec := ""
	if width > 0 {
		colSpec = fmt.Sprintf(",c=%d", width)
	}

	const chunkSize = 4096
	chunks := splitIntoChunks(encoded, chunkSize)
	if len(chunks) == 0 {
		return ""
	}

	var b strings.Builder
	for i, chunk := range chunks {
		more := 1
		if i == len(chunks)-1 {
			more = 0
		}
		if i == 0 {
			b.WriteString(fmt.Sprintf("\033_Ga=T,f=100,m=%d%s;%s\033\\", more, colSpec, chunk))
		} else {
			b.WriteString(fmt.Sprintf("\033_Gm=%d;%s\033\\", more, chunk))
		}
	}
	return b.String()
}

// RenderImageFallback returns an ASCII placeholder for terminals that
// cannot display inline images. Shows dimensions in a bracketed format.
func RenderImageFallback(width, height int) string {
	label := fmt.Sprintf("[image %dx%d]", width, height)
	return Dim(label)
}

func splitIntoChunks(s string, size int) []string {
	if len(s) == 0 {
		return nil
	}
	chunks := make([]string, 0, (len(s)/size)+1)
	for len(s) > size {
		chunks = append(chunks, s[:size])
		s = s[size:]
	}
	if len(s) > 0 {
		chunks = append(chunks, s)
	}
	return chunks
}

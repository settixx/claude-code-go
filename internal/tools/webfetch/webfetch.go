package webfetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	toolName       = "WebFetch"
	maxBodyBytes   = 50 * 1024
	requestTimeout = 30 * time.Second
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

type Tool struct {
	toolutil.BaseTool
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"web_fetch", "fetch_url"},
			ToolSearchHint:  "fetch URL, download web page, HTTP GET",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Fetch content from a URL and return it as readable text. " +
		"HTML tags are stripped for readability. Output is truncated to 50 KB.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "The URL to fetch.",
			},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "Optional prompt to guide extraction or summarization of the page content.",
			},
		},
		Required: []string{"url"},
	}
}

func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	rawURL, err := toolutil.RequireString(input, "url")
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	body, err := fetchURL(ctx, rawURL)
	if err != nil {
		return &types.ToolResult{Data: fmt.Sprintf("Fetch failed: %v", err)}, nil
	}

	cleaned := stripHTML(body)
	cleaned = collapseWhitespace(cleaned)
	cleaned = toolutil.TruncateResult(cleaned, maxBodyBytes)

	return &types.ToolResult{Data: cleaned}, nil
}

func fetchURL(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("bad request: %w", err)
	}
	req.Header.Set("User-Agent", "TiCode/1.0")
	req.Header.Set("Accept", "text/html, text/plain, */*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	limited := io.LimitReader(resp.Body, maxBodyBytes+1024)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	return string(data), nil
}

func stripHTML(s string) string {
	s = regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)<style[^>]*>[\s\S]*?</style>`).ReplaceAllString(s, "")
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	return s
}

func collapseWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blankRun := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankRun++
			if blankRun <= 1 {
				out = append(out, "")
			}
			continue
		}
		blankRun = 0
		out = append(out, trimmed)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

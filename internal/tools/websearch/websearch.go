package websearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	toolName   = "WebSearch"
	maxResults = 8
	maxBody    = 512 * 1024
	userAgent  = "Mozilla/5.0 (compatible; TiCode/1.0)"
)

var (
	reResultLink = regexp.MustCompile(`<a[^>]+class="result__a"[^>]+href="([^"]*)"[^>]*>([^<]*)</a>`)
	reSnippet    = regexp.MustCompile(`<a[^>]+class="result__snippet"[^>]*>([\s\S]*?)</a>`)
	reHTMLTags   = regexp.MustCompile(`<[^>]*>`)
)

type searchResult struct {
	Title   string
	URL     string
	Snippet string
}

type Tool struct {
	toolutil.BaseTool
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"web_search"},
			ToolSearchHint:  "search the web, find information online",
			ReadOnly:        true,
			ConcurrencySafe: true,
		},
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Search the web for real-time information using DuckDuckGo. " +
		"Returns summarized results with titles, URLs and snippets.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query to look up on the web.",
			},
			"allowed_domains": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Optional list of domains to restrict search results to.",
			},
		},
		Required: []string{"query"},
	}
}

func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	query, err := toolutil.RequireString(input, "query")
	if err != nil {
		return nil, err
	}

	results, err := searchDuckDuckGo(ctx, query)
	if err != nil {
		return &types.ToolResult{Data: fmt.Sprintf("Search failed: %v", err)}, nil
	}
	if len(results) == 0 {
		return &types.ToolResult{Data: "No results found for: " + query}, nil
	}

	domains := extractAllowedDomains(input)
	if len(domains) > 0 {
		results = filterByDomains(results, domains)
	}

	return &types.ToolResult{Data: formatResults(query, results)}, nil
}

func searchDuckDuckGo(ctx context.Context, query string) ([]searchResult, error) {
	searchURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from DuckDuckGo", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, err
	}

	return parseResults(string(body)), nil
}

func parseResults(html string) []searchResult {
	linkMatches := reResultLink.FindAllStringSubmatch(html, -1)
	snippetMatches := reSnippet.FindAllStringSubmatch(html, -1)

	var results []searchResult
	for i, m := range linkMatches {
		if len(results) >= maxResults {
			break
		}
		r := searchResult{
			Title: cleanHTML(m[2]),
			URL:   resolveURL(m[1]),
		}
		if i < len(snippetMatches) && len(snippetMatches[i]) > 1 {
			r.Snippet = cleanHTML(snippetMatches[i][1])
		}
		if r.URL == "" || r.Title == "" {
			continue
		}
		results = append(results, r)
	}
	return results
}

func resolveURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "//duckduckgo.com/l/?uddg=") {
		if u, err := url.QueryUnescape(strings.TrimPrefix(raw, "//duckduckgo.com/l/?uddg=")); err == nil {
			if idx := strings.Index(u, "&"); idx != -1 {
				u = u[:idx]
			}
			return u
		}
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return "https://duckduckgo.com" + raw
	}
	return raw
}

func cleanHTML(s string) string {
	s = reHTMLTags.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#x27;", "'")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.TrimSpace(s)
	return s
}

func formatResults(query string, results []searchResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))
	for i, r := range results {
		b.WriteString(fmt.Sprintf("%d. %s\n   %s\n", i+1, r.Title, r.URL))
		if r.Snippet != "" {
			b.WriteString(fmt.Sprintf("   %s\n", r.Snippet))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func extractAllowedDomains(input map[string]interface{}) []string {
	raw, ok := input["allowed_domains"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	domains := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok && s != "" {
			domains = append(domains, strings.ToLower(s))
		}
	}
	return domains
}

func filterByDomains(results []searchResult, domains []string) []searchResult {
	filtered := make([]searchResult, 0, len(results))
	for _, r := range results {
		for _, d := range domains {
			if strings.Contains(strings.ToLower(r.URL), d) {
				filtered = append(filtered, r)
				break
			}
		}
	}
	return filtered
}

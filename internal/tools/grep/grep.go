package grep

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	toolName       = "Grep"
	maxResultChars = 20_000
	defaultLimit   = 250
)

var skipDirs = map[string]bool{
	".git":         true,
	".svn":         true,
	".hg":          true,
	"node_modules": true,
	".bzr":         true,
	".jj":          true,
}

type Tool struct {
	toolutil.BaseTool
}

func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"grep", "search", "Search"},
			ToolSearchHint:  "search file contents with regex",
			ToolMaxChars:    maxResultChars,
			ReadOnly:        true,
			Destructive:     false,
			ConcurrencySafe: true,
		},
	}
}

func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Search file contents using a regular expression pattern. " +
		"Returns matching lines in file:line:content format. " +
		"Walks the directory tree, skipping VCS and node_modules dirs.", nil
}

func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The regular expression pattern to search for",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File or directory to search in (default: cwd)",
			},
			"include": map[string]interface{}{
				"type":        "string",
				"description": "Glob pattern to filter files (e.g. *.go, *.ts)",
			},
			"-A": map[string]interface{}{
				"type":        "integer",
				"description": "Number of lines to show after each match",
			},
			"-B": map[string]interface{}{
				"type":        "integer",
				"description": "Number of lines to show before each match",
			},
			"-C": map[string]interface{}{
				"type":        "integer",
				"description": "Number of lines to show before and after each match",
			},
			"output_mode": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"content", "files_with_matches", "count"},
				"description": "Output mode: content (default), files_with_matches, or count",
			},
			"multiline": map[string]interface{}{
				"type":        "boolean",
				"description": "Enable multiline matching (dot matches newlines)",
			},
			"head_limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of matches to return",
			},
			"-i": map[string]interface{}{
				"type":        "boolean",
				"description": "Case insensitive search",
			},
		},
		Required: []string{"pattern"},
	}
}

func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	pattern, err := toolutil.RequireString(input, "pattern")
	if err != nil {
		return nil, err
	}
	searchPath := toolutil.OptionalString(input, "path", ".")
	include := toolutil.OptionalString(input, "include", "")
	outputMode := toolutil.OptionalString(input, "output_mode", "content")
	multiline := toolutil.OptionalBool(input, "multiline", false)
	caseInsensitive := toolutil.OptionalBool(input, "-i", false)
	headLimit := toolutil.OptionalInt(input, "head_limit", 0)
	afterCtx := toolutil.OptionalInt(input, "-A", 0)
	beforeCtx := toolutil.OptionalInt(input, "-B", 0)
	bothCtx := toolutil.OptionalInt(input, "-C", 0)

	if bothCtx > 0 {
		if afterCtx == 0 {
			afterCtx = bothCtx
		}
		if beforeCtx == 0 {
			beforeCtx = bothCtx
		}
	}

	if !filepath.IsAbs(searchPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to determine working directory: %w", err)
		}
		searchPath = filepath.Join(cwd, searchPath)
	}

	rePattern := pattern
	if caseInsensitive {
		rePattern = "(?i)" + rePattern
	}
	if multiline {
		rePattern = "(?s)" + rePattern
	}
	re, err := regexp.Compile(rePattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	info, err := os.Stat(searchPath)
	if err != nil {
		return nil, fmt.Errorf("path %q does not exist: %w", searchPath, err)
	}

	opts := &searchOpts{
		re:         re,
		include:    include,
		outputMode: outputMode,
		afterCtx:   afterCtx,
		beforeCtx:  beforeCtx,
		headLimit:  headLimit,
		multiline:  multiline,
	}

	var results []string
	if info.IsDir() {
		results, err = searchDir(ctx, searchPath, opts)
	} else {
		results, err = searchSingleFile(ctx, searchPath, opts)
	}
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &types.ToolResult{Data: "No matches found"}, nil
	}

	limit := defaultLimit
	if headLimit > 0 && headLimit < limit {
		limit = headLimit
	}
	truncated := len(results) > limit
	if truncated {
		results = results[:limit]
	}

	result := strings.Join(results, "\n")
	if truncated {
		result += fmt.Sprintf("\n\n[Showing first %d matches; results truncated]", limit)
	}

	result = toolutil.TruncateResult(result, maxResultChars)
	return &types.ToolResult{Data: result}, nil
}

type searchOpts struct {
	re         *regexp.Regexp
	include    string
	outputMode string
	afterCtx   int
	beforeCtx  int
	headLimit  int
	multiline  bool
}

func searchDir(ctx context.Context, root string, opts *searchOpts) ([]string, error) {
	var allMatches []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if info.IsDir() && skipDirs[info.Name()] {
			return filepath.SkipDir
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		if !matchesInclude(path, opts.include) {
			return nil
		}

		fileMatches, err := searchSingleFile(ctx, path, opts)
		if err != nil {
			return nil
		}
		allMatches = append(allMatches, fileMatches...)
		return nil
	})

	return allMatches, err
}

func searchSingleFile(ctx context.Context, path string, opts *searchOpts) ([]string, error) {
	if opts.multiline {
		return searchFileMultiline(ctx, path, opts)
	}
	return searchFileLine(ctx, path, opts)
}

func searchFileLine(ctx context.Context, path string, opts *searchOpts) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	var allLines []string
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 256*1024)
	scanner.Buffer(buf, 2*1024*1024)

	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	switch opts.outputMode {
	case "files_with_matches":
		return searchFileMatchOnly(path, allLines, opts)
	case "count":
		return searchFileCount(ctx, path, allLines, opts)
	default:
		return searchFileContent(ctx, path, allLines, opts)
	}
}

func searchFileMatchOnly(path string, lines []string, opts *searchOpts) ([]string, error) {
	for _, line := range lines {
		if opts.re.MatchString(line) {
			return []string{path}, nil
		}
	}
	return nil, nil
}

func searchFileCount(_ context.Context, path string, lines []string, opts *searchOpts) ([]string, error) {
	count := 0
	for _, line := range lines {
		if opts.re.MatchString(line) {
			count++
		}
	}
	if count == 0 {
		return nil, nil
	}
	return []string{fmt.Sprintf("%s:%d", path, count)}, nil
}

func searchFileContent(ctx context.Context, path string, lines []string, opts *searchOpts) ([]string, error) {
	needContext := opts.beforeCtx > 0 || opts.afterCtx > 0

	if !needContext {
		return searchFileSimple(ctx, path, lines, opts)
	}
	return searchFileWithContext(ctx, path, lines, opts)
}

func searchFileSimple(ctx context.Context, path string, lines []string, opts *searchOpts) ([]string, error) {
	var matches []string
	for i, line := range lines {
		select {
		case <-ctx.Done():
			return matches, ctx.Err()
		default:
		}
		if opts.re.MatchString(line) {
			matches = append(matches, fmt.Sprintf("%s:%d:%s", path, i+1, line))
			if opts.headLimit > 0 && len(matches) >= opts.headLimit {
				break
			}
		}
	}
	return matches, nil
}

func searchFileWithContext(ctx context.Context, path string, lines []string, opts *searchOpts) ([]string, error) {
	var matches []string
	printed := make(map[int]bool)
	lastPrinted := -2

	matchLines := findMatchingLineNums(lines, opts)

	for _, lineIdx := range matchLines {
		select {
		case <-ctx.Done():
			return matches, ctx.Err()
		default:
		}

		start := lineIdx - opts.beforeCtx
		if start < 0 {
			start = 0
		}
		end := lineIdx + opts.afterCtx
		if end >= len(lines) {
			end = len(lines) - 1
		}

		if lastPrinted >= 0 && start > lastPrinted+1 {
			matches = append(matches, "--")
		}

		for i := start; i <= end; i++ {
			if printed[i] {
				continue
			}
			printed[i] = true
			sep := "-"
			if i == lineIdx {
				sep = ":"
			}
			matches = append(matches, fmt.Sprintf("%s%s%d%s%s", path, sep, i+1, sep, lines[i]))
			lastPrinted = i
		}
	}
	return matches, nil
}

func findMatchingLineNums(lines []string, opts *searchOpts) []int {
	var nums []int
	for i, line := range lines {
		if opts.re.MatchString(line) {
			nums = append(nums, i)
			if opts.headLimit > 0 && len(nums) >= opts.headLimit {
				break
			}
		}
	}
	return nums
}

func searchFileMultiline(_ context.Context, path string, opts *searchOpts) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	content := string(data)

	switch opts.outputMode {
	case "files_with_matches":
		if opts.re.MatchString(content) {
			return []string{path}, nil
		}
		return nil, nil
	case "count":
		locs := opts.re.FindAllStringIndex(content, -1)
		if len(locs) == 0 {
			return nil, nil
		}
		return []string{fmt.Sprintf("%s:%d", path, len(locs))}, nil
	default:
		return searchMultilineContent(path, content, opts)
	}
}

func searchMultilineContent(path, content string, opts *searchOpts) ([]string, error) {
	locs := opts.re.FindAllStringIndex(content, -1)
	if opts.headLimit > 0 && len(locs) > opts.headLimit {
		locs = locs[:opts.headLimit]
	}

	var matches []string
	for _, loc := range locs {
		lineNum := strings.Count(content[:loc[0]], "\n") + 1
		matchText := content[loc[0]:loc[1]]
		if len(matchText) > 200 {
			matchText = matchText[:200] + "..."
		}
		matchText = strings.ReplaceAll(matchText, "\n", "\\n")
		matches = append(matches, fmt.Sprintf("%s:%d:%s", path, lineNum, matchText))
	}
	return matches, nil
}

func matchesInclude(path, include string) bool {
	if include == "" {
		return true
	}
	base := filepath.Base(path)
	ok, _ := filepath.Match(include, base)
	return ok
}

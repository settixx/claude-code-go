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

// directories that are always skipped during tree walk.
var skipDirs = map[string]bool{
	".git":         true,
	".svn":         true,
	".hg":          true,
	"node_modules": true,
	".bzr":         true,
	".jj":          true,
}

// Tool searches file contents with regular expressions.
type Tool struct {
	toolutil.BaseTool
}

// New creates a ready-to-use GrepTool.
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

// Description returns a human-readable description for the model.
func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Search file contents using a regular expression pattern. " +
		"Returns matching lines in file:line:content format. " +
		"Walks the directory tree, skipping VCS and node_modules dirs.", nil
}

// InputSchema returns the JSON Schema for the tool's input.
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
		},
		Required: []string{"pattern"},
	}
}

// Call walks the directory tree and returns matching lines.
func (t *Tool) Call(ctx context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	pattern, err := toolutil.RequireString(input, "pattern")
	if err != nil {
		return nil, err
	}
	searchPath := toolutil.OptionalString(input, "path", ".")
	include := toolutil.OptionalString(input, "include", "")

	if !filepath.IsAbs(searchPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to determine working directory: %w", err)
		}
		searchPath = filepath.Join(cwd, searchPath)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	info, err := os.Stat(searchPath)
	if err != nil {
		return nil, fmt.Errorf("path %q does not exist: %w", searchPath, err)
	}

	var matches []string
	if info.IsDir() {
		matches, err = searchDir(ctx, searchPath, re, include)
	} else {
		matches, err = searchFile(ctx, searchPath, re)
	}
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return &types.ToolResult{Data: "No matches found"}, nil
	}

	truncated := len(matches) > defaultLimit
	if truncated {
		matches = matches[:defaultLimit]
	}

	result := strings.Join(matches, "\n")
	if truncated {
		result += fmt.Sprintf("\n\n[Showing first %d matches; results truncated]", defaultLimit)
	}

	result = toolutil.TruncateResult(result, maxResultChars)
	return &types.ToolResult{Data: result}, nil
}

func searchDir(ctx context.Context, root string, re *regexp.Regexp, include string) ([]string, error) {
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

		if !matchesInclude(path, include) {
			return nil
		}

		fileMatches, err := searchFile(ctx, path, re)
		if err != nil {
			return nil
		}
		allMatches = append(allMatches, fileMatches...)
		return nil
	})

	return allMatches, err
}

func searchFile(ctx context.Context, path string, re *regexp.Regexp) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	var matches []string
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 256*1024)
	scanner.Buffer(buf, 2*1024*1024)
	lineNum := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return matches, ctx.Err()
		default:
		}

		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, fmt.Sprintf("%s:%d:%s", path, lineNum, line))
		}
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

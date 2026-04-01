package glob

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/settixx/claude-code-go/internal/tools/toolutil"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	toolName       = "Glob"
	maxResultChars = 100_000
	maxResults     = 100
)

// Tool finds files matching a glob pattern.
type Tool struct {
	toolutil.BaseTool
}

// New creates a ready-to-use GlobTool.
func New() *Tool {
	return &Tool{
		BaseTool: toolutil.BaseTool{
			ToolName:        toolName,
			ToolAliases:     []string{"glob", "find_files"},
			ToolSearchHint:  "find files by name pattern or wildcard",
			ToolMaxChars:    maxResultChars,
			ReadOnly:        true,
			Destructive:     false,
			ConcurrencySafe: true,
		},
	}
}

// Description returns a human-readable description for the model.
func (t *Tool) Description(_ map[string]interface{}) (string, error) {
	return "Find files matching a glob pattern, sorted by modification time (newest first). " +
		"Supports ** for recursive matching. Results are limited to 100 files.", nil
}

// InputSchema returns the JSON Schema for the tool's input.
func (t *Tool) InputSchema() types.ToolInputSchema {
	return types.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The glob pattern to match files against",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Root directory to search in (default: cwd)",
			},
		},
		Required: []string{"pattern"},
	}
}

// Call finds files matching the glob pattern and returns them sorted by mtime.
func (t *Tool) Call(_ context.Context, input map[string]interface{}) (*types.ToolResult, error) {
	pattern, err := toolutil.RequireString(input, "pattern")
	if err != nil {
		return nil, err
	}
	rootDir := toolutil.OptionalString(input, "path", ".")

	if !filepath.IsAbs(rootDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to determine working directory: %w", err)
		}
		rootDir = filepath.Join(cwd, rootDir)
	}

	matches, truncated, err := findGlob(rootDir, pattern)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return &types.ToolResult{Data: "No files found"}, nil
	}

	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(m)
		sb.WriteByte('\n')
	}
	if truncated {
		sb.WriteString("(Results are truncated. Consider using a more specific path or pattern.)\n")
	}

	result := toolutil.TruncateResult(sb.String(), maxResultChars)
	return &types.ToolResult{Data: result}, nil
}

// fileWithTime pairs a path with its modification timestamp.
type fileWithTime struct {
	path    string
	modTime int64
}

// findGlob walks rootDir, matches files against pattern using recursive glob,
// and returns paths sorted by modification time (newest first).
func findGlob(rootDir, pattern string) ([]string, bool, error) {
	if !strings.HasPrefix(pattern, "**/") && !filepath.IsAbs(pattern) {
		pattern = "**/" + pattern
	}

	var files []fileWithTime

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if shouldSkipDir(info) {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}

		if matchGlob(pattern, rel) {
			files = append(files, fileWithTime{
				path:    path,
				modTime: info.ModTime().UnixNano(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to walk directory %q: %w", rootDir, err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	truncated := len(files) > maxResults
	if truncated {
		files = files[:maxResults]
	}

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.path
	}
	return paths, truncated, nil
}

func shouldSkipDir(info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	name := info.Name()
	return name == ".git" || name == "node_modules" || name == ".svn" || name == ".hg"
}

// matchGlob performs simple glob matching supporting *, ?, and ** (doublestar).
func matchGlob(pattern, name string) bool {
	if !strings.Contains(pattern, "**") {
		ok, _ := filepath.Match(pattern, name)
		return ok
	}
	return matchDoublestar(pattern, name)
}

// matchDoublestar handles ** recursive patterns by splitting on ** segments.
func matchDoublestar(pattern, name string) bool {
	parts := strings.SplitN(pattern, "**", 2)
	if len(parts) != 2 {
		ok, _ := filepath.Match(pattern, name)
		return ok
	}

	prefix := parts[0]
	suffix := strings.TrimPrefix(parts[1], "/")
	suffix = strings.TrimPrefix(suffix, string(filepath.Separator))

	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/")
		prefix = strings.TrimSuffix(prefix, string(filepath.Separator))
	}

	if prefix != "" && !hasPrefix(name, prefix) {
		return false
	}

	if suffix == "" {
		return true
	}

	segments := splitPath(name)
	for i := 0; i <= len(segments); i++ {
		remaining := filepath.Join(segments[i:]...)
		if matchGlob(suffix, remaining) {
			return true
		}
	}
	return false
}

func hasPrefix(name, prefix string) bool {
	ok, _ := filepath.Match(prefix, splitPath(name)[0])
	return ok || strings.HasPrefix(name, prefix+"/") || strings.HasPrefix(name, prefix+string(filepath.Separator))
}

func splitPath(p string) []string {
	return strings.FieldsFunc(p, func(r rune) bool {
		return r == '/' || r == filepath.Separator
	})
}

package fileread

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line one\nline two\nline three\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	tool := New()
	ctx := context.Background()

	result, err := tool.Call(ctx, map[string]interface{}{
		"file_path": path,
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	data, ok := result.Data.(string)
	if !ok {
		t.Fatalf("result.Data is %T, want string", result.Data)
	}

	if !strings.Contains(data, "line one") {
		t.Errorf("output should contain 'line one', got %q", data)
	}
	if !strings.Contains(data, "line three") {
		t.Errorf("output should contain 'line three', got %q", data)
	}
}

func TestReadFileLineNumberFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "numbered.txt")
	content := "alpha\nbeta\ngamma\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	tool := New()
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"file_path": path,
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	data := result.Data.(string)
	lines := strings.Split(strings.TrimRight(data, "\n"), "\n")

	for _, line := range lines {
		if !strings.Contains(line, "|") {
			t.Errorf("line %q missing pipe separator", line)
		}
	}

	if !strings.Contains(lines[0], "1|") || !strings.Contains(lines[0], "alpha") {
		t.Errorf("first line = %q, want line number 1 with 'alpha'", lines[0])
	}
}

func TestReadFileWithOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "offset.txt")
	content := "a\nb\nc\nd\ne\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	tool := New()
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"file_path": path,
		"offset":    float64(3),
		"limit":     float64(2),
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	data := result.Data.(string)
	if !strings.Contains(data, "c") {
		t.Errorf("output should contain 'c' (line 3), got %q", data)
	}
	if !strings.Contains(data, "d") {
		t.Errorf("output should contain 'd' (line 4), got %q", data)
	}
	if strings.Contains(data, "e") {
		t.Errorf("output should not contain 'e' (beyond limit), got %q", data)
	}
}

func TestReadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	tool := New()
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"file_path": path,
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	data := result.Data.(string)
	if data != "File is empty." {
		t.Errorf("output = %q, want %q", data, "File is empty.")
	}
}

func TestReadNonexistentFile(t *testing.T) {
	tool := New()
	_, err := tool.Call(context.Background(), map[string]interface{}{
		"file_path": "/nonexistent/path/to/file.txt",
	})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReadToolMetadata(t *testing.T) {
	tool := New()
	if tool.Name() != "FileRead" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "FileRead")
	}
	if !tool.IsReadOnly(nil) {
		t.Error("FileRead should be read-only")
	}
	if !tool.IsConcurrencySafe(nil) {
		t.Error("FileRead should be concurrency-safe")
	}
}

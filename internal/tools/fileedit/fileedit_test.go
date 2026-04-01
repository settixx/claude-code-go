package fileedit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBasicReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	original := "hello world\nfoo bar\nbaz qux\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	tool := New()
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"file_path":  path,
		"old_string": "foo bar",
		"new_string": "FOO BAR",
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	data := result.Data.(string)
	if !strings.Contains(data, "updated successfully") {
		t.Errorf("result = %q, want success message", data)
	}

	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "FOO BAR") {
		t.Error("file should contain 'FOO BAR' after edit")
	}
	if strings.Contains(string(content), "foo bar") {
		t.Error("file should not contain 'foo bar' after edit")
	}
}

func TestUniquenessCheckRejectsMultipleMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dup.txt")
	content := "abc\nabc\nabc\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	tool := New()
	_, err := tool.Call(context.Background(), map[string]interface{}{
		"file_path":  path,
		"old_string": "abc",
		"new_string": "xyz",
	})

	if err == nil {
		t.Fatal("expected error for multiple occurrences without replace_all")
	}
	if !strings.Contains(err.Error(), "3 occurrences") {
		t.Errorf("error = %q, want mention of '3 occurrences'", err.Error())
	}
}

func TestReplaceAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "replall.txt")
	content := "aaa bbb aaa ccc aaa\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	tool := New()
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"file_path":   path,
		"old_string":  "aaa",
		"new_string":  "XXX",
		"replace_all": true,
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	data := result.Data.(string)
	if !strings.Contains(data, "3 occurrences") {
		t.Errorf("result = %q, want mention of 3 occurrences", data)
	}

	content2, _ := os.ReadFile(path)
	if strings.Contains(string(content2), "aaa") {
		t.Error("file should not contain 'aaa' after replace_all")
	}
}

func TestOldStringNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notfound.txt")
	if err := os.WriteFile(path, []byte("existing content"), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	tool := New()
	_, err := tool.Call(context.Background(), map[string]interface{}{
		"file_path":  path,
		"old_string": "nonexistent string",
		"new_string": "replacement",
	})
	if err == nil {
		t.Error("expected error when old_string not found")
	}
}

func TestIdenticalStringsRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "same.txt")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	tool := New()
	_, err := tool.Call(context.Background(), map[string]interface{}{
		"file_path":  path,
		"old_string": "content",
		"new_string": "content",
	})
	if err == nil {
		t.Error("expected error when old_string == new_string")
	}
}

func TestCreateNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "new.txt")

	tool := New()
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"file_path":  path,
		"old_string": "",
		"new_string": "brand new content",
	})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	data := result.Data.(string)
	if !strings.Contains(data, "created successfully") {
		t.Errorf("result = %q, want creation message", data)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(content) != "brand new content" {
		t.Errorf("file content = %q, want %q", string(content), "brand new content")
	}
}

func TestFileEditMetadata(t *testing.T) {
	tool := New()
	if tool.Name() != "FileEdit" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "FileEdit")
	}
	if tool.IsReadOnly(nil) {
		t.Error("FileEdit should not be read-only")
	}
}

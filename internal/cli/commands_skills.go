package cli

import (
	"fmt"
	"os"

	"github.com/settixx/claude-code-go/internal/tui"
)

func cmdSkills(_ string, _ *CommandContext) error {
	skills := []struct {
		name string
		desc string
	}{
		{"file_read", "Read file contents"},
		{"file_write", "Write or create files"},
		{"file_edit", "Apply surgical edits to files"},
		{"bash", "Execute shell commands"},
		{"grep", "Search file contents with regex"},
		{"glob", "Find files by pattern"},
		{"web_search", "Search the web"},
		{"web_fetch", "Fetch a URL"},
		{"notebook_edit", "Edit Jupyter notebooks"},
	}

	fmt.Fprintln(os.Stdout, tui.Bold("Available skills:"))
	fmt.Fprintln(os.Stdout)
	for _, s := range skills {
		fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Cyan(s.name), tui.Dim(s.desc))
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, tui.Dim("Skills are enabled based on the current permission mode."))
	return nil
}

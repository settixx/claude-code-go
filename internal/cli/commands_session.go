package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/settixx/claude-code-go/internal/interfaces"
	"github.com/settixx/claude-code-go/internal/storage"
	"github.com/settixx/claude-code-go/internal/tui"
	"github.com/settixx/claude-code-go/internal/types"
)

func registerSessionCommands(reg *CommandRegistry) {
	reg.Register(&Command{
		Name:        "resume",
		Description: "List and resume previous sessions",
		Handler:     cmdResume,
	})
	reg.Register(&Command{
		Name:        "session",
		Description: "Session management (list, info)",
		Handler:     cmdSession,
	})
	reg.Register(&Command{
		Name:        "export",
		Description: "Export conversation to file",
		Handler:     cmdExport,
	})
	reg.Register(&Command{
		Name:        "memory",
		Description: "Show or edit memory files",
		Handler:     cmdMemory,
	})
}

func cmdResume(args string, ctx *CommandContext) error {
	if ctx.Storage == nil {
		fmt.Fprintln(os.Stdout, tui.Red("No storage backend configured."))
		return nil
	}

	sessions, err := ctx.Storage.List()
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}
	if len(sessions) == 0 {
		fmt.Fprintln(os.Stdout, tui.Dim("No previous sessions found."))
		return nil
	}

	if args != "" {
		return resumeByArg(args, sessions, ctx)
	}

	fmt.Fprintln(os.Stdout, tui.Bold("Recent sessions:"))
	fmt.Fprintln(os.Stdout)
	limit := min(len(sessions), 10)
	for i, s := range sessions[:limit] {
		ts := time.Unix(s.UpdatedAt, 0).Format("2006-01-02 15:04")
		title := s.Title
		if title == "" {
			title = "(no title)"
		}
		title = tui.Truncate(title, 60)
		fmt.Fprintf(os.Stdout, "  %s  %s  %s  %s\n",
			tui.Cyan(fmt.Sprintf("[%d]", i+1)),
			tui.Dim(ts),
			tui.Dim(fmt.Sprintf("%3d msgs", s.MessageCount)),
			title,
		)
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, tui.Dim("Use /resume <number> or /resume <session-id> to load a session."))
	return nil
}

func resumeByArg(args string, sessions []interfaces.SessionInfo, ctx *CommandContext) error {
	if idx, err := strconv.Atoi(args); err == nil && idx >= 1 && idx <= len(sessions) {
		return loadSession(sessions[idx-1].ID, ctx)
	}

	sid := types.SessionId(args)
	msgs, err := ctx.Storage.Load(sid)
	if err != nil {
		return fmt.Errorf("load session %s: %w", args, err)
	}
	ctx.SessionID = args
	ctx.Messages = msgs
	ctx.MessageCount = len(msgs)
	fmt.Fprintf(os.Stdout, "%s Resumed session %s (%d messages)\n",
		tui.Green("✓"), tui.Cyan(args), len(msgs))
	return nil
}

func loadSession(sid types.SessionId, ctx *CommandContext) error {
	msgs, err := ctx.Storage.Load(sid)
	if err != nil {
		return fmt.Errorf("load session %s: %w", string(sid), err)
	}
	ctx.SessionID = string(sid)
	ctx.Messages = msgs
	ctx.MessageCount = len(msgs)
	fmt.Fprintf(os.Stdout, "%s Resumed session %s (%d messages)\n",
		tui.Green("✓"), tui.Cyan(string(sid)), len(msgs))
	return nil
}

func cmdSession(_ string, ctx *CommandContext) error {
	sid := ctx.SessionID
	if sid == "" {
		fmt.Fprintln(os.Stdout, tui.Dim("No active session."))
		return nil
	}

	startStr := "(unknown)"
	if !ctx.StartTime.IsZero() {
		startStr = ctx.StartTime.Format("2006-01-02 15:04:05")
	}

	fmt.Fprintln(os.Stdout, tui.Bold("Session details:"))
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("session_id"), sid)
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("started"), startStr)
	fmt.Fprintf(os.Stdout, "  %-18s %d\n", tui.Dim("messages"), ctx.MessageCount)
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("model"), ctx.Model)
	fmt.Fprintf(os.Stdout, "  %-18s %d in / %d out\n", tui.Dim("tokens"), ctx.TokensIn, ctx.TokensOut)
	fmt.Fprintf(os.Stdout, "  %-18s %s\n", tui.Dim("estimated cost"), tui.FormatCost(ctx.CostUSD))
	return nil
}

func cmdExport(args string, ctx *CommandContext) error {
	if len(ctx.Messages) == 0 {
		fmt.Fprintln(os.Stdout, tui.Dim("No messages to export."))
		return nil
	}

	format, outPath := parseExportArgs(args, ctx.SessionID)

	switch format {
	case "json":
		return exportJSON(outPath, ctx.Messages)
	default:
		return exportMarkdown(outPath, ctx.Messages)
	}
}

func parseExportArgs(args, sessionID string) (format, outPath string) {
	format = "markdown"
	parts := strings.Fields(args)

	for _, p := range parts {
		switch strings.ToLower(p) {
		case "json":
			format = "json"
		case "markdown", "md":
			format = "markdown"
		default:
			outPath = p
		}
	}

	if outPath != "" {
		return format, outPath
	}

	sid := sessionID
	if sid == "" {
		sid = "conversation"
	}
	ext := "md"
	if format == "json" {
		ext = "json"
	}
	return format, fmt.Sprintf("./session-%s.%s", sid, ext)
}

func exportMarkdown(path string, msgs []types.Message) error {
	var b strings.Builder
	b.WriteString("# Conversation Export\n\n")

	for _, msg := range msgs {
		role := roleLabel(msg)
		b.WriteString(fmt.Sprintf("## %s\n\n", role))

		if msg.Text != "" {
			b.WriteString(msg.Text)
			b.WriteString("\n\n")
			continue
		}
		for _, block := range msg.Content {
			if block.Text != "" {
				b.WriteString(block.Text)
				b.WriteString("\n\n")
			}
		}
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write export: %w", err)
	}
	fmt.Fprintf(os.Stdout, "%s Exported %d messages to %s\n",
		tui.Green("✓"), len(msgs), tui.Cyan(path))
	return nil
}

func exportJSON(path string, msgs []types.Message) error {
	data, err := json.MarshalIndent(msgs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write export: %w", err)
	}
	fmt.Fprintf(os.Stdout, "%s Exported %d messages to %s\n",
		tui.Green("✓"), len(msgs), tui.Cyan(path))
	return nil
}

func roleLabel(msg types.Message) string {
	if msg.Role != "" {
		return capitalizeFirst(msg.Role)
	}
	switch msg.Type {
	case types.MsgUser:
		return "User"
	case types.MsgAssistant:
		return "Assistant"
	case types.MsgSystem:
		return "System"
	default:
		return string(msg.Type)
	}
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func cmdMemory(args string, ctx *CommandContext) error {
	dir := ctx.CWD
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
	}

	if args == "" {
		return showMemory(dir)
	}
	return appendMemory(dir, args)
}

func showMemory(dir string) error {
	content, err := storage.LoadMemory(dir)
	if err != nil {
		return fmt.Errorf("read memory: %w", err)
	}
	if content == "" {
		fmt.Fprintln(os.Stdout, tui.Dim("No memory file found. Use /memory <text> to create one."))
		fmt.Fprintf(os.Stdout, tui.Dim("  Path: %s\n"), filepath.Join(dir, ".claude", "memory.md"))
		return nil
	}
	fmt.Fprintln(os.Stdout, tui.Bold("Memory contents:"))
	fmt.Fprintln(os.Stdout, tui.HorizontalRule())
	fmt.Fprintln(os.Stdout, content)
	fmt.Fprintln(os.Stdout, tui.HorizontalRule())
	return nil
}

func appendMemory(dir, text string) error {
	existing, err := storage.LoadMemory(dir)
	if err != nil {
		return fmt.Errorf("read memory: %w", err)
	}

	newContent := existing
	if newContent != "" && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += text + "\n"

	if err := storage.SaveMemory(dir, newContent); err != nil {
		return fmt.Errorf("write memory: %w", err)
	}
	fmt.Fprintf(os.Stdout, "%s Memory updated: %s\n",
		tui.Green("✓"), filepath.Join(dir, ".claude", "memory.md"))
	return nil
}

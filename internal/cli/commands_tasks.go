package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/settixx/claude-code-go/internal/tui"
	"github.com/settixx/claude-code-go/internal/types"
)

func cmdTasks(_ string, ctx *CommandContext) error {
	if ctx.StateStore == nil {
		fmt.Fprintln(os.Stdout, tui.Dim("(Not yet connected to a task store)"))
		return nil
	}

	appState := ctx.StateStore.Get()
	if len(appState.Tasks) == 0 {
		fmt.Fprintln(os.Stdout, tui.Dim("No tasks."))
		return nil
	}

	tasks := sortedTasks(appState.Tasks)

	fmt.Fprintln(os.Stdout, tui.Bold("Tasks:"))
	fmt.Fprintln(os.Stdout)
	for _, t := range tasks {
		icon := taskStatusIcon(t.Status)
		name := t.Name
		if name == "" {
			name = t.ID
		}
		fmt.Fprintf(os.Stdout, "  %s %-12s %-16s %s\n",
			icon, tui.Dim(string(t.Kind)), tui.Cyan(name), tui.Dim(string(t.Status)))
	}
	fmt.Fprintln(os.Stdout)
	return nil
}

func sortedTasks(m map[string]*types.TaskState) []*types.TaskState {
	tasks := make([]*types.TaskState, 0, len(m))
	for _, t := range m {
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks
}

func taskStatusIcon(status types.TaskStatus) string {
	switch status {
	case types.TaskRunning:
		return tui.Green("●")
	case types.TaskPending:
		return tui.Yellow("○")
	case types.TaskComplete:
		return tui.Green("✓")
	case types.TaskFailed:
		return tui.Red("✗")
	case types.TaskStopped:
		return tui.Dim("■")
	default:
		return tui.Dim("?")
	}
}

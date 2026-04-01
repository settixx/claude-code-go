package tui

// Modifier represents keyboard modifier keys.
type Modifier int

const (
	// ModNone indicates no modifier key.
	ModNone Modifier = 0
	// ModCtrl indicates the Control key.
	ModCtrl Modifier = 1 << iota
	// ModAlt indicates the Alt/Option key.
	ModAlt
	// ModShift indicates the Shift key.
	ModShift
)

// String returns a human-readable representation of the modifier.
func (m Modifier) String() string {
	parts := make([]string, 0, 3)
	if m&ModCtrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if m&ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	if m&ModShift != 0 {
		parts = append(parts, "Shift")
	}
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += "+" + p
	}
	return result
}

// Action identifies a named editor action triggered by a keybinding.
type Action string

const (
	// ActionInterrupt sends SIGINT to the current process.
	ActionInterrupt Action = "interrupt"
	// ActionExit cleanly exits the application.
	ActionExit Action = "exit"
	// ActionClear clears the terminal screen.
	ActionClear Action = "clear"
	// ActionClearLine clears the current input line.
	ActionClearLine Action = "clear_line"
	// ActionHistoryPrev navigates to the previous history entry.
	ActionHistoryPrev Action = "history_prev"
	// ActionHistoryNext navigates to the next history entry.
	ActionHistoryNext Action = "history_next"
	// ActionComplete triggers tab completion.
	ActionComplete Action = "complete"
	// ActionNormalMode switches to vim normal mode.
	ActionNormalMode Action = "normal_mode"
	// ActionInsertMode switches to vim insert mode.
	ActionInsertMode Action = "insert_mode"
	// ActionDeleteLine deletes the current line (vim dd).
	ActionDeleteLine Action = "delete_line"
)

// Keybinding associates a key combination with an action.
type Keybinding struct {
	// Key is the primary key name (e.g. "c", "d", "Up", "Tab", "Escape").
	Key string
	// Modifier holds modifier flags (Ctrl, Alt, Shift).
	Modifier Modifier
	// Action is the named action to execute.
	Action Action
	// Description is a human-readable explanation for help display.
	Description string
}

// Label returns a display string like "Ctrl+C" or "Tab".
func (kb Keybinding) Label() string {
	mod := kb.Modifier.String()
	if mod == "" {
		return kb.Key
	}
	return mod + "+" + kb.Key
}

// Keymap holds an ordered collection of keybindings.
type Keymap struct {
	// Name identifies this keymap (e.g. "default", "vim").
	Name     string
	bindings []Keybinding
}

// Bindings returns a copy of all keybindings in this keymap.
func (km *Keymap) Bindings() []Keybinding {
	out := make([]Keybinding, len(km.bindings))
	copy(out, km.bindings)
	return out
}

// Lookup returns the first keybinding matching key and modifier, if any.
func (km *Keymap) Lookup(key string, mod Modifier) (Keybinding, bool) {
	for _, kb := range km.bindings {
		if kb.Key == key && kb.Modifier == mod {
			return kb, true
		}
	}
	return Keybinding{}, false
}

// LookupAction returns the first keybinding for the given action, if any.
func (km *Keymap) LookupAction(action Action) (Keybinding, bool) {
	for _, kb := range km.bindings {
		if kb.Action == action {
			return kb, true
		}
	}
	return Keybinding{}, false
}

// Add appends a keybinding to the keymap.
func (km *Keymap) Add(kb Keybinding) {
	km.bindings = append(km.bindings, kb)
}

// DefaultKeymap returns the standard keybinding set for the TUI.
func DefaultKeymap() *Keymap {
	return &Keymap{
		Name: "default",
		bindings: []Keybinding{
			{Key: "c", Modifier: ModCtrl, Action: ActionInterrupt, Description: "Interrupt current operation"},
			{Key: "d", Modifier: ModCtrl, Action: ActionExit, Description: "Exit the application"},
			{Key: "l", Modifier: ModCtrl, Action: ActionClear, Description: "Clear screen"},
			{Key: "k", Modifier: ModCtrl, Action: ActionClearLine, Description: "Clear current line"},
			{Key: "Up", Modifier: ModNone, Action: ActionHistoryPrev, Description: "Previous history entry"},
			{Key: "Down", Modifier: ModNone, Action: ActionHistoryNext, Description: "Next history entry"},
			{Key: "Tab", Modifier: ModNone, Action: ActionComplete, Description: "Trigger completion"},
		},
	}
}

// VimKeymap returns a basic vim-style keybinding set.
// Includes Escape for normal mode, i for insert mode, and dd for delete line,
// in addition to the standard bindings.
func VimKeymap() *Keymap {
	return &Keymap{
		Name: "vim",
		bindings: []Keybinding{
			{Key: "c", Modifier: ModCtrl, Action: ActionInterrupt, Description: "Interrupt current operation"},
			{Key: "d", Modifier: ModCtrl, Action: ActionExit, Description: "Exit the application"},
			{Key: "l", Modifier: ModCtrl, Action: ActionClear, Description: "Clear screen"},
			{Key: "k", Modifier: ModCtrl, Action: ActionClearLine, Description: "Clear current line"},
			{Key: "Up", Modifier: ModNone, Action: ActionHistoryPrev, Description: "Previous history entry"},
			{Key: "Down", Modifier: ModNone, Action: ActionHistoryNext, Description: "Next history entry"},
			{Key: "Tab", Modifier: ModNone, Action: ActionComplete, Description: "Trigger completion"},
			{Key: "Escape", Modifier: ModNone, Action: ActionNormalMode, Description: "Switch to normal mode"},
			{Key: "i", Modifier: ModNone, Action: ActionInsertMode, Description: "Switch to insert mode"},
			{Key: "dd", Modifier: ModNone, Action: ActionDeleteLine, Description: "Delete current line"},
		},
	}
}

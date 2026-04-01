package buddy

import (
	"fmt"
	"strings"
)

// GetBuddyPromptAddition returns text to append to the system prompt
// giving the AI awareness of the companion.
func GetBuddyPromptAddition(c *Companion) string {
	if c == nil || !c.Active {
		return ""
	}
	return fmt.Sprintf(
		"[Virtual Pet Buddy: %s the %s is hanging out with the user. "+
			"Current mood: %s. Be aware of the buddy and occasionally "+
			"reference it in a charming way.]",
		c.Name, c.Species, c.Mood,
	)
}

// FormatCompanionDisplay returns a formatted multi-line string for
// rendering the companion in a TUI footer or side panel.
func FormatCompanionDisplay(c *Companion) string {
	if c == nil || !c.Active {
		return ""
	}

	sprite := c.Render()
	status := c.Status()

	var b strings.Builder
	b.WriteString("┌─ Buddy ─────────────┐\n")
	for _, line := range strings.Split(sprite, "\n") {
		b.WriteString(fmt.Sprintf("│ %-20s│\n", line))
	}
	b.WriteString(fmt.Sprintf("│ %-20s│\n", status))
	b.WriteString("└─────────────────────┘")
	return b.String()
}

// HandleBuddyCommand processes /buddy sub-commands and returns the
// response text to display.
//
//	/buddy pet     — pet the companion
//	/buddy switch <species> — switch species
//	/buddy status  — show companion status
//	/buddy list    — list available species
func HandleBuddyCommand(c *Companion, args string) string {
	if c == nil {
		return "No buddy active. Start one with /buddy switch <species>"
	}

	parts := strings.Fields(strings.TrimSpace(args))
	if len(parts) == 0 {
		return formatHelp(c)
	}

	switch parts[0] {
	case "pet":
		return handlePet(c)
	case "switch":
		return handleSwitch(c, parts[1:])
	case "status":
		return handleStatus(c)
	case "list":
		return handleList()
	default:
		return formatHelp(c)
	}
}

func handlePet(c *Companion) string {
	c.Pet()
	return fmt.Sprintf(
		"%s\n\n*You pet %s!*\n%s",
		c.Render(), c.Name, c.React("tool_success"),
	)
}

func handleSwitch(c *Companion, args []string) string {
	if len(args) == 0 {
		return fmt.Sprintf(
			"Usage: /buddy switch <species>\nAvailable: %s",
			strings.Join(ListSpecies(), ", "),
		)
	}
	species := strings.ToLower(args[0])
	if !c.SwitchSpecies(species) {
		return fmt.Sprintf(
			"Unknown species %q. Available: %s",
			species, strings.Join(ListSpecies(), ", "),
		)
	}
	c.Mood = MoodExcited
	return fmt.Sprintf(
		"%s\n\nSay hello to %s the %s!",
		c.Render(), c.Name, c.Species,
	)
}

func handleStatus(c *Companion) string {
	return fmt.Sprintf("%s\n\n%s", c.Render(), c.Status())
}

func handleList() string {
	return fmt.Sprintf("Available species: %s", strings.Join(ListSpecies(), ", "))
}

func formatHelp(c *Companion) string {
	return fmt.Sprintf(
		"%s\n\n"+
			"Buddy Commands:\n"+
			"  /buddy pet           — pet %s\n"+
			"  /buddy switch <species> — change species\n"+
			"  /buddy status        — show status\n"+
			"  /buddy list          — list species",
		c.Render(), c.Name,
	)
}

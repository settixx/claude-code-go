package buddy

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Mood represents the companion's emotional state.
type Mood string

const (
	MoodIdle     Mood = "idle"
	MoodHappy    Mood = "happy"
	MoodThinking Mood = "thinking"
	MoodSleeping Mood = "sleeping"
	MoodExcited  Mood = "excited"
)

var validSpecies = []string{"duck", "cat", "ghost", "robot", "bear"}

// Companion is the virtual pet buddy.
type Companion struct {
	Name    string
	Species string
	Mood    Mood
	Sprites SpriteSet
	Active  bool

	frame     int
	lastPetAt time.Time
	petCount  int
}

// NewCompanion creates a companion of the given species.
// Falls back to "duck" for unknown species.
func NewCompanion(species string) *Companion {
	if !isValidSpecies(species) {
		species = "duck"
	}
	return &Companion{
		Name:    speciesName(species),
		Species: species,
		Mood:    MoodIdle,
		Sprites: GetSprites(species),
		Active:  true,
	}
}

// Pet triggers a pet interaction — the buddy gets happy.
func (c *Companion) Pet() {
	c.lastPetAt = time.Now()
	c.petCount++
	c.Mood = MoodHappy
	c.frame = 0
}

// React generates a reaction string for a given event type.
func (c *Companion) React(event string) string {
	switch event {
	case "tool_success":
		c.Mood = MoodHappy
		return c.pickReaction(successReactions)
	case "tool_error":
		c.Mood = MoodExcited
		return c.pickReaction(errorReactions)
	case "user_message":
		c.Mood = MoodIdle
		return c.pickReaction(userMessageReactions)
	case "assistant_response":
		c.Mood = MoodThinking
		return c.pickReaction(thinkingReactions)
	case "session_start":
		c.Mood = MoodExcited
		return c.pickReaction(greetReactions)
	case "session_end":
		c.Mood = MoodSleeping
		return c.pickReaction(farewellReactions)
	case "idle":
		c.Mood = MoodSleeping
		return c.pickReaction(sleepReactions)
	default:
		return ""
	}
}

// Render returns the current sprite frame as a string.
func (c *Companion) Render() string {
	frames := c.framesForMood()
	if len(frames) == 0 {
		return ""
	}
	idx := c.frame % len(frames)
	c.frame++
	return frames[idx]
}

// Tick advances the animation frame — call periodically.
func (c *Companion) Tick() {
	c.frame++
}

// Status returns a one-line summary of the companion state.
func (c *Companion) Status() string {
	return fmt.Sprintf("%s the %s — mood: %s, pets: %d", c.Name, c.Species, c.Mood, c.petCount)
}

// SwitchSpecies changes the companion to a different species.
func (c *Companion) SwitchSpecies(species string) bool {
	if !isValidSpecies(species) {
		return false
	}
	c.Species = species
	c.Name = speciesName(species)
	c.Sprites = GetSprites(species)
	c.frame = 0
	return true
}

// ListSpecies returns all available species names.
func ListSpecies() []string {
	out := make([]string, len(validSpecies))
	copy(out, validSpecies)
	return out
}

func (c *Companion) framesForMood() []string {
	switch c.Mood {
	case MoodHappy:
		return c.Sprites.Happy
	case MoodThinking:
		return c.Sprites.Thinking
	case MoodSleeping:
		return c.Sprites.Sleeping
	case MoodExcited:
		return c.Sprites.Excited
	default:
		return c.Sprites.Idle
	}
}

func (c *Companion) pickReaction(pool []string) string {
	if len(pool) == 0 {
		return ""
	}
	base := pool[rand.Intn(len(pool))]
	return strings.ReplaceAll(base, "{name}", c.Name)
}

func isValidSpecies(s string) bool {
	for _, v := range validSpecies {
		if v == s {
			return true
		}
	}
	return false
}

func speciesName(species string) string {
	names := map[string]string{
		"duck":  "Quackers",
		"cat":   "Whiskers",
		"ghost": "Boo",
		"robot": "Beep",
		"bear":  "Honey",
	}
	if n, ok := names[species]; ok {
		return n
	}
	return "Buddy"
}

var successReactions = []string{
	"{name} does a little dance!",
	"{name} is impressed!",
	"*{name} nods approvingly*",
	"{name}: Nice one!",
	"*{name} gives a tiny high-five*",
}

var errorReactions = []string{
	"{name} winces a little...",
	"*{name} pats you on the shoulder*",
	"{name}: It's okay, we'll get it next time!",
	"*{name} looks concerned*",
	"{name}: Hmm, let me see...",
}

var userMessageReactions = []string{
	"*{name} perks up*",
	"*{name} tilts head curiously*",
	"{name} is listening...",
	"*{name} wiggles attentively*",
}

var thinkingReactions = []string{
	"*{name} watches intently*",
	"*{name} strokes chin thoughtfully*",
	"{name} is pondering...",
	"*{name} follows along*",
}

var greetReactions = []string{
	"{name} waves hello!",
	"*{name} bounces with excitement*",
	"{name}: Let's build something cool!",
	"{name} is ready to go!",
}

var farewellReactions = []string{
	"*{name} waves goodbye*",
	"{name}: See you next time!",
	"*{name} yawns and curls up*",
	"{name}: Good session!",
}

var sleepReactions = []string{
	"*{name} dozes off*",
	"{name} fell asleep... zzz",
	"*{name} snores softly*",
}

# Ti Code

**An AI coding agent that lives in your terminal. Single binary. Zero dependencies. Built with Go.**

[![Go 1.23+](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status: WIP](https://img.shields.io/badge/Status-WIP-orange)]()

**English** | [简体中文](./README.zh-CN.md)

---

Ti Code is a terminal-native AI coding agent. It talks to LLMs, reads and writes your code, runs commands, and stays out of your way — all from a single binary with zero runtime dependencies.

The TUI is built on [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) and the [Charm](https://charm.sh/) ecosystem. If you've used Claude Code or Gemini CLI, the workflow will feel familiar.

> **Note** — This project is under active development. Contributions and feedback are welcome.

## Highlights

- **Single binary** — `go build` → ship. No Node.js, no `node_modules`, no runtime.
- **Fast startup** — native Go binary, cold start in single-digit milliseconds.
- **15 built-in tools** — file read/write/edit, bash, glob, grep, web fetch/search, notebook, todo, config, and more.
- **Multi-model** — Anthropic Claude (default), with model alias shortcuts (`opus`, `sonnet`, `haiku`).
- **Streaming** — real-time token streaming with thinking/extended thinking support.
- **MCP protocol** — built-in MCP client/server for tool extensibility.
- **Session persistence** — JSON-based session storage with resume support.
- **Permission system** — configurable permission modes with write-tool classification.
- **Multi-agent coordinator** — worker pool, team management, worktree isolation.
- **Buddy system** — virtual companion with mood states and sprites (duck, cat, ghost, robot, bear).
- **Cost tracking** — per-request and cumulative token/cost accounting.
- **Slash commands** — `/help`, `/model`, `/status`, `/cost`, `/session`, `/config`, and more.

## Quick Start

```bash
git clone https://github.com/settixx/claude-code-go.git
cd claude-code-go
make build
```

Set your API key and run:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
./bin/ticode
```

That's it. No `npm install`, no config files required.

## Usage

```bash
ticode                                    # Interactive REPL
ticode -p "explain this function"         # Single query (print mode)
ticode -m opus "review main.go"           # Use a specific model
ticode --resume <session-id>              # Resume a previous session
cat file.go | ticode -p                   # Pipe stdin
ticode --version                          # Version info
```

### Flags

| Flag | Short | Description |
|---|---|---|
| `--print` | `-p` | Non-interactive print mode |
| `--model` | `-m` | Model selection (e.g. `opus`, `sonnet`, `haiku`) |
| `--resume` | `-r` | Resume a session by ID |
| `--verbose` | `-v` | Enable verbose output |
| `--permission-mode` | | Set permission mode (`default`) |
| `--agent` | | Select agent |
| `--debug` | | Enable debug output |

### Slash Commands

| Command | Description |
|---|---|
| `/help` | Show available commands |
| `/model` | View or switch the active model |
| `/status` | Show session status |
| `/cost` | Display token usage and cost |
| `/config` | View or modify configuration |
| `/session` | Session management |
| `/exit` | Exit the REPL |

## Architecture

```
cmd/ticode/          CLI entry point
internal/
├── api/             LLM client, streaming, models, cost tracking, retry
├── bridge/          Bridge protocol for IDE/headless integration
├── buddy/           Virtual companion system (sprites, mood, observer)
├── cli/             Flag parsing, REPL loop, slash command registry
├── config/          Paths, CLAUDE.md parsing, config loading
├── coordinator/     Multi-agent: worker pool, teams, worktrees, mailbox
├── errors/          Structured error types
├── interfaces/      Core interface definitions
├── mcp/             MCP protocol client/server, transport, config
├── permissions/     Permission checker, rules, classifier
├── query/           Query engine, agent loop, budget, history
├── state/           Application state store
├── storage/         Session persistence (file-based)
├── tools/           15 built-in tools (bash, file*, glob, grep, web*, ...)
├── tui/             Bubble Tea v2 app, input, renderer, themes, keymap
├── types/           Shared types (message, config, tool, query, IDs)
└── version/         Build version injection
```

## Built-in Tools

### Core (Tier 1)

| Tool | Description |
|---|---|
| `Bash` | Execute shell commands |
| `FileRead` | Read file contents |
| `FileWrite` | Write files |
| `FileEdit` | Patch files with search/replace |
| `Glob` | Find files by pattern |
| `Grep` | Search file contents with regex |

### Extended (Tier 2)

| Tool | Description |
|---|---|
| `WebSearch` | Search the web |
| `WebFetch` | Fetch URL content |
| `Notebook` | Edit Jupyter notebooks |
| `Todo` | Task management |
| `Config` | View/modify configuration |
| `Sleep` | Delay execution |
| `AskUser` | Prompt user for input |
| `Brief` | Summarize content |
| `ToolSearch` | Discover available tools |

## Tech Stack

| Component | Library |
|---|---|
| TUI framework | [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) |
| Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Widgets | [Bubbles](https://github.com/charmbracelet/bubbles) |
| Forms | [Huh](https://github.com/charmbracelet/huh) |
| Markdown | [Glamour](https://github.com/charmbracelet/glamour) |
| Syntax | [Chroma](https://github.com/alecthomas/chroma) |
| JSON | `encoding/json` + [jsonc](https://github.com/tidwall/jsonc) |

## Development

```bash
make build                 # Build binary to ./bin/ticode
make run                   # Build and run
make test                  # Run tests
make lint                  # golangci-lint
make build-all             # Cross-compile (darwin/linux/windows × amd64/arm64)
make release               # goreleaser release
make clean                 # Remove build artifacts
```

### Cross-compilation

The project uses GoReleaser for releases and includes a `build-all` make target:

```bash
# Manual cross-compile
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ticode-linux ./cmd/ticode
```

### Conventions

- Max 3 levels of nesting — extract early, return early
- One loop does one thing — extract if body exceeds 10 lines
- Interfaces define behavior, structs implement it
- Table-driven tests: `[]struct{ name, input, want }`

## Roadmap

**Phase 1 — Foundation** `in progress`

- [x] CLI skeleton with flag parsing
- [x] Bubble Tea v2 REPL (input + streaming output)
- [x] Claude API client with streaming
- [x] Core tools: bash, file read/write/edit, glob, grep
- [x] Extended tools: web search/fetch, notebook, todo, config
- [x] Slash command system
- [x] Permission checker
- [x] Session persistence
- [x] Cost tracking

**Phase 2 — Agent**

- [x] Multi-agent coordinator (worker pool, teams)
- [x] Worktree isolation for parallel agents
- [x] MCP client/server protocol
- [x] Bridge protocol for IDE integration
- [ ] Agent loop refinement (plan → act → observe)
- [ ] Glamour markdown rendering in REPL

**Phase 3 — Protocol**

- [ ] LSP integration
- [ ] Git-aware context
- [ ] Memory system (CLAUDE.md integration)

**Phase 4 — Polish**

- [ ] Vim keybindings
- [ ] Custom themes
- [ ] `brew install ticode` / `scoop install ticode`
- [ ] Test coverage > 80%

## Contributing

Contributions welcome. Fork, branch, test, lint, PR.

```bash
git checkout -b feature/your-thing
make test
make lint
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Disclaimer

- Not affiliated with, endorsed by, or maintained by [Anthropic](https://anthropic.com).
- "Claude" is a trademark of Anthropic. This project does not claim any rights to it.

If you believe any content in this repository raises a concern, please [open an issue](https://github.com/settixx/claude-code-go/issues).

## License

[MIT](LICENSE)

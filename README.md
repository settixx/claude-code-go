<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/License-MIT-blue" />
  <img src="https://img.shields.io/badge/Status-WIP-orange" />
</p>

<h1 align="center">Ti Code</h1>

<p align="center">
  An AI coding agent that lives in your terminal. Built with Go.
</p>

<p align="center">
  <strong>English</strong> | <a href="./README.zh-CN.md">简体中文</a>
</p>

---

Ti Code is an AI coding agent for the terminal. It talks to LLMs, reads and writes your code, runs commands, and stays out of your way — all from a single binary with zero runtime dependencies.

The TUI is built on [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) and the [Charm](https://charm.sh/) ecosystem. If you've used Claude Code or Gemini CLI, the workflow will feel familiar.

> [!NOTE]
> This project is under active development. Contributions and feedback are welcome.

## Why rewrite in Go?

The original codebase is ~500K lines of TypeScript running on Node/Bun with React (Ink) for terminal UI. It works, but:

- **Shipping is painful.** Users need Node.js installed. `node_modules` is 164 MB. Cold start takes 150 ms+.
- **Distribution should be trivial.** `go build` → single binary → done. Cross-compile with `GOOS` / `GOARCH`.
- **Concurrency is a first-class citizen.** Goroutines and channels map naturally to streaming LLM responses, parallel tool execution, and subprocess management.
- **The TUI ecosystem is mature.** Bubble Tea has 41K+ stars, a v2 release (Feb 2026), and 18,000+ dependents. It's production-proven.

This is not a port. The architecture is redesigned around Go idioms — interfaces, composition, and the MVU pattern.

## Getting started

```bash
git clone https://github.com/YOUR_USERNAME/ti-code.git
cd ti-code
go build -o ti-code ./cmd/ti-code
```

Set an API key and run:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
./ti-code
```

That's it. No `npm install`, no runtime, no config files required.

### Other providers

```bash
export OPENAI_API_KEY="sk-..."           # OpenAI
export OLLAMA_HOST="http://localhost:11434"  # Local models
```

### Usage

```bash
ti-code                                    # Interactive REPL
ti-code -p "explain this function"         # Single query
ti-code --print -p "review main.go"        # Non-interactive print mode
ti-code --version                          # Version info
```

## Tech stack

| What | Library |
|---|---|
| TUI | [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) |
| Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Widgets | [Bubbles](https://github.com/charmbracelet/bubbles) (input, viewport, spinner, table) |
| Forms | [Huh](https://github.com/charmbracelet/huh) |
| CLI | [Cobra](https://github.com/spf13/cobra) + [Viper](https://github.com/spf13/viper) |
| Markdown | [Glamour](https://github.com/charmbracelet/glamour) |
| Syntax | [Chroma](https://github.com/alecthomas/chroma) |
| DB | [go-sqlite3](https://github.com/mattn/go-sqlite3) |
| JSON | `encoding/json` + [jsonc](https://github.com/tidwall/jsonc) |

## Roadmap

**Phase 1 — Foundation** 🚧

- [ ] Cobra CLI skeleton
- [ ] Bubble Tea v2 REPL (input + streaming output viewport)
- [ ] Claude API client with streaming
- [ ] Core tools: file read, file write, bash
- [ ] Config via env vars + TOML

**Phase 2 — Agent**

- [ ] Full tool suite (glob, grep, file edit, web fetch)
- [ ] Agent loop: plan → act → observe
- [ ] SQLite sessions
- [ ] Multi-provider support
- [ ] Glamour markdown rendering

**Phase 3 — Protocol**

- [ ] MCP server & client
- [ ] LSP integration
- [ ] Git-aware context
- [ ] Cost tracking

**Phase 4 — Polish**

- [ ] Vim keybindings
- [ ] Themes
- [ ] `brew install` / `scoop install`
- [ ] Test coverage >80%

## Development

```bash
go run ./cmd/ti-code                                          # Dev run
go build -ldflags="-s -w" -o ti-code ./cmd/ti-code            # Release build
go test ./...                                                 # Tests
golangci-lint run                                             # Lint
GOOS=linux GOARCH=amd64 go build -o ti-code-linux ./cmd/ti-code  # Cross-compile
```

### Conventions

- Max 3 levels of nesting — extract early, return early
- One loop does one thing — extract if body exceeds 10 lines
- Interfaces define behavior, structs implement it
- Table-driven tests: `[]struct{ name, input, want }`

## Contributing

Contributions welcome. Fork, branch, test, lint, PR.

```bash
git checkout -b feature/your-thing
go test ./...
golangci-lint run
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Disclaimer

- Not affiliated with, endorsed by, or maintained by [Anthropic](https://anthropic.com).
- "Claude" is a trademark of Anthropic. This project does not claim any rights to it.

If you believe any content in this repository raises a concern, please [open an issue](https://github.com/YOUR_USERNAME/ti-code/issues).

## License

[MIT](LICENSE)

---

<p align="center">
  Built with Go and the <a href="https://charm.sh/">Charm</a> ecosystem.
</p>

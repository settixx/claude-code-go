<div align="center">

<br>

```
  ████████╗██╗     ██████╗  ██████╗ ██████╗ ███████╗
  ╚══██╔══╝██║    ██╔════╝ ██╔═══██╗██╔══██╗██╔════╝
     ██║   ██║    ██║      ██║   ██║██║  ██║█████╗
     ██║   ██║    ██║      ██║   ██║██║  ██║██╔══╝
     ██║   ██║    ╚██████╗ ╚██████╔╝██████╔╝███████╗
     ╚═╝   ╚═╝     ╚═════╝  ╚═════╝ ╚═════╝ ╚══════╝
```

**An agentic coding assistant that lives in your terminal.**\
**Single binary. Zero dependencies. Built with Go.**

[![Go 1.23+](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Status: WIP](https://img.shields.io/badge/Status-Active_Development-orange?style=flat-square)]()

[Quick Start](#quick-start) · [Features](#features) · [Usage](#usage) · [Architecture](#architecture) · [Contributing](#contributing)

**English** | [简体中文](./README.zh-CN.md)

</div>

---

Ti Code is a terminal-native AI coding agent. It talks to LLMs, reads and writes your code, runs commands, and stays out of your way — all from a single binary with zero runtime dependencies.

The TUI is built on [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) and the [Charm](https://charm.sh/) ecosystem. If you've used Claude Code or Gemini CLI, the workflow will feel familiar.

> [!NOTE]
> This project is under active development. Contributions and feedback are welcome.

## Why Go?

| Pain point | How Ti Code fixes it |
|---|---|
| Node.js + `node_modules` (164 MB) | `go build` → **single binary**, done |
| Cold start 150 ms+ | Native binary, single-digit ms startup |
| Complex distribution | Cross-compile with `GOOS` / `GOARCH` |
| Concurrency bolted on | Goroutines + channels are first-class |

This is not a port. The architecture is redesigned around Go idioms — interfaces, composition, and the MVU pattern.

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

## Features

<table>
<tr>
<td>

**Core**
- 25+ built-in tools (file ops, bash, grep, web, agent, MCP, ...)
- Real-time token streaming with thinking support
- Multi-model support with alias shortcuts
- Permission system with security validator chain
- CLAUDE.md config with frontmatter, file locking & trust system

</td>
<td>

**Agent**
- Multi-agent coordinator with worker pool & orchestrator
- 5 built-in subagent types (explore, plan, code-reviewer, shell, refactor)
- Task management (create, list, stop, output)
- Team-based agent coordination

</td>
</tr>
<tr>
<td>

**Skills & MCP**
- Skills engine (bundled + user + project skills)
- 6 bundled skills (commit, debug, review, simplify, verify, remember)
- MCP protocol with SSE transport & OAuth
- MCP prompts, resources, and tool discovery

</td>
<td>

**DX**
- 20+ slash commands (`/commit`, `/review`, `/plan`, `/agents`, `/tasks`, ...)
- Plan mode toggle for read-only exploration
- Context compaction with token-aware budget
- Bridge protocol with WebSocket renderer for IDE integration
- Notification system with rate limiting

</td>
</tr>
</table>

## Usage

```bash
ticode                                    # Interactive REPL
ticode -p "explain this function"         # Single query (print mode)
ticode -m opus "review main.go"           # Use a specific model
ticode --resume <session-id>              # Resume a previous session
cat file.go | ticode -p                   # Pipe stdin
ticode --version                          # Version info
```

<details>
<summary><b>All Flags</b></summary>

| Flag | Short | Description |
|---|---|---|
| `--print` | `-p` | Non-interactive print mode |
| `--model` | `-m` | Model selection (`opus`, `sonnet`, `haiku`, or full ID) |
| `--resume` | `-r` | Resume a session by ID |
| `--verbose` | `-v` | Enable verbose output |
| `--permission-mode` | | Set permission mode |
| `--agent` | | Select agent |
| `--debug` | | Enable debug output |

</details>

<details>
<summary><b>Slash Commands</b></summary>

| Command | Description |
|---|---|
| `/help` | Show available commands |
| `/model` | View or switch the active model |
| `/status` | Show session status |
| `/cost` | Display token usage and cost |
| `/config` | View or modify configuration |
| `/session` | Session management |
| `/commit` | Generate commit message from staged changes |
| `/diff` | Show git diff summary |
| `/review` | Request a code review of recent changes |
| `/doctor` | Run environment diagnostics |
| `/resume` | List and resume previous sessions |
| `/export` | Export conversation to file |
| `/memory` | Show or edit memory files |
| `/plan` | Toggle plan mode (read-only) |
| `/agents` | List active agents |
| `/tasks` | List active tasks |
| `/mcp` | Show MCP server status |
| `/skills` | List available skills |
| `/buddy` | Toggle buddy companion display |
| `/exit` | Exit the REPL |

</details>

## Built-in Tools

<details>
<summary><b>Tier 1 — Core</b> (always loaded)</summary>

| Tool | What it does |
|---|---|
| **Bash** | Execute shell commands with security validator chain |
| **FileRead** | Read file contents |
| **FileWrite** | Write files |
| **FileEdit** | Patch files with search/replace |
| **Glob** | Find files by pattern |
| **Grep** | Search file contents with regex |

</details>

<details>
<summary><b>Tier 2 — Extended</b></summary>

| Tool | What it does |
|---|---|
| **WebSearch** | Search the web |
| **WebFetch** | Fetch URL content |
| **Notebook** | Edit Jupyter notebooks |
| **Config** | View/modify configuration |
| **AskUser** | Prompt user for input |
| **Brief** | Summarize content |
| **ToolSearch** | Discover available tools |
| **LSP** | Language server protocol integration |

</details>

<details>
<summary><b>Tier 3 — Agent & MCP</b></summary>

| Tool | What it does |
|---|---|
| **Agent** | Spawn subagents (explore, plan, code-reviewer, shell, refactor) |
| **SendMessage** | Send follow-up messages to running agents |
| **TaskCreate** | Create background tasks |
| **TaskList** | List active tasks |
| **TaskOutput** | Read task output |
| **TaskStop** | Stop a running task |
| **TaskUpdate** | Update task status |
| **TeamCreate** | Create multi-agent teams |
| **TeamDelete** | Delete agent teams |
| **Worktree** | Manage git worktrees for isolated agent work |
| **EnterPlanMode** | Switch to read-only plan mode |
| **ExitPlanMode** | Return to normal mode |
| **MCPTool** | Execute MCP server tools |
| **ListMCPResources** | List available MCP resources |
| **ReadMCPResource** | Read a specific MCP resource |

</details>

## Subagent Types

| Agent | Model | Purpose |
|---|---|---|
| `explore` | fast | Quick codebase exploration — find files, search patterns |
| `plan` | default | Design implementation approaches before coding |
| `code-reviewer` | default | Analyze code quality and identify issues |
| `shell` | fast | Run terminal commands and return results |
| `refactor` | default | Restructure code for clarity and performance |

## Skills Engine

Ti Code includes a skill system that extends agent capabilities with specialized knowledge and workflows.

| Skill | What it does |
|---|---|
| `commit` | Generate well-formed git commits |
| `debug` | Systematic debugging workflow |
| `review` | Thorough code review checklist |
| `simplify` | Reduce code complexity |
| `verify` | Validate changes against requirements |
| `remember` | Persist context across sessions |

Skills are loaded from three sources: **bundled** (shipped with the binary), **user** (`~/.ticode/skills/`), and **project** (`.ticode/skills/` in repo root).

## Architecture

```
cmd/ticode/             Entry point
internal/
├── api/                LLM client · streaming · models · cost · retry
├── bridge/             IDE / headless integration · WebSocket renderer
├── buddy/              Virtual companion (sprites · mood · observer)
├── cli/                Flags · REPL · slash commands (core + dev + session + agents + tasks + MCP + plan)
├── config/             Paths · CLAUDE.md · frontmatter · file locking · trust
├── coordinator/        Multi-agent: orchestrator · pool · teams · worktrees
├── errors/             Structured error types
├── interfaces/         Core interface definitions
├── mcp/                MCP client · SSE transport · OAuth · prompts · bootstrap · adapter
├── notify/             Desktop notifications · rate limiting
├── permissions/        Checker · rules · classifier · security chain
├── query/              Engine · compaction · token budget · system prompt
├── skills/             Skill registry · loader · discovery · skill-as-tool
├── state/              Application state store
├── storage/            Session persistence (file-based JSON)
├── tools/
│   ├── agent/          Subagent spawner · runner · definitions
│   ├── bash/           Bash tool · security · readonly · quoting · semantics
│   ├── tasktool/       Task CRUD (create / get / list / stop / output / update)
│   ├── teamtool/       Team create / delete
│   ├── planmode/       Enter / exit plan mode
│   ├── worktree/       Git worktree management
│   ├── mcptool/        MCP tool execution
│   ├── lsp/            Language server protocol tool
│   └── ...             14 more built-in tools
├── tui/                Bubble Tea v2 · model · input · viewport · permissions · buddy widget · cost tracker · git branch · markdown · syntax highlight
├── types/              Shared types (message · config · tool · IDs)
└── version/            Build version injection
test/
├── bench_test.go       Performance benchmarks
├── build_test.go       Build validation
└── golden_test.go      Golden file / snapshot tests
```

## Tech Stack

| Component | Library |
|---|---|
| TUI | [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) |
| Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Widgets | [Bubbles](https://github.com/charmbracelet/bubbles) |
| Forms | [Huh](https://github.com/charmbracelet/huh) |
| Markdown | [Glamour](https://github.com/charmbracelet/glamour) |
| Syntax | [Chroma v2](https://github.com/alecthomas/chroma) |
| Markdown | [Glamour](https://github.com/charmbracelet/glamour) |
| WebSocket | [gorilla/websocket](https://github.com/gorilla/websocket) |
| JSON | `encoding/json` + [jsonc](https://github.com/tidwall/jsonc) |

## Development

```bash
make build          # Build → ./bin/ticode
make run            # Build and run
make test           # Run tests
make lint           # golangci-lint
make build-all      # Cross-compile (darwin/linux/windows × amd64/arm64)
make release        # goreleaser
make clean          # Remove artifacts
```

### Conventions

- Max **3 levels** of nesting — extract early, return early
- One loop does one thing — extract if body exceeds 10 lines
- Interfaces define behavior, structs implement it
- Table-driven tests: `[]struct{ name, input, want }`

## Roadmap

| Phase | Focus | Status |
|---|---|---|
| **1 — Foundation** | CLI, TUI, API client, core tools, permissions, sessions, cost tracking | ✅ Done |
| **2 — Agent** | Multi-agent coordinator, skills engine, agent tools, task management, MCP SSE/OAuth, bash security, context compaction | 🚧 In Progress |
| **3 — Protocol** | LSP integration, git-aware context, memory system | 📋 Planned |
| **4 — Polish** | Vim bindings, themes, `brew install`, test coverage > 80% | 📋 Planned |

## Contributing

Contributions welcome. Fork, branch, test, lint, PR.

```bash
git checkout -b feature/your-thing
make test && make lint
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Disclaimer

- Not affiliated with, endorsed by, or maintained by [Anthropic](https://anthropic.com).
- "Claude" is a trademark of Anthropic. This project does not claim any rights to it.

If you believe any content in this repository raises a concern, please [open an issue](https://github.com/settixx/claude-code-go/issues).

## License

[MIT](LICENSE)

---

<div align="center">

Built with Go and the [Charm](https://charm.sh/) ecosystem.

</div>

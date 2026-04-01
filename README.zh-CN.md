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

**一个住在终端里的 AI 编程助手。**\
**单文件二进制。零依赖。Go 构建。**

[![Go 1.23+](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Status: WIP](https://img.shields.io/badge/Status-Active_Development-orange?style=flat-square)]()

[快速开始](#快速开始) · [功能特性](#功能特性) · [使用方式](#使用方式) · [架构](#架构) · [贡献](#贡献)

[English](./README.md) | **简体中文**

</div>

---

Ti Code 是一个终端原生 AI 编程助手。它与大语言模型对话，读写你的代码，执行命令，同时不打扰你的工作流 —— 所有功能来自一个零运行时依赖的单文件二进制。

TUI 基于 [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) 和 [Charm](https://charm.sh/) 生态构建。如果你用过 Claude Code 或 Gemini CLI，上手会很快。

> [!NOTE]
> 本项目正在积极开发中，欢迎贡献代码和反馈意见。

## 为什么用 Go？

| 痛点 | Ti Code 的解法 |
|---|---|
| Node.js + `node_modules`（164 MB） | `go build` → **单文件二进制**，完事 |
| 冷启动 150 ms+ | 原生二进制，毫秒级启动 |
| 分发复杂 | `GOOS` / `GOARCH` 交叉编译 |
| 并发是后加的 | Goroutine + Channel 是一等公民 |

这不是一次移植，而是围绕 Go 惯用方式重新设计的架构 —— 接口、组合、MVU 模式。

## 快速开始

```bash
git clone https://github.com/settixx/claude-code-go.git
cd claude-code-go
make build
```

设置 API Key 并运行：

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
./bin/ticode
```

就这样。不需要 `npm install`，不需要配置文件。

## 功能特性

<table>
<tr>
<td>

**核心**
- 25+ 内置工具（文件操作、Bash、Grep、Web、Agent、MCP……）
- 实时 Token 流式输出 + 思考过程支持
- 多模型支持与别名快捷切换
- 权限系统 + 安全校验链

</td>
<td>

**Agent**
- 多 Agent 协调器（Worker Pool + Orchestrator）
- 5 种内置子代理（explore、plan、code-reviewer、shell、refactor）
- 任务管理（创建、列表、停止、输出）
- 团队级 Agent 协调

</td>
</tr>
<tr>
<td>

**Skills & MCP**
- 技能引擎（内置 + 用户 + 项目级技能）
- 6 个内置技能（commit、debug、review、simplify、verify、remember）
- MCP 协议：SSE 传输 + OAuth 认证
- MCP Prompts、Resources、工具发现

</td>
<td>

**开发体验**
- 15+ 斜杠命令（`/commit`、`/review`、`/export`、`/doctor`……）
- Plan 模式用于只读探索
- 上下文压缩 + Token 预算管理
- 会话持久化、恢复与导出

</td>
</tr>
</table>

## 使用方式

```bash
ticode                                    # 交互式 REPL
ticode -p "解释这个函数"                    # 单次查询（打印模式）
ticode -m opus "review main.go"           # 指定模型
ticode --resume <session-id>              # 恢复历史会话
cat file.go | ticode -p                   # 管道输入
ticode --version                          # 版本信息
```

<details>
<summary><b>所有命令行参数</b></summary>

| 参数 | 缩写 | 说明 |
|---|---|---|
| `--print` | `-p` | 非交互式打印模式 |
| `--model` | `-m` | 模型选择（`opus`、`sonnet`、`haiku` 或完整 ID） |
| `--resume` | `-r` | 通过 ID 恢复会话 |
| `--verbose` | `-v` | 详细输出 |
| `--permission-mode` | | 权限模式 |
| `--agent` | | 选择 Agent |
| `--debug` | | 调试输出 |

</details>

<details>
<summary><b>斜杠命令</b></summary>

| 命令 | 说明 |
|---|---|
| `/help` | 显示可用命令 |
| `/model` | 查看或切换模型 |
| `/status` | 显示会话状态 |
| `/cost` | 显示 Token 用量与费用 |
| `/config` | 查看或修改配置 |
| `/session` | 会话管理 |
| `/commit` | 从暂存变更生成 Commit 消息 |
| `/diff` | 显示 Git Diff 摘要 |
| `/review` | 请求代码审查 |
| `/doctor` | 运行环境诊断 |
| `/resume` | 列出并恢复历史会话 |
| `/export` | 导出对话到文件 |
| `/memory` | 查看或编辑记忆文件 |
| `/exit` | 退出 REPL |

</details>

## 内置工具

<details>
<summary><b>Tier 1 — 核心</b>（始终加载）</summary>

| 工具 | 功能 |
|---|---|
| **Bash** | 执行 Shell 命令（含安全校验链） |
| **FileRead** | 读取文件内容 |
| **FileWrite** | 写入文件 |
| **FileEdit** | 搜索/替换方式修补文件 |
| **Glob** | 按模式查找文件 |
| **Grep** | 正则搜索文件内容 |

</details>

<details>
<summary><b>Tier 2 — 扩展</b></summary>

| 工具 | 功能 |
|---|---|
| **WebSearch** | 搜索网络 |
| **WebFetch** | 抓取 URL 内容 |
| **Notebook** | 编辑 Jupyter Notebook |
| **Config** | 查看/修改配置 |
| **AskUser** | 向用户提问 |
| **Brief** | 内容摘要 |
| **ToolSearch** | 发现可用工具 |
| **LSP** | 语言服务器协议集成 |

</details>

<details>
<summary><b>Tier 3 — Agent & MCP</b></summary>

| 工具 | 功能 |
|---|---|
| **Agent** | 生成子代理（explore、plan、code-reviewer、shell、refactor） |
| **SendMessage** | 向运行中的 Agent 发送后续消息 |
| **TaskCreate** | 创建后台任务 |
| **TaskList** | 列出活跃任务 |
| **TaskOutput** | 读取任务输出 |
| **TaskStop** | 停止运行中的任务 |
| **TaskUpdate** | 更新任务状态 |
| **TeamCreate** | 创建多 Agent 团队 |
| **TeamDelete** | 删除 Agent 团队 |
| **Worktree** | 管理 Git Worktree 用于隔离的 Agent 工作 |
| **EnterPlanMode** | 切换到只读 Plan 模式 |
| **ExitPlanMode** | 返回正常模式 |
| **MCPTool** | 执行 MCP 服务器工具 |
| **ListMCPResources** | 列出可用 MCP 资源 |
| **ReadMCPResource** | 读取指定 MCP 资源 |

</details>

## 子代理类型

| Agent | 模型 | 用途 |
|---|---|---|
| `explore` | fast | 快速代码库探索 —— 查找文件、搜索模式 |
| `plan` | default | 编码前设计实现方案 |
| `code-reviewer` | default | 分析代码质量、发现问题 |
| `shell` | fast | 执行终端命令并返回结果 |
| `refactor` | default | 重构代码以提升清晰度和性能 |

## 技能引擎

Ti Code 内置技能系统，通过专业知识和工作流扩展 Agent 能力。

| 技能 | 功能 |
|---|---|
| `commit` | 生成规范的 Git Commit |
| `debug` | 系统化调试工作流 |
| `review` | 全面代码审查清单 |
| `simplify` | 降低代码复杂度 |
| `verify` | 验证变更是否满足需求 |
| `remember` | 跨会话持久化上下文 |

技能从三个来源加载：**内置**（随二进制分发）、**用户级**（`~/.ticode/skills/`）、**项目级**（仓库根目录 `.ticode/skills/`）。

## 架构

```
cmd/ticode/             入口
internal/
├── api/                LLM 客户端 · 流式传输 · 模型 · 费用 · 重试
├── bridge/             IDE / 无头集成协议
├── buddy/              虚拟伙伴（精灵 · 心情 · 观察者）
├── cli/                参数解析 · REPL · 斜杠命令（核心 + 开发 + 会话）
├── config/             路径 · CLAUDE.md · 配置加载
├── coordinator/        多 Agent：编排器 · Worker Pool · 团队 · Worktree
├── errors/             结构化错误类型
├── interfaces/         核心接口定义
├── mcp/                MCP 客户端 · SSE 传输 · OAuth · Prompts · 工具
├── permissions/        检查器 · 规则 · 分类器 · 安全链
├── query/              引擎 · 上下文压缩 · Token 预算 · 系统提示词
├── skills/             技能注册表 · 加载器 · 发现 · 技能即工具
├── state/              应用状态存储
├── storage/            会话持久化（文件 JSON）
├── tools/
│   ├── agent/          子代理生成器 · 运行器 · 定义
│   ├── bash/           Bash 工具 · 安全 · 只读 · 引号 · 语义
│   ├── tasktool/       任务 CRUD（create / get / list / stop / output / update）
│   ├── teamtool/       团队 create / delete
│   ├── planmode/       进入 / 退出 Plan 模式
│   ├── worktree/       Git Worktree 管理
│   ├── mcptool/        MCP 工具执行
│   ├── lsp/            语言服务器协议工具
│   └── ...             其余 14 个内置工具
├── tui/                Bubble Tea v2 · 模型 · 输入 · 视口 · 权限
├── types/              共享类型（消息 · 配置 · 工具 · ID）
└── version/            构建版本注入
test/
├── bench_test.go       性能基准测试
├── build_test.go       构建验证
└── golden_test.go      Golden File / 快照测试
```

## 技术栈

| 组件 | 库 |
|---|---|
| TUI 框架 | [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) |
| 样式 | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| 组件库 | [Bubbles](https://github.com/charmbracelet/bubbles) |
| 表单 | [Huh](https://github.com/charmbracelet/huh) |
| Markdown | [Glamour](https://github.com/charmbracelet/glamour) |
| 语法高亮 | [Chroma](https://github.com/alecthomas/chroma) |
| JSON | `encoding/json` + [jsonc](https://github.com/tidwall/jsonc) |

## 开发指南

```bash
make build          # 构建 → ./bin/ticode
make run            # 构建并运行
make test           # 运行测试
make lint           # golangci-lint
make build-all      # 交叉编译（darwin/linux/windows × amd64/arm64）
make release        # goreleaser
make clean          # 清理构建产物
```

### 编码约定

- 最大嵌套 **3 层** —— 尽早提取函数，尽早返回
- 一个循环只做一件事 —— 循环体超过 10 行就提取
- 接口定义行为，结构体实现行为
- 表驱动测试：`[]struct{ name, input, want }`

## 路线图

| 阶段 | 重点 | 状态 |
|---|---|---|
| **1 — 基础** | CLI、TUI、API 客户端、核心工具、权限、会话、费用追踪 | ✅ 完成 |
| **2 — Agent** | 多 Agent 协调器、技能引擎、Agent 工具、任务管理、MCP SSE/OAuth、Bash 安全、上下文压缩 | 🚧 进行中 |
| **3 — 协议** | LSP 集成、Git 上下文感知、记忆系统 | 📋 计划中 |
| **4 — 打磨** | Vim 快捷键、主题定制、`brew install`、测试覆盖率 > 80% | 📋 计划中 |

## 贡献

欢迎贡献。Fork → 分支 → 测试 → 检查 → PR。

```bash
git checkout -b feature/your-thing
make test && make lint
```

详见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 免责声明

- 本项目与 [Anthropic](https://anthropic.com) 无任何关联、背书或维护关系。
- "Claude" 是 Anthropic 的商标，本项目不主张任何相关权利。

如果您认为本仓库中存在任何问题，请[提交 Issue](https://github.com/settixx/claude-code-go/issues)。

## 许可证

[MIT](LICENSE)

---

<div align="center">

使用 Go 和 [Charm](https://charm.sh/) 生态构建。

</div>

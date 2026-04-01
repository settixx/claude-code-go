<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/License-MIT-blue" />
  <img src="https://img.shields.io/badge/Status-WIP-orange" />
</p>

<h1 align="center">Ti Code</h1>

<p align="center">
  一个住在终端里的 AI 编程助手，用 Go 构建。
</p>

<p align="center">
  <a href="./README.md">English</a> | <strong>简体中文</strong>
</p>

---

Ti Code 是一个终端 AI 编程助手。它与大语言模型对话，读写你的代码，执行命令，同时不打扰你的工作流 —— 所有功能来自一个零运行时依赖的单文件二进制。

TUI 基于 [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) 和 [Charm](https://charm.sh/) 生态构建。如果你用过 Claude Code 或 Gemini CLI，上手会很快。

> [!NOTE]
> 本项目正在积极开发中，欢迎贡献代码和反馈意见。

## 为什么用 Go 重写？

原始代码库约 50 万行 TypeScript，运行在 Node/Bun 上，终端 UI 使用 React（Ink）。能用，但是：

- **分发很痛苦。** 用户必须安装 Node.js，`node_modules` 体积 164 MB，冷启动 150 ms 以上。
- **分发应该很简单。** `go build` → 单文件二进制 → 完事。通过 `GOOS` / `GOARCH` 交叉编译。
- **并发是一等公民。** Goroutine 和 Channel 天然适合 LLM 流式响应、并行工具执行和子进程管理。
- **TUI 生态已经成熟。** Bubble Tea 拥有 41K+ Star、v2 版本（2026 年 2 月发布）、18,000+ 下游依赖，久经生产验证。

这不是一次移植，而是围绕 Go 的惯用方式重新设计架构 —— 接口、组合、MVU 模式。

## 快速开始

```bash
git clone https://github.com/YOUR_USERNAME/ti-code.git
cd ti-code
go build -o ti-code ./cmd/ti-code
```

设置 API Key 并运行：

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
./ti-code
```

就这样。不需要 `npm install`，不需要运行时，不需要配置文件。

### 其他模型提供商

```bash
export OPENAI_API_KEY="sk-..."           # OpenAI
export OLLAMA_HOST="http://localhost:11434"  # 本地模型
```

### 使用方式

```bash
ti-code                                    # 交互式 REPL
ti-code -p "解释这个函数"                    # 单次查询
ti-code --print -p "审查 main.go"           # 非交互式输出模式
ti-code --version                          # 版本信息
```

## 技术栈

| 组件 | 库 |
|---|---|
| TUI 框架 | [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) |
| 样式 | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| 组件库 | [Bubbles](https://github.com/charmbracelet/bubbles)（输入框、视口、加载动画、表格） |
| 表单 | [Huh](https://github.com/charmbracelet/huh) |
| CLI 解析 | [Cobra](https://github.com/spf13/cobra) + [Viper](https://github.com/spf13/viper) |
| Markdown | [Glamour](https://github.com/charmbracelet/glamour) |
| 语法高亮 | [Chroma](https://github.com/alecthomas/chroma) |
| 数据库 | [go-sqlite3](https://github.com/mattn/go-sqlite3) |
| JSON | `encoding/json` + [jsonc](https://github.com/tidwall/jsonc) |

## 路线图

**第一阶段 — 基础框架** 🚧

- [ ] Cobra CLI 骨架
- [ ] Bubble Tea v2 REPL（输入 + 流式输出视口）
- [ ] Claude API 流式调用客户端
- [ ] 核心工具：文件读取、文件写入、Bash
- [ ] 通过环境变量 + TOML 管理配置

**第二阶段 — Agent 核心**

- [ ] 完整工具集（Glob、Grep、文件编辑、Web 抓取）
- [ ] Agent 循环：计划 → 执行 → 观察
- [ ] SQLite 会话持久化
- [ ] 多模型提供商支持
- [ ] Glamour Markdown 渲染

**第三阶段 — 协议层**

- [ ] MCP 服务端与客户端
- [ ] LSP 集成
- [ ] Git 上下文感知
- [ ] 费用追踪

**第四阶段 — 打磨**

- [ ] Vim 快捷键模式
- [ ] 主题定制
- [ ] `brew install` / `scoop install`
- [ ] 测试覆盖率 >80%

## 开发指南

```bash
go run ./cmd/ti-code                                          # 开发运行
go build -ldflags="-s -w" -o ti-code ./cmd/ti-code            # 构建发布版
go test ./...                                                 # 运行测试
golangci-lint run                                             # 代码检查
GOOS=linux GOARCH=amd64 go build -o ti-code-linux ./cmd/ti-code  # 交叉编译
```

### 编码约定

- 最大嵌套 3 层 —— 尽早提取函数，尽早返回
- 一个循环只做一件事 —— 循环体超过 10 行就提取
- 接口定义行为，结构体实现行为
- 表驱动测试：`[]struct{ name, input, want }`

## 贡献

欢迎贡献。Fork → 分支 → 测试 → 检查 → PR。

```bash
git checkout -b feature/your-thing
go test ./...
golangci-lint run
```

详见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 免责声明

- 本项目与 [Anthropic](https://anthropic.com) 无任何关联、背书或维护关系。
- "Claude" 是 Anthropic 的商标，本项目不主张任何相关权利。

如果您认为本仓库中存在任何问题，请[提交 Issue](https://github.com/YOUR_USERNAME/ti-code/issues)。

## 许可证

[MIT](LICENSE)

---

<p align="center">
  使用 Go 和 <a href="https://charm.sh/">Charm</a> 生态构建。
</p>

---
name: remember
description: Save important context to project memory
tags: [memory, context]
user_invocable: true
auto_run: true
---

# Memory Skill

Persist important decisions, patterns, or context so they survive across sessions.

## Instructions

1. Identify what needs to be remembered:
   - Architecture decisions and their rationale
   - Project conventions and coding standards
   - Common pitfalls or gotchas
   - User preferences for this project
2. Read the existing `.claude/memory.md` if it exists.
3. Append the new information under an appropriate heading.
4. Keep entries concise — bullet points preferred over prose.
5. Avoid duplicating information already present.

Memory file location: `.claude/memory.md` in the project root.

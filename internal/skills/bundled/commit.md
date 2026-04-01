---
name: commit
description: Review changes and create a well-formed git commit
tags: [git, workflow]
user_invocable: true
---

# Commit Skill

Review the staged git changes and generate a clear, conventional commit message.

## Instructions

1. Run `git diff --cached` to inspect staged changes.
2. Summarize the nature of the changes (bug fix, feature, refactor, docs, etc.).
3. Draft a concise commit message following Conventional Commits format:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `refactor:` for code restructuring
   - `docs:` for documentation changes
   - `chore:` for maintenance tasks
4. Include a short body if the change is non-trivial.
5. Run `git commit -m "<message>"` to create the commit.
6. Verify with `git log -1` to confirm.

Keep the subject line under 72 characters. Focus on *why* the change was made, not *what* was changed.

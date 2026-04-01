---
name: review
description: Analyze code changes for bugs, style issues, and security concerns
tags: [code-review, quality]
user_invocable: true
---

# Code Review Skill

Perform a thorough code review of the specified changes.

## Instructions

1. Identify the scope of changes — look at the diff or specified files.
2. Check for:
   - **Correctness**: Logic errors, off-by-one, nil/null dereference, race conditions.
   - **Security**: Injection vulnerabilities, hardcoded secrets, unsafe deserialization.
   - **Style**: Naming conventions, code organization, consistent formatting.
   - **Performance**: Unnecessary allocations, N+1 queries, missing indexes.
   - **Testing**: Are new paths covered? Are edge cases handled?
3. Provide actionable feedback grouped by severity:
   - 🔴 **Critical** — must fix before merge
   - 🟡 **Suggestion** — would improve quality
   - 🟢 **Nit** — minor style preference
4. Summarize overall impression and whether the change is ready to merge.

---
name: simplify
description: Refactor code for clarity while preserving behavior
tags: [refactoring, code-quality]
user_invocable: true
---

# Simplification Skill

Refactor code to improve clarity, reduce complexity, and make it easier to maintain — without changing behavior.

## Instructions

1. **Identify complexity**: Find deeply nested logic, long functions, duplicate code, or unclear naming.
2. **Apply transformations**:
   - Extract helper functions for repeated patterns
   - Use guard clauses to flatten nested conditionals
   - Replace manual loops with idiomatic constructs (comprehensions, iterators)
   - Rename variables/functions for clarity
   - Remove dead code
3. **Preserve behavior**: Every refactoring step must keep existing tests passing.
4. **Verify**: Run the test suite after each significant change.
5. **Summarize**: List what was simplified and why.

Aim for code that is easy to read top-to-bottom without jumping between many abstractions.

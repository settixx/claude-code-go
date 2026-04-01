---
name: debug
description: Systematically investigate and fix bugs
tags: [debugging, workflow]
user_invocable: true
---

# Debug Skill

Systematically investigate and resolve a reported bug.

## Instructions

1. **Reproduce**: Understand the expected vs actual behavior. Run the failing case if possible.
2. **Isolate**: Narrow the scope — identify the file, function, and line range involved.
3. **Hypothesize**: Form 2-3 likely root causes based on the symptoms.
4. **Verify**: Add logging or read code to confirm/reject each hypothesis.
5. **Fix**: Apply the minimal correct fix.
6. **Validate**: Re-run the failing case and any related tests.
7. **Explain**: Summarize the root cause and why the fix is correct.

Prefer reading code and reasoning over trial-and-error. Always check error handling paths and boundary conditions.

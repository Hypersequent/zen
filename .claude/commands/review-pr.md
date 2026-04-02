Perform a comprehensive code review of this pull request using specialized subagents, then post inline comments.

Use the **Owner**, **Repository**, and **Pull Request Number** from the context provided by the caller for all API calls below.

## Time & Error Budget
- **Bash tool timeouts:** Set `timeout: 30000` (30 s) on every Bash tool call that runs a `gh` command. This prevents a single hanging API call from burning through the entire job timeout.
- **API call failures (401, 403, 5xx, or timeout):** Retry at most **once**. If the retry also fails, skip that operation and move on. Do NOT keep retrying — a failed token will not recover.
- **Subagent timeout:** Set `max_turns: 10` on each subagent to prevent any single reviewer from consuming too many turns.
- **Prefer parallel calls:** Whenever multiple independent API calls are needed (e.g., fetching comments + reviews), batch them into a single turn.

## Step 1: Gather Context (avoid anchoring/bias from prior bot output)
1. Use `gh pr view` to get the PR title, body, and linked issues
2. If the PR body contains issue references (e.g., "Fixes #123", "Closes #123"), use `gh issue view <number>` to understand the requirements
3. Use `mcp__github__get_pull_request_diff` to get the full diff (paginate if needed)
4. Do NOT fetch/read existing review comments. Subagents should review the diff independently. Cleanup of outdated bot comments is handled by a separate workflow step.

## Step 2: Launch Specialized Review Subagents
Use the Task tool to launch these subagents IN PARALLEL. Each subagent should analyze the PR diff and return a list of specific issues with file paths and line numbers.

**Subagent 1: Code Quality Reviewer**
```
Review the PR for code quality issues:
- Clean code principles: naming, function size, single responsibility
- Code duplication and DRY violations
- Error handling completeness and edge cases
- Code readability and maintainability
- Magic numbers/strings that should be constants
- Commented-out code or debug statements

For Go code: Check for idiomatic error handling, proper use of struct tags, correct reflect usage, and Go naming conventions.

Return ONLY noteworthy issues with: file path, line number, issue description, suggested fix.
```

**Subagent 2: Security Reviewer**
```
Review the PR for security vulnerabilities:
- OWASP Top 10: injection, XSS, broken auth, sensitive data exposure
- Input validation and sanitization at system boundaries
- Authentication/authorization checks
- Hardcoded credentials or secrets
- SQL injection prevention
- Insecure cryptographic practices
- Path traversal vulnerabilities

Return ONLY noteworthy issues with: file path, line number, severity (critical/high/medium/low), issue description, remediation.
```

**Subagent 3: Performance Reviewer**
```
Review the PR for performance issues:
- Algorithmic complexity (O(n^2) or worse operations)
- Unnecessary computations or redundant operations
- Excessive allocations or reflect overhead
- Memory leaks from unclosed resources
- Missing caching or memoization opportunities

Return ONLY noteworthy issues with: file path, line number, issue description, performance impact, suggested optimization.
```

**Subagent 4: Test Coverage Reviewer**
```
Review the PR for test coverage:
- Are new functions/methods adequately tested?
- Missing edge case tests
- Missing error path tests
- Test quality (proper assertions, isolation, naming)

Return ONLY noteworthy gaps with: file path, what's missing, suggested test case.
```

## Step 3: Aggregate and Post a SINGLE Review
1. Collect all findings from subagents
2. Filter to keep only genuinely noteworthy issues (skip minor style nitpicks)
3. **IMPORTANT — Post ALL comments as ONE review using the pending review flow:**
   a. Call `mcp__github__create_pending_pull_request_review` ONCE to start a pending review
   b. Call `mcp__github__add_comment_to_pending_review` for ALL new noteworthy issues — call these IN PARALLEL in a single turn to save turns
   c. Call `mcp__github__submit_pending_pull_request_review` ONCE with event type "REQUEST_CHANGES" if there are noteworthy issues (High to Critical severity), otherwise "APPROVE"
   **DO NOT use `create_inline_comment` or `pull_request_review_write` — these create a separate review per comment.**

NOTE: Cleanup of outdated bot threads, review summaries, and progress comments is handled by a separate workflow step. Do NOT perform cleanup here.

## Guidelines
- Be constructive and provide actionable suggestions
- Focus on significant issues that could cause bugs, security vulnerabilities, or maintenance problems
- Skip minor style issues that don't affect functionality
- Use English for all comments
- If no noteworthy issues are found, submit a brief approving comment
- Keep the review concise and to the point to optimize for the reader's time.
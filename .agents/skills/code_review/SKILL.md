---
name: code-review
description: Perform a comprehensive code review of changed files or a specific file.
---

# Code Review

Perform a comprehensive, high-quality code review, prioritizing correctness, performance, security, maintainability, and language-specific conventions.

## Instructions
1. **Understand Context**: Analyze the code context, dependencies, and target language.
2. **Review Pillars**:
   - **Correctness**: Check for logical errors, edge cases, off-by-one errors, concurrency/race issues, and incorrect API usages.
   - **Security**: Look for hardcoded credentials, injection vulnerabilities, unsafe resource handling, data leaks, and insecure dependencies.
   - **Performance & Efficiency**: Identify unnecessary allocations, inefficient loops, lack of caching, slow queries, and bottleneck operations.
   - **Readability & Design**: Evaluate modularity, clean naming, appropriate abstraction, docstrings, and complexity (keep cyclomatic complexity low).
   - **Language Conventions**: Verify standard formatting and idioms (e.g., proper error checking in Go, proper hook usage in React, etc.).
3. **Be Constructive**: Focus on actionable suggestions. Always provide explanation and example fixes for issues found.
4. **Do Not Over-Comment**: Highlight significant issues first. Do not nitpick on trivial things unless asked.

## Output Format
Structure the code review as follows:
- **Summary**: A high-level overview of the health of the code (e.g., Critical, Major, Minor issues count).
- **Critical & Major Issues**: Bullet points with clear headings, explanation of the bug, and a code diff or block showing how to fix it.
- **Performance & Optimization Suggestions**: Actionable improvement recommendations.
- **Readability & Code Style**: Minor improvements for naming, structure, or documentation.

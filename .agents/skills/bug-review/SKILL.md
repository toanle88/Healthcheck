---
name: bug-review
description: Troubleshoot and diagnose a bug, exception, log error, or failing test.
---

# Bug Investigate

Systematically reproduce, diagnose, locate, and fix bugs, runtime crashes, failed assertions, or log errors.

## Instructions
1. **Analyze the Symptom**:
   - Parse any error messages, stack traces, log files, or description of the unexpected behavior.
   - Note the exact files, line numbers, or components implicated in the logs.
2. **Locate and Read the Code**:
   - Open and analyze the files mentioned in the traceback.
   - Trace the flow of data and control leading to the failure point.
3. **Formulate Hypotheses**:
   - Determine the possible causes of the bug (e.g., race conditions, unhandled nil pointers, off-by-one errors, state mismatches, incorrect API usage, or environment config issues).
4. **Isolate and Reproduce**:
   - Suggest or run small reproduction scripts or tests if applicable.
   - Trace the inputs that trigger the edge case or error.
5. **Develop a Solution**:
   - Formulate the most robust, minimal fix that addresses the root cause without introducing side effects.
   - Ensure the solution respects existing design patterns and codebase constraints.
6. **Prevent Regression**:
   - Propose a test case (unit or integration test) that would fail under the original bug but passes with the proposed fix.

## Output Format
Structure the bug investigation response as follows:
- **Diagnostic Summary**: A brief, clear explanation of the symptom and user impact.
- **Root Cause Analysis**: Detailed explanation of *why* the failure occurs, pointing to specific file paths, line numbers, and variables.
- **Proposed Solution**: Fenced diff block showing the recommended code changes.
- **Verification Plan**: Step-by-step instructions (e.g., specific test commands) to verify the fix works and does not break other parts of the application.

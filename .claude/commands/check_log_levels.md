Cross-check log levels in Go code against the logging guidelines in CONTRIBUTING.md.

## Process

1. Search for all logging statements in the codebase using patterns:
   - `logger.Debug(`, `logger.Info(`, `logger.Warn(`, `logger.Error(`
   - `logger.WithError(`
   - Other common logging patterns

2. For each logging statement found, verify:
   - **Severity appropriateness**: Is the log level correct per guidelines?
     - `Debug`: High noise, developer diagnostics (branch decisions, cache hits/misses, detailed timings)
     - `Info`: Low noise, lifecycle/business events, important operations (service started, subscription created)
     - `Warn`: Unexpected but recoverable (auto-retries, fallbacks, suspicious inputs, CSRF forbidden, rate limits)
     - `Error`: Operation failed, needs action (5xx errors, data loss risk, config invalid at startup)

   - **Message style**:
     - Message is short, stable, human-readable
     - No variable data embedded in message (should be in attributes)
     - Good: `logger.Info("user created", slog.String("user_id", userID))`
     - Bad: `logger.Info("user created: %s", user.ID)`

   - **Noise level**:
     - Avoid high volume Info logs (per-request logs should be Debug)
     - Error only when action/investigation is required

   - **Security**:
     - No secrets or PII in logs

   - **No duplicate error logs**: Check if error is logged multiple times across layers

3. Generate a report with:
   - File path and line number
   - Current log statement
   - Issue identified (if any)
   - Recommended fix (if needed)

4. Optionally fix issues if requested by user

## Usage

- `/check_log_levels` - Check all logging statements in the codebase (report only, console output)
- `/check_log_levels <path>` - Check logging in specific file or directory (report only, console output)
- `/check_log_levels --report <filename.md>` - Check and export report to markdown file
- `/check_log_levels <path> --report <filename.md>` - Check specific path and export report to markdown file
- `/check_log_levels --fix` - Check and fix all logging issues, commit each file separately
- `/check_log_levels --fix <path>` - Check and fix logging in specific file or directory, commit each file separately
- `/check_log_levels --fix --report <filename.md>` - Fix issues, commit, and export detailed report to markdown file

## Fix and Commit Mode (--fix flag)

When `--fix` flag is used:

1. For each file with logging issues:
   - Apply all necessary fixes to that file
   - Create a git commit for that file with message format:
     ```
     Fix log levels in <filename>

     - [List specific changes made, e.g.:]
     - Change Info to Debug for high-volume per-request logs
     - Fix message format: move variable data to attributes
     - Change Error to Warn for recoverable retry scenarios

     ```

2. After fixing each file:
   - Show summary of changes made
   - List files that had no issues (unchanged)

3. At the end, provide:
   - Total files fixed and committed
   - Total files checked but unchanged
   - Summary of all changes across all files

## Report Export Mode (--report flag)

When `--report <filename.md>` flag is used:

1. Generate a comprehensive markdown report including:
   - **Executive Summary**: Overview of findings
   - **Statistics**: Total files checked, issues found, issues fixed (if --fix used)
   - **Issues by Severity**: Group issues by type (Error→Warn, Info→Debug, etc.)
   - **Detailed Findings**: For each file with issues:
     - File path with clickable links
     - Line numbers with clickable links to specific lines
     - Current log statement (code block)
     - Issue description
     - Recommended fix (or actual fix applied if --fix used)
     - Rationale for the change
   - **Files Without Issues**: List of checked files with no problems
   - **Git Commits**: List of commits created (if --fix used)
   - **Appendix**: Full logging guidelines reference

2. Report format:
   - Use proper markdown formatting with headers, code blocks, and lists
   - Include clickable file links: `[filename.go](path/to/filename.go)`
   - Include clickable line links: `[filename.go:123](path/to/filename.go#L123)`
   - Use badges/emojis for visual clarity (✅ correct, ❌ issue, ⚠️ warning)
   - Include timestamps and metadata

3. Save the report to the specified filename in the current working directory

## Examples

```bash
# Check and display report in console
/check_log_levels pkg/lib/config

# Check and export to markdown file
/check_log_levels pkg/lib/config --report log-audit-report.md

# Fix issues and export comprehensive report
/check_log_levels pkg/lib/config --fix --report log-fix-report.md

# Check entire codebase and export
/check_log_levels --report full-log-audit.md
```

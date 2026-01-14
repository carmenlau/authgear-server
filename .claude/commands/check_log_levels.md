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

- `/check_log_levels` - Check all logging statements in the codebase
- `/check_log_levels <path>` - Check logging in specific file or directory

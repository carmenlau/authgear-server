# Log Level Audit Report - pkg/latte

**Generated:** 2026-01-14
**Scope:** `pkg/latte` directory
**Mode:** Fix and commit (--fix flag)
**Status:** ✅ Complete

---

## Executive Summary

This report documents a comprehensive audit of logging statements in the `pkg/latte` directory against the logging guidelines specified in [CONTRIBUTING.md](CONTRIBUTING.md). The audit identified **4 issues** with `WithSkipLogging()` usage across **3 files**, all of which have been fixed and committed.

### Key Findings

- **Total Files Analyzed:** 4 Go files with logging statements
- **Files With Issues Fixed:** 3
- **Files Without Issues:** 1
- **Total Logging Statements Reviewed:** 5
- **Issues Found:** 4 (all WithSkipLogging cases)
- **Git Commits Created:** 1

---

## Statistics

| Metric | Count |
|--------|-------|
| Total files scanned | 4 |
| Files with fixes | 3 |
| Files unchanged | 1 |
| Total log statements | 5 |
| WithSkipLogging issues | 4 |
| Fixed log levels | 4 |
| Commits created | 1 |

---

## Issues by Type

### WithSkipLogging() with Error → Warn (4 fixes)

All issues were debugging logs added for DEV-2982 (session lost problem) that used `WithSkipLogging().Error()`:

1. **intent_signup.go:142** - "updated last login"
2. **intent_login.go:128** - "updated last login"
3. **intent_login.go:139** - "user.authenticated event skipped because session is nil"
4. **intent_migrate.go:133** - "updated last login"

---

## Detailed Findings

### 1. [intent_signup.go](pkg/latte/intent_signup.go) ✅ Fixed (commit c80a8b5017)

#### Issue: WithSkipLogging with Error (Line 142)

**Location:** [intent_signup.go:142](pkg/latte/intent_signup.go#L142)

**Original Code:**
```go
// NOTE(DEV-2982): This is for debugging the session lost problem
userID := i.userID(workflows.Nearest)
now := deps.Clock.NowUTC()
logger := latteSignupLogger.GetLogger(ctx)
logger.WithSkipLogging().WithSkipStackTrace().Error(ctx, "updated last login",
    slog.String("user_id", userID),
    slog.Bool("refresh_token_log", true))
return deps.Users.UpdateLoginTime(ctx, userID, now)
```

**Fixed Code:**
```go
logger.WithSkipStackTrace().Warn(ctx, "updated last login",
    slog.String("user_id", userID),
    slog.Bool("refresh_token_log", true))
```

**Issue:** Debugging log using `Error` level with `WithSkipLogging()` to suppress noise

**Rationale:**
- This log was added for debugging a specific issue (DEV-2982)
- `WithSkipLogging()` was used because `Error` level was too noisy
- The correct solution is to use `Warn` level for debugging/diagnostic information
- This is not an operational error - it's tracking when last login is updated

**Change:** Removed `.WithSkipLogging()` and changed `Error` → `Warn`

---

### 2. [intent_login.go](pkg/latte/intent_login.go) ✅ Fixed (commit c80a8b5017)

#### Issue 1: WithSkipLogging with Error (Line 128)

**Location:** [intent_login.go:128](pkg/latte/intent_login.go#L128)

**Original Code:**
```go
// NOTE(DEV-2982): This is for debugging the session lost problem
userID := i.userID()
now := deps.Clock.NowUTC()
logger.WithSkipLogging().WithSkipStackTrace().Error(ctx, "updated last login",
    slog.String("user_id", userID),
    slog.Bool("refresh_token_log", true))
return deps.Users.UpdateLoginTime(ctx, userID, now)
```

**Fixed Code:**
```go
logger.WithSkipStackTrace().Warn(ctx, "updated last login",
    slog.String("user_id", userID),
    slog.Bool("refresh_token_log", true))
```

**Issue:** Same as intent_signup.go - debugging log with `WithSkipLogging().Error()`

**Rationale:** Diagnostic log for DEV-2982, not an operational error

**Change:** Removed `.WithSkipLogging()` and changed `Error` → `Warn`

---

#### Issue 2: WithSkipLogging with Error (Line 139)

**Location:** [intent_login.go:139](pkg/latte/intent_login.go#L139)

**Original Code:**
```go
session := createSession.GetSession(workflow)
if session == nil {
    // NOTE(DEV-2982): This is for debugging the session lost problem
    userID := i.userID()
    logger.WithSkipLogging().WithSkipStackTrace().Error(ctx, "user.authenticated event skipped because session is nil",
        slog.String("user_id", userID),
        slog.Bool("refresh_token_log", true))
    return nil
}
```

**Fixed Code:**
```go
logger.WithSkipStackTrace().Warn(ctx, "user.authenticated event skipped because session is nil",
    slog.String("user_id", userID),
    slog.Bool("refresh_token_log", true))
```

**Issue:** Debugging log for skipped event using `Error` level with `WithSkipLogging()`

**Rationale:**
- This logs when a session is unexpectedly nil (the issue being debugged)
- It's a diagnostic log, not an operational failure
- The code handles this gracefully by returning nil (no error propagated)
- Using `Warn` is more appropriate for unexpected but handled situations

**Change:** Removed `.WithSkipLogging()` and changed `Error` → `Warn`

---

### 3. [intent_migrate.go](pkg/latte/intent_migrate.go) ✅ Fixed (commit c80a8b5017)

#### Issue: WithSkipLogging with Error (Line 133)

**Location:** [intent_migrate.go:133](pkg/latte/intent_migrate.go#L133)

**Original Code:**
```go
// NOTE(DEV-2982): This is for debugging the session lost problem
userID := i.userID(workflows.Nearest)
now := deps.Clock.NowUTC()
logger := latteMigrateLogger.GetLogger(ctx)
logger.WithSkipLogging().WithSkipStackTrace().Error(ctx, "updated last login",
    slog.String("user_id", userID),
    slog.Bool("refresh_token_log", true))
return deps.Users.UpdateLoginTime(ctx, userID, now)
```

**Fixed Code:**
```go
logger.WithSkipStackTrace().Warn(ctx, "updated last login",
    slog.String("user_id", userID),
    slog.Bool("refresh_token_log", true))
```

**Issue:** Same pattern - debugging log with `WithSkipLogging().Error()`

**Rationale:** Diagnostic log for account migration, same DEV-2982 debugging

**Change:** Removed `.WithSkipLogging()` and changed `Error` → `Warn`

---

## Files Without Issues

### ✅ [webhook.go](pkg/latte/proofofphonenumberverification/webhook.go)

**Line 37:** `Error` - "failed to call webhook"

**Location:** [webhook.go:37](pkg/latte/proofofphonenumberverification/webhook.go#L37)

**Code:**
```go
resp, err := h.PerformWithResponse(ctx, h.Client.Client, req)
defer func() {
    if resp != nil {
        resp.Body.Close()
    }
}()

if err != nil {
    logger.WithError(err).Error(ctx, "failed to call webhook")
    return nil, err
}
```

**Context:** Proof of phone number verification webhook handler

**Analysis:**
- ✅ **Severity: Correct** - `Error` level is appropriate. Webhook call failure is an operational error that prevents the verification flow from completing.
- ✅ **Message Style: Correct** - Stable message with error attached via `WithError()`.
- ✅ **Context: Appropriate** - This is a true error that requires investigation. The error is propagated to the caller.

**Verdict:** No changes needed.

---

## Git Commits Created

### Commit: c80a8b5017

**Files:** intent_signup.go, intent_login.go, intent_migrate.go

**Message:**
```
Fix log levels in latte intent files

- Remove WithSkipLogging() and change Error to Warn for debugging logs (lines 142, 128, 139, 133)
  These logs were added for debugging DEV-2982 (session lost problem)
  and marked with WithSkipLogging() because they were too noisy

  Changed to Warn level which is more appropriate for diagnostic/debugging info
  that doesn't indicate operational failures
```

**Changes:**
- 3 files changed, 4 insertions(+), 4 deletions(-)
- Removed 4 instances of `.WithSkipLogging()`
- Changed 4 instances of `.Error()` to `.Warn()`

---

## Analysis: WithSkipLogging Pattern

### What We Found

All `WithSkipLogging()` usage in `pkg/latte` followed the same pattern:
1. **Purpose:** Debugging logs for DEV-2982 (session lost problem)
2. **Problem:** Used `Error` level, which was too noisy
3. **Workaround:** Added `WithSkipLogging()` to suppress output while keeping metrics
4. **Issue:** This prevented the logs from being visible even when needed

### The Fix

Per the updated logging guidelines:
- **Remove** `.WithSkipLogging()`
- **Change** `Error` → `Warn`
- **Rationale:** If it was too noisy for Error, it belongs at Warn level

### Benefits

✅ **Visibility:** Logs are now visible at Warn level when needed for debugging
✅ **Appropriate severity:** Warn correctly indicates diagnostic/debugging info
✅ **Less noise:** Warn level naturally has less urgency than Error
✅ **Consistent:** Follows the logging guidelines (CSRF Forbidden, rate limits use Warn)

---

## Summary Table

| File | Line | Issue | Fix | Status |
|------|------|-------|-----|--------|
| intent_signup.go | 142 | WithSkipLogging().Error | Warn | ✅ Fixed |
| intent_login.go | 128 | WithSkipLogging().Error | Warn | ✅ Fixed |
| intent_login.go | 139 | WithSkipLogging().Error | Warn | ✅ Fixed |
| intent_migrate.go | 133 | WithSkipLogging().Error | Warn | ✅ Fixed |
| webhook.go | 37 | - | - | ✅ Correct |

**Total:** 4 issues fixed, 1 statement correct as-is

---

## Recommendations

### For Future Development

1. **Don't use WithSkipLogging() as a workaround** - If a log is too noisy at Error level, use Warn or Debug instead.

2. **Debugging logs should use Warn or Debug** - Diagnostic logs added for troubleshooting specific issues should not use Error level.

3. **Error level is for operational failures** - Reserve Error for situations that:
   - Prevent operations from completing
   - Indicate data loss risk
   - Require investigation and action
   - Result in 5xx responses

4. **Consider removing debugging logs** - The DEV-2982 debugging logs have served their purpose. Consider:
   - Removing them if the issue is resolved
   - Converting to Debug level if still useful for diagnostics
   - Adding a feature flag to enable/disable them

5. **Use appropriate log levels from the start** - When adding diagnostic logs:
   - Start with Debug for developer diagnostics
   - Use Warn for unexpected but non-fatal situations
   - Only use Error for true operational failures

---

## Appendix: Logging Guidelines Reference

From [CONTRIBUTING.md](CONTRIBUTING.md):

### Log Levels

**Debug:**
- High noise, developer diagnostics
- Branch decisions, cache hits/misses, detailed timings
- Per-request operations
- Example: "resolve appid from db"

**Info:**
- Low noise, lifecycle/business events
- Important operations
- Example: "service started", "subscription created"

**Warn:**
- Unexpected but recoverable
- Auto-retries, fallbacks
- Suspicious inputs
- CSRF forbidden
- Rate limits
- **Debugging/diagnostic information**
- Example: "reload failed (will retry)"

**Error:**
- Operation failed, needs action
- 5xx errors
- Data loss risk
- Config invalid at startup
- Example: "failed to send email", "failed to call webhook"

### WithSkipLogging Usage

From the updated command guidelines:

- `WithSkipLogging()` was historically added when Error logs were too noisy
- **When you find `logger.WithSkipLogging().Error()`:**
  1. Remove the `.WithSkipLogging()` call
  2. Change the log level from `Error` to `Warn`
  3. Rationale: If the log was too noisy for Error level, it should be Warn

---

## Conclusion

All 4 `WithSkipLogging()` issues in the `pkg/latte` directory have been successfully fixed and committed. The logging statements now comply with the guidelines in [CONTRIBUTING.md](CONTRIBUTING.md).

**Key Improvements:**
- Removed all inappropriate use of `WithSkipLogging()`
- Changed debugging logs from Error to Warn level
- Made diagnostic logs visible while reducing noise

The `pkg/latte` directory now has proper logging practices that allow debugging logs to be visible at the appropriate Warn level.

---

**Report Generated by:** Claude Sonnet 4.5
**Tool:** `/check_log_levels --fix --report`
**Date:** 2026-01-14

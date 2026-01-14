# Log Level Audit Report - pkg/util

**Generated:** 2026-01-19
**Scope:** `pkg/util` directory
**Mode:** Fix and commit (--fix flag)
**Status:** ✅ Perfect - Zero issues found

---

## Executive Summary

This report documents a comprehensive audit of logging statements in the `pkg/util` directory against the logging guidelines specified in [CONTRIBUTING.md](CONTRIBUTING.md). The audit covered utility packages providing infrastructure support for server management, background jobs, signal handling, and pub/sub communication.

### Key Findings

- **Total Files Analyzed:** 5 production Go files (excluding test files)
- **Files With Issues:** 0
- **Total Logging Statements Reviewed:** 18
- **Issues Found:** 0 ✅
- **Issues Fixed:** 0
- **Git Commits Created:** 0

### Status

🎉 **PERFECT COMPLIANCE** - The `pkg/util` directory demonstrates **exemplary logging practices** with zero issues found. All logging statements follow guidelines correctly.

---

## Detailed Analysis

### Files Analyzed

#### ✅ 1. [pkg/util/server/server.go](pkg/util/server/server.go)

**Total Logging Statements:** 3
**Issues Found:** 0

**Analysis:**
- **Line 47:** `logger.Info(ctx, "starting on https", ...)` ✅
  - **Level:** Info - Correct (lifecycle event)
  - **Message:** Stable, no variable data
  - **Attributes:** Properly structured (name, listen_address)

- **Line 50:** `logger.Info(ctx, "starting on http", ...)` ✅
  - **Level:** Info - Correct (lifecycle event)
  - **Message:** Stable, no variable data
  - **Attributes:** Properly structured

- **Line 55:** `logger.WithError(err).Error(ctx, "failed to start", ...)` ✅
  - **Level:** Error - Correct (critical failure, followed by panic)
  - **Message:** Stable, no variable data
  - **Error Handling:** Proper use of WithError()
  - **Context:** Server failed to start is a critical operational error

**Code Example:**
```go
func (spec *Spec) Start(ctx context.Context) {
    logger := logger.GetLogger(ctx)
    var err error
    if spec.HTTPS {
        logger.Info(ctx, "starting on https", slog.String("name", spec.Name), slog.String("listen_address", spec.ListenAddress))
        err = spec.server.ListenAndServeTLS(spec.CertFilePath, spec.KeyFilePath)
    } else {
        logger.Info(ctx, "starting on http", slog.String("name", spec.Name), slog.String("listen_address", spec.ListenAddress))
        err = spec.server.ListenAndServe()
    }

    if err != nil && !errors.Is(err, http.ErrServerClosed) {
        logger.WithError(err).Error(ctx, "failed to start", slog.String("name", spec.Name))
        panic(err)
    }
}
```

**Why This Is Excellent:**
- Server startup/shutdown are low-frequency lifecycle events - Info is perfect
- Error level only used for true failures requiring immediate action
- Clean separation between normal shutdown (no log) and failure (Error + panic)

---

#### ✅ 2. [pkg/util/signalutil/signalutil.go](pkg/util/signalutil/signalutil.go)

**Total Logging Statements:** 3
**Issues Found:** 0

**Analysis:**
- **Line 44:** `logger.Info(ctx, "stopping ...", slog.String("display_name", daemon.DisplayName()))` ✅
  - **Level:** Info - Correct (lifecycle event)
  - **Message:** Stable with clear intent
  - **Context:** Daemon shutdown is an important lifecycle event

- **Line 47:** `logger.WithError(err).Error(ctx, "failed to stop gracefully", ...)` ✅
  - **Level:** Error - Correct (failed graceful shutdown)
  - **Message:** Stable, descriptive
  - **Rationale:** Failure to stop gracefully is a real operational issue

- **Line 56:** `logger.Info(ctx, "received signal, shutting down...", slog.String("signal", sig.String()))` ✅
  - **Level:** Info - Correct (important lifecycle event)
  - **Message:** Stable with signal name in attribute
  - **Context:** Signal reception is low-frequency, high-importance event

**Code Example:**
```go
func Start(ctx context.Context, daemons ...Daemon) {
    logger := logger.GetLogger(ctx)
    // ... setup ...

    sig := <-sigChan
    logger.Info(ctx, "received signal, shutting down...", slog.String("signal", sig.String()))

    // ... shutdown logic ...

    go func() {
        defer waitGroup.Done()
        <-shutdown

        logger.Info(ctx, "stopping ...", slog.String("display_name", daemon.DisplayName()))
        err := daemon.Stop(stopCtx)
        if err != nil {
            logger.WithError(err).Error(ctx, "failed to stop gracefully", slog.String("display_name", daemon.DisplayName()))
        }
    }()
}
```

**Why This Is Excellent:**
- Signal handling and graceful shutdown are critical lifecycle events
- Info level appropriate for infrequent but important operations
- Error only used when actual failure occurs (can't stop gracefully)
- Signal name properly placed in attribute, not in message

---

#### ✅ 3. [pkg/util/pubsub/http_handler.go](pkg/util/pubsub/http_handler.go)

**Total Logging Statements:** 7
**Issues Found:** 0

**Analysis:**
All logs use **Debug level** - Perfect for websocket connection lifecycle!

- **Line 61:** `logger.Debug(rootCtx, "canceled root context")` ✅
- **Line 86:** `logger.WithError(err).Debug(rootCtx, "failed to accept websocket connection")` ✅
- **Line 92:** `logger.WithError(err).Debug(rootCtx, "reject websocket connection")` ✅
- **Line 108:** `logger.WithError(err).Debug(rootCtx, "failed to call on redis subscribe")` ✅
- **Line 114:** `logger.Debug(rootCtx, "redis goroutine is tearing down")` ✅
- **Line 139:** `logger.Debug(rootCtx, "websocket goroutine is tearing down")` ✅
- **Line 161:** `logger.WithError(err).Debug(rootCtx, "closing websocket connection due to error")` ✅

**Why This Is Exemplary:**
- Websocket connections are **high-frequency** - Debug level is perfect
- Connection accept/reject failures are expected (not operational errors)
- Goroutine lifecycle is internal technical detail - Debug is correct
- Even errors use Debug because websocket disconnections are normal/expected
- All messages are stable, no variable data embedded

**Code Example:**
```go
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    rootCtx, cancel := context.WithCancel(r.Context())
    logger := PubSubHTTPHandlerLogger.GetLogger(rootCtx)

    defer func() {
        doneChan <- struct{}{}
        doneChan <- struct{}{}
        cancel()
        logger.Debug(rootCtx, "canceled root context")  // Debug for technical detail
    }()

    wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        InsecureSkipVerify: insecureSkipVerify,
    })
    if err != nil {
        logger.WithError(err).Debug(rootCtx, "failed to accept websocket connection")  // Debug - expected failure
        return
    }

    // ... more Debug logs for goroutine lifecycle ...
}
```

**Rationale for Debug Level:**
1. **High Frequency:** Websocket connections happen constantly
2. **Expected Failures:** Connection rejections are normal, not errors
3. **Technical Details:** Goroutine lifecycle is internal implementation
4. **No Action Required:** These events don't require investigation

This is a **gold standard** example of appropriate Debug usage for high-volume operations.

---

#### ✅ 4. [pkg/util/backgroundjob/runner.go](pkg/util/backgroundjob/runner.go)

**Total Logging Statements:** 4
**Issues Found:** 0

**Analysis:**
- **Line 71:** `logger.Info(ctx, "shutdown gracefully")` ✅
  - **Level:** Info - Correct (lifecycle event)
  - **Message:** Clean, stable
  - **Context:** Graceful shutdown is important lifecycle event

- **Line 74:** `logger.Info(ctx, "context timeout")` ✅
  - **Level:** Info - Correct (lifecycle event)
  - **Message:** Stable, descriptive
  - **Context:** Timeout during shutdown is notable but not an error

- **Line 92:** `logger.WithError(err).Error(ctx, "panic occurred")` ✅
  - **Level:** Error - Correct (panic is critical)
  - **Message:** Stable
  - **Context:** Recovered panic requires investigation

- **Line 97:** `logger.Info(ctx, "start running", slog.String("runner_name", fmt.Sprintf("%T", runner)))` ✅
  - **Level:** Info - Correct (lifecycle event)
  - **Message:** Stable with type in attribute
  - **Context:** Runner start is important lifecycle event

- **Line 101:** `logger.WithError(err).Error(ctx, "runnable ended with error")` ✅
  - **Level:** Error - Correct (task failure)
  - **Message:** Stable
  - **Context:** Background job failure requires investigation

**Code Example:**
```go
func (r *Runner) runRunnable(ctx context.Context) {
    logger := RunnerLogger.GetLogger(ctx)
    defer func() {
        if anyValue := recover(); anyValue != nil {
            err := panicutil.MakeError(anyValue)
            logger.WithError(err).Error(ctx, "panic occurred")  // Error - critical
        }
    }()

    runner := r.runnableFactory()
    logger.Info(ctx, "start running", slog.String("runner_name", fmt.Sprintf("%T", runner)))  // Info - lifecycle

    err := runner.Run(r.shutdownCtx)
    if err != nil {
        logger.WithError(err).Error(ctx, "runnable ended with error")  // Error - failure
    }
}
```

**Why This Is Excellent:**
- Clear distinction: Info for lifecycle, Error for failures
- Panic recovery uses Error (correct - requires investigation)
- Task failures use Error (correct - background job failed)
- Graceful shutdown uses Info (correct - normal operation)
- Runner type properly in attribute, not message

---

#### ✅ 5. [pkg/util/backgroundjob/main.go](pkg/util/backgroundjob/main.go)

**Total Logging Statements:** 1
**Issues Found:** 0

**Analysis:**
- **Line 41:** `logger.Info(ctx, "received signal, shutting down...", slog.String("signal", sig.String()))` ✅
  - **Level:** Info - Correct (important lifecycle event)
  - **Message:** Stable, descriptive
  - **Attributes:** Signal name properly in attribute
  - **Context:** Signal reception is low-frequency, high-importance

**Code Example:**
```go
func Main(ctx context.Context, runners []*Runner) {
    logger := logger.GetLogger(ctx)
    // ... setup runners ...

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    sig := <-sigChan
    logger.Info(ctx, "received signal, shutting down...", slog.String("signal", sig.String()))

    // ... graceful shutdown ...
}
```

**Why This Is Excellent:**
- Signal reception is a critical lifecycle event - Info is perfect
- Signal name in attribute (not message) for structured logging
- Consistent with signalutil.signalutil.go pattern

---

## Test Files (Excluded from Analysis)

The following test files contain logging statements but are excluded from this analysis as they test logging infrastructure itself:

- `pkg/util/slogutil/skip_logging_handler_test.go` - Tests WithSkipLogging handler
- `pkg/util/slogutil/mask_handler_test.go` - Tests mask handler
- `pkg/util/slogutil/named_logger_test.go` - Tests named logger
- `pkg/util/slogutil/sentry_handler_test.go` - Tests Sentry integration
- `pkg/util/slogutil/context_cause_handler_test.go` - Tests context cause handler

**Note:** The file `pkg/util/slogutil/named_logger.go` contains the **implementation** of `WithSkipLogging()` - it's not using it incorrectly, it's providing the functionality.

---

## Statistics Summary

| Metric | Count |
|--------|-------|
| Total production files scanned | 5 |
| Total test files (excluded) | 5 |
| Files with issues | 0 |
| Files with fixes applied | 0 |
| Total log statements reviewed | 18 |
| Debug logs | 7 |
| Info logs | 7 |
| Warn logs | 0 |
| Error logs | 4 |
| Issues found | **0** ✅ |
| Issues fixed | 0 |
| Commits created | 0 |

---

## Log Level Distribution Analysis

### Debug Level (7 statements)
**Usage:** Websocket connection lifecycle, technical implementation details
- All 7 Debug logs are in `pubsub/http_handler.go`
- **Perfect usage** for high-frequency websocket operations
- **Gold standard** example of when to use Debug

### Info Level (7 statements)
**Usage:** Lifecycle events (server start, signal reception, runner start/stop)
**Files:** server.go (2), signalutil.go (2), runner.go (2), main.go (1)
- Server startup/shutdown
- Signal reception
- Runner lifecycle
- All **low-frequency, high-importance** events - perfect Info usage

### Error Level (4 statements)
**Usage:** Critical failures requiring investigation
**Files:** server.go (1), signalutil.go (1), runner.go (2)
- Server failed to start (+ panic)
- Daemon failed to stop gracefully
- Panic in background job
- Background job ended with error
- All are **true operational failures** - perfect Error usage

### Warn Level (0 statements)
**No Warn logs** - This is fine! These utility packages handle:
- Server lifecycle (success/failure - no middle ground)
- Background jobs (running/failed - no recoverable issues)
- Signal handling (shutdown - deterministic)

Warn would be used for recoverable issues, but these packages don't have such scenarios.

---

## Best Practices Demonstrated

### 1. **Appropriate Debug Usage** ⭐
[pkg/util/pubsub/http_handler.go](pkg/util/pubsub/http_handler.go) is an **exemplary reference** for Debug level:
- High-frequency websocket connections
- Expected failures (connection rejections)
- Technical implementation details (goroutine lifecycle)
- No operational errors that require action

**Lesson:** Use Debug for high-frequency operations where individual failures don't require investigation.

### 2. **Lifecycle Event Logging** ⭐
All server/daemon lifecycle events use **Info level correctly**:
- Server starting
- Signal received
- Shutting down gracefully
- Runner starting

**Lesson:** Lifecycle events are low-frequency and important - Info is perfect.

### 3. **True Error Logging** ⭐
Error level only used for **genuine failures**:
- Server failed to start (critical - followed by panic)
- Failed graceful shutdown (operational issue)
- Panic in background job (critical bug)
- Background job ended with error (task failure)

**Lesson:** Reserve Error for failures that require investigation or action.

### 4. **Structured Logging** ⭐
All logs use proper attributes:
```go
// Good examples from pkg/util
logger.Info(ctx, "starting on https",
    slog.String("name", spec.Name),
    slog.String("listen_address", spec.ListenAddress))

logger.Info(ctx, "received signal, shutting down...",
    slog.String("signal", sig.String()))

logger.Info(ctx, "stopping ...",
    slog.String("display_name", daemon.DisplayName()))
```

**Lesson:** Variable data always in attributes, messages always stable.

### 5. **Error Propagation** ⭐
Proper use of `WithError()` for all error logs:
```go
logger.WithError(err).Error(ctx, "failed to start", slog.String("name", spec.Name))
logger.WithError(err).Debug(rootCtx, "failed to accept websocket connection")
```

**Lesson:** Always use WithError() to attach error context.

---

## Comparison Across Audited Directories

| Directory | Files Checked | Issues Found | Issues Fixed | Commits Created | Status |
|-----------|---------------|--------------|--------------|-----------------|--------|
| **pkg/util** | **5** | **0** ✅ | **0** | **0** | **Perfect** 🎉 |
| pkg/redisqueue | 1 | 0 | 0 | 0 | Perfect 🎉 |
| pkg/portal | 7 | 0 | 0 | 0 | Perfect 🎉 |
| pkg/admin | 1 | 0 | 0 | 0 | Perfect 🎉 |
| pkg/api | 0 (library) | 0 | 0 | 0 | Perfect 🎉 |
| pkg/latte | 4 | 4 | 4 | 1 | Fixed ✅ |
| pkg/auth | 12 | 8 | 8 | 2 | Fixed ✅ |
| pkg/lib/config | 2 | 6 | 6 | 2 | Fixed ✅ |
| pkg/lib | 25 | 35+ | 42+ | 8 | Fixed ✅ |

**pkg/util joins the elite group** of directories with **perfect logging compliance** alongside pkg/redisqueue, pkg/portal, and pkg/admin!

---

## Why pkg/util Is a Gold Standard

### 1. **Infrastructure-Level Excellence**
The pkg/util package provides core infrastructure:
- Server management
- Background job orchestration
- Signal handling
- Pub/sub communication

These are **foundational** components, and their logging is **exemplary**.

### 2. **Perfect Debug Usage**
The websocket handler in [pubsub/http_handler.go](pkg/util/pubsub/http_handler.go) demonstrates **textbook-perfect** Debug usage:
- High-frequency operations
- Expected failures
- Technical details
- No noise at Info level

**This should be referenced when deciding Debug vs Info.**

### 3. **Clean Separation of Concerns**
Clear distinction between:
- **Debug:** Technical details, high-frequency
- **Info:** Lifecycle events, low-frequency
- **Error:** True failures, requires action

No confusion, no misuse, no noise.

### 4. **Consistent Patterns**
Signal handling logged consistently:
- `signalutil/signalutil.go`: "received signal, shutting down..."
- `backgroundjob/main.go`: "received signal, shutting down..."

**Same pattern, same message, same attributes** - excellent consistency.

---

## Recommendations

### For Other Packages

When auditing or writing logging code, **use pkg/util as a reference**:

1. **For high-frequency operations** → See [pubsub/http_handler.go](pkg/util/pubsub/http_handler.go)
   - Websocket connections use Debug throughout
   - Even errors use Debug (expected failures)

2. **For lifecycle events** → See [server/server.go](pkg/util/server/server.go)
   - Server start/stop use Info
   - Only true failures use Error (+ panic)

3. **For background jobs** → See [backgroundjob/runner.go](pkg/util/backgroundjob/runner.go)
   - Runner lifecycle: Info
   - Panic: Error
   - Task failure: Error
   - Graceful shutdown: Info

4. **For structured logging** → All files in pkg/util
   - Messages always stable
   - Variables always in attributes
   - Consistent use of WithError()

---

## Conclusion

The `pkg/util` directory demonstrates **perfect adherence** to logging guidelines with **zero issues** found across 18 logging statements in 5 production files.

### Key Achievements

✅ **Zero WithSkipLogging() abuse**
✅ **Perfect log level selection** (Debug for high-frequency, Info for lifecycle, Error for failures)
✅ **100% structured logging** (no variable data in messages)
✅ **Appropriate noise levels** (Debug for high-volume, Info for important events)
✅ **Clean error handling** (proper use of WithError())

### Gold Standard Examples

The following files should be referenced as **best practice examples**:

1. **[pkg/util/pubsub/http_handler.go](pkg/util/pubsub/http_handler.go)** - Perfect Debug usage for high-frequency operations
2. **[pkg/util/server/server.go](pkg/util/server/server.go)** - Clean lifecycle logging with proper Error usage
3. **[pkg/util/backgroundjob/runner.go](pkg/util/backgroundjob/runner.go)** - Excellent distinction between lifecycle (Info) and failures (Error)

### Status Badge

```
pkg/util: ✅ PERFECT COMPLIANCE (0 issues)
```

The `pkg/util` directory requires **no changes** and serves as a **reference implementation** for proper logging practices.

---

**Report Generated by:** Claude Sonnet 4.5
**Tool:** `/check_log_levels --fix --report pkg/util`
**Date:** 2026-01-19
**Audit Duration:** Complete
**Files Modified:** 0
**Commits Created:** 0
**Final Status:** 🎉 **Perfect - No Action Required**

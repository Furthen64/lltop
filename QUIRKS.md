# Code Quirks and Improvement Opportunities

## Critical Issues

### 1. Destructive `make clean` target — ✅ COMPLETED

**File:** `Makefile:23`

**Fix applied:** Separated the targets. `clean` now only removes `bin/`. Added a new `factory-reset` target that depends on `clean` and also removes `~/.config/lltop`. The help text has been updated to reflect the new target.

### 2. Redundant port validation — ✅ COMPLETED

**File:** `internal/config/validate.go:67-68`

**Fix applied:** Removed the redundant port range check from `ValidateLaunchProfile()`. The check at line 35 in `ValidateProfileConfig()` already covers this case.

### 3. Brittle "no such file or directory" string matching — ✅ COMPLETED

**File:** `internal/history/failure_match.go:43`

**Fix applied:** Replaced `strings.Contains(err.Error(), "no such file or directory")` with the idiomatic `os.IsNotExist(err)`. Added `"os"` to the imports.

---

## Magic Numbers and Duplicated Constants

### 4. Log line limit `500` duplicated across 4 locations — ✅ COMPLETED

**Files:**
- `internal/runner/runner.go:45,161-162`
- `internal/ui/model.go:252-253,279-280`

**Fix applied:** Extracted `const MaxLogLines = 500` to `internal/config/config.go`. All four sites now reference `config.MaxLogLines`.

### 5. Log channel buffer `1024` duplicated — ✅ COMPLETED

**File:** `internal/runner/runner.go:43,67`

**Fix applied:** Extracted `const LogChBuffer = 1024` to `internal/runner/runner.go`. Both `New()` and `Launch()` now reference `LogChBuffer`.

### 6. Status strings as bare literals — ✅ COMPLETED

**Files:** `internal/runner/runner.go`, `internal/ui/views.go`

**Fix applied:** Added `StatusStopped`, `StatusRunning`, `StatusStopping`, `StatusFailed` constants to `internal/runner/runner.go`. All references across `runner.go`, `views.go`, and `views_test.go` now use the constants instead of bare string literals.

### 7. Error kind strings as bare literals — ✅ COMPLETED

**File:** `internal/parser/llama_logs.go:141-156`

**Fix applied:** Added `ErrorKind` type and named constants (`ErrorKindCUDAOutOfMemory`, `ErrorKindLoadModel`, `ErrorKindBind`, `ErrorKindUnknownArg`, `ErrorKindInvalidArg`, `ErrorKindOpenModel`) to `internal/parser/llama_logs.go`. All error kind assignments now use the constants.

### 8. Default editor strings duplicated — ✅ COMPLETED

**Files:**
- `internal/config/config.go:47-49` -- defines `"notepad"` / `"nano"` defaults
- `internal/ui/model.go:978-980` -- duplicates the same defaults

**Fix applied:** Exported `defaultEditor()` as `DefaultEditor()` in `config.go`. The `openEditor` function in `model.go` now calls `config.DefaultEditor()` instead of duplicating the OS check. Removed unused `runtime` import from `model.go`.

### 9. `"llama-server"` binary name appears in 3 files — ✅ COMPLETED

**Files:** `internal/config/first_run_wizard.go:207,279` and `internal/ui/external_process.go:55`

**Fix applied:** Added `const LlamaServerBinary = "llama-server"` to `internal/config/config.go`. All three sites now reference `config.LlamaServerBinary`.

### 10. Scanner buffer sizes as magic numbers

**File:** `internal/runner/runner.go:144`

```go
scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
```

Extract `64*1024` (initial) and `1024*1024` (max) to named constants.

---

## Duplicate Code Patterns

### 11. Log line truncation logic copied 3 times across 2 files

**Files:**
- `internal/runner/runner.go:161-162`
- `internal/ui/model.go:252-253,279-280`

The identical pattern `if len(lines) > 500 { lines = append([]string(nil), lines[len(lines)-500:]...) }` should be a shared helper function.

### 12. Filesystem walk depth logic duplicated

**File:** `internal/config/first_run_wizard.go:93-108` and `:224-239`

The `filepath.WalkDir` callback that computes depth via `filepath.Rel` + `strings.Count` and returns `filepath.SkipDir` is nearly identical in `DiscoverModelFiles` and `findNamedExecutables`. Factor into a shared `walkWithDepth` helper.

### 13. Stats snapshot constructed identically in 2 places

**Files:**
- `cmd/lltop/main.go:232-245`
- `internal/ui/model.go:296-309`

Both the headless runner and the TUI model build a `history.StatsSnapshot` from the same source fields. A helper method would eliminate duplication.

### 14. `LoadRunRecords` called twice per history refresh

**File:** `internal/ui/model.go:785-826`

`refreshRunHistoryState()` loads all records, then calls `refreshHistorySummary()` which loads all records again from disk. Pass the already-loaded records down instead.

---

## Performance Issues (Hot Path)

### 15. `detectExternalLlamaServer` spawns subprocess every render cycle

**Files:** `internal/ui/views.go:247-248,374-375` and `internal/ui/model.go:426-427`

This function runs `ps` (Linux) or `tasklist` (Windows) and is called at least 3 times per render. In a TUI where `View()` fires frequently, this is excessive process spawning. Cache the result in the model and re-detect only on relevant events.

### 16. `os.Stat` called for every profile's model file on every render

**File:** `internal/ui/views.go:100,142-151`

`modelFileSizeText()` calls `os.Stat()` in `renderProfiles()`, which runs every frame. Cache file sizes with an invalidation strategy.

### 17. `BuildCommand` called on every render

**File:** `internal/ui/views.go:340-356`

`renderedLaunchText` builds the full command string every frame. Cache and invalidate on profile/config change.

### 18. All log lines re-colorized on every viewport refresh

**File:** `internal/ui/model.go:573-581`

`refreshViewport()` re-colorizes every existing log line from scratch. Since `colorizeLogLine` is a pure function, only new lines need colorization.

---

## Suboptimal Patterns

### 19. `append([]T(nil), slice...)` instead of `slices.Clone`

**Files:** multiple locations across `internal/history/`, `internal/ui/`, and `internal/runner/`

The codebase targets Go 1.24, which has `slices.Clone` in the standard library. This is more idiomatic and self-documenting.

### 20. `atoi`/`atof` wrappers silently discard errors

**File:** `internal/parser/llama_logs.go:197-204`

```go
func atoi(s string) int {
    v, _ := strconv.Atoi(s)
    return v
}
```

While regex-validated captures make this safe in practice, the names evoke C conventions and give false confidence. Rename to `parseAtoi` or `mustParseInt` and add a comment explaining why the error discard is safe.

### 21. `\x00` null-byte separator for ExtraArgs comparison

**File:** `internal/history/failure_match.go:37,78`

Using a null byte to join `ExtraArgs` for comparison is unusual. A sorted slice comparison or `slices.Equal` would be more conventional and readable.

### 22. Inconsistent nil vs. empty slice initialization

**File:** `internal/ui/model.go`

In `NewModel` (line 102-103), slices are initialized as `[]string{}` and `[]history.Issue{}`. In `startSelected()` (lines 507-508), they're reset to `nil`. Pick one convention and stick with it.

---

## Error Handling

### 23. `os.Remove` and `tmp.Close()` errors silently ignored

**File:** `internal/ui/model.go:744,745,750,778`

Three `os.Remove` calls and one `tmp.Close()` discard errors with `_`. Stale temp files will accumulate silently. At minimum log the error, or use `defer` patterns.

### 24. Scanner errors logged but don't affect runner state

**File:** `internal/runner/runner.go:149`

Scanner errors (e.g., line too long, pipe corruption) are appended as log lines but don't set an error state. Pipe corruption would be invisible to error handling logic.

---

## Missing Documentation

### 25. Regex patterns have no example log lines

**File:** `internal/parser/llama_logs.go:51-62`

The regex patterns for parsing llama.cpp output are complex and undocumented. Each should have a comment with a sample matching log line, e.g.:

```go
// Matches: "prompt eval time =     810.49 ms /   114 tokens (...)"
promptEvalRe = regexp.MustCompile(`...`)
```

### 26. `filterConflictingExtraArgs` name is misleading

**File:** `internal/runner/command.go:83-102`

The function name suggests general conflict filtering, but it only handles `--flash-attn`/`-fa` flags. Other profile fields (`--temp`, `--ctx`, `--port`, etc.) that could be overridden via `extra_args` are not checked. Either rename to `filterFlashAttnExtraArgs` or implement comprehensive deduplication.

---

## Edge Cases

### 27. No symlink protection in filesystem walks

**File:** `internal/config/first_run_wizard.go:93-118`

`filepath.WalkDir` follows symlinks by default. A directory symlinked to an ancestor could cause infinite loops during model/executable discovery.

### 28. `colorizeLogLine` matches user content as errors

**File:** `internal/ui/views.go:567-586`

A user prompt containing the word "error" (e.g., "Please fix the error in the code") would be highlighted red. Consider more specific patterns, like anchoring to common log prefixes.

---

## Minor

### 29. `go.mod` specifies Go 1.24.13

Most projects specify only major.minor (`go 1.24`). The patch version is informational and doesn't enforce a toolchain, but the specific number may give false confidence.

### 30. `golang.org/x/text` at v0.3.8 is old

Run `go mod tidy` to update indirect dependencies.

### 31. No lint/vet targets in Makefile

Consider adding `make lint` (`go vet ./...`) and `make fmt` (`gofmt -w`) targets to catch common issues.

### 32. `panic` in `mustLoadHintRules` doesn't identify the bad rule

**File:** `internal/parser/llama_logs.go:175-185`

The panic message `"parser: invalid hint rule in hint_rules.json"` doesn't include the index or content of the invalid rule. Include diagnostic info in the panic message.

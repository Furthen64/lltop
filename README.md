# lltop


WORK IN PROGRESS
PROOF OF CONCEPT

This is meant to be something that can be put in the "llmgr" github repo.


`lltop` is a terminal launcher and monitor for `llama-server`, written in Go. It manages reusable launch profiles, starts and stops local servers, tails live logs, extracts useful runtime stats from server output, and stores a JSON record for each run.

The project is structured as a small CLI/TUI application with clear package boundaries:

- `cmd/lltop`: process entrypoint, CLI flag handling, first-run bootstrap, headless execution path, and TUI startup
- `internal/config`: global config loading, profile schema, validation, first-run wizard, and profile generation from discovered models
- `internal/runner`: command assembly plus supervised `llama-server` process execution and log capture
- `internal/parser`: regex-based parsing of `llama-server` log lines into structured runtime signals
- `internal/history`: persisted run records and recent-failure matching
- `internal/ui`: Bubble Tea interface, external-process detection, live log viewport, and user interactions

## What It Does

`lltop` sits between the operator and `llama-server`.

- It loads a global config from `~/.config/lltop/config.toml`
- It loads one or more profile files from `~/.config/lltop/profiles/*.toml`
- It can launch a selected profile either through the TUI or headlessly with `-run`
- It captures stdout and stderr into timestamped log files
- It parses logs to surface token throughput, GPU memory usage, offload counts, chat format, prompt progress, and known error patterns
- It writes a JSON run record after each completed run
- It warns before re-running a recently failed startup scenario
- It can detect an already-running external `llama-server` and present that state in the UI

## Architecture

### 1. Startup and Bootstrap

Program flow starts in [cmd/lltop/main.go](/home/fat64/github/lltop/cmd/lltop/main.go).

On startup:

1. Global config is loaded with `config.LoadGlobalConfig()`.
2. If no config file exists yet, the first-run wizard asks for a `llama-server` path and optional models directory.
3. The wizard writes the config, ensures app directories exist, creates a starter profile, and can auto-generate additional profiles from discovered model files.
4. Profiles are loaded from disk and the program branches into CLI utility mode, headless run mode, or the interactive TUI.

The first-run experience is deliberately thin: it collects only the data needed to make the program operable, then derives defaults and scaffolding from there.

### 2. Configuration Model

There are two layers of configuration:

- Global config: application-wide defaults such as `llama_server`, profile/log/run directories, editor, restart/failure confirmation behavior, and default host/port
- Profile config: launch-specific `llama-server` arguments such as model path, context size, GPU offload, cache types, batching, chat template, and free-form extra args

Profiles are TOML files represented by `config.Profile`. Defaults are applied lazily through `ApplyDefaults`, which keeps the on-disk profile format compact while still producing a complete launch specification at runtime.

Validation is split intentionally:

- `ValidateProfileConfig` checks basic schema-level constraints
- `ValidateLaunchProfile` checks launch-time requirements such as executable existence and model path existence

That separation keeps disk parsing lightweight while still preventing bad launches.

### 3. Command Assembly and Process Supervision

[internal/runner/command.go](/home/fat64/github/lltop/internal/runner/command.go) is the translation layer from a profile into a concrete `llama-server` command.

It builds:

- `Path`: executable to start
- `Args`: argv for `exec.Command`
- `Display`: shell-quoted human-readable command for the UI, clipboard, and run history

[internal/runner/runner.go](/home/fat64/github/lltop/internal/runner/runner.go) is the runtime supervisor. Its responsibilities are:

- validate launch inputs
- create a timestamped log file under `logs/`
- start the child process
- stream stdout and stderr concurrently
- append log lines to in-memory buffers and a log channel
- expose stop and kill controls
- publish completion state through `DoneCh`

This package is small but central. Everything above it treats process execution as an event source.

### 4. Log Parsing and Derived State

[internal/parser/llama_logs.go](/home/fat64/github/lltop/internal/parser/llama_logs.go) converts raw `llama-server` output into structured fields.

The parser currently extracts:

- prompt and eval token throughput
- token counts and elapsed times
- offloaded layer counts
- GPU memory totals and breakdowns
- prompt-processing progress
- chat format and context slot size
- common startup/runtime error categories such as bind failures, load failures, invalid arguments, and CUDA OOM

This parsing layer is what allows the UI and run-history system to work from logs alone instead of integrating directly with `llama-server` internals.

### 5. TUI Layer

[internal/ui/model.go](/home/fat64/github/lltop/internal/ui/model.go) contains the Bubble Tea state machine. The UI is organized around three visible panels:

- profile list
- live log viewport
- current server status and key bindings

User actions include:

- launch selected profile
- graceful stop or force kill
- restart with optional confirmation
- open or create profile files in an external editor
- duplicate a profile
- copy the resolved launch command
- toggle log auto-scroll
- inspect externally started `llama-server` processes

The UI keeps very little business logic of its own. It mostly coordinates `config`, `runner`, `parser`, and `history`.

One notable design choice is external-process awareness. If `lltop` did not launch the running `llama-server`, it can still detect the process, infer the most likely log file, tail it, and present that state as "externally running". That makes the tool operationally useful even when process ownership is mixed.

### 6. Run History and Failure Avoidance

Every completed run is serialized to JSON through [internal/history/run_record.go](/home/fat64/github/lltop/internal/history/run_record.go) and [internal/history/store.go](/home/fat64/github/lltop/internal/history/store.go).

Each record stores:

- profile and resolved launch parameters
- generated command
- start/end timestamps and duration
- exit code and exit reason
- final observed throughput and GPU stats
- captured issues inferred from logs

`history.FindRecentFailure` compares the selected profile against recent failed startup scenarios and can trigger a confirmation prompt before repeating the same failing launch. This is a practical safety feature, not just an audit trail.

## Runtime Data Flow

The core runtime loop is:

1. Load config and profiles.
2. Resolve a profile into a concrete command.
3. Launch `llama-server`.
4. Stream stdout/stderr to both disk and memory.
5. Parse log lines into structured stats and issues.
6. Render those stats in the TUI or capture them in headless mode.
7. Persist a run record when the process exits.

That is the main architectural through-line of the project.

## CLI Modes

`lltop` supports both interactive and non-interactive usage:

```bash
lltop
lltop -list-profiles
lltop -show-command PROFILE_NAME
lltop -validate PROFILE_NAME
lltop -run PROFILE_NAME
```

- `-list-profiles`: prints available profile names
- `-show-command`: prints the resolved `llama-server` command for a profile
- `-validate`: validates that a profile is launchable on the current machine
- `-run`: executes a profile headlessly while still collecting parsed stats and writing run history

## Configuration Files

Global config lives at:

```text
~/.config/lltop/config.toml
```

Profiles live under:

```text
~/.config/lltop/profiles/
```

Run records and logs live under:

```text
~/.config/lltop/runs/
~/.config/lltop/logs/
```

Example profiles are included in [examples/profiles](/home/fat64/github/lltop/examples/profiles).

## Build and Test

Build with:

```bash
make build
```

Run locally with:

```bash
make run
```

Run tests with:

```bash
go test ./...
```

Current test coverage is concentrated in:

- `internal/config`
- `internal/parser`
- `internal/runner`
- `internal/ui`

## Architectural Assessment

The codebase is small, coherent, and easy to reason about. The strongest parts of the design are:

- clean separation between config, execution, parsing, persistence, and UI
- a simple event-driven runtime model
- log-derived observability that avoids tight coupling to `llama-server`
- headless and TUI modes sharing the same core building blocks

The main tradeoffs and likely future pressure points are:

- the parser is regex-heavy, so upstream log format drift will require maintenance
- the UI model carries several responsibilities and will grow harder to change if more screens are added
- external process detection is platform-sensitive and currently more capable on Linux than on Windows
- history currently has persistence but not richer query/reporting features

For the current project size, these tradeoffs are reasonable. The architecture matches the scope well.

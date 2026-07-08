# lltop

`lltop` is a terminal profile manager and live monitor for
[`llama.cpp`](https://github.com/ggml-org/llama.cpp) `llama-server`.

It is built for running on the same machine as `llama-server`, typically over
SSH inside `tmux` or `screen`. `lltop` starts and supervises a selected server
profile, tails live logs, extracts useful runtime stats from `llama-server`
output, and writes a JSON run record for each completed run.

This project is currently a work in progress/proof of concept.

## Features

- Interactive Bubble Tea TUI for selecting and launching profiles
- TOML profile files for reusable `llama-server` configurations
- First-run setup wizard for `llama-server` and model discovery
- Headless commands for listing, validating, showing, and running profiles
- Live stdout/stderr capture into timestamped log files
- Runtime parsing for token throughput, GPU memory, offloaded layers, prompt
  progress, chat format, context slot size, and common startup errors
- JSON-backed log hint rules for known benign or explanatory backend messages
- Run history saved as JSON for completed launches
- Recent-failure warning before repeating the same failing startup scenario
- Detection of externally started `llama-server` processes

## Requirements

- Go 1.24 or newer
- A built `llama-server` binary from `llama.cpp`
- One or more local model files, usually `.gguf`
- Linux is the primary target environment

The codebase uses Go modules. Runtime configuration is stored under
`~/.config/lltop` by default.

## Build

```bash
make build
```

The binary is written to:

```text
bin/lltop
```

You can also build directly with Go:

```bash
go build -o bin/lltop ./cmd/lltop
```

Install to `~/.local/bin` with:

```bash
make install
```

## First Run

Start the TUI:

```bash
./bin/lltop
```

On first launch, `lltop` creates its config directory and runs a small setup
wizard in an interactive terminal. The wizard asks for:

- the `llama-server` executable path, or a directory containing it
- an optional models directory

If a models directory is provided, `lltop` scans up to three directory levels
for `.gguf` and `.bin` files and generates starter profiles for discovered
models.

If setup runs in a non-interactive context, `lltop` does not write the setup
files yet. Run it again in an interactive terminal to complete first-run setup.

## Usage

Interactive mode:

```bash
lltop
```

CLI modes:

```bash
lltop -list-profiles
lltop -show-command PROFILE_NAME
lltop -validate PROFILE_NAME
lltop -run PROFILE_NAME
```

- `-list-profiles` prints available profile names
- `-show-command` prints the resolved `llama-server` command
- `-validate` checks that the profile can launch on the current machine
- `-run` starts the profile without the TUI, streams logs, and writes run
  history

## TUI Keys

```text
Up/Down      Select profile
Enter        Launch selected profile
s            Stop gracefully
S            Force kill
r            Restart
e            Edit selected profile
n            Create new profile
d            Duplicate selected profile
a            Annotate latest run for selected profile
v            Show generated command
c            Copy generated command
l            Toggle log autoscroll
h or ?       Toggle expanded help
q            Quit
```

When log autoscroll is disabled:

```text
PgUp/PgDown  Scroll log
Ctrl+U/D     Half-page log scroll
Home/End     Jump to top/bottom
```

## Configuration files

Global config:

```text
~/.config/lltop/config.toml
```

Profiles:

```text
~/.config/lltop/profiles/*.toml
```

Run records and logs:

```text
~/.config/lltop/runs/
~/.config/lltop/logs/
```

Example global config:

```toml
llama_server = "/home/user/llama.cpp/build/bin/llama-server"
models_dir = "/home/user/models"
default_profile = "starter"
profiles_dir = "/home/user/.config/lltop/profiles"
runs_dir = "/home/user/.config/lltop/runs"
logs_dir = "/home/user/.config/lltop/logs"
editor = "nano"
confirm_restart = true
confirm_recent_failure = true
recent_failure_window_seconds = 120
startup_failure_seconds = 20
default_host = "0.0.0.0"
default_port = 8080
```

Paths support `~` and environment variable expansion.

## Log Hints

`lltop` can attach short explanatory notes to known log lines that are useful
context but not necessarily hard failures. These rules are defined in:

```text
internal/parser/hint_rules.json
```

Each rule currently has:

- `kind`: stable identifier for the hint
- `message`: text shown in the status panel
- `match_all`: string fragments that must all appear in the log line

This is intended for "if this, then that" style guidance. For example, the
`common_fit_params` warning about `n_gpu_layers already set by user` can be
shown as a note explaining that `llama.cpp` tried to auto-fit VRAM settings
but skipped that step because the user already set `n_gpu_layers` explicitly.

## Profiles

Profiles are TOML launch specifications. A profile-level `llama_server` can
override the global `llama_server`; otherwise the global executable path is
used.

```toml
name = "coding-q4"
description = "Qwen coder profile using q4 KV cache and high GPU offload."
model = "/path/to/model.gguf"
host = "0.0.0.0"
port = 8080
alias = "qwen"
ctx = 65536
ngl = 999
cache_k = "q4_0"
cache_v = "q4_0"
flash_attn = "auto"
temp = 0.1
top_p = 0.95
top_k = 40
min_p = 0.05
batch = 512
ubatch = 256
parallel = 1
threads = 0
jinja = true
metrics = true
no_mmap = true
chat_template = "chatml"
reasoning = "auto"
reasoning_budget = -1
extra_args = []
```

Example profiles are available in [`examples/profiles`](examples/profiles).

### Generated Command

Profiles are translated into a `llama-server` command. For example, the profile
above generates arguments like:

```bash
llama-server -m /path/to/model.gguf --host 0.0.0.0 --port 8080 -a qwen -c 65536 -ngl 999 --cache-type-k q4_0 --cache-type-v q4_0 --flash-attn auto --temp 0.1 --top-p 0.95 --top-k 40 --min-p 0.05 -b 512 -ub 256 --parallel 1 --metrics --jinja --reasoning auto --reasoning-budget -1 --no-mmap --chat-template chatml
```

`threads = 0` means `--threads` is omitted. `flash_attn` and `reasoning`
accept `auto`, `on`, or `off`. `reasoning_budget = -1` leaves thinking
unrestricted, while `0` is preserved as an explicit immediate stop. Any values
in `extra_args` are appended after the generated arguments, except conflicting
`-fa` or `--flash-attn` entries, which are ignored in favor of the first-class
`flash_attn` field.

## Run History

Each completed run writes:

- a timestamped log file under `~/.config/lltop/logs`
- a JSON run record under `~/.config/lltop/runs`

Run records include profile details, resolved command, timestamps, exit code,
final observed throughput and GPU stats, parsed issues from the logs, and
optional notes/annotations for the selected run.

The status panel also summarizes per-profile history from those run records,
including latest, average, median, range, and a small sparkline for ingestion
and generation throughput.

The recent-failure check compares the selected profile against recent failed
startup scenarios. If the same scenario failed within the configured window,
the TUI asks for confirmation before launching it again.

## Development

Run tests:

```bash
go test ./...
```

Download dependencies:

```bash
make deps
```

Tidy modules:

```bash
make tidy
```

Clean generated local files:

```bash
make clean
```

Note: `make clean` removes both `bin/` and `~/.config/lltop`.

## Project Layout

```text
cmd/lltop/          CLI entrypoint, flags, first-run bootstrap, TUI startup
internal/config/    global config, profile schema, validation, setup wizard
internal/runner/    command generation and supervised process execution
internal/parser/    llama-server log parsing
internal/history/   JSON run records and recent-failure matching
internal/ui/        Bubble Tea TUI and external process/log detection
examples/profiles/  starter profile examples
```

The main runtime flow is:

1. Load global config and profiles.
2. Resolve the selected profile into a concrete command.
3. Launch `llama-server`.
4. Stream stdout/stderr to disk and memory.
5. Parse log lines into structured stats and issues.
6. Render live state in the TUI or print logs in headless mode.
7. Persist a run record after process exit.

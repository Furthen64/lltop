# lltop — llama.cpp server profile manager and live TUI

## Project summary

`lltop` is a Linux terminal application for managing `llama.cpp` `llama-server` profiles. It should feel like a focused, btop-inspired mission-control panel for local LLM hosting.

The app runs directly on the Linux LLM server. Users connect over SSH and keep it alive inside `tmux` or `screen`. `lltop` itself does not need multi-server support in v0.1.

Primary goals:

- Select, launch, stop, and restart `llama-server` profiles.
- Edit profile settings using TOML files.
- Show live `llama-server` logs front and center. Color highlighting on key metrics and patterns in log.
- Show current process/server status.
- Parse useful llama.cpp runtime info such as tokens/sec, prompt eval speed, offloaded layers, context size, KV cache mode, errors, and startup failures.
- Save run history so the user can compare tuning experiments.
- Warn before repeating a recently failed launch scenario. 

Non-goals for v0.1:

- Do not implement multi-server orchestration.
- Do not require a daemon/backend service.
- Do not require systemd integration.
- Do not build a full web UI.

Use btop only as visual and UX inspiration: dense terminal layout, bordered panels, compact status, keyboard-driven controls, live graphs later.

---

## Implementation tech stack

Language: **Go**

Suggested TUI libraries:

- Bubble Tea
- Lip Gloss
- Bubbles

Alternative acceptable Go TUI library:

- tview

Default recommendation: Bubble Tea + Lip Gloss + Bubbles.

The app should build to a single Linux binary:

```bash
go build -o lltop ./cmd/lltop
```

---

## Target usage model

The user connects to a Linux server and attaches to a persistent terminal session:

```bash
ssh computer1
tmux attach -t lltop
```

Inside that session, `lltop` is already running or can be started:

```bash
lltop
```

`lltop` manages `llama-server` as a child process:

- Launch selected profile.
- Track child PID.
- Capture stdout/stderr.
- Send graceful stop signal.
- Send force kill signal.
- Detect process exit.
- Save run summary.

`tmux` or `screen` handles reconnect behavior. `lltop` does not need to implement remote reconnect logic.

---

## Config directory layout

Default config root:

```text
~/.config/lltop/
```

Suggested structure:

```text
~/.config/lltop/
  config.toml
  profiles/
    coding-q4.toml
    coding-q8.toml
    vision-q8.toml
  runs/
    2026-05-25_1713_coding-q4.json
    2026-05-25_1726_coding-q8.json
  logs/
    current.log
    2026-05-25_1713_coding-q4.log
```

Optional project-local override later:

```text
./.lltop/
```

But for v0.1, use only:

```text
~/.config/lltop/
```

---

## Global config format

File:

```text
~/.config/lltop/config.toml
```

Example:

```toml
llama_server = "/home/uht/llama/build/bin/llama-server"
default_profile = "coding-q4"

profiles_dir = "/home/user1/.config/lltop/profiles"
runs_dir = "/home/user1/.config/lltop/runs"
logs_dir = "/home/user1/.config/lltop/logs"

editor = ""
confirm_restart = true
confirm_recent_failure = true

recent_failure_window_seconds = 120
startup_failure_seconds = 20

default_host = "0.0.0.0"
default_port = 8080
```

Rules:

- If `editor` is empty, use `$EDITOR`.
- If `$EDITOR` is empty, fall back to `nano`.
- Create missing config directories on first run.
- If no profiles exist, create a starter profile template.

---

## Profile format

Profiles are TOML files.

Example:

```toml
name = "coding-q4"
description = "Qwen coder profile using q4 KV cache and high GPU offload."

llama_server = "/home/user1/llama/build/bin/llama-server"
model = "/home/user1/models/Qwen3-Coder-30B-A3B-Instruct-UD-Q2_K_XL.gguf"

host = "0.0.0.0"
port = 8080
alias = "qwen"

ctx = 65536
ngl = 999

cache_k = "q4_0"
cache_v = "q4_0"

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

extra_args = []
```

Important defaults for new profiles:

```toml
metrics = true
jinja = true
temp = 0.1
cache_k = "q4_0"
cache_v = "q4_0"
ctx = 65536
parallel = 1
```

If `threads = 0`, omit the `--threads` argument.

---

## llama-server command generation

The profile above should generate a command roughly like:

```bash
/home/user1/llama/build/bin/llama-server \
  -m /home/user1/models/Qwen3-Coder-30B-A3B-Instruct-UD-Q2_K_XL.gguf \
  --host 0.0.0.0 \
  --port 8080 \
  -a qwen \
  -c 65536 \
  -ngl 999 \
  --cache-type-k q4_0 \
  --cache-type-v q4_0 \
  --temp 0.1 \
  --top-p 0.95 \
  --top-k 40 \
  --min-p 0.05 \
  -b 512 \
  -ub 256 \
  --parallel 1 \
  --metrics \
  --jinja
```

The generated command should be visible in the UI before launch or in a details panel.

Validation before launch:

- `llama_server` path exists.
- `llama_server` is executable.
- `model` path exists.
- `port` is valid.
- `ctx > 0`.
- `ngl >= 0`.
- `batch > 0`.
- `ubatch > 0`.
- `parallel > 0`.

---

## Main UI layout

The main screen should prioritize:

1. Profile list
2. Live logs

Other panels should use less space.

Suggested layout:

```text
┌ profiles ─────────────┐ ┌ live llama-server log ──────────────────────────────┐
│ > coding-q4           │ │ 13:54 prompt processing progress ... progress=1.0   │
│   coding-q8           │ │ 13:54 prompt done, n_tokens=8665                   │
│   vision-q8           │ │ 14:30 eval time = ... 3.45 tokens per second       │
│                       │ │                                                     │
│ Enter launch          │ │                                                     │
│ e edit                │ │                                                     │
│ n new                 │ │                                                     │
│ d duplicate           │ │                                                     │
└───────────────────────┘ └─────────────────────────────────────────────────────┘

┌ current server ───────────────────────────────────────────────────────────────┐
│ status: running  pid: 41965  profile: coding-q4  model: Qwen3-Coder...        │
│ ctx: 65536  ngl: 40  kv: q4_0/q4_0  prompt: 140 tok/s  eval: 3.76 tok/s       │
│ offload: 41/49 layers  vram: 14.0/16.0 GiB  cpu: 48%                         │
└───────────────────────────────────────────────────────────────────────────────┘

┌ keys ─────────────────────────────────────────────────────────────────────────┐
│ Enter launch  s graceful stop  S force kill  r restart  e edit  q quit       │
└───────────────────────────────────────────────────────────────────────────────┘
```

The first version can be simpler. The key requirement is that profiles and logs are visible at the same time.

---

## Keyboard controls

Required v0.1 controls:

```text
Up/Down       Move profile selection
Enter         Launch selected profile if no server running
s             Graceful stop
S             Force kill
r             Restart current profile; confirm if currently running
e             Edit selected profile in $EDITOR
n             Create new profile from template
d             Duplicate selected profile
v             View generated command for selected profile
l             Toggle/focus log panel
h or ?        Help
q             Quit lltop, without killing llama-server unless configured otherwise
Ctrl+C        Quit lltop
```

Stop behavior:

- `s` sends graceful stop signal.
- `S` force kills the child process.
- `r` asks confirmation if a server is already running.

Suggested signal behavior:

- Graceful stop: send SIGINT first.
- If still running after timeout, offer force kill.
- Force kill: send SIGKILL.

---

## Process handling

`lltop` should launch `llama-server` as a child process and capture both stdout and stderr into the live log panel.

Requirements:

- Track PID.
- Track start time.
- Track selected profile.
- Stream stdout/stderr to UI.
- Save stdout/stderr to a timestamped log file.
- Detect normal exit.
- Detect error exit.
- Preserve last N log lines in memory for UI.
- Save run summary JSON at process exit.

When `lltop` quits while `llama-server` is running:

v0.1 default:

- Quit the TUI but leave the child process running only if this can be implemented safely.
- Otherwise, ask the user whether to stop the server before exiting.

Simpler acceptable v0.1 behavior:

- Warn that quitting will stop the child process.
- Ask confirmation.
- Later versions can implement detach behavior.

---

## Log parsing requirements

Parse llama.cpp logs for useful status lines.

Minimum patterns:

### Prompt eval timing

Example:

```text
prompt eval time =     810.49 ms /   114 tokens (    7.11 ms per token,   140.66 tokens per second)
```

Extract:

- prompt eval ms
- prompt tokens
- prompt ms/token
- prompt tokens/sec

### Generation eval timing

Example:

```text
eval time =  119350.76 ms /   449 tokens (  265.81 ms per token,     3.76 tokens per second)
```

Extract:

- generation eval ms
- generated tokens
- generation ms/token
- generation tokens/sec

This is the main generation speed number.

### Total timing

Example:

```text
total time =  120161.25 ms /   563 tokens
```

Extract:

- total ms
- total tokens

### Offloaded layers

Example:

```text
load_tensors: offloaded 26/49 layers to GPU
```

Extract:

- offloaded layers
- total layers

### GPU model buffer size

Examples:

```text
CUDA0 model buffer size =  6649.71 MiB
CUDA_Host model buffer size =  5893.77 MiB
CPU model buffer size =   127.51 MiB
```

Extract:

- CUDA0 model buffer MiB
- CUDA host model buffer MiB
- CPU model buffer MiB

### Memory breakdown

Example:

```text
llama_memory_breakdown_print: |   - CUDA0 (RTX 4070 Ti SUPER) | 15942 = 7251 + (7713 =  6649 +     900 +     163) +         978 |
```

Extract where possible:

- GPU name
- total MiB
- free MiB
- self MiB
- model MiB
- context MiB
- compute MiB
- unaccounted MiB

### Prompt progress

Example:

```text
prompt processing progress, n_tokens = 9863, batch.n_tokens = 27, progress = 1.000000
```

Extract:

- n_tokens
- batch tokens
- progress

UI meaning:

- `progress < 1.0`: prompt eval / prefill phase.
- `progress = 1.0`: prompt is loaded; generation follows.

### Chat format

Example:

```text
params_from_: Chat format: Qwen3 Coder
```

Extract:

- chat format

### Context and prompt size

Example:

```text
new prompt, n_ctx_slot = 65536, n_keep = 0, task.n_tokens = 9863
```

Extract:

- context slot size
- n_keep
- task prompt tokens

### Cancellation

Example:

```text
stop: cancel task, id_task = 1179
```

Track cancellation as an abnormal or user-initiated run event.

### Startup errors

Detect obvious startup problems:

- CUDA OOM
- model file not found
- failed to bind/listen on port
- unknown argument
- invalid argument
- failed to load model
- missing tokenizer/chat template issues
- metrics endpoint not enabled warning, if relevant
- process exits within startup failure window

---

## Run history

Every launch should create a run record.

Path:

```text
~/.config/lltop/runs/YYYY-MM-DD_HHMMSS_profile-name.json
```

Example schema:

```json
{
  "run_id": "2026-05-25_1713_coding-q4",
  "profile_name": "coding-q4",
  "started_at": "2026-05-25T17:13:00+02:00",
  "ended_at": "2026-05-25T17:16:40+02:00",
  "duration_seconds": 220,
  "exit_code": 0,
  "exit_reason": "normal",
  "llama_server": "/home/uht/llama/build/bin/llama-server",
  "model": "/home/uht/models/Qwen3-Coder-30B-A3B-Instruct-UD-Q2_K_XL.gguf",
  "host": "0.0.0.0",
  "port": 8080,
  "alias": "qwen",
  "ctx": 65536,
  "ngl": 40,
  "cache_k": "q4_0",
  "cache_v": "q4_0",
  "temp": 0.1,
  "top_p": 0.95,
  "top_k": 40,
  "min_p": 0.05,
  "batch": 512,
  "ubatch": 256,
  "parallel": 1,
  "metrics": true,
  "jinja": true,
  "generated_command": "/home/uht/llama/build/bin/llama-server ...",
  "last_prompt_tokens_per_second": 140.66,
  "last_eval_tokens_per_second": 3.76,
  "last_generated_tokens": 449,
  "last_prompt_tokens": 114,
  "offloaded_layers": 41,
  "total_layers": 49,
  "gpu_total_mib": 15942,
  "gpu_free_mib": 7251,
  "gpu_model_mib": 6649,
  "gpu_context_mib": 900,
  "gpu_compute_mib": 163,
  "issues": []
}
```

If issues are detected:

```json
"issues": [
  {
    "severity": "error",
    "kind": "cuda_oom",
    "message": "CUDA out of memory detected during startup.",
    "seen_at_seconds": 8
  }
]
```

---

## Recent failure warning

If the user tries to launch a profile/scenario that recently failed, warn before launching.

Scenario identity should include:

- profile name
- model path
- ctx
- ngl
- cache_k
- cache_v
- batch
- ubatch
- parallel
- extra_args

If a matching scenario failed within `recent_failure_window_seconds`, show a confirmation:

```text
This same launch scenario failed 10 seconds into the previous run.

Profile: coding-q8
Issue: CUDA out of memory during startup
Previous run: 2026-05-25 17:13:22

Run it again anyway? [y/N]
```

Default answer should be No.

Startup failure window:

```text
startup_failure_seconds = 20
```

If the process exits within this window with a non-zero exit code, classify it as a startup failure.

---

## Metrics and resource monitoring

Stage this work.

### v0.1

Required:

- Parse stdout/stderr logs.
- Track child process PID and status.
- Show basic process memory/CPU if easy.

### v0.2

Use llama-server `/metrics` when enabled.

Profiles should include:

```toml
metrics = true
```

Metrics endpoint:

```text
http://127.0.0.1:<port>/metrics
```

Parse useful Prometheus metrics if available.

### v0.3

Add GPU/system panels using:

- `nvidia-smi`
- `/proc/<pid>`
- `ps`
- `/proc/meminfo`

Do not block the UI if `nvidia-smi` is slow or unavailable.

---

## Profile editing

Support two profile editing methods.

### External editor

Key:

```text
e
```

Behavior:

- Open selected TOML profile in `$EDITOR`.
- Pause TUI while editor is active.
- Reload and validate profile after editor exits.
- Show validation errors in UI.

### Built-in editor

For v0.1, a minimal form is enough if time permits.

Fields:

- name
- description
- llama_server
- model
- host
- port
- alias
- ctx
- ngl
- cache_k
- cache_v
- temp
- top_p
- top_k
- min_p
- batch
- ubatch
- parallel
- jinja
- metrics
- extra_args

Built-in editor can be v0.2 if external editor works well.

---

## Startup experience

On first run:

1. Create config directories.
2. Create default `config.toml`.
3. Create starter profile.
4. Show a clear message telling the user what was created.
5. Let the user edit the starter profile before launch.

Starter profile should not assume model path exists. It can use:

```toml
model = "/path/to/model.gguf"
```

If user tries to launch with placeholder path, show validation error.

---

## Suggested repository structure

```text
lltop/
  README.md
  task.md
  go.mod
  cmd/
    lltop/
      main.go
  internal/
    app/
      app.go
      keys.go
      layout.go
    config/
      config.go
      profile.go
      validate.go
    runner/
      runner.go
      command.go
      signals.go
    parser/
      llama_logs.go
      llama_logs_test.go
    history/
      run_record.go
      store.go
      failure_match.go
    metrics/
      prometheus.go
      nvidia.go
      proc.go
    ui/
      model.go
      views.go
      styles.go
  examples/
    profiles/
      coding-q4.toml
      coding-q8.toml
```

---

## MVP milestones

### Milestone 1 — CLI skeleton and profile loading

Deliver:

- Go project builds.
- Loads global config.
- Loads profiles from TOML.
- Lists profiles.
- Validates profile fields.
- Can print generated llama-server command.

Commands:

```bash
lltop --list-profiles
lltop --show-command coding-q4
lltop --validate coding-q4
```

### Milestone 2 — Basic launcher

Deliver:

- Launch selected profile from CLI.
- Capture stdout/stderr.
- Save log file.
- Track process exit code.
- Save run JSON.

Example:

```bash
lltop --run coding-q4
```

### Milestone 3 — Log parser

Deliver:

- Parse prompt eval tok/s.
- Parse generation eval tok/s.
- Parse offloaded layer count.
- Parse prompt progress.
- Parse memory breakdown where possible.
- Add unit tests using real llama.cpp log snippets.

### Milestone 4 — First TUI

Deliver:

- Profile list panel.
- Live log panel.
- Current status panel.
- Keyboard controls:
  - Up/Down
  - Enter launch
  - s graceful stop
  - S force kill
  - r restart
  - e external edit
  - q quit

### Milestone 5 — Run history and recent failure warning

Deliver:

- Save run history JSON.
- Detect startup failures.
- Detect repeated failed scenario.
- Warn before launching same scenario again.

### Milestone 6 — Metrics/resource panel

Deliver:

- Query `/metrics` if enabled.
- Show basic process CPU/RAM.
- Optional: show `nvidia-smi` GPU usage.

---

## Acceptance criteria for v0.1

The project is useful when the user can:

1. Start `lltop` inside tmux over SSH.
2. See a list of TOML profiles.
3. Select a profile and launch `llama-server`.
4. Watch live llama.cpp logs in the TUI.
5. See whether server is running and what PID it has.
6. See latest generation tokens/sec after a completed request.
7. See offloaded layers when available.
8. Stop gracefully with `s`.
9. Force kill with `S`.
10. Restart with `r`, with confirmation.
11. Edit a profile with `$EDITOR`.
12. Save run logs and run JSON.
13. Get warned before repeating a recently failed startup scenario.

---

## Design notes

Use this mental model:

```text
profile = desired launch configuration
runner = child-process lifecycle manager
parser = extracts meaning from llama.cpp logs
history = stores runs and failure memory
ui = btop-inspired control surface
```

Keep these layers separate.

Avoid mixing:

- TUI rendering with process control
- log parsing with UI state
- profile TOML loading with command execution
- run history storage with profile validation

This will keep the project agent-friendly and easier to expand.

---

## Important llama.cpp interpretation notes

Generation speed is the `eval time` tokens/sec line, not the `prompt eval time` line.

Example:

```text
eval time = 119350.76 ms / 449 tokens (265.81 ms per token, 3.76 tokens per second)
```

This means:

```text
generation speed = 3.76 tokens/sec
```

Prompt eval speed is separate:

```text
prompt eval time = 810.49 ms / 114 tokens (7.11 ms per token, 140.66 tokens per second)
```

This means prompt ingestion/prefill speed, not answer generation speed.

Prompt progress:

```text
progress < 1.0
```

means prompt eval/prefill is ongoing.

```text
progress = 1.0
```

means prompt eval is done and answer generation follows.

If GPU usage is high during prompt progress but low after progress reaches 1.0, the decode/generation phase may be bottlenecked by CPU layers, sampling, memory transfer, or incomplete GPU offload.

---

## Future ideas after v0.1

- Built-in profile form editor.
- Profile comparison screen.
- Benchmark mode.
- One-button duplicate profile with changed `ngl`/KV cache.
- Suggested warning when CPU decode bottleneck is suspected.
- Tmux helper command:
  - `lltop tmux-start`
  - `lltop tmux-attach`
- systemd user service integration.
- Remote controller mode.
- More complete GPU graphs.
- Export benchmark table to Markdown/CSV.
- HTTP API for controlling the local server.

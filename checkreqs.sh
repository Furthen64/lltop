#!/usr/bin/env bash
set -u
set -o pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

if [[ -t 1 ]]; then
  green=$'\033[32m'
  yellow=$'\033[33m'
  red=$'\033[31m'
  blue=$'\033[34m'
  bold=$'\033[1m'
  reset=$'\033[0m'
else
  green=""
  yellow=""
  red=""
  blue=""
  bold=""
  reset=""
fi

failures=0
warnings=0
next_steps=()

info() {
  printf "%s==>%s %s\n" "$blue" "$reset" "$*"
}

pass() {
  printf "%s[ok]%s %s\n" "$green" "$reset" "$*"
}

warn() {
  warnings=$((warnings + 1))
  printf "%s[warn]%s %s\n" "$yellow" "$reset" "$*"
}

fail() {
  failures=$((failures + 1))
  printf "%s[fail]%s %s\n" "$red" "$reset" "$*"
}

add_next_step() {
  local step=$1
  local existing
  for existing in "${next_steps[@]:-}"; do
    if [[ "$existing" == "$step" ]]; then
      return
    fi
  done
  next_steps+=("$step")
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

dir_writable() {
  local dir=$1
  local probe

  mkdir -p "$dir" 2>/dev/null || return 1
  probe=$(mktemp "$dir/.lltop-write-check.XXXXXX" 2>/dev/null) || return 1
  rm -f "$probe"
}

version_ge() {
  local left=$1
  local right=$2
  [[ "$(printf '%s\n%s\n' "$right" "$left" | sort -V | tail -n1)" == "$left" ]]
}

recommend_go_install() {
  if have_cmd apt-get; then
    printf 'Install Go: sudo apt-get update && sudo apt-get install -y golang-go'
  elif have_cmd dnf; then
    printf 'Install Go: sudo dnf install -y golang'
  elif have_cmd pacman; then
    printf 'Install Go: sudo pacman -S go'
  elif have_cmd zypper; then
    printf 'Install Go: sudo zypper install -y go'
  elif have_cmd apk; then
    printf 'Install Go: sudo apk add go'
  else
    printf 'Install Go 1.24.13 or newer, then re-run ./checkreqs.sh'
  fi
}

min_go_version=$(awk '/^go / { print $2; exit }' go.mod)

printf "%slltop build environment check%s\n" "$bold" "$reset"
printf "Repository: %s\n" "$PWD"
printf "Minimum Go version: %s\n\n" "${min_go_version:-unknown}"

info "Checking host platform"
os_name=$(uname -s 2>/dev/null || echo "unknown")
arch_name=$(uname -m 2>/dev/null || echo "unknown")
printf "OS: %s\n" "$os_name"
printf "Arch: %s\n" "$arch_name"
if [[ "$os_name" != "Linux" ]]; then
  warn "Linux is the primary target environment in this repo; other platforms may work but are less tested."
else
  pass "Linux host detected."
fi

printf "\n"
info "Checking required commands"
missing_cmds=()
found_cmds=()
for cmd in bash go git make; do
  if have_cmd "$cmd"; then
    found_cmds+=("$cmd")
  else
    missing_cmds+=("$cmd")
    fail "Missing required command: $cmd"
    if [[ "$cmd" == "go" ]]; then
      add_next_step "$(recommend_go_install)"
    else
      add_next_step "Install missing command: $cmd"
    fi
  fi
done
if ((${#found_cmds[@]} > 0)); then
  pass "Found: ${found_cmds[*]}"
fi

printf "\n"
info "Checking Go toolchain"
if have_cmd go; then
  go_version_raw=$(go version 2>/dev/null || true)
  if [[ -n "$go_version_raw" ]]; then
    printf "%s\n" "$go_version_raw"
    go_version=$(awk '{print $3}' <<<"$go_version_raw")
    go_version=${go_version#go}
    if [[ -n "$min_go_version" ]] && version_ge "$go_version" "$min_go_version"; then
      pass "Go version $go_version satisfies go.mod requirement $min_go_version."
    elif [[ -n "$min_go_version" ]]; then
      fail "Go version $go_version is older than required version $min_go_version."
      add_next_step "$(recommend_go_install)"
    else
      warn "Could not read the minimum Go version from go.mod."
    fi
  else
    fail "Unable to read Go version."
    add_next_step "Verify the Go installation, then re-run ./checkreqs.sh"
  fi

  for env_name in GOPATH GOMODCACHE GOCACHE; do
    env_value=$(go env "$env_name" 2>/dev/null || true)
    if [[ -n "$env_value" ]]; then
      if ! dir_writable "$env_value"; then
        fail "$env_name is not writable. Fix permissions or point it at a writable directory before building."
        add_next_step "Fix $env_name permissions or set it to a writable directory"
      fi
    else
      warn "Could not read $env_name from go env."
    fi
  done
fi

printf "\n"
info "Checking module download"
if have_cmd go; then
  if go mod download; then
    pass "Go modules downloaded successfully."
  else
    fail "go mod download failed. Check network access and your Go proxy settings."
    add_next_step "Run 'go mod download' after fixing network access or Go proxy settings"
  fi
fi

printf "\n"
info "Checking clean build"
if have_cmd go; then
  tmp_bin=$(mktemp /tmp/lltop-build-check.XXXXXX)
  if go build -o "$tmp_bin" ./cmd/lltop; then
    pass "Temporary build succeeded."
    rm -f "$tmp_bin"
  else
    fail "go build failed for ./cmd/lltop."
    add_next_step "Run 'go build ./cmd/lltop' and fix the reported build error"
    rm -f "$tmp_bin"
  fi
fi

printf "\n"
info "Checking runtime-only extras"
if command -v llama-server >/dev/null 2>&1; then
  pass "Found llama-server in PATH."
else
  warn "llama-server not found in PATH. This does not block go build, but it is required to run lltop against llama.cpp."
  add_next_step "Build llama.cpp and make 'llama-server' available in PATH before running lltop"
fi

if find examples -type f -name '*.toml' -print -quit | grep -q .; then
  pass "Example profiles are present."
else
  warn "No example profiles were found under examples/."
fi

printf "\n"
if (( failures > 0 )); then
  printf "%sResult:%s %d failure(s), %d warning(s)\n" "$red" "$reset" "$failures" "$warnings"
  if ((${#next_steps[@]} > 0)); then
    printf "\n%sAction required%s\n" "$bold" "$reset"
    step_num=1
    for step in "${next_steps[@]}"; do
      printf "%d. %s\n" "$step_num" "$step"
      step_num=$((step_num + 1))
    done
  fi
  exit 1
fi

printf "%sResult:%s environment is ready for go build" "$green" "$reset"
if (( warnings > 0 )); then
  printf " (%d warning(s))" "$warnings"
fi
printf "\n"
if (( warnings > 0 && ${#next_steps[@]} > 0 )); then
  printf "\n%sBefore runtime use%s\n" "$bold" "$reset"
  step_num=1
  for step in "${next_steps[@]}"; do
    printf "%d. %s\n" "$step_num" "$step"
    step_num=$((step_num + 1))
  done
fi

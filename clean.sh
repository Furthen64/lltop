#!/usr/bin/env bash

set -euo pipefail

app_dir="${HOME}/.config/lltop"

if [[ ! -e "$app_dir" ]]; then
    printf 'Nothing to remove. Not found: %s\n' "$app_dir"
    exit 0
fi

printf 'The following lltop files will be removed:\n'
find "$app_dir" -mindepth 0 -print | sort
printf '\n'

read -r -p 'Type "yes" to delete these files: ' reply

if [[ "$reply" != "yes" ]]; then
    printf 'Aborted. Nothing was removed.\n'
    exit 0
fi

rm -rf -- "$app_dir"
printf 'Removed: %s\n' "$app_dir"

#!/usr/bin/env bash
# Moved to `lab webui comfyui-env` (issue #6; same arguments, same results, checked side by side). This wrapper stays until
# nothing calls it (see ~/.cache/lab-wrapper-calls.log), then it goes. Help: lab webui comfyui-env --help
# Full path on purpose: the Rerun button (hub/actions) runs benchmarks without ~/.local/bin on its PATH.
LAB="$HOME/.local/bin/lab"
[ -x "$LAB" ] || { echo "make-comfyui-env.sh: lab is not installed. Run: ~/homelab/cli/build.sh install" >&2; exit 1; }
echo "$(date '+%F %T') hub/openwebui/make-comfyui-env.sh $(ps -o comm= $PPID 2>/dev/null)" >> "$HOME/.cache/lab-wrapper-calls.log" 2>/dev/null || true
exec "$LAB" webui comfyui-env "$@"

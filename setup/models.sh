#!/usr/bin/env bash
# Download every model listed in setup/models-<machine>.txt into /srv/models, then check sha256.
#   ~/homelab/setup/models.sh                 (uses this machine's name)
#   ~/homelab/setup/models.sh zaphod gemma    (only lines whose destination contains "gemma")
# Safe to re-run: finished files are skipped, partial ones resume. Pinned revisions: same bytes every time.
# Takes hours for a full machine. Run inside tmux, or with nohup.
set -uo pipefail
H=${1:-$(hostname)}; FILTER=${2:-}
LIST=$(dirname "$(readlink -f "$0")")/models-$H.txt
[ -f "$LIST" ] || { echo "no list $LIST"; exit 1; }
cd /srv/models || exit 1
fail=0
while read -r repo rev path dest size sha; do
  case "$repo" in ''|\#*) continue ;; esac
  [ -n "$FILTER" ] && [[ "$dest" != *"$FILTER"* ]] && continue
  mkdir -p "$(dirname "$dest")"
  if [ -f "$dest" ] && [ "$(stat -c %s "$dest")" = "$size" ]; then echo "have $dest"
  else
    echo "$(date +%T) downloading $dest ($((size/1000000)) MB)"
    wget -q -c -O "$dest" "https://huggingface.co/$repo/resolve/$rev/$path" || { echo "FAILED download $dest"; fail=1; continue; }
  fi
  echo "$(date +%T) checking $dest"
  echo "$sha  $dest" | sha256sum -c --quiet || { echo "CHECKSUM FAILED $dest"; fail=1; }
done < "$LIST"
grep -h '^# UNKNOWN' "$LIST" | sed 's/^/not in the list, get by hand: /'
[ $fail = 0 ] && echo MODELS-OK || echo MODELS-FAILED

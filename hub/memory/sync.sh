#!/usr/bin/env bash
# Keep LocalRecall's copy of the lab's notes current (plans/agent-infra.md step 4).
#   ./sync.sh            upload changed Markdown now; good result ends with "sync: N uploaded, M removed, K unchanged"
#   ./sync.sh install    run it every hour at :41 (cron); log in ~/.cache/memory-sync.log
set -euo pipefail
HUB=$(cd "$(dirname "$0")/.." && pwd)
LOG=$HOME/.cache/memory-sync.log
case "${1:-run}" in
  install)
    line="41 * * * * $HUB/memory/sync.sh >> $LOG 2>&1"
    # "|| true": with no crontab yet, crontab -l and grep both fail, which must not stop the script
    { crontab -l 2>/dev/null | grep -v 'memory/sync.sh' || true; echo "$line"; } | crontab -
    crontab -l | grep -q 'memory/sync.sh' || { echo "install FAILED"; exit 1; }
    echo "installed: $line" ;;
  run)
    echo "== $(date '+%F %T')"
    cd "$HUB" && docker compose --profile sync run --rm -T memory-sync ;;
  *) sed -n 2,4p "$0"; exit 1 ;;
esac

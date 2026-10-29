#!/usr/bin/env bash
# Nightly backup of zaphod's data that is not in git -> marvin:~/backups/zaphod/<date>/
#   ~/homelab/setup/backup-to-marvin.sh            run a backup now
#   ~/homelab/setup/backup-to-marvin.sh install    run it every night at 04:17 (crontab)
#   ~/homelab/setup/backup-to-marvin.sh list       show the snapshots on marvin
# What: hub/data (Open WebUI chats/uploads, Beszel, MLflow, ComfyUI, ...), llama-swap activity, all .env files.
# Not: model files (setup/models.sh gets them back), caches, Caddy certificates (Caddy fetches new ones).
# Databases are copied with SQLite's online backup, so services keep running. Keeps the last 14 snapshots;
# unchanged files are hard links, so each extra snapshot costs only what changed.
# Temporary until the NAS is rebuilt (see setup/REBUILD.md section 6).
set -euo pipefail
REPO=$HOME/homelab
DEST_HOST=marvin
DEST=backups/zaphod
KEEP=14
STAGE=$HOME/.cache/backup-stage
LOG=$HOME/.cache/backup-to-marvin.log

case "${1:-run}" in
  install)
    line="17 4 * * * $REPO/setup/backup-to-marvin.sh >> $LOG 2>&1"
    # "|| true": with no crontab yet, crontab -l and grep both fail, which must not stop the script
    { crontab -l 2>/dev/null | grep -v 'backup-to-marvin.sh' || true; echo "$line"; } | crontab -
    crontab -l | grep -q 'backup-to-marvin.sh' || { echo "install FAILED"; exit 1; }
    echo "installed: $line"; exit 0 ;;
  list) ssh $DEST_HOST "du -sh $DEST/* 2>/dev/null; echo; du -sh $DEST"; exit 0 ;;
  run) ;;
  *) sed -n '2,11p' "$0"; exit 1 ;;
esac

echo "== $(date '+%F %T') backup start"
# The copy step runs as root in a container; clean up its leftovers the same way.
clean() { mkdir -p "$STAGE"; docker run --rm -v "$STAGE:/out" alpine:3.22 sh -c 'rm -rf /out/* /out/.[!.]*' ; }
clean

# 1. Copy hub/data and llama-swap/data inside a container (some files belong to root),
#    using SQLite's backup API for every database so the copy is consistent while services run.
#    Not mounted :ro because SQLite needs the -shm file of a live database; every file is still opened read-only.
docker run --rm -i --network none \
  -v "$REPO/hub/data:/src/hub-data" \
  -v "$REPO/zaphod/llama-swap/data:/src/llama-swap-data" \
  -v "$STAGE:/out" \
  python:3.13-slim-trixie python3 - <<'EOF'
import os, shutil, sqlite3
SKIP_DIRS = {"caddy", "caddy-config", "searxng-cache", "cache"}   # rebuildable
SKIP_SUFFIX = ("-wal", "-shm", "-journal")
n_db = n_file = 0
def own_all():
    for root, dirs, files in os.walk("/out"):
        for x in dirs + files:
            os.chown(os.path.join(root, x), 1000, 1000)
    os.chown("/out", 1000, 1000)
try:
  for top in ("hub-data", "llama-swap-data"):
      for root, dirs, files in os.walk(f"/src/{top}"):
          dirs[:] = [d for d in dirs if d not in SKIP_DIRS]
          out = root.replace("/src", "/out", 1)
          os.makedirs(out, exist_ok=True)
          for f in files:
              s, d = os.path.join(root, f), os.path.join(out, f)
              if f.endswith(SKIP_SUFFIX):
                  continue
              with open(s, "rb") as fh:
                  is_db = fh.read(16) == b"SQLite format 3\x00"
              if is_db:
                  src = sqlite3.connect(f"file:{s}?mode=ro", uri=True)
                  dst = sqlite3.connect(d)
                  src.backup(dst); dst.close(); src.close(); n_db += 1
              else:
                  shutil.copy2(s, d); n_file += 1
finally:
    own_all()
print(f"copied {n_db} databases (online backup) and {n_file} other files")
EOF

for must in hub-data/open-webui/webui.db hub-data/beszel/data.db hub-data/mlflow/mlflow.db; do
  [ -s "$STAGE/$must" ] || { echo "BACKUP FAILED: $must missing from the copy"; exit 1; }
done

# 2. Secrets (also in Proton Pass)
mkdir -p "$STAGE/env"
for f in hub/.env zaphod/llama-swap/.env; do
  [ -f "$REPO/$f" ] && install -m 600 "$REPO/$f" "$STAGE/env/$(echo "$f" | tr / _)"
done
chmod 700 "$STAGE/env"

# 3. Send to marvin as a dated snapshot; unchanged files are hard links to the previous snapshot.
SNAP=$(date +%F_%H%M)
ssh $DEST_HOST "mkdir -p $DEST && chmod 700 backups"
PREV=$(ssh $DEST_HOST "ls -1d $DEST/20* 2>/dev/null | tail -1" || true)
rsync -a --delete ${PREV:+--link-dest="$HOME/$PREV"} "$STAGE/" "$DEST_HOST:$DEST/$SNAP/"
ssh $DEST_HOST "cd $DEST && ls -1d 20* | head -n -$KEEP | xargs -r rm -rf"
echo "== $(date '+%F %T') done: $DEST_HOST:$DEST/$SNAP ($(du -sh "$STAGE" | cut -f1))"
clean

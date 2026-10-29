#!/usr/bin/env bash
# Real-use speed of one model during a time window, from llama-swap's own request log (data/activity.sqlite).
# Run on the machine that served the model:
#   ./activity.sh <model> <start-epoch> <end-epoch>                 (zaphod)
#   ssh marvin bash -s -- big <start> <end> < activity.sh           (marvin)
# Prints key=value lines; ./log turns them into live_* metrics on the run. Read-only.
# Caveat: every request to that model in the window counts, including a chat you had at the same time.
MODEL="$1" START="$2" END="$3"
DB_DIR=$(readlink -f "$HOME/llama-swap/data")
[ -f "$DB_DIR/activity.sqlite" ] || { echo "live_error=no activity.sqlite in $DB_DIR"; exit 0; }
docker run --rm -i --network none -v "$DB_DIR:/d" -e MODEL="$MODEL" -e START="$START" -e END="$END" \
  python:3.13-slim-trixie python3 - <<'EOF'
import os, sqlite3, statistics as st
c = sqlite3.connect("file:/d/activity.sqlite?mode=ro", uri=True)
rows = c.execute("""select input_tokens, cache_tokens, output_tokens, draft_tokens, draft_acc_tokens,
                    prompt_per_second, tokens_per_second, duration_ms from activity
                    where model_id = ? and ts_created between ? and ? and resp_status_code = 200""",
                 (os.environ["MODEL"], int(os.environ["START"]), int(os.environ["END"]))).fetchall()
print(f"live_requests={len(rows)}")
if rows:
    inp, cache, out, dr, acc, pps, tps, ms = zip(*rows)
    writing = [t for t, o in zip(tps, out) if t and o >= 50]          # short replies give noisy speeds
    reading = [p for p, i in zip(pps, inp) if p and i >= 500]         # only prompts with real new text to read
    if writing:   # left out when no reply was long enough to time (not the same as 0)
        print(f"live_writing_tps={round(st.median(writing), 1)}")
    if reading:   # left out when no prompt had 500+ new tokens to read
        print(f"live_reading_tps={round(st.median(reading), 1)}")
    # llama-swap stores -1 when there is no draft model: count only real draft numbers
    d = [(x, y) for x, y in zip(dr, acc) if x and x > 0]
    if d:
        print(f"live_draft_acceptance_pct={round(100 * sum(y for _, y in d) / sum(x for x, _ in d), 1)}")
    cache = [max(x or 0, 0) for x in cache]
    print(f"live_cached_pct={round(100 * sum(cache) / (sum(cache) + sum(inp)), 1) if sum(cache) + sum(inp) else 0}")
    print(f"live_input_tokens={sum(inp)}")
    print(f"live_output_tokens={sum(out)}")
    print(f"live_busy_minutes={round(sum(ms) / 60000, 1)}")
EOF

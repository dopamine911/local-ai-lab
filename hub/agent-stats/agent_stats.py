"""Read-only agent statistics for the Homepage start page, one instance per machine (zaphod, marvin).

GET /  ->  {
  "totals":  last 24 h from llama-swap's request log: requests, tokens in/out, average and top writing speed,
             average reading speed (all models on this machine)
  "now":     loaded models, slots in use / total, active requests, pending requests (needs llama-server --metrics;
             null when that page is off), highest context use of any slot
  "models":  the same per loaded model
  "slots":   one row per slot that has work: "<model> #<slot>" and "<used>K / <size>K (<pct>%)"
}
Sources: llama-swap API (/running, /upstream/<model>/slots, /upstream/<model>/metrics) and its activity.sqlite,
opened read-only. Env: LISTEN (host:port), LLAMA_SWAP (base URL), API_KEY (llama-swap key), DB (activity.sqlite path).
"""
import json, os, sqlite3, statistics, time, urllib.request
from http.server import BaseHTTPRequestHandler, HTTPServer

BASE = os.environ.get("LLAMA_SWAP", "http://127.0.0.1:8080")
KEY = os.environ.get("API_KEY", "")
DB = os.environ.get("DB", "/data/activity.sqlite")


def get(path, raw=False):
    req = urllib.request.Request(BASE + path, headers={"Authorization": "Bearer " + KEY})
    with urllib.request.urlopen(req, timeout=5) as r:
        data = r.read()
    return data.decode() if raw else json.loads(data)


def last_day():
    out = {"requests": 0, "tokens_in": 0, "tokens_out": 0, "avg_write_tps": None, "top_write_tps": None,
           "avg_read_tps": None, "per_model": {}}
    try:
        c = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
        rows = c.execute("""select model_id, input_tokens, output_tokens, prompt_per_second, tokens_per_second
                            from activity where ts_created > ? and resp_status_code = 200""",
                         (int(time.time()) - 86400,)).fetchall()
        c.close()
    except sqlite3.Error:
        return out
    rows = [x for x in rows if "/" not in x[0]]   # "marvin/big" etc.: passed on to a peer, counted on that machine
    def summary(rs):
        w = [t for _, _, o, _, t in rs if t and t > 0 and o and o >= 50]    # short replies give noisy speeds
        r = [p for _, i, _, p, _ in rs if p and p > 0 and i and i >= 500]   # only prompts with real reading
        return {"requests": len(rs), "tokens_in": sum(max(x[1] or 0, 0) for x in rs),
                "tokens_out": sum(max(x[2] or 0, 0) for x in rs),
                "avg_write_tps": round(statistics.mean(w), 1) if w else None,
                "top_write_tps": round(max(w), 1) if w else None,
                "avg_read_tps": round(statistics.mean(r), 0) if r else None}
    out.update(summary(rows))
    for m in sorted({x[0] for x in rows}):
        out["per_model"][m] = summary([x for x in rows if x[0] == m])
    return out


def metric(text, name):
    for line in text.splitlines():
        if line.startswith(name + " ") or line.startswith(name + "{"):
            try:
                return float(line.rsplit(" ", 1)[1])
            except ValueError:
                return None
    return None


def now():
    res = {"models_loaded": 0, "slots_busy": 0, "slots_total": 0, "active": 0, "pending": None,
           "max_context_pct": 0, "models": [], "slots": []}
    try:
        running = [m for m in get("/running").get("running", []) if m.get("state") == "ready"]
    except Exception:
        return res
    for m in running:
        mid = m["model"]
        info = {"id": mid, "slots_busy": 0, "slots_total": 0, "pending": None}
        try:
            slots = get(f"/upstream/{mid}/slots")
        except Exception:
            slots = []                                       # speech/embedding servers have no slots
        for s in slots if isinstance(slots, list) else []:
            info["slots_total"] += 1
            busy = bool(s.get("is_processing"))
            info["slots_busy"] += busy
            n_ctx = s.get("n_ctx") or 0
            used = (s.get("n_prompt_tokens") or 0) + ((s.get("next_token") or [{}])[0].get("n_decoded") or 0
                                                       if isinstance(s.get("next_token"), list) else 0)
            if busy and n_ctx:
                pct = round(100 * used / n_ctx)
                res["max_context_pct"] = max(res["max_context_pct"], pct)
                res["slots"].append({"slot": f"{mid} #{s.get('id')}",
                                     "context": f"{used / 1000:.1f}K / {n_ctx // 1000}K ({pct}%)"})
        try:
            mt = get(f"/upstream/{mid}/metrics", raw=True)
            info["pending"] = metric(mt, "llamacpp:requests_deferred")
        except Exception:
            pass
        if info["pending"] is not None:
            res["pending"] = (res["pending"] or 0) + info["pending"]
        res["slots_busy"] += info["slots_busy"]; res["slots_total"] += info["slots_total"]
        res["models"].append(info)
    res["models_loaded"] = len(running)
    res["active"] = res["slots_busy"]
    if not res["slots"]:
        res["slots"].append({"slot": "no slot busy", "context": ""})
    return res


class H(BaseHTTPRequestHandler):
    def do_GET(self):
        n = now()
        body = json.dumps({"totals": last_day(), "now": n, "slots": n.pop("slots")}).encode()
        self.send_response(200); self.send_header("Content-Type", "application/json"); self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a):
        pass


host, port = os.environ.get("LISTEN", "127.0.0.1:9102").rsplit(":", 1)
HTTPServer((host, int(port)), H).serve_forever()

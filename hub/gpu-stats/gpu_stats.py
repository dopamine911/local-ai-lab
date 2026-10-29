"""Tiny read-only GPU stats endpoint for the Homepage start page (AMD cards, from /sys).

GET /  ->  {"cards": [{"name", "busy_pct", "mem_used_gb", "mem_total_gb", "temp_c", "power_w"}, ...]}
Env: LISTEN (host:port), CARD_NAMES ("0000:06:00.0=card 1,0000:0b:00.0=card 2"),
     MEMORY=vram (dedicated cards) or gtt (shared memory, e.g. Strix Halo).
"""
import glob, json, os
from http.server import BaseHTTPRequestHandler, HTTPServer

NAMES = dict(x.split("=", 1) for x in os.environ.get("CARD_NAMES", "").split(",") if "=" in x)
MEM = os.environ.get("MEMORY", "vram")

def read(path, scale=1.0):
    try:
        with open(path) as f:
            return round(int(f.read()) / scale, 1)
    except (OSError, ValueError):
        return None

def cards():
    out = []
    for dev in sorted(glob.glob("/sys/class/drm/card[0-9]*/device")):
        if not os.path.exists(f"{dev}/mem_info_{MEM}_used"):
            continue
        pci = os.path.basename(os.path.realpath(dev))
        hw = (glob.glob(f"{dev}/hwmon/hwmon*") or [""])[0]
        temps = [read(p, 1000) for p in sorted(glob.glob(f"{hw}/temp*_input"))]
        out.append({
            "name": NAMES.get(pci, pci),
            "busy_pct": read(f"{dev}/gpu_busy_percent"),
            "mem_used_gb": read(f"{dev}/mem_info_{MEM}_used", 1024**3),
            "mem_total_gb": read(f"{dev}/mem_info_{MEM}_total", 1024**3),
            "temp_c": max([t for t in temps if t is not None], default=None),   # hottest sensor (junction)
            "power_w": read(f"{hw}/power1_average", 1e6) or read(f"{hw}/power1_input", 1e6),
        })
    return out

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        body = json.dumps({"cards": cards()}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a):
        pass

host, port = os.environ.get("LISTEN", "127.0.0.1:9101").rsplit(":", 1)
HTTPServer((host, int(port)), H).serve_forever()

#!/usr/bin/env python3
# Writing speed of real requests on marvin, from its llama-swap log: grouped by conversation size, before/after a
# time cut (edit `cut`), alone vs the other slot busy, with MTP draft acceptance. Not a controlled test.
# Use: ssh marvin 'cd ~/llama-swap; docker compose logs -t --since 26h' | python3 tuning/marvin-log-speeds.py
import re, sys, statistics as st
from datetime import datetime
cut = "2026-11-04T03:19:20"
T = lambda s: datetime.fromisoformat(s[:26])
launch = re.compile(r"launch_slot_: id\s+(\d) \| task (\d+)")
ev = re.compile(r"id\s+(\d) \| task (\d+) \|\s+eval time =\s*([\d.]+) ms /\s*(\d+) tokens")
acc = re.compile(r"task (\d+) \| draft acceptance = ([\d.]+)")
rel = re.compile(r"release: id\s+(\d) \| task (\d+) \| stop processing: n_tokens = (\d+)")
start, sp, ac, rows = {}, {}, {}, []
lines = [l for l in sys.stdin if len(l.split()) > 2 and l.split()[2].startswith("2026-")]
for l in lines:
    ts = l.split()[2]
    if (m := launch.search(l)): start[m.group(2)] = (T(ts), m.group(1))
    elif (m := ev.search(l)) and "prompt eval" not in l:
        if int(m.group(4)) >= 200: sp[m.group(2)] = (int(m.group(4)) / float(m.group(3)) * 1000, float(m.group(3)))
    elif (m := acc.search(l)): ac[m.group(1)] = float(m.group(2))
    elif (m := rel.search(l)):
        t = m.group(2)
        if t in sp and t in start:
            end = T(ts); wstart = end.timestamp() - sp[t][1] / 1000
            rows.append(dict(t=t, slot=m.group(1), ws=wstart, we=end.timestamp(), v=sp[t][0], acc=ac.get(t), ctx=int(m.group(3)), after=ts >= cut))
# all tasks' busy intervals (launch -> release) per slot, to see if the other slot was busy while this one wrote
iv = {}
for l in lines:
    ts = l.split()[2]
    if (m := launch.search(l)): iv[m.group(2)] = [T(ts).timestamp(), None, m.group(1)]
    elif (m := rel.search(l)) and m.group(2) in iv: iv[m.group(2)][1] = T(ts).timestamp()
ivs = [v for v in iv.values() if v[1]]
for r in rows:
    r["both"] = any(s != r["slot"] and a < r["we"] and b > r["ws"] for a, b, s in ivs)
def med(xs): return f"{st.median(xs):5.1f} (n={len(xs):2d})" if xs else "   -      "
for b, lo, hi in (("30-60K", 30e3, 60e3), ("60-90K", 60e3, 90e3), ("90K+", 90e3, 1e9)):
    for k in (False, True):
        sel = [r for r in rows if r["after"] == k and lo <= r["ctx"] < hi]
        alone = [r["v"] for r in sel if not r["both"]]; both = [r["v"] for r in sel if r["both"]]
        a = [r["acc"] for r in sel if r["acc"] is not None]
        print(f"{b:7s} {'after 256K ' if k else 'before 128K'} | alone {med(alone)} | other slot busy {med(both)} | draft acceptance {st.median(a):.2f}" if a else f"{b} {k} -")

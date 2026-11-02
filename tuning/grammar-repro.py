#!/usr/bin/env python3
# Repro for llama.cpp's "failed to parse grammar" bug (llama.cpp#31342): tools + a required JSON format.
# Usage (key read from the env file, never printed):
#   (set -a; . ~/llama-swap/.env; set +a; python3 tuning/grammar-repro.py coder [variant numbers...])
# Good result on a fixed llama.cpp: every variant OK. On b11870/b12020: 3, 4 and 5 give 400.
import json, os, sys, urllib.request
URL = os.environ.get("URL", "http://127.0.0.1:8080/v1/chat/completions")
KEY = os.environ["ZAPHOD_KEY"]; MODEL = sys.argv[1] if len(sys.argv) > 1 else "coder"
TOOLS = [{"type":"function","function":{"name":"get_weather","description":"Get weather","parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}}]
SCHEMA = {"type":"object","properties":{"location":{"type":"string"},"temperature_celsius":{"type":"number"}},"required":["location","temperature_celsius"],"additionalProperties":False}
JS = {"type":"json_schema","json_schema":{"name":"w","strict":True,"schema":SCHEMA}}
U = [{"role":"user","content":"Weather in Tokyo as JSON."}]
T2 = U + [{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Tokyo\"}"}}]},
          {"role":"tool","tool_call_id":"c1","content":"{\"temp_c\":18,\"condition\":\"cloudy\"}"}]
V = {
 "1 schema, no tools":        dict(messages=U, response_format=JS),
 "2 tools, no schema":        dict(messages=U, tools=TOOLS),
 "3 tools + schema, turn 1":  dict(messages=U, tools=TOOLS, response_format=JS),
 "4 tools + json_object":     dict(messages=U, tools=TOOLS, response_format={"type":"json_object"}),
 "5 tools + schema, turn 2":  dict(messages=T2, tools=TOOLS, response_format=JS),
 "6 schema, turn 2, no tools":dict(messages=T2, response_format=JS),
 "7 tools+schema, tool_choice none": dict(messages=T2, tools=TOOLS, tool_choice="none", response_format=JS),
 "8 tools+schema, tool_choice required": dict(messages=T2, tools=TOOLS, tool_choice="required", response_format=JS),
}
only = sys.argv[2:] 
for name, body in V.items():
    if only and name[0] not in only: continue
    body = dict(model=MODEL, max_tokens=300, temperature=0, **body)
    req = urllib.request.Request(URL, json.dumps(body).encode(), {"Content-Type":"application/json","Authorization":"Bearer "+KEY})
    try:
        r = json.load(urllib.request.urlopen(req, timeout=600)); m = r["choices"][0]["message"]
        out = "tool_calls" if m.get("tool_calls") else repr((m.get("content") or "")[:80])
        print(f"{name:36} OK   {out}")
    except urllib.error.HTTPError as e:
        print(f"{name:36} {e.code}  {e.read().decode()[:150]}")

#!/bin/sh
# Container start: Enso's web app (API :5179, page :5178, both 127.0.0.1 inside the container), plus a forwarder so
# the relay container can reach the page. Enso checks the Host header is loopback, so open it on zaphod as
# http://localhost:5178 (or through `ssh -L 5178:127.0.0.1:5178 zaphod`).
set -eu
socat TCP-LISTEN:15178,fork,reuseaddr TCP:127.0.0.1:5178 &
cd /opt/enso
exec bun run dev

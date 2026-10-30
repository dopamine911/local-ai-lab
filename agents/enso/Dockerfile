# Enso (Daniel's harness around the pi coding agent) in a container, for local models on zaphod.
# Plan: plans/agent-infra.md step 1. How to run: agents/enso/README.md.
#
# Every input is pinned to its latest version: the base image by digest, Ubuntu packages by the archive snapshot of
# one date, bun by sha256, Enso by commit, and pi and every other package by Enso's bun.lock (pi 1.1.0, the version
# Enso supports). The network is used at build time only; the container runs on agents-internal (no internet).
#
#   ./agents/enso/enso.sh build
#
# Ubuntu 26.04, tag resolute-20261024.1
FROM docker.io/library/ubuntu@sha256:6a06111c3b0a8600feadb270aec771dc239bf407b25625f93f44ab00e11f4d3c
ENV DEBIAN_FRONTEND=noninteractive
# The archive as it was on this date. The first install (ca-certificates alone) skips the TLS check because the base
# image has no CA certificates and the snapshot server is HTTPS only; apt still checks every package's signature.
ARG UBUNTU_SNAPSHOT=20261030T000000Z
RUN sed -i -E "s#http://(archive|security).ubuntu.com/ubuntu/#https://snapshot.ubuntu.com/ubuntu/${UBUNTU_SNAPSHOT}/#" \
        /etc/apt/sources.list.d/ubuntu.sources \
    && apt-get -o Acquire::https::Verify-Peer=false update -q \
    && apt-get -o Acquire::https::Verify-Peer=false install -y -q --no-install-recommends ca-certificates \
    && apt-get update -q \
    && apt-get install -y -q --no-install-recommends nodejs git ripgrep unzip socat less jq python3 \
    && rm -rf /var/lib/apt/lists/*

# bun 1.4.4 (latest, 2026-10-22; Enso's packageManager says 1.4.0, engines >=1.4.0), checked against bun's SHASUMS256.txt
ADD --checksum=sha256:bf8ccbfb2b0b80159e0f9a6d15c55317add8077534dbc6130a3edb52fccb395c \
    https://github.com/oven-sh/bun/releases/download/bun-v1.4.4/bun-linux-x64.zip /tmp/bun.zip
RUN unzip -q /tmp/bun.zip -d /tmp && install -m 755 /tmp/bun-linux-x64/bun /usr/local/bin/bun && rm -rf /tmp/bun*

# Enso at a fixed commit (main, 2026-10-15). Its bun.lock pins pi 1.1.0 and the rest.
ARG ENSO_COMMIT=537559ac5e53f0cbc5794349fc61bee6367e1b77
ADD https://github.com/dopamine911/enso-harness.git#${ENSO_COMMIT} /opt/enso
# The package cache stays in the image so `bun install --frozen-lockfile` also works offline in the workspace clone.
ENV BUN_INSTALL_CACHE_DIR=/opt/bun-cache
RUN chown -R ubuntu:ubuntu /opt/enso && mkdir -p /opt/bun-cache && chown ubuntu:ubuntu /opt/bun-cache
USER ubuntu
# No .git in the ADDed tree, so the postinstall's `git config core.hooksPath` fails harmlessly (`|| true`).
RUN cd /opt/enso && bun install --frozen-lockfile

USER root
# `pi` is the copy Enso's lockfile installed (1.1.0), not a newer one from npm.
RUN printf '#!/bin/sh\nexec node /opt/enso/node_modules/@earendil-works/pi-coding-agent/dist/bundle/cli.js "$@"\n' \
        > /usr/local/bin/pi && chmod 755 /usr/local/bin/pi
COPY --chmod=755 enso-entry.sh /usr/local/bin/enso-entry

USER ubuntu
WORKDIR /work
# No install/update telemetry or provider attribution headers (pi settings.md); there is no internet anyway.
ENV PI_TELEMETRY=0 ENSO_HOME=/home/ubuntu/.enso
ENTRYPOINT ["enso-entry"]

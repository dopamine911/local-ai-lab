# llama-swap 269 (2026-11-01; llama.cpp 0ec4027 = b12020), pinned by digest. Was 267 (b11870): issue #1.
FROM ghcr.io/mostlygeek/llama-swap:unified-vulkan-269@sha256:bcb532139b072e4c34ba1dece004db1c5c25cbffd8d099fa26d92caa29edc2a3 AS official

# Ubuntu 26.04, pinned (was the floating ubuntu:26.04). Docker allows no comment after an instruction (fixed 2026-11-02).
FROM ubuntu:resolute-20261024.1@sha256:6a06111c3b0a8600feadb270aec771dc239bf407b25625f93f44ab00e11f4d3c
# Mesa (the Vulkan driver) pinned: a rebuild must not change the driver along with llama.cpp (one change at a time).
# 2026-11-09: 26.0.8-1ubuntu0.3 (Ubuntu's newest) → 26.2.6 from the kisak PPA, measured on gpt-oss
# (plans/gpt-oss-timeline.md). Key: kisak-mesa.asc, fingerprint EB8B81E14DA65431D7504EA8F63F0F2B90935439 (checked
# against Launchpad). Plain http is fine: apt checks the signature. PPAs drop old versions: if this pin stops
# resolving, use the saved image (marvin ~/backups) or go back to 26.0.8 (remove the PPA lines).
COPY kisak-mesa.asc /etc/apt/keyrings/kisak-mesa.asc
RUN printf 'Types: deb\nURIs: http://ppa.launchpadcontent.net/kisak/kisak-mesa/ubuntu\nSuites: resolute\nComponents: main\nSigned-By: /etc/apt/keyrings/kisak-mesa.asc\n' \
      > /etc/apt/sources.list.d/kisak-mesa.sources
RUN apt-get update && apt-get install -y --no-install-recommends \
      libvulkan1 mesa-vulkan-drivers=26.2.6~kisak1~r libgomp1 openssl ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
RUN apt-get update && (apt-get install -y --no-install-recommends rocm-smi || true) \
    && rm -rf /var/lib/apt/lists/*
COPY --from=official /usr/local/bin/llama-swap /usr/local/bin/llama-server \
     /usr/local/bin/llama-bench /usr/local/bin/run.sh /usr/local/bin/
RUN ! ldd /usr/local/bin/llama-server /usr/local/bin/llama-bench | grep 'not found'
WORKDIR /models
ENTRYPOINT ["run.sh"]
CMD []

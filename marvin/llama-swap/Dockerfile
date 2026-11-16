# llama-swap 273 (2026-11-15; llama.cpp e574a17 = v1.0.0 + 48), pinned by digest. Was 269 (llama.cpp 0ec4027 = b12020):
# plans/llama-cpp-v1-timeline.md. Rollback: image llama-swap-marvin:269 (269 digest sha256:bcb53213...a2edc2a3).
FROM ghcr.io/mostlygeek/llama-swap:unified-vulkan-273@sha256:c5f59a9bf87eebd716ca61c7dfe8bb9f1466dd356d4b2ef8b85d1e6cc9e1824f AS official

# Ubuntu 26.04, pinned (was the floating ubuntu:26.04). Docker allows no comment after an instruction (fixed 2026-11-02).
FROM ubuntu:resolute-20261024.1@sha256:6a06111c3b0a8600feadb270aec771dc239bf407b25625f93f44ab00e11f4d3c
# Mesa (the Vulkan driver) pinned: a rebuild must not change the driver along with llama.cpp (one change at a time).
# Same version as in llama-swap-marvin:267 (checked 2026-11-02). Bump it on purpose, measured.
# 2026-11-09: 26.2.6 from the kisak PPA tried, +0.8% writing / +1.3% reading on gpt-oss (not clear of heat): rolled back (marvin/SETUP.md rows 24-25).
RUN apt-get update && apt-get install -y --no-install-recommends \
      libvulkan1 mesa-vulkan-drivers=26.0.8-1ubuntu0.3 libgomp1 openssl ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
RUN apt-get update && (apt-get install -y --no-install-recommends rocm-smi || true) \
    && rm -rf /var/lib/apt/lists/*
COPY --from=official /usr/local/bin/llama-swap /usr/local/bin/llama-server \
     /usr/local/bin/llama-bench /usr/local/bin/run.sh /usr/local/bin/
RUN ! ldd /usr/local/bin/llama-server /usr/local/bin/llama-bench | grep 'not found'
WORKDIR /models
ENTRYPOINT ["run.sh"]
CMD []

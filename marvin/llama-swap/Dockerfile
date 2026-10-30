FROM ghcr.io/mostlygeek/llama-swap:unified-vulkan-267 AS official

FROM ubuntu:resolute-20261024.1@sha256:6a06111c3b0a8600feadb270aec771dc239bf407b25625f93f44ab00e11f4d3c   # Ubuntu 26.04, pinned (was the floating ubuntu:26.04)
RUN apt-get update && apt-get install -y --no-install-recommends \
      libvulkan1 mesa-vulkan-drivers libgomp1 openssl ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
RUN apt-get update && (apt-get install -y --no-install-recommends rocm-smi || true) \
    && rm -rf /var/lib/apt/lists/*
COPY --from=official /usr/local/bin/llama-swap /usr/local/bin/llama-server \
     /usr/local/bin/llama-bench /usr/local/bin/run.sh /usr/local/bin/
RUN ! ldd /usr/local/bin/llama-server /usr/local/bin/llama-bench | grep 'not found'
WORKDIR /models
ENTRYPOINT ["run.sh"]
CMD []

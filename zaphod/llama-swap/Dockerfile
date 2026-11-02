# llama-swap 269 (2026-11-01; llama.cpp 0ec4027, 2026-10-31), pinned by digest. Was 267 (llama.cpp 74bca39): issue #1.
FROM ghcr.io/mostlygeek/llama-swap:unified-vulkan-269@sha256:bcb532139b072e4c34ba1dece004db1c5c25cbffd8d099fa26d92caa29edc2a3 AS official

# Ubuntu 26.04, pinned (was the floating ubuntu:26.04). Docker allows no comment after an instruction (fixed 2026-11-02).
FROM ubuntu:resolute-20261024.1@sha256:6a06111c3b0a8600feadb270aec771dc239bf407b25625f93f44ab00e11f4d3c
# Mesa (the Vulkan driver) pinned: a rebuild must not change the driver along with llama.cpp (one change at a time).
# Same version as in :267-media (checked 2026-11-02). Bump it on purpose, measured.
RUN apt-get update && apt-get install -y --no-install-recommends \
      libvulkan1 mesa-vulkan-drivers=26.0.8-1ubuntu0.3 libgomp1 openssl ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
RUN apt-get update && (apt-get install -y --no-install-recommends rocm-smi || true) \
    && rm -rf /var/lib/apt/lists/*
COPY --from=official /usr/local/bin/llama-swap /usr/local/bin/llama-server \
     /usr/local/bin/llama-bench /usr/local/bin/run.sh /usr/local/bin/
# Speech and images (image tag 267-media): whisper-server (speech-to-text), sd-server (images),
# audiocpp_server and llama-tts (text-to-speech). whisper-server needs the shared ggml/whisper
# libraries; llama-server has ggml built in, so these libraries don't affect it.
RUN apt-get update && apt-get install -y --no-install-recommends libmp3lame0 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=official /usr/local/bin/whisper-server /usr/local/bin/sd-server \
     /usr/local/bin/audiocpp_server /usr/local/bin/llama-tts /usr/local/bin/
COPY --from=official /usr/local/bin/espeak-ng-data /usr/local/bin/espeak-ng-data
COPY --from=official /usr/local/lib/libggml*.so* /usr/local/lib/libwhisper.so* /usr/local/lib/
RUN ldconfig && ! ldd /usr/local/bin/llama-server /usr/local/bin/llama-bench \
      /usr/local/bin/whisper-server /usr/local/bin/sd-server /usr/local/bin/audiocpp_server \
      /usr/local/bin/llama-tts | grep 'not found'
WORKDIR /models
ENTRYPOINT ["run.sh"]
CMD []

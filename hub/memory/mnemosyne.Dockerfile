# Mnemosyne 4.0.0 (mnemosyne-oss/mnemosyne, MIT): short memories in one SQLite file, MCP over streamable HTTP with a
# bearer token.   docker compose --profile images build mnemosyne     (from hub/)
# Pinned: Python base by digest (3.12.16-slim), mnemosyne-memory 4.0.0 (latest, 2026-10-27).
# Embeddings: its default local model (FastEmbed BAAI/bge-small-en-v1.5, 384 dims, CPU), downloaded HERE at build
# time; at run time nothing is downloaded (HF_HUB_OFFLINE) and there is no internet. Not llama-swap: 4.0 refuses to
# send an API key over plain http, and the local model keeps memory working while llama-swap swaps models.
FROM docker.io/library/python:3.12.16-slim@sha256:8523d6abeb48ceae0b7240d30c5808fc26cb79c1a635bacdf19cf3baecfbec07
ENV MNEMOSYNE_FASTEMBED_CACHE_DIR=/opt/fastembed
RUN pip install --no-cache-dir "mnemosyne-memory[mcp,embeddings]==4.0.0" \
    && python -c "from fastembed import TextEmbedding; TextEmbedding('BAAI/bge-small-en-v1.5', cache_dir='/opt/fastembed')" \
    && chmod -R a+rX /opt/fastembed
USER 1000:1000
ENV HOME=/data MNEMOSYNE_DATA_DIR=/data HF_HUB_OFFLINE=1
ENTRYPOINT ["mnemosyne", "mcp", "--transport", "streamable-http", "--host", "0.0.0.0", "--port", "8080", "--path", "/mcp"]

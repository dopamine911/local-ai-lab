# LocalRecall's MCP server (mudler/mcps, stdio only) behind mcp-proxy, so MCPJungle can reach it over streamable HTTP.
#   docker compose --profile images build mcp-localrecall     (from hub/)
# Pinned: the MCP server image by digest (v0.0.4, built 2026-02-20; a static binary), the Python base by digest
# (3.12-slim, 2026-08-13), mcp-proxy 0.13.0 and every dependency as published on PyPI by 2026-10-16 (uv --exclude-newer).
FROM ghcr.io/mudler/mcps/localrecall@sha256:53e635373eef4ec727a2fb65d25cc8c58e92015461e227e19763c74c4c3e415f AS lrmcp
FROM docker.io/library/python@sha256:2c941e860699f878900b0edc2403613c234d4b32eda3cc9fa7036991a2a63c4a
RUN pip install --no-cache-dir uv==0.13.0 \
    && uv pip install --system --no-cache --exclude-newer 2026-10-16T00:00:00Z mcp-proxy==0.13.0 \
    && pip uninstall -y uv
COPY --from=lrmcp /app/main /usr/local/bin/localrecall-mcp
USER 65534:65534
# --pass-environment: the LOCALRECALL_* variables reach the stdio server
ENTRYPOINT ["mcp-proxy", "--host", "0.0.0.0", "--port", "8080", "--pass-environment", "--", "/usr/local/bin/localrecall-mcp"]

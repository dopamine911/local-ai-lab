# LocalRecall's MCP server (mudler/mcps, stdio only) behind mcp-proxy, so MCPJungle can reach it over streamable HTTP.
#   docker compose --profile images build mcp-localrecall     (from hub/)
# Pinned: the MCP server image by digest (v0.0.4, built 2026-02-20; a static binary), the Python base by digest
# (3.12.16-slim), mcp-proxy 0.13.0 (latest).
FROM ghcr.io/mudler/mcps/localrecall@sha256:53e635373eef4ec727a2fb65d25cc8c58e92015461e227e19763c74c4c3e415f AS lrmcp
FROM docker.io/library/python:3.12.16-slim@sha256:8523d6abeb48ceae0b7240d30c5808fc26cb79c1a635bacdf19cf3baecfbec07
RUN pip install --no-cache-dir mcp-proxy==0.13.0
COPY --from=lrmcp /app/main /usr/local/bin/localrecall-mcp
USER 65534:65534
# --pass-environment: the LOCALRECALL_* variables reach the stdio server
ENTRYPOINT ["mcp-proxy", "--host", "0.0.0.0", "--port", "8080", "--pass-environment", "--", "/usr/local/bin/localrecall-mcp"]

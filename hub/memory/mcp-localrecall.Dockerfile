# LocalRecall's MCP server (mudler/mcps, stdio only) behind mcp-proxy, so MCPJungle can reach it over streamable HTTP.
#   docker compose --profile images build mcp-localrecall     (from hub/)
# Pinned: the MCP server image by digest (v0.0.4, built 2026-02-20; a static binary), the Python base by digest
# (3.12.16-slim), mcp-proxy (the pip pin below; Renovate moves it: then update the image tag in hub/compose.yaml).
FROM ghcr.io/mudler/mcps/localrecall:v0.0.4@sha256:53e635373eef4ec727a2fb65d25cc8c58e92015461e227e19763c74c4c3e415f AS lrmcp
FROM docker.io/library/python:3.12.16-slim@sha256:b18b261e1b4974b4f321451ad5251ded85abb22c2b8b262da527dda1d031ab30
RUN pip install --no-cache-dir mcp-proxy==0.14.0
COPY --from=lrmcp /app/main /usr/local/bin/localrecall-mcp
USER 65534:65534
# --pass-environment: the LOCALRECALL_* variables reach the stdio server
# mcp-proxy 0.13.3 checks the Host header (DNS-rebinding protection): allow only the name MCPJungle uses on the
# Docker network (found 2026-11-02 after the Renovate bump: "Invalid Host header: mcp-localrecall:8080").
ENTRYPOINT ["mcp-proxy", "--host", "0.0.0.0", "--port", "8080", "--allowed-host", "mcp-localrecall:8080", "--pass-environment", "--", "/usr/local/bin/localrecall-mcp"]

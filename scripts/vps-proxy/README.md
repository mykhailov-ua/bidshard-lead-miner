# VPS Squid proxy (optional)

Datacenter HTTP proxy for parser crawl tests. Does not replace residential proxy for Cloudflare igaming.

| Script | Purpose |
| --- | --- |
| `install-on-vps.sh` | Install Squid on Ubuntu VPS |
| `setup-docker-proxy.sh` | Local Docker Squid (`make vps-proxy-docker`) |
| `check-proxy.sh` | Curl smoke test |
| `print-env-snippet.sh` | Print `PARSER_PROXY_LIST=...` line |

Docs: [docs/OPS.md#vps-squid-optional](../../docs/OPS.md#vps-squid-optional), [docs/DEPLOY_VPS.md#appendix-a-vps-squid-datacenter-egress-only](../../docs/DEPLOY_VPS.md#appendix-a-vps-squid-datacenter-egress-only).

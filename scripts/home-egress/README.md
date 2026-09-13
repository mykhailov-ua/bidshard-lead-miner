# Home ISP egress bridge

Route **parser HTTP crawl** (forum, tgweb, lander, SERP) through your **home public IP** when paid residential proxies fail.

Parser on VPS uses `network_mode: host`, so a reverse SSH tunnel to `127.0.0.1` on the VPS is enough.

Telegram MTProto stays separate (`TELEGRAM_PROXY_URL`); this is only `PARSER_PROXY_LIST`.

## Topology

```
VPS parser  --HTTP--> 127.0.0.1:19888 (tunnel)
                           |
                      SSH -R
                           |
Home PC     Squid :3128 --> your ISP IP --> target sites
```

## On home machine

```bash
cp scripts/home-egress/env.example scripts/home-egress/.env.local
# edit PROXY_PASS, VPS_SSH_* if needed

make home-egress-proxy-up    # Squid on 127.0.0.1:3128
make home-egress-tunnel      # keep running (autossh if installed)
```

Copy `scripts/home-egress/.credentials` to VPS `.env` (or run `make vps-apply-home-proxy`).

## On VPS

```bash
# In /opt/lead-intent-processor/.env - replace commercial residential list:
# PARSER_PROXY_LIST=http://parser:YOUR_PASS@127.0.0.1:19888

docker compose restart parser
make vps-check-home-tunnel   # from dev laptop, optional
```

## Requirements

- Home PC always on while crawling, or run tunnel on a home router/NAS.
- SSH from home to VPS (same key as deploy).
- VPS `sshd` allows `AllowTcpForwarding yes` (default).
- Home router: no inbound port forward required (outbound SSH only).

## Security

- Tunnel binds **127.0.0.1** on VPS only (not public).
- Squid listens **127.0.0.1** on home only.
- Use strong `PROXY_PASS`; do not commit `.credentials` or `.env.local`.

See [docs/OPS.md](../../docs/OPS.md#home-isp-egress-bridge).

Optional VPS **systemd failover**: probes residential 403 rate and switches parser to this tunnel when the pool is bad (`make vps-install-proxy-failover`). Home tunnel must stay up while mode=home.

# BidShard ICP (lead-intent-processor)

Who we target for [BidShard](https://bidshard.com/) outbound. Parser scoring and source mix follow this doc.

## Yes (primary ICP)

- In-house **media buying teams** running paid traffic (Meta, Google, native, push).
- **Self-hosted stack** on own VPS: Keitaro, Binom, Voluum, or evaluating migration.
- Pain: junk traffic bleed, postback mismatch, tracker + cloaker + IVT stack cost, margin pressure.
- Team size: solo power buyer with DevOps access, or 3+ buyers with roles/permissions.
- Geo: outside RU/BY (parser hard-rejects RU/BY signals).

### Crypto-gray buyer (secondary ICP, high LTV)

Operators who **settle in USDT** and optimize for **clean click / FTD quality** on gray verticals (igaming, nutra, CPI).

**Signals (need 2+ tiers, not one keyword):**

| Tier | Examples |
|------|----------|
| Payout | TRC20, ERC20, USDT, crypto payout/settlement, weekly/daily USDT, no KYC, Capitalist, PST |
| Stack | Keitaro, Binom, Zeustrack, HideClick, Cloaking.House, FraudFilter, JS fingerprint, MaxMind, IPQS |
| Pain | shaving, scrubbing, bot click fraud, fake leads, auto-fill bots, trash deposits, CR drop, balance frozen, incentivized/fraudulent traffic |

**Sources:** webmaster supergroups of crypto CPA networks (not AM support), private farm/agency-cab chats where buyers fund via crypto, tracker/cloak operator groups.

**Not ICP:** account/card **sellers**, pump channels, CPA network AM outreach (see instant_drop / supply filters).

Implementation: [MILESTONE.md](MILESTONE.md) **M11**.

## No (not sales target now)

| Segment | Why skip |
|---------|----------|
| CPA networks (MaxBounty, Dr.Cash, etc.) as **B2B buyer** | Vendor lock-in, legacy ops, different product (offer marketplace not tracker license) |
| SSP / publisher adops (ads.txt contacts on taboola.com-scale domains) | Adops, not media buyer with Keitaro |
| Job candidates from Djinni/Hirify | 0-1 good per 10-20; use jobs only as **company OSINT** (employer name) |
| Course sellers, account farms, design shops | `instant_drop` filter |
| Programmatic / SSP / brand AdTech | Anti-ICP keywords |

## Affiliate inside a CPA network

The **individual affiliate** running Keitaro on their VPS is ICP. The **network entity** is not.

## Source priority (VPS profile)

| Priority | Sources |
|----------|---------|
| Hot | `forum`, `telegram` (realtime), `github`, `reddit` pain queries |
| Warm | `serp`, `webpain`, `reviews`, `tgweb` |
| Intel only | `supply` (ads.txt) - off 24/7 poll in `config/env/.env.bidshard-icp.example` |

## Qualification signals

**Boost:** postback, keitaro/binom/voluum, self-hosted, nginx/htop logs, CAPI, migration, parallel pilot language.

**Drop:** seller spam, agency outreach, job/tutorial noise, seller author handles (shop/store/rent).

**LLM defer (not reject):** ban-context (FB ban, pixel, conversion attribution disputes).

## Env profile

```bash
cat config/env/.env.bidshard-icp.example >> .env
```

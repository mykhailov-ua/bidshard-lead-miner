# HEADLESS - browser environment reference

Dry comparison of what automated and headless Chromium-family sessions often expose versus what a typical interactive user session presents. Not a product spec; observations used in industry probes and in BidShard tracker client sensors (`ad-event-processor/internal/track/antifraud_telemetry.js`, fingerprint hydrators). Overlap is not universal: patched automation, real GPU in CI, or mobile WebViews sit between the columns.

---

## 1. Terms

| Term | Meaning here |
|------|----------------|
| **Headless browser** | Chromium (or other engine) without a user-visible window; often launched with `--headless` or equivalent API flag. |
| **Automated session** | Page controlled by WebDriver, CDP, Playwright, Puppeteer, Selenium, or in-page scripts driving navigation without physical input. |
| **Legitimate interactive client** | Normal browser or in-app WebView where the user (or OS IME) generates pointer, touch, keyboard, and scroll input; window manager owns outer dimensions. |

Headless and automated are related but not identical: headless can be driven manually via CDP; headed Chrome can be fully automated.

---

## 2. Automation API and globals

| Signal | Often in automated / default headless tooling | Typical legitimate interactive client |
|--------|-----------------------------------------------|-------------------------------------|
| `navigator.webdriver` | `true` in WebDriver-controlled Chrome; may be patched to `false` in stealth setups | `false` in stock Chrome, Firefox, Safari |
| `Navigator.prototype.webdriver` getter | Present; stealth may replace getter (descriptor / `this` binding anomalies) | Standard built-in getter |
| ChromeDriver DOM hook | `document.$cdc_*` properties on `document` | Absent |
| Playwright | `window.__playwright`, `__pwInitScripts` | Absent |
| Puppeteer / CDP injection | Stack or globals referencing puppeteer, cdp, evaluation frames | Absent |
| Selenium | `__selenium_unwrapped`, `__webdriver_evaluate`, `__driver_evaluate` | Absent |
| PhantomJS (legacy) | `_phantom`, `callPhantom` | Absent |
| `Function.prototype.toString` on builtins | Patched to hide hooks; may not contain `[native code]` | Native functions report `[native code]` |
| `navigator.permissions.query` | Sometimes proxied | Unmodified native |

Invoking certain getters with wrong `this` (e.g. `Navigator.prototype.webdriver.get.call({})`) can throw `TypeError` in an unmodified engine; patched getters may throw other errors or succeed. Error stacks from probe calls may contain framework names when automation wraps native code.

---

## 3. Window and screen geometry

| Signal | Often in headless / automation | Typical legitimate desktop | Typical legitimate mobile |
|--------|-------------------------------|----------------------------|---------------------------|
| `window.outerWidth` / `outerHeight` | `0` or equal to inner (no chrome UI) | Outer > inner (toolbars, borders) | Differs from inner; safe areas |
| `window.innerWidth` vs `outerWidth` | Frequently equal on desktop headless | Inner smaller than outer | N/A same rule |
| `screen.width` / `screen.height` | Sometimes `0` or placeholder | Positive, matches display | Device screen dimensions |
| Headed automation | May set viewport explicitly; outer may still look synthetic | OS-decorated window | Full-screen or browser chrome |

Mobile legitimate clients are usually excluded from desktop-only viewport rules (no expectation of outer chrome in the same way).

---

## 4. Graphics stack (WebGL)

| Signal | Often in headless / software rendering | Typical legitimate client |
|--------|----------------------------------------|---------------------------|
| `WEBGL_renderer` | `SwiftShader`, `LLVMpipe`, `Mesa Offscreen`, ANGLE on software path | GPU vendor string (Intel, NVIDIA, Apple, Adreno, etc.) |
| Vendor vs UA | Chrome UA with Mozilla-only WebGL vendor (or reverse) can occur when UA and GL are inconsistent | Vendor/renderer broadly aligned with browser engine and OS |
| Shader compile timing | Very fast software path on some hosts | Hardware-dependent, usually higher on real GPUs |
| Canvas / WebGL / Audio hashes | Stable per environment; headless clusters share hashes | Stable per device profile; wider distribution |

---

## 5. Navigator, locale, and plugins

| Signal | Often in minimal automation context | Typical legitimate client |
|--------|-------------------------------------|---------------------------|
| `navigator.userAgent` | Set explicitly or default headless UA | User or enterprise policy; matches install |
| `navigator.language` vs `navigator.languages` | Mismatch if only UA or lang overridden | Primary language appears in `languages` list |
| `navigator.plugins.length` | Often `0` in headless Chromium | Non-zero in desktop Chrome (PDF viewer, etc.); mobile varies |
| `hardwareConcurrency` | Default VM count | Device CPU count |
| Timezone (`Intl`, `Date`) | May disagree with IP geolocation if VM defaults to UTC | Usually aligned with user locale or travel (not strict) |

---

## 6. Input events and motion

| Signal | Often in scripted automation | Typical legitimate interactive client |
|--------|------------------------------|---------------------------------------|
| `Event.isTrusted` | `false` for `element.click()`, CDP `Input.dispatch*`, synthetic DOM events | `true` for OS-dispatched pointer, touch, keyboard |
| Pointer path | Straight lines, constant speed, collinear samples | Curved paths, variable speed, subpixel coordinates |
| Scroll | Programmatic `scrollTo` / single-step jumps | Wheel/touch inertia; jerk and velocity vary |
| Touch (mobile) | Missing force/radius; absent gyro if not simulated | `force`, `radiusX/Y` on capable devices; motion sensors when permitted |
| Event rate | Metronomic intervals (low coefficient of variation) | Higher variance in inter-event timing |
| Dwell vs depth | Short time to full page depth (footer) with little pointer activity | Longer exploration; pointer and scroll activity correlate |

CDP can synthesize events that appear trusted in some builds; probes that combine low `isTrusted` ratio with low kinematic variance target that gap.

---

## 7. Rendering loop and timing

| Signal | Often in headless / throttled automation | Typical legitimate client |
|--------|----------------------------------------|---------------------------|
| `requestAnimationFrame` delta CV | Can be very stable or anomalously jittery under virtualization | Moderate variance; tied to display refresh |
| Page lifetime | Navigation + scrape: short dwell, deterministic order | Variable dwell; background tabs pause RAF |
| RTT measurements from page | May be missing if no secondary fetch or blocked network | Periodic fetches; RTT samples present when script runs probes |

---

## 8. Network-facing observations (environment)

These are properties of the host and path, not of headless alone:

| Topic | Note |
|-------|------|
| WebRTC local candidate | Desktop browsers often expose a private LAN IP; missing or public IP mismatch is common in misconfigured or blocked WebRTC setups |
| TLS fingerprint (JA3/JA4) | Headless Chrome often matches Chrome family; differs from Safari/Firefox |
| IP anonymity flags | Datacenter, VPN, or proxy egress independent of headless flag |

---

## 9. Common stacks (factual defaults)

| Stack | Default headless | Typical automation surface |
|-------|------------------|----------------------------|
| Chromium `--headless=new` | No window; WebGL may be SwiftShader unless GPU passed | CDP; optional `webdriver` |
| Playwright `chromium.launch(headless=True)` | Same family; default context without extra stealth | CDP under the hood; globals unless disabled |
| Selenium + ChromeDriver | WebDriver mode; CDC artifacts historically | `navigator.webdriver === true` unless `--disable-blink-features=AutomationControlled` and patches |
| Real user Chrome/Firefox/Safari | Headed | No WebDriver flag; physical input |

Stealth plugins patch subsets of section 2 and 3; sections 6-7 still differ when input is synthesized.

---

## 10. Headless in this repository

Playwright backs **tgweb / lander** contact discovery when HTTP+RSC is not enough:

| Piece | Role |
|-------|------|
| `sources/headless/profile.py` | Desktop Chrome profile (UA, `Sec-Ch-Ua*`, viewport) aligned with `internal/httpclient/client.go` |
| `sources/headless/interact.py` | Post-load wheel + pointer motion and dwell (section 6) |
| `sources/headless/fetch.py` | Launch, context, `goto`, interact, `content()` |
| `internal/sources/lander/headless.go` | Go pool; one Python subprocess per fetch |
| `cmd/parser/headless.go` | Nightly defer-queue drain |

Default launch: Chromium `--headless=new`, `--disable-blink-features=AutomationControlled`, `ignore_default_args` drops `--enable-automation`. Optional `PARSER_HEADLESS_HEADED=true` for a visible window (use `xvfb-run` on Linux servers).

---

## 11. Source alignment

Detailed bit names for automation globals and runtime tampering checks are defined in `ad-event-processor/pkg/antifraudtelemetry/score.go` and collected in `internal/track/antifraud_telemetry.js`. Sections 2-7 are the checklist we close for **lead fetch** (not tracker scoring).

---

## 12. Interactive client profile (lead-intent-processor)

Goal: behave like a **normal desktop Chrome user** opening an affiliate/lander URL from search or outreach, so antifraud probes in sections 2-7 see fewer automation-only gaps.

### What we align

| HEADLESS section | Implementation |
|------------------|----------------|
| 2 Automation API | No `--enable-automation`; `AutomationControlled` disabled; no custom `__playwright` hiding scripts |
| 3 Geometry | 1920x1080 viewport + screen (override via env) |
| 5 Navigator / locale | `user_agent`, `locale`, `timezone_id`, `Accept-Language` consistent with HTTP crawl |
| 6 Input | `PARSER_HEADLESS_INTERACTIVE=true` (default): `mouse.wheel`, curved `mouse.move`, variable `wait_for_timeout` |
| 7 Timing | Initial dwell + optional `networkidle` after motion; not instant scrape-and-exit |

We do **not** guarantee GPU WebGL strings (section 4) in Docker; use host Chrome via `PARSER_HEADLESS_CHANNEL=chrome` when a site blocks software GL.

### Scope and limits

- URLs: **tgweb / lander** registries, seeds, `headless_queue.json` only (same as HTTP crawl admission).
- **No** LinkedIn or login-gated social (processor rejects LinkedIn leads).
- **No** CAPTCHA solving or credential stuffing.
- Egress: `PARSER_PROXY_LIST` (first URL), same as HTTP; align timezone/locale with proxy geo when sites geo-gate.

### Env (Playwright sidecar)

| Variable | Default | Meaning |
|----------|---------|---------|
| `PARSER_HEADLESS_INTERACTIVE` | `true` | Wheel + pointer + dwell after `goto` |
| `PARSER_HEADLESS_HEADED` | `false` | Headed Chromium (needs display or xvfb) |
| `PARSER_HEADLESS_LOCALE` | `en-US` | `navigator.languages` |
| `PARSER_HEADLESS_TIMEZONE` | `UTC` | `Intl` / `Date` timezone |
| `PARSER_HEADLESS_USER_AGENT` | Chrome 122 (httpclient) | Override only when bumping Chrome version |
| `PARSER_HEADLESS_VIEWPORT_WIDTH/HEIGHT` | 1920 / 1080 | Desktop window |
| `PARSER_HEADLESS_TIMEOUT_MS` | 30000 | Navigation timeout |
| `PARSER_HEADLESS_CHANNEL` | (bundled Chromium) | `chrome` for system Google Chrome |
| `PARSER_HEADLESS_SEED` | (random) | Fixed RNG for reproducing motion in debug |

Parser toggles unchanged: `PARSER_LANDER_HEADLESS`, `PARSER_LANDER_HEADLESS_DEFER`, drain cron, `PARSER_LANDER_HEADLESS_MAX_BROWSERS`.

### Ops checklist

1. Prefer `PARSER_LANDER_HEADLESS_DEFER=true` + `scripts/ops/headless-crawl-cron.sh` on VPS; set `PARSER_HEADLESS_TIMEZONE` to match residential proxy region.
2. For stubborn landers: `PARSER_HEADLESS_HEADED=true` inside `xvfb-run` or `PARSER_HEADLESS_CHANNEL=chrome` on a workstation.
3. Image: `Dockerfile.playwright` / `docker-compose.headless.yaml`.
4. Queue depth: `parser auto status`, `parser_headless_*` metrics.

See: [docs/OPS.md](docs/OPS.md), [docs/DEPLOY.md](docs/DEPLOY.md), [docs/CRAWL_EGRESS_ANTIFRAUD.md](docs/CRAWL_EGRESS_ANTIFRAUD.md) (residential proxy, Cloudflare risk, minimization ceiling).

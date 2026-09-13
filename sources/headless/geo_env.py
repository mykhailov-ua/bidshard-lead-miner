"""Print shell exports for locale/timezone inferred from proxy (cron / ops)."""

from __future__ import annotations

import os
import sys

from sources.headless.geo_from_proxy import active_proxy_url, infer_locale_timezone


def main() -> int:
    if os.environ.get("PARSER_HEADLESS_LOCALE", "").strip():
        locale_set = True
    else:
        locale_set = False
    if os.environ.get("PARSER_HEADLESS_TIMEZONE", "").strip():
        tz_set = True
    else:
        tz_set = False
    if locale_set and tz_set:
        return 0

    url = active_proxy_url()
    if not url:
        return 0
    inferred = infer_locale_timezone(url)
    if not inferred:
        return 0
    locale, tz = inferred
    if not locale_set:
        print(f'export PARSER_HEADLESS_LOCALE="{locale}"')
    if not tz_set:
        print(f'export PARSER_HEADLESS_TIMEZONE="{tz}"')
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

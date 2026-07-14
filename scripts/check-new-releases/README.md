# check-new-releases

Two complementary tools for finding Windows product edition IDs that exist on
Microsoft's side but aren't yet in `cli/catalog.go` / `products.json` /
`validContributeProducts`. Neither touches those files automatically --
naming a new entry correctly (build number, refresh qualifiers like "V2",
etc.) still needs a human checking Microsoft's own release-health pages.
Both only *discover* candidates.

Only Windows 11 is worth checking at all: Windows 8.1 is fully frozen (no
updates since Jan 2023) and Windows 10 is past end-of-life (ESU security
patches only, delivered via Windows Update, not new consumer ISOs). Neither
will ever produce a new release again.

## Method 1 -- `msdls_v3.py` (brute-force range scan)

Checks every ID in a range you specify, one by one, against Microsoft's
SKU-info API. Use this when you already have a rough idea of where to look
(e.g. "the last release was around 3260, check up to 3350") or want to
directly merge results into a JSON catalog file.

```bash
pip install requests
python msdls_v3.py --first 3320 --last 3330
```

Optionally write results to a JSON file (merges with an existing one if
present, preserving fields like `badge`/`archs`/`related`, marking IDs that
stopped responding as `"active": false`):

```bash
python msdls_v3.py --first 3320 --last 3330 --write scan-results.json
```

Don't point `--write` directly at `frontend/public/data/products.json` --
write to a scratch file first and review before merging into the real
catalog.

**How it identifies a release:** reads `ProductDisplayName` from the first
SKU in the API response (e.g. `"Windows 11 25H2__V2"`). This field name was
wrong in an earlier version of this script (checked `EditionName`/
`ReleaseName`/`FriendlyName`, none of which exist in the real response) --
confirmed live and fixed 2026-07-14.

**Session handling:** registers the session via `vlscppe`'s permit endpoint
before scanning. A single SKU lookup was confirmed to work without this, but
this script's use case -- many rapid consecutive requests across a range --
is a much higher-risk pattern for triggering a Sentinel block than a
one-shot lookup, so the permit call is worth keeping. The CLI/backend's full
session flow additionally replays an `ov-df.microsoft.com` fingerprint; add
that too if the permit call alone isn't enough under real scanning load.

## Method 2 -- `main.go` (auto-discovery)

Scrapes Microsoft's public `/windows11` download page for the *current*
flagship product edition ID (no range-guessing needed), compares it against
`cli/catalog.go`, and if it's new, probes a bounded range of adjacent IDs to
discover the accompanying variant family (Home China, Pro China, ARM64).
Use this when you just want to know "is there anything new" without knowing
where to look.

```bash
cd scripts/check-new-releases
go run .
```

If a new release is found, it makes real requests to Microsoft to probe the
variant family -- let it finish rather than re-running immediately.

Flags:
- `-catalog <path>` — path to `cli/catalog.go` (default: `../../cli/catalog.go`)
- `-probe-range <n>` — how many adjacent IDs to check after a new flagship is found (default: 20)

**Known limitation:** the ARM64 family's offset from the flagship isn't
consistent release to release (observed: +3 for the 25H2 family, +18 for
24H2's). The default probe range of 20 covers both known cases, but if no
ARM64 variant shows up, try a wider `-probe-range`.

## Which one to use

| | Method 1 (Python) | Method 2 (Go) |
|---|---|---|
| Need to guess a starting range? | Yes | No -- finds the current flagship automatically |
| Gets the real Microsoft name? | Yes (`ProductDisplayName`) | Yes (`ProductDisplayName`) |
| Can write results to a catalog-shaped JSON? | Yes (`--write`) | No -- prints a report only |
| Best for | "I want to scan a specific range and get a mergeable file" | "Just tell me if anything's new" |

Both were verified end-to-end against live Microsoft endpoints on
2026-07-14, which surfaced a real gap: product IDs `3322`/`3323`/`3325`/`3326`
(Home/Pro China variants of the 25H2 "V2" refresh) existed on Microsoft's
side but weren't yet in the catalog. Since fixed.

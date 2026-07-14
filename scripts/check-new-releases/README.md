# check-new-releases

Checks Microsoft's public Windows 11 download page for a new "flagship" product
edition ID not yet in `cli/catalog.go`, and if found, probes nearby IDs to
discover the accompanying variant family (Home China, Pro China, ARM64, ...).

Windows 8.1 and Windows 10 aren't checked -- 8.1 is fully frozen and Windows 10
is past end-of-life, so neither will ever produce a new consumer ISO release.
Windows 11 is the only OS still shipping feature updates.

This tool only *discovers* candidate IDs. It doesn't touch `cli/catalog.go`,
`products.json`, or `validContributeProducts` automatically -- naming a new
entry correctly (build number, "Updated Oct"-style qualifiers, etc.) needs a
human checking Microsoft's own release-health pages.

## Usage

```bash
cd scripts/check-new-releases
go run .
```

If a new release is found, it makes real requests to Microsoft to probe the
variant family -- let it finish rather than re-running immediately.

Flags:
- `-catalog <path>` — path to `cli/catalog.go` (default: `../../cli/catalog.go`)
- `-probe-range <n>` — how many adjacent IDs to check after a new flagship is found (default: 20)

## Known limitation

The ARM64 family's offset from the flagship isn't consistent release to
release (observed: +3 for the 25H2 family, +18 for 24H2's). The default probe
range of 20 covers both known cases, but if no ARM64 variant shows up, try a
wider `-probe-range`.

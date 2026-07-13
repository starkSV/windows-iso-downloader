# Contributing to MSDL

Thanks for considering a contribution. This project is a small, community-run tool — bug reports, product additions, and fixes are all welcome.

## Reporting bugs

Open a [GitHub issue](https://github.com/starkSV/windows-iso-downloader/issues). Include:

- Whether you hit it via the web app, the CLI, or both
- CLI version (`msdl --help` prints it) or which page on msdl.tech-latest.com
- The exact error message, if any
- Product/language you were trying to fetch

## Development setup

See the [README's "Running Locally" section](./README.md#running-locally) for backend and frontend setup. For the CLI:

```bash
cd cli
go run . --help
go test ./...
```

## Adding a new consumer Windows release

1. Find the product ID at `www.microsoft.com/software-download-connector/api/`
2. Add it to `frontend/public/data/products.json` (name, archs, badge, related, active)
3. Add it to `cli/catalog.go`'s `consumerProducts` slice
4. Update the product table in `README.md`

## Adding a new evaluation edition

1. Find the fwlink URL at `microsoft.com/en-us/evalcenter/download-*`
2. Add the slug and fwlink to the `evalProducts` map in `backend/main.go`
3. Add the same slug to `cli/catalog.go`'s `evalProducts` slice
4. Add the product config to `frontend/src/data/evalProducts.ts`
5. Update the eval table in `README.md`

## Making changes

- Small, one-line config fixes can go straight to `main`.
- Anything touching behavior (new feature, bug fix, refactor) should go through a feature branch and PR.
- **Test locally before opening a PR** — build and run the affected component (`go build`, `go test ./...`, `npm run dev`) rather than relying on review to catch it.
- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `docs:`, `chore:`.
- Keep PRs focused — one fix or feature per PR, not a bundle of unrelated changes.

## CLI releases

CLI releases are cut by the maintainer via git tag (`cli/vX.Y.Z`), which triggers the GitHub Actions build for all platforms. Contributors don't need to worry about tagging or versioning — just get the fix or feature merged to `main`.

Package manager updates (winget, the [Homebrew tap](https://github.com/starkSV/homebrew-msdl), and the [`msdl-bin` AUR package](./aur/msdl-bin)) are submitted manually by the maintainer after each release, not automated in CI.

## Code style

No linter is enforced yet — match the surrounding code's style (Go: stdlib-first, minimal dependencies; frontend: existing Tailwind/component patterns). Don't add abstractions, comments, or error handling beyond what the change actually needs.

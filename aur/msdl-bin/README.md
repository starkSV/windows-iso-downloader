# msdl-bin (AUR)

> **⚠️ Publishing blocked as of 2026-07-13:** Arch Linux disabled new AUR account
> registrations on 2026-06-15 after a malware campaign compromised 1,500+ AUR
> packages across several waves. There's no announced reopening date. This
> package is ready to publish (`PKGBUILD` + `.SRCINFO` below) the moment
> registration reopens — check the [aur-general mailing list](https://lists.archlinux.org/mailman3/lists/aur-general.lists.archlinux.org/)
> or [aur.archlinux.org](https://aur.archlinux.org/register/) periodically.
> Until then, macOS/Linux users should use the [Homebrew tap](https://github.com/starkSV/homebrew-msdl) instead.

`PKGBUILD` and `.SRCINFO` for the [msdl-bin](https://aur.archlinux.org/packages/msdl-bin) AUR package, tracked here so version bumps have the same history/review as the winget manifests.

`.SRCINFO` is generated with the real `makepkg` tool (via an ephemeral `archlinux` Docker container — no VM or native Arch install needed) whenever `PKGBUILD` changes:

```bash
docker run --rm -v "$(pwd)":/pkg -w /pkg archlinux:latest bash -c "
  useradd -m builder && chown -R builder:builder /pkg &&
  pacman -Sy --noconfirm --needed base-devel &&
  su builder -c 'makepkg --printsrcinfo' > .SRCINFO
"
```

Publishing (requires an AUR account with an SSH key registered at aur.archlinux.org — this step can't be automated on someone else's behalf):

```bash
git clone ssh://aur@aur.archlinux.org/msdl-bin.git
cp PKGBUILD .SRCINFO msdl-bin/
cd msdl-bin
git add PKGBUILD .SRCINFO
git commit -m "Update to <version>"
git push
```

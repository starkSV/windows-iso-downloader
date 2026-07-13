# msdl-bin (AUR)

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

# msdl-bin (AUR)

`PKGBUILD` for the [msdl-bin](https://aur.archlinux.org/packages/msdl-bin) AUR package, tracked here so version bumps have the same history/review as the winget manifests.

This file is not auto-published — after editing it here, push it to the AUR repo manually from an Arch (or Arch container) environment:

```bash
git clone ssh://aur@aur.archlinux.org/msdl-bin.git
cp PKGBUILD msdl-bin/
cd msdl-bin
makepkg --printsrcinfo > .SRCINFO
git add PKGBUILD .SRCINFO
git commit -m "Update to <version>"
git push
```

Requires an AUR account with an SSH key registered at aur.archlinux.org.

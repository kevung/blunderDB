# AUR package: `blunderdb-bin`

`blunderdb-bin` is a precompiled (`-bin`) AUR package. Its recipe is generated
from [`PKGBUILD.in`](PKGBUILD.in) by substituting the version and the tarball
checksum, then pushed to the AUR. It sources the
`blunderDB-linux-webkit2gtk-4.1-<version>.tar.gz` release asset (which already
bundles the binary, `.desktop`, icon, metainfo and LICENSE).

Two ways to publish:

- **Automatically** — `.github/workflows/aur.yml` runs on every published
  GitHub release (opt-in; see *CI setup* below).
- **Manually** — `scripts/aur-publish.sh <version> --push` (from the repo root).

Both require a one-time AUR account + SSH key setup.

---

## 1. Create an AUR account

1. Register at <https://aur.archlinux.org/register>. The account needs an **SSH
   public key** (the AUR is git-over-SSH only; there is no password push).
2. You can reuse an existing key or, recommended, create one dedicated to AUR:

   ```bash
   ssh-keygen -t ed25519 -f ~/.ssh/aur -C "aur@blunderdb" -N ""
   ```

   This writes the private key `~/.ssh/aur` and the public key `~/.ssh/aur.pub`.
3. Paste the **public** key (`cat ~/.ssh/aur.pub`) into *My Account → SSH Public
   Key* on the AUR website and save.
4. (Optional) Make `ssh` use that key for the AUR host by adding to `~/.ssh/config`:

   ```
   Host aur.archlinux.org
     IdentityFile ~/.ssh/aur
     User aur
   ```
5. Test the connection (expect a help banner, not a shell — interactive shell is
   disabled on the AUR):

   ```bash
   ssh aur@aur.archlinux.org help
   ```

The AUR creates the package repository automatically on the **first push** if the
name `blunderdb-bin` is still available.

---

## 2. First / manual publish

After a GitHub release exists for `<version>` (so the `.tar.gz` asset is
available), from the repo root on an Arch machine (needs `makepkg`):

```bash
# Dry run — renders ./PKGBUILD and prints the computed sha256, no push:
scripts/aur-publish.sh <version>

# Publish — clones the AUR repo, regenerates .SRCINFO, commits and pushes:
scripts/aur-publish.sh <version> --push
```

The script downloads the release tarball, computes its checksum, fills
`PKGBUILD.in`, and pushes. Verify afterwards at
<https://aur.archlinux.org/packages/blunderdb-bin>.

---

## 3. CI setup (automatic publish on release)

`.github/workflows/aur.yml` publishes on `release: published`. It **no-ops
unless `AUR_SSH_PRIVATE_KEY` is set**, so it never fails a release before you opt
in. To enable it, add three repository secrets:

```bash
gh secret set AUR_SSH_PRIVATE_KEY < ~/.ssh/aur     # the PRIVATE key file
gh secret set AUR_USERNAME --body "your-aur-username"
gh secret set AUR_EMAIL    --body "kevin.unger@proton.me"
```

- `AUR_SSH_PRIVATE_KEY` — contents of the **private** key whose public half you
  registered on the AUR (step 1). Keep it secret; never commit it.
- `AUR_USERNAME` / `AUR_EMAIL` — used as the git commit author on the AUR repo.

Once set, every published release triggers the workflow: it waits for the 4.1
tarball asset, computes the checksum, renders the PKGBUILD and pushes via
[`KSXGitHub/github-actions-deploy-aur`]. You can also run it on demand from the
Actions tab (*Run workflow* → enter the version).

[`KSXGitHub/github-actions-deploy-aur`]: https://github.com/KSXGitHub/github-actions-deploy-aur

---

## 4. Updating

- **Automatic:** nothing to do — each release republishes via CI (if secrets set).
- **Manual:** `scripts/aur-publish.sh <new-version> --push`.

AUR users get the update through their helper (`yay -Syu`, `paru -Syu`).

If the packaging itself changes (new files, dependencies, install paths), edit
`PKGBUILD.in` and bump `pkgrel` for a packaging-only change, or just release a new
`<version>` for an upstream change (which resets `pkgrel` to 1).

# Release Process

This document describes the release pipeline for `vmvarela/github`, published to both the
[Terraform Registry](https://registry.terraform.io/providers/vmvarela/github) and the
[OpenTofu Registry](https://search.opentofu.org/provider/vmvarela/github).

## Versioning

This fork uses **CalVer** (`vYY.MM.N`) to avoid conflicts with upstream SemVer tags (`v6.x.x`):

| Field | Meaning | Example |
|-------|---------|---------|
| `YY` | 2-digit year | `26` |
| `MM` | Month (no zero-pad) | `3` |
| `N` | Sequential release within the month, starting at 0 | `0` |

Examples: `v26.3.0`, `v26.3.1`, `v26.4.0`

## One-Time Setup

### 1. GPG Signing Key

The Terraform Registry requires provider binaries to be signed with a GPG key. The public key must
be registered in your Terraform Registry account, and the private key must be stored as a repository
secret.

```bash
# Generate a new GPG key (RSA 4096, no expiry)
gpg --full-generate-key

# Export the private key (base64-encoded, for the secret)
gpg --export-secret-keys --armor <KEY_ID> | base64

# Export the public key (to register in the Terraform Registry)
gpg --export --armor <KEY_ID>
```

Register the **public key** at:
- Terraform Registry: Settings → GPG Keys → Add a GPG key
  (`https://registry.terraform.io/settings/gpg-keys`)
- OpenTofu Registry: follows the same Terraform Registry protocol; no separate key registration
  is needed

### 2. GitHub Repository Secrets

Store the following secrets under **Settings → Environments → `release`** in this repository:

| Secret | Value |
|--------|-------|
| `GPG_PRIVATE_KEY` | Base64-encoded private GPG key from the step above |
| `PASSPHRASE` | Passphrase used when generating the GPG key |

The `release` environment is referenced by `.github/workflows/release.yaml` and gates the signing
step.

### 3. Register the Provider

**Terraform Registry** (`registry.terraform.io`):

1. Sign in at <https://registry.terraform.io> with your GitHub account.
2. Click **Publish → Provider**.
3. Select the `vmvarela/terraform-provider-github` repository.
4. Confirm the GPG public key is already registered (step 1 above).
5. Click **Publish Provider**.

The registry will detect the first CalVer tag (`v26.x.x`) automatically on the next push.

**OpenTofu Registry** (`search.opentofu.org`):

The OpenTofu Registry mirrors providers published to the Terraform Registry via the same GitHub
release assets. No separate registration is required — once the Terraform Registry picks up a
release, the OpenTofu Registry will index it automatically, provided the
`terraform-registry-manifest.json` is included in the release (already configured in
`.goreleaser.yml`).

## Release Workflow

### Normal Release (new CalVer tag on `master`)

```bash
# 1. Ensure master is up to date with upstream and all feature branches
git checkout master
git fetch upstream main
git merge upstream/main
git merge billing-usage
git merge cost-centers
git merge enterprise-scim
git merge enterprise-teams

# 2. Run the build and tests to confirm everything is green
make build
make test

# 3. Create and push a CalVer tag
#    Format: vYY.MM.N  (e.g. first release of April 2026 -> v26.4.0)
git tag v26.4.0
git push origin master --tags
```

The [release workflow](.github/workflows/release.yaml) triggers automatically on `v*` tags and:

1. Builds binaries for all supported platforms (linux, darwin, windows, freebsd — amd64/arm64/386/arm)
2. Generates an SBOM with Syft
3. Signs the SHA256SUMS file with the GPG key from the `release` environment
4. Signs the SBOM with Cosign (keyless, via OIDC)
5. Creates a GitHub Release with all assets
6. Attests build provenance with `actions/attest-build-provenance`

Both registries poll GitHub Releases and will pick up the new version within minutes.

### Patch Release (bug fix on current month)

```bash
# Increment only the patch counter
git tag v26.4.1
git push origin master --tags
```

### Feature Branch Merge Before Release

When a feature branch has new commits that should be included in the next release:

```bash
git checkout master
git merge <feature-branch>
git push origin master
# Then follow the normal release steps above
```

## Branch Protection Notes

- **`main`** tracks `upstream/main` — never commit here directly.
- **`master`** is the integration and release branch — all tags must be pushed from here.
- Feature branches (`billing-usage`, `cost-centers`, `enterprise-scim`, `enterprise-teams`) each
  have an open PR against upstream. Keep them rebased on `upstream/main`.

## Verifying a Release

After the workflow completes:

```bash
# Verify the GitHub release assets exist
gh release view v26.4.0

# Confirm the provider appears in the Terraform Registry (may take a few minutes)
curl -s https://registry.terraform.io/v1/providers/vmvarela/github/versions | jq '.versions[-1].version'

# Verify build attestation
gh attestation verify ./dist/terraform-provider-github_v26.4.0_linux_amd64.zip \
  --repo vmvarela/terraform-provider-github
```

Full attestation verification instructions are in [VERIFY_ATTESTATIONS.md](VERIFY_ATTESTATIONS.md).

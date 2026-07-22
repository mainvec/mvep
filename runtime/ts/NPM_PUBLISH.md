# Publishing `@mainvec/mvep` to npm

The [`npm-publish.yml`](../.github/workflows/npm-publish.yml) GitHub Actions workflow publishes `@mainvec/mvep` to the npm registry automatically when a version tag (`v*`) is pushed. On pull requests it runs a **dry-run** (`npm publish --dry-run`) to catch metadata and build issues without publishing.

## Prerequisites

### 1. Create an npm access token

Create a **granular automation** access token on npm:

1. Sign in at https://www.npmjs.com/ as the package owner (the account that owns the `mainvec` organization).
2. Go to **Access Tokens** → **Generate New Token** → **Granular Access Token**.
3. Configure the token:
   - **Name**: `runtime/go-github-actions-publish`
   - **Expiration**: choose a sensible rotation window (e.g. 90 days)
   - **Packages and scopes**: select the `@mainvec` scope, **Read and write** for packages
   - **Organizations**: `mainvec`
4. Copy the token (`npm_xxx…`) — you'll only see it once.

> Prefer a **granular automation token** over a legacy automation token — it is scoped and revocable. Automation tokens can bypass 2FA for publishing, which is required for CI.

If the package requires two-factor authentication for publish, ensure the token is an **automation** token type so it can publish non-interactively.

### 2. Add the token as a GitHub repository secret

```bash
gh secret set NPM_TOKEN --repo mainvec/runtime/go --body "npm_xxx..."
```

The workflow reads this secret via `${{ secrets.NPM_TOKEN }}` and sets it as `NODE_AUTH_TOKEN` for the `npm publish` step.

### 3. Ensure 2FA is compatible

If your npm account enforces 2FA for publishes, the token **must** be an automation-type token (granular automation tokens qualify). Do **not** enable "require 2FA at publish time" with a non-automation token, or CI publishes will fail.

## How a release works

1. Update the version in [`package.json`](./package.json) — keep it in sync with the monorepo tag (e.g. `0.5.0`).
2. Commit the version bump and create an annotated tag:
   ```bash
   git tag -a v0.5.0 -m "v0.5.0"
   git push origin v0.5.0
   ```
3. Pushing the `v*` tag triggers `npm-publish.yml`. It:
   - installs deps (`npm ci`)
   - builds (`npm run build`)
   - tests (`npm test`)
   - verifies with `npm publish --dry-run`
   - publishes with `npm publish --access public`

`--access public` is required because `@mainvec/mvep` is a **scoped** package, and scoped packages are private by default on the free npm tier.

## Verifying a publish

```bash
npm view @mainvec/mvep version           # latest published version
npm view @mainvec/mvep versions --json   # all published versions
```

## What gets published

The `files` field in `package.json` allowlists `dist` and `src`. The `.npmignore` further excludes tests, configs, and editor files. The published tarball contains only:

- `dist/` — compiled JS + `.d.ts` type declarations (the runtime entry points)
- `src/` — original TypeScript source (for source maps / debugging)
- `README.md` and `LICENSE` (included automatically by npm)

Inspect locally with:

```bash
cd runtime/ts
npm pack --dry-run   # lists files that would be published
```

## Rollback / unpublish

npm does not allow unpublishing once a package has been depended on for more than 24 hours. To retract a broken release, **deprecate** it instead:

```bash
npm deprecate @mainvec/mvep@0.5.0 "use 0.5.1 instead"
```

Then publish a fixed version (e.g. `0.5.1`) with a new tag.
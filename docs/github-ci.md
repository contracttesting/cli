# Using ctio with GitHub CI

Versions are git commit SHAs. The CLI auto-detects the version from the checkout (`git rev-parse --short HEAD`) when `--version` is omitted, so pipelines need no version wrangling. Branches map to environments in the workflow config — the broker only knows versions and environments.

## Setup (once per environment)

```bash
ctio create-environment staging
ctio create-environment production
```

## Deploy pipeline

The branch→environment mapping lives here, not in the broker:

```yaml
# .github/workflows/deploy.yml
on:
  push:
    branches: [main, develop]

jobs:
  deploy:
    runs-on: ubuntu-latest
    env:
      ENV: ${{ github.ref_name == 'main' && 'production' || 'staging' }}
    steps:
      - uses: actions/checkout@v4

      - run: ctio publish api.yaml --participant users
      - run: ctio can-i-deploy users --environment $ENV

      # ... deploy the service ...

      # last step, only after the deployment succeeded
      - run: ctio record-deployment users --environment $ENV
```

`can-i-deploy` exits non-zero when the version is incompatible with the counterparts currently deployed in the environment, failing the job before the deploy step. `record-deployment` is gated by the broker: it requires a fresh passing compatibility check for that exact version and environment.

## Pull request pipeline (status check, no deployment)

Publish and check against the branch's target environment; never record a deployment:

```yaml
# .github/workflows/pr.yml
on: pull_request

jobs:
  contract-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: ctio publish api.yaml --participant users
      - run: ctio can-i-deploy users --environment production
```

Every PR commit gets its own version (its SHA). Commits that don't touch the contract publish as aliases of the same contract snapshot — the version is still addressable, and `can-i-deploy` still answers for it.

## Reverts and rollbacks

A revert deploy runs the exact same three steps as any deploy:

- `git revert` produces a new commit → `publish` aliases it to the old contract snapshot.
- A reset/redeploy of an already-published SHA → `publish` is an idempotent success.

Never skip `can-i-deploy` because "the contract didn't change" — the counterparts deployed in the environment might have changed since. The check runs against what is deployed *now*, and the record-deployment gate rejects a deploy without a fresh check anyway. After `record-deployment`, the reverted SHA becomes the environment's latest-deployed version.

## Overriding the version

Auto-detection needs a git checkout (`actions/checkout`). Outside a git repo, or to pin an explicit version, pass the flag:

```bash
ctio publish api.yaml --participant users --version 76a39e5
ctio can-i-deploy users --version 76a39e5 --environment production
ctio record-deployment users --version 76a39e5 --environment production
```

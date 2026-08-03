# Defer the golangci-lint-action major bump in Dependabot

**Branch:** `chore/dependabot-defer-golangci-action-major`
**File:** `.github/dependabot.yml`

---

## Problem

The weekly `github-actions` Dependabot run opens a major bump of
`golangci/golangci-lint-action` (v6 → v9, PR #17). This bump **fails CI**: action
v9 defaults to golangci-lint **v2**, whose `.golangci.yml` config schema is
incompatible with the v1-style config this repo uses. Taking it requires a
deliberate golangci-lint v1→v2 config migration, not an automated bump.

#17 was closed manually, but with no ignore rule Dependabot **recreates it on the
next weekly run** — the same red PR reappears indefinitely.

**Root cause:** no `ignore` entry scopes out the `golangci-lint-action` semver-major,
so Dependabot keeps proposing the breaking v9 bump.

---

## Fix

Add a narrow `ignore` rule to the **existing** `github-actions` ecosystem block —
scoped to this one action's major only. All other GitHub Actions majors keep
flowing (checkout, setup-*, upload-artifact, fetch-metadata were merged this cycle);
this defers **only** the action whose major is a breaking migration.

### Change — `.github/dependabot.yml`

**Exact old block:**

```yaml
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
```

**Exact new block:**

```yaml
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    ignore:
      # golangci-lint-action v9 defaults to golangci-lint v2, whose config schema
      # breaks this repo's v1-style .golangci.yml. Deferred until a deliberate
      # golangci-lint v1→v2 migration. Other Actions majors still flow — narrow
      # this rule (or remove it) when the migration is scheduled.
      - dependency-name: "golangci/golangci-lint-action"
        update-types: ["version-update:semver-major"]
```

---

## Files Changed

| File | Change |
|------|--------|
| `.github/dependabot.yml` | Add `github-actions` ignore for `golangci/golangci-lint-action` semver-major |

---

## Definition of Done

- [ ] `.github/dependabot.yml` parses as valid YAML
- [ ] Only the one file changed
- [ ] Committed and pushed to `chore/dependabot-defer-golangci-action-major`
- [ ] PR opened, CI green, merged
- [ ] After merge, confirm a future weekly run no longer recreates the v9 bump

---

## What NOT to Do

- Do NOT broaden the ignore to all `github-actions` majors — the other majors are wanted.
- Do NOT commit to `main`.
- Do NOT bump golangci-lint or migrate the config here — that is separate, deliberate work.

# Issue: README/docs structure drift

**Date:** 2026-03-17
**Status:** Closed

## Summary
Basket service documentation diverged from the shared template. `docs/testing/README.md` and `docs/issues/` were missing, and the root README mixed architecture/API/testing guidance without the standardized section order.

## Impact
- Operators lacked a consistent location for test commands and troubleshooting steps.
- Future automation relying on docs/ structure could not find testing metadata.
- README navigation varied across repos, increasing onboarding time.

## Resolution
- Added `docs/testing/README.md` describing Go unit/integration test workflows, coverage, and linting.
- Created `docs/issues/` with this log entry.
- Rewrote `README.md` to the standard template (description → quick start → usage → architecture → directory layout → documentation links → releases → related → license).

## Follow Up
- Mirror the same structure across the remaining shopping-cart services per platform spec.
- Update `docs/testing/README.md` whenever new suites or tooling are added.

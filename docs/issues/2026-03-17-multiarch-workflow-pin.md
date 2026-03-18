# Issue: Multi-arch workflow pin update

**Date:** 2026-03-17
**Status:** Closed

## Summary
The reusable workflow pin in `.github/workflows/go-ci.yml` referenced SHA `8363caf`, which produced amd64-only images. The infra repo added `platforms: linux/amd64,linux/arm64` at SHA `999f8d7`, so the pin needed updating.

## Fix
- Updated the workflow reference to `build-push-deploy.yml@999f8d70277b92d928412ff694852b05044dbb75`.
- Ensures CI publishes both amd64 and arm64 images, unblocking ArgoCD sync on the arm64 Ubuntu node.

## Follow Up
- Monitor CI runs to confirm multi-arch manifests are pushed.
- Re-pin if infra workflow is updated again.

# Upgrade Guide

This document provides guidance for upgrading between major versions of the telegraf-sidecar-operator Helm chart.

## v1.3.x to v2.x

### Breaking Changes

#### Feature Gates System

The chart now uses a unified feature gates system instead of individual boolean flags for experimental features.

**Before (v1.3.x):**
```yaml
sidecar:
  enableNativeSidecars: true
```

**After (v2.x):**
```yaml
featureGates:
  - "operator.nativesidecars"
```

`sidecar.enableNativeSidecars` has been removed. To enable native sidecars, add `operator.nativesidecars` to the `featureGates` list.

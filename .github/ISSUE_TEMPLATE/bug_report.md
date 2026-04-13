---
name: Bug Report
about: Report a bug in AstroCTFb
title: "[Bug]: "
labels: bug
assignees: ""
---

## Describe the Bug

A clear description of what the bug is.

## To Reproduce

Steps to reproduce the behavior:

1. Start AstroCTFb with `make -C backend compose-full`
2. Navigate to / call the endpoint…
3. Observe the error

```bash
# If applicable - curl or relevant command
curl -X POST http://localhost:8090/api/v1/...
```

## Expected Behavior

What you expected to happen.

## Actual Behavior

What actually happened. Include full error output or HTTP response if applicable.

## Environment

- **AstroCTFb version / commit**: (e.g., `git rev-parse --short HEAD`)
- **Go version**: (`go version`)
- **OS**: (e.g., Ubuntu 22.04, macOS 15)
- **Deployment**: (e.g., `compose-infra` + local binary, `compose-full`, production Docker)
- **Storage provider**: (e.g., local, SeaweedFS / S3)
- **Competition mode**: (e.g., `flexible`, `teams`, `solo`)

## Logs

```text
# Paste relevant backend logs here (LOG_LEVEL=debug for more detail)
```

## Additional Context

Any other context: config values (without secrets), migration state, repro rate, etc.

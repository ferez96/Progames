# BUG-001: MaxStdoutLineBytes defaults to 64 bytes instead of 64 KiB

## Severity

High — affects all bot matches in production.

## Spec reference

`SPECS.md` §14.4: "Max single stdout line read: 64 KiB — Longer input → treat as invalid move / crash per runner."

## What the code does

`internal/config/config.go:33`:
```go
envInt("PROGAMES_MAX_STDOUT_LINE_BYTES", 64)
```

Default is **64 bytes**, not 64 KiB (65536 bytes). Any bot whose stdout line exceeds 64 characters is treated as an invalid move or crash, causing an immediate game loss.

## Impact

Any bot that prints anything beyond the bare `x,y` coordinate — debug output, trailing whitespace, logging — risks hitting the limit. The practical effect is that most real bots will fail silently on move output during match execution.

## Fix

`internal/config/config.go:33`: change default from `64` to `64*1024`.

## Done when

- Default value is `65536` (64 KiB).
- Existing runner length-check logic (`runner.go`, `container.go`) unchanged — the limit value is already plumbed correctly; only the default is wrong.
- Config-level test or comment documents the unit explicitly to prevent regression.

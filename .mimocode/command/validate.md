---
description: Run the full Go validation cycle: vet, format, test, build. Use after code changes to confirm nothing is broken. Accepts optional package path filter.
---

# Go Validation Cycle

Run the standard k13d validation pipeline. Execute ALL steps even if an earlier one warns (but STOP on error).

## Steps

1. **go vet** — static analysis
2. **gofmt** — auto-format changed files
3. **go test -race** — unit tests with race detector
4. **go build** — verify compilation

## Command

```bash
cd /Users/chime/develop/antigravity/k13d && \
  go vet $ARGUMENTS 2>&1 && \
  gofmt -s -w $ARGUMENTS 2>&1 && \
  go test -race $ARGUMENTS 2>&1 && \
  go build ./... 2>&1 && \
  echo "✓ All validation passed"
```

## Usage

```
/validate                  # validate entire project
/validate ./pkg/cli/...    # validate specific package
/validate ./pkg/config/... ./pkg/ui/...  # multiple packages
```

## Notes

- If `$ARGUMENTS` is empty, validates the whole project (`./...`).
- `gofmt -s -w` modifies files in place — this is intentional.
- `go build ./...` always runs on the full project regardless of filter, to catch cross-package breakage.
- If any step fails, stop and report the error. Do NOT continue to the next step.

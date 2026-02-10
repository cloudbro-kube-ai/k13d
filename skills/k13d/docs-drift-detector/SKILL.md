---
name: check-drift
description: 코드 변경이 문서에 영향을 미치는지 감지합니다. 문서 drift를 찾고 업데이트가 필요한 파일을 알려줍니다. Triggers - '/check-drift', 'check documentation drift', '문서 drift 확인', '문서 동기화 확인'
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash
---

# Documentation Drift Detector

코드 변경이 문서에 영향을 미치는지 감지하는 스킬입니다.

## 사용법

```
/check-drift
```

## 워크플로우

### 1. 변경된 파일 분석

최근 변경된 파일을 확인합니다:

```bash
git diff --name-only HEAD~5
# 또는
git diff --name-only main...HEAD
```

### 2. 탐지 규칙 적용

다음 패턴을 확인합니다:

| 코드 변경 | 영향받는 문서 |
|-----------|---------------|
| `go.mod` (Go 버전) | README.md 배지, CLAUDE.md, installation.md |
| `cmd/*/main.go` (CLI 플래그) | reference/cli.md, reference/env-vars.md |
| `pkg/ui/*.go` (키바인딩) | user-guide/shortcuts.md, concepts/tui-architecture.md |
| `pkg/web/*.go` (API) | reference/api.md, features/web-ui.md |
| `pkg/config/*.go` (설정) | getting-started/configuration.md |
| `pkg/ai/**/*.go` (AI) | ai-llm/*.md, concepts/ai-assistant.md |
| `pkg/mcp/*.go` (MCP) | concepts/mcp-integration.md |

### 3. Drift 리포트 생성

```yaml
drifts:
  - id: cli-flag-added
    severity: high
    source: cmd/kube-ai-dashboard-cli/main.go
    change: "Added CLI flag: --trace-level"
    affected_docs:
      - file: docs-site/docs/reference/cli.md
        action: "Add --trace-level to CLI options table"
      - file: docs-site/docs/reference/env-vars.md
        action: "Document TRACE_LEVEL environment variable"
```

## 탐지 패턴

### Go 버전 변경
```
파일: go.mod
패턴: ^go (\d+\.\d+)
영향: README.md, CLAUDE.md, installation.md
```

### CLI 플래그 추가
```
파일: cmd/**/main.go
패턴: flag\.(String|Bool|Int|Duration)\("([^"]+)"
영향: reference/cli.md, reference/env-vars.md
```

### TUI 키바인딩 변경
```
파일: pkg/ui/app*.go
패턴: case tcell\.Key|KeyRune.*'.'
영향: user-guide/shortcuts.md, concepts/tui-architecture.md
```

### Web API 엔드포인트
```
파일: pkg/web/*.go
패턴: http\.Handle(Func)?\("(/[^"]+)"
영향: reference/api.md, features/web-ui.md
```

### 설정 스키마 변경
```
파일: pkg/config/*.go
패턴: type \w+Config struct|`yaml:"([^"]+)"`
영향: getting-started/configuration.md
```

## 출력 예시

```
## Documentation Drift Report

### High Severity (2)

1. **CLI Flag Added** in `cmd/kube-ai-dashboard-cli/main.go`
   - Change: Added `--trace-level` flag
   - Affects:
     - `docs-site/docs/reference/cli.md` → Add to CLI options table
     - `docs-site/docs/reference/env-vars.md` → Document TRACE_LEVEL

2. **Config Field Added** in `pkg/config/config.go`
   - Change: Added `AIModelProfiles` field
   - Affects:
     - `docs-site/docs/getting-started/configuration.md` → Document model_profiles

### Medium Severity (1)

3. **Keybinding Changed** in `pkg/ui/app_actions.go`
   - Change: Added Ctrl+R refresh shortcut
   - Affects:
     - `docs-site/docs/user-guide/shortcuts.md` → Add to shortcuts table

---

Run `/update-docs` to generate documentation updates.
```

## 다음 단계

Drift가 감지되면 `/update-docs`를 실행하여 문서 업데이트를 생성할 수 있습니다.

---
name: sync-changelog
description: Git 커밋에서 CHANGELOG.md 항목을 생성합니다. Conventional Commit 형식을 파싱하여 릴리즈 노트를 자동 생성합니다. Triggers - '/sync-changelog', 'sync changelog', 'generate changelog', '변경로그 동기화', '변경로그 생성'
allowed-tools:
  - Read
  - Write
  - Edit
  - Bash
  - Grep
---

# Changelog Sync

Git 커밋에서 CHANGELOG.md 항목을 생성하는 스킬입니다.

## 사용법

```
/sync-changelog
/sync-changelog for v0.8.0
/sync-changelog from v0.7.0 to HEAD
```

## 워크플로우

### 1. 커밋 범위 결정

```bash
# 최근 태그 이후 커밋
git log --oneline $(git describe --tags --abbrev=0)..HEAD

# 특정 버전 범위
git log --oneline v0.7.0..v0.8.0

# 또는 main 브랜치와 비교
git log --oneline main..HEAD
```

### 2. Conventional Commit 파싱

커밋 메시지를 분석합니다:

| Prefix | Category | Example |
|--------|----------|---------|
| `feat:` | Added | feat(tui): add Ctrl+R refresh |
| `fix:` | Fixed | fix(web): resolve auth timeout |
| `docs:` | Documentation | docs: update README |
| `refactor:` | Changed | refactor(ai): simplify agent loop |
| `test:` | Testing | test: add config unit tests |
| `chore:` | Maintenance | chore: update dependencies |
| `perf:` | Performance | perf(tui): optimize render loop |
| `security:` | Security | security: fix XSS vulnerability |

### 3. 항목 그룹화

```markdown
## [0.8.0] - 2026-02-09

### Added
- **TUI**: Add Ctrl+R for refresh (#123)
- **Config**: Support multiple AI model profiles (#125)

### Fixed
- **Web**: Resolve authentication session timeout (#124)
- **AI**: Fix streaming response memory leak (#126)

### Changed
- **AI**: Simplify agent state machine (#127)

### Security
- **Web**: Fix potential XSS in error messages (#128)
```

### 4. 미리보기 & 승인

```
## Proposed Changelog Entry

## [0.8.0] - 2026-02-09

### Added
- **TUI**: Add Ctrl+R for refresh (#123)
- **Config**: Support multiple AI model profiles (#125)

### Fixed
- **Web**: Resolve authentication session timeout (#124)

---

Insert this at line 8 of CHANGELOG.md?
[A]pprove / [E]dit / [C]ancel
```

## Conventional Commit 형식

### 기본 형식
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### 예시
```
feat(tui): add autocomplete dropdown for commands

- Shows dropdown when 2+ completions match
- Navigate with Up/Down, select with Tab/Enter
- Dismiss with Esc

Closes #42
```

### 파싱 결과
```yaml
type: feat
scope: tui
description: add autocomplete dropdown for commands
body: |
  - Shows dropdown when 2+ completions match
  - Navigate with Up/Down, select with Tab/Enter
  - Dismiss with Esc
references:
  - "#42"
```

## CHANGELOG.md 형식

[Keep a Changelog](https://keepachangelog.com/) 형식을 따릅니다:

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New feature X

## [0.7.0] - 2026-02-01

### Added
- Feature A
- Feature B

### Fixed
- Bug fix C
```

## 출력 예시

```
## Changelog Sync

Analyzing commits from v0.7.0 to HEAD...

Found 12 commits:
- 5 feat (features)
- 4 fix (bug fixes)
- 2 docs (documentation)
- 1 chore (maintenance)

### Proposed Entry

## [0.8.0] - 2026-02-09

### Added
- **TUI**: Add autocomplete dropdown for commands (#42)
- **TUI**: Add Ctrl+R refresh shortcut (#45)
- **Config**: Support model profiles for LLM switching (#48)
- **Plugin**: TUI integration for plugins.yaml (#50)
- **AI**: Preserve chat history across questions (#52)

### Fixed
- **TUI**: Fix screen ghosting during AI streaming (#43)
- **Web**: Resolve session timeout on long AI responses (#46)
- **AI**: Fix memory leak in streaming handler (#49)
- **Config**: Fix aliases not merging with built-ins (#51)

---

Insert after line 7 (after ## [Unreleased] section)?
[A]pprove / [E]dit / [C]ancel: A

✓ Changelog updated!

Note: docs-site/docs/reference/changelog.md uses snippets,
      so it will automatically reflect this change.
```

## Snippets 연동

`docs-site/docs/reference/changelog.md`는 다음과 같이 설정되어 있습니다:

```markdown
# Changelog

--8<-- "CHANGELOG.md:3:"
```

따라서 루트 `CHANGELOG.md`만 업데이트하면 docs-site에 자동 반영됩니다.

## 팁

### PR 병합 시 자동화
```bash
# PR 병합 후 changelog 동기화
git checkout main
git pull
claude "/sync-changelog"
```

### 릴리즈 워크플로우
```bash
# 1. Changelog 동기화
claude "/sync-changelog for v0.8.0"

# 2. 버전 태그
git tag v0.8.0

# 3. 푸시
git push && git push --tags
```

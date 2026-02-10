---
name: update-docs
description: 코드 변경에 따른 문서 업데이트를 생성합니다. Drift 감지 결과를 바탕으로 문서 수정안을 제안하고 사용자 승인 후 적용합니다. Triggers - '/update-docs', 'update documentation', '문서 업데이트', '문서 수정'
allowed-tools:
  - Read
  - Write
  - Edit
  - Grep
  - Glob
  - Bash
---

# Documentation Updater

코드 변경에 따른 문서 업데이트를 생성하는 스킬입니다.

## 사용법

```
/update-docs
/update-docs for pkg/config/config.go changes
```

## 워크플로우

### 1. Drift 확인

먼저 `/check-drift`를 실행하거나, 이미 감지된 drift 리포트를 확인합니다.

### 2. 문서 컨텍스트 수집

영향받는 문서의 현재 상태를 읽습니다:
- 파일 구조 및 포맷
- 관련 섹션 내용
- 스타일 가이드 (CLAUDE.md)

### 3. 코드 분석

변경된 코드에서 문서에 필요한 정보를 추출합니다:
- 새 CLI 플래그: 이름, 타입, 기본값, 설명
- 새 설정 필드: YAML 태그, 타입, 용도
- 새 키바인딩: 키, 동작, 컨텍스트
- 새 API 엔드포인트: 경로, 메서드, 파라미터

### 4. Diff 미리보기 생성

```markdown
## File: docs-site/docs/reference/cli.md

### Section: Command Line Options

**Current:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --web | bool | false | Enable web UI mode |
| --port | int | 8080 | Web server port |

**Proposed:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --web | bool | false | Enable web UI mode |
| --port | int | 8080 | Web server port |
| --trace-level | string | info | Set trace level (debug\|info\|warn\|error) |

---

Approve this change? [A]pprove / [E]dit / [S]kip / [Q]uit
```

### 5. 사용자 승인

- **[A]pprove**: 변경 적용
- **[E]dit**: 수정 요청 (Claude가 재작성)
- **[S]kip**: 이 파일 건너뛰기
- **[Q]uit**: 모든 업데이트 취소

## 스타일 가이드

### MkDocs 호환성
- YAML frontmatter 유지
- 올바른 heading 레벨 사용
- 내부 링크 형식: `[text](../path/file.md)`

### 코드 예시
- 언어 태그 필수: ` ```yaml `, ` ```go `, ` ```bash `
- 실행 가능한 예시 선호
- 플레이스홀더는 `<your-value>` 형식

### 테이블 형식
- Markdown 테이블 사용
- 정렬: 왼쪽 정렬 기본
- 필수 컬럼: Name, Type, Default, Description

### 톤 & 스타일
- CLAUDE.md의 프로젝트 용어 사용
- 간결하고 명확한 설명
- 한글/영어 혼용 시 기존 문서 스타일 따름

## 안전 기능

### Never Auto-Commit
모든 변경은 사용자 승인 필요. 스킬은:
1. 메모리에서 패치 준비
2. Diff 미리보기 표시
3. 승인 후에만 파일 쓰기

### 백업 & 롤백
```bash
# 변경 전 백업 (git stash)
git stash push -m "docs-backup-before-update"

# 롤백이 필요한 경우
git stash pop
# 또는
git restore docs-site/docs/reference/cli.md
```

## 출력 예시

```
## Documentation Update Preview

### 1/3: docs-site/docs/reference/cli.md

**Change:** Add --trace-level flag documentation

```diff
 | --port | int | 8080 | Web server port |
+| --trace-level | string | info | Set trace level (debug|info|warn|error) |
```

Approve? [A/E/S/Q]: A
✓ Applied update to cli.md

### 2/3: docs-site/docs/getting-started/configuration.md

**Change:** Document model_profiles configuration

```diff
 llm:
   provider: openai
   model: gpt-4
+
+  # Multiple model profiles (v0.8.0+)
+  model_profiles:
+    - name: fast
+      model: gpt-3.5-turbo
+    - name: accurate
+      model: gpt-4
```

Approve? [A/E/S/Q]: E
What would you like to change? Add temperature field example
[Regenerating...]

```diff
+  model_profiles:
+    - name: fast
+      model: gpt-3.5-turbo
+      temperature: 0.3
+    - name: accurate
+      model: gpt-4
+      temperature: 0.1
```

Approve? [A/E/S/Q]: A
✓ Applied update to configuration.md

### Summary
- 2 files updated
- 1 file skipped
- Run `git diff docs-site/` to review all changes
```

## 다음 단계

문서 업데이트 후:
1. `git diff docs-site/`로 변경사항 확인
2. 로컬에서 `mkdocs serve`로 미리보기 (선택)
3. 변경사항 커밋

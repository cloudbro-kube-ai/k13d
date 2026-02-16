# k13d Living Documentation Skills

이 디렉토리에는 k13d 문서를 코드와 동기화 상태로 유지하기 위한 Claude 스킬이 포함되어 있습니다.

## 사용 가능한 스킬

### 1. Docs Drift Detector (`/check-drift`)

**목적:** 코드 변경이 문서에 영향을 미치는지 감지

**사용법:**
```
/check-drift
```

**동작:**
1. 최근 변경된 파일 분석
2. `config/docs-detection-rules.yaml` 규칙과 매칭
3. 영향받는 문서 목록 출력
4. 업데이트 제안

---

### 2. Docs Updater (`/update-docs`)

**목적:** 감지된 drift를 기반으로 문서 업데이트 생성

**사용법:**
```
/update-docs
/update-docs for pkg/config/config.go changes
```

**동작:**
1. Drift 리포트 확인 (또는 `/check-drift` 먼저 실행)
2. 현재 문서 상태 분석
3. 업데이트 diff 미리보기 생성
4. 사용자 승인 후 적용

---

### 3. Changelog Sync (`/sync-changelog`)

**목적:** Git 커밋에서 CHANGELOG.md 항목 생성

**사용법:**
```
/sync-changelog
/sync-changelog for v0.8.0
```

**동작:**
1. 최근 태그 이후 커밋 분석
2. Conventional Commit 형식 파싱
3. CHANGELOG.md 항목 생성
4. 사용자 승인 후 삽입

---

## 스킬 테스트

```bash
# Drift detector 테스트
claude "/check-drift"

# Docs updater 테스트
claude "/update-docs"

# Changelog sync 테스트
claude "/sync-changelog"
```

## 설정

탐지 규칙은 `config/docs-detection-rules.yaml`에 정의되어 있습니다.

## 안전 기능

- **모든 쓰기 작업은 사람 승인 필요**
- 자동 커밋 없음
- Diff 미리보기 필수
- 롤백 지원 (`git restore`)

## 문서 구조

이 스킬들은 다음 문서 구조를 대상으로 합니다:

```
docs-site/docs/           # Single Source of Truth
├── getting-started/      # 설치, 설정
├── features/             # 기능 설명
├── concepts/             # 아키텍처, MCP
├── user-guide/           # 사용법, 단축키
├── ai-llm/               # AI 관련
├── deployment/           # 배포 가이드
└── reference/            # CLI, API, 환경변수
```

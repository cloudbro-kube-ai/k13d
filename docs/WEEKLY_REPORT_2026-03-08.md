# k13d 주간 회의록 (2026-02-23 ~ 2026-03-08)

## 1. 이번 주 주요 성과

### v0.9.5 릴리스 (2026-03-01)

#### [Feature] TUI & Web UI 아키텍처 개선 (`8c84033`)

- Web UI 프론트엔드 모듈 분리: `VirtualScroller.js`, `api.js`, `i18n.js` 신규 추가
- AI Provider 인터페이스 개선 및 Azure OpenAI 어댑터 수정
- `index.html` 구조 리팩터링 (867줄 변경)
- `app.js` 대규모 정리 (17,504줄 변경)
- TUI `app.go` 기능 추가 및 테스트 보강
- 총 30개 파일 변경, +10,006 / -9,359줄

#### [Bug Fix] Web UI 로그인 폼 이슈 (6건)

| 이슈 | 원인 | 해결 |
|------|------|------|
| 로그인 입력란 미노출 | CSS inline style과 .active 클래스 충돌 | classList 기반으로 통일 |
| auth-mode local에서 토큰 폼 표시 | JS 비동기 API 의존 | 서버사이드 `__AUTH_MODE__` 주입 |
| K13D 로고 우측 밀림 | ASCII art overflow | `overflow: hidden` 적용 |
| 전체 JS 미실행 | 라인 723 orphan 코드 (`") {"`) 구문 오류 | 잔존 코드 제거 |
| renderTableBody 미정의 | 함수 참조만 있고 정의 누락 | `generateRowHTML()`, `renderTableBody()` 구현 |
| 무한 리로드 루프 | 인증 전 API 호출 → 401 → logout → reload 반복 | `loadClusterContexts()`를 `showApp()` 안으로 이동 |

#### [Bug Fix] Lint 오류 수정 (2건)

- `pkg/ai/client.go`: `w.Write` 반환값 미검사
- `pkg/web/server.go`: `w.Write` 반환값 미검사

### 문서 업데이트 (2026-02-25)

#### [Docs] `-auth-mode local` 문서화 및 전체 문서 갱신 (`28d4ab9`)

- README: 인증 모드 테이블 (5개 모드), CLI 플래그, AI Provider 테이블 (8개) 추가
- README: MCP 설정 예시, Shell Completion 섹션 추가
- Installation Guide: 인증 모드 테이블 및 local auth 예시 추가
- Docker Guide: Quick Start에 `-auth-mode local` 추가
- Configuration Guide: Anthropic, Gemini, Bedrock 프로바이더 추가

## 2. 변경 파일 요약

| 영역 | 파일 수 | 주요 파일 |
|------|---------|-----------|
| Web UI (Frontend) | 6 | `index.html`, `app.js`, `api.js`, `views.css`, `VirtualScroller.js`, `i18n.js` |
| Web Server (Backend) | 8 | `server.go`, `handlers_settings.go`, `handlers_ai.go`, 테스트 파일들 |
| AI / Provider | 4 | `client.go`, `azopenai.go`, `interface.go`, `providers_test.go` |
| TUI | 3 | `app.go`, `app_test.go`, `tui_test_fixtures.go` |
| Docs | 5 | `README.md`, `CHANGELOG.md`, `installation.md`, `docker.md`, `configuration.md` |

## 3. 커밋 이력

| 날짜 | 커밋 | 설명 |
|------|------|------|
| 02-25 | `28d4ab9` | docs: add -auth-mode local documentation and refresh all docs |
| 03-01 | `8c84033` | feat: TUI and Web UI architecture, stability, and UX enhancements |
| 03-01 | `ad16a74` | fix: linting error in pkg/ai/client.go |
| 03-01 | `eb4d1bd` | fix(web): fix login form visibility by unifying display control |
| 03-01 | `c759d3c` | fix(web): show token login form by default with active class |
| 03-01 | `69222c9` | fix(web): server-side auth mode injection and login layout fix |
| 03-01 | `b8e9a8a` | fix(web): remove broken orphan code causing JS syntax error |
| 03-01 | `44dcb3c` | fix(web): server-side inline style injection for login form visibility |
| 03-01 | `4673f57` | fix(web): define generateRowHTML and renderTableBody functions |
| 03-01 | `a46694c` | fix(web): prevent infinite reload loop on login page |
| 03-01 | `1c86da5` | fix: check w.Write error return to satisfy errcheck linter |
| 03-01 | `a15c20e` | docs: add v0.9.5 changelog for web UI login and stability fixes |

## 4. CloudBro 최종 발표 안내

### 일정

| 항목 | 내용 |
|------|------|
| 일시 | 2026-04-04 (토) 13:00 ~ 17:00 |
| 장소 | 디캠프 마포 5층 박병원홀 (공덕역 4번 출구) |
| 주소 | 서울 마포구 마포대로 122 |
| 발표 형식 | 15분 발표 + 5분 Q&A (팀당 20분) |

### 타임라인

| 시간 | 내용 |
|------|------|
| 13:00 - 13:10 | 오프닝 |
| 13:10 - 14:50 | 시즌 2 최종 발표 (5팀 × 20분) |
| 15:00 - 15:45 | 기술검증단 발표 (3분 × 15분) |
| 이후 | 별도 세션 (TBD) |

### 발표 자료 구성 (필수 항목)

1. 문제 정의
2. 프로젝트 팀의 접근 및 차별점
3. 아키텍처 전체 구조 Overview
4. 라이브 데모
5. 적용 시나리오
6. 향후 로드맵

## 5. 내부 릴리스 계획

### v1.0.0 릴리스 (3월 중순 목표)

- **목표**: v1.0.0 정식 릴리스
- **예상 일정**: 3월 중순 (오프라인 미팅에서 확정 예정)
- **주요 작업**:
  - [ ] `app.js` (약 10,900줄) 멀티 파일 분리 — 유지보수성 개선
  - [ ] Dependabot 보안 취약점 1건 확인 및 조치
  - [ ] 라이브 데모용 안정성 강화 (4/4 발표 대비)
  - [ ] PoC 시나리오 정의 및 환경 구성
  - [ ] 전체 기능 검증 및 QA

### 오프라인 미팅 계획

- **목적**: v1.0.0 릴리스 일정 확정 및 최종 발표 준비 점검
- **시기**: 3월 중 (일정 조율 중)

## 6. 앞으로 해야 할 것

### 사전 팀 WIKI 최종 수정

- **기한: ~3/8 (일) 23:59까지**
- [ ] 해결하고자 하는 문제 작성
- [ ] 적용 가능한 대상 기업/조직 정의 (도메인, 조직 상황 등)
- [ ] 작동 방식 및 기술 스택 정리
- [ ] 완성도 및 PoC 가능 범위 작성 (기간, 필요 환경, 예상 결과물)

### 최종 발표 자료 제출

- **기한: ~4/1 (수) 23:59까지**
- **참고**: 발표 순서는 제출 시간 역순 (가장 늦게 제출한 팀이 1번)
- [ ] CloudBro PPT 템플릿 수령 (3/8 경 공유 예정)
- [ ] 문제 정의 슬라이드 작성
- [ ] 접근 방식 및 차별점 슬라이드 작성
- [ ] 아키텍처 Overview 슬라이드 작성
- [ ] 라이브 데모 준비 및 시나리오 구성
- [ ] 적용 시나리오 슬라이드 작성
- [ ] 향후 로드맵 슬라이드 작성

### 프로젝트 개발 TODO

- [ ] `app.js` (약 10,900줄) 멀티 파일 분리 — 유지보수성 개선
- [ ] v1.0.0 릴리스 태그 확정 및 배포 (3월 중순)
- [ ] Dependabot 보안 취약점 1건 확인
- [ ] 라이브 데모용 안정성 강화 (4/4 발표 대비)
- [ ] PoC 시나리오 정의 및 환경 구성

## 7. 주요 일정 요약

| 기한 | 항목 |
|------|------|
| 3/8 (일) | 팀 WIKI 최종 수정 제출 |
| 3/8 경 | PPT 발표 템플릿 공유 (CloudBro → 팀) |
| 3월 중 | 오프라인 미팅 (v1.0.0 릴리스 일정 확정) |
| 3월 중순 | v1.0.0 내부 릴리스 목표 |
| 3월 둘째주~ | 행사 참여자 모집 시작 |
| 4/1 (수) | 최종 발표 자료 제출 |
| 4/4 (토) | 최종 발표 (디캠프 마포) |

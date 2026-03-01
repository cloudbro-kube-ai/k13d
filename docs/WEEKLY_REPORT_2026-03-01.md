# k13d 주간 회의록 (2026-02-23 ~ 2026-03-01)

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

## 4. 다음 주 계획 / 논의 사항

- [ ] `app.js` (약 10,900줄) 멀티 파일 분리 검토 — 유지보수성 개선
- [ ] v0.9.5 릴리스 태그 최종 확정 및 배포
- [ ] Dependabot 보안 취약점 1건 확인 필요 (GitHub 알림)

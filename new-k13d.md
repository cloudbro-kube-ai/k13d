# k13d 기능 개선 사항 (feature/chkwak vs main)

> **기간**: 2024년 ~ 2026년 6월  
> **커밋 수**: 23개 커밋  
> **변경 파일**: 68개 파일 (추가 10,260줄, 삭제 1,694줄)

---

## 1. Web UI 주요 기능

### 1.1 Cluster Visualizer (클러스터 시각화)
- **애니메이션 트래픽 시각화**: Pod 간 통신을 실시간 애니메이션으로 표시
- **CSS 애니메이션**: `views/cluster-visualizer.css` (873줄)
- **JS 구현**: `features/custom-views/cluster-visualizer.js` (527줄)

### 1.2 아이콘 시스템 교체
- 기존 아이콘 → **Lucide 아이콘** 전체 교체
- `icons.js` (180줄), `lucide.js` 추가
- 상단 헤더 버튼을 **아이콘만 표시**로 변경

### 1.3 사이드바 디자인 개선
- **TailAdmin 스타일** 적용
- `layout.css` 대폭 개선 (402줄 변경)
- `components.css`, `views.css` 스타일 업데이트

### 1.4 다크 모드 지원
- **다크 모드 토글 버튼** 헤더에 추가
- Sun/Moon 아이콘으로 현재 모드 표시
- Ollama 미니멀리스트 테마 추가 (`theme-light.css` 94줄)

### 1.5 한글 i18n 지원
- 전체 UI 한글 번역 적용
- 언어 설정을 **영어/한국어만**으로 제한
- `i18n.js` 대폭 개선 (666줄 변경)

### 1.6 로그인 화면 개선
- 브랜딩 정리
- 미니멀리스트 디자인 적용

---

## 2. CLI 기능 (신규)

### 2.1 CLI REPL 구현
- `pkg/cli/` 패키지 신규 생성 (총 1,294줄)
  - `repl.go` - 메인 REPL 루프 (227줄)
  - `tui_repl.go` - TUI 기반 REPL (469줄)
  - `input.go` - 입력 처리 (201줄)
  - `output.go` - 출력 포맷팅 (156줄)
  - `history.go` - 히스토리 관리 (100줄)
  - `splash.go` - 스플래시 스크린 (154줄)

### 2.2 한글 입력 지원
- readLine에서 **한국어(UTF-8) 입력** 지원
- `b5ead6b3` 커밋으로 수정

### 2.3 CLI 로고 변경
- 새로운 ASCII 아트 로고 적용

---

## 3. TUI 개선

### 3.1 화면 깜빡임(Flicker) 개선
- **3단계 개선**: 
  1. 화면 깜빡임 최소화 및 렌더링 안정성 향상
  2. 스마트 레이아웃 재구성으로 추가 개선
  3. navigateTo에서 requestSync 제거로 이중 표시 문제 해결

### 3.2 Diff 렌더링
- `pkg/ui/render_diff.go` (311줄) - Diff 렌더링 구현
- `pkg/ui/render_diff_test.go` (220줄) - 테스트 코드

### 3.3 웰컴 스크린 (신규)
- `pkg/ui/welcome.go` (360줄)
- 시작 시 로고 및 태그라인 표시
- 메뉴 기반 네비게이션

---

## 4. 설정 시스템 개선

### 4.1 설정 탭 개선
- 사용자 뱃지 클릭 시 **Settings/Admin 탭** 자동 열기
- `handlers_settings.go` (147줄) 신규 API 엔드포인트

### 4.2 스타일 설정 시스템
- `pkg/config/styles.go` (71줄) - 스타일 설정 구조
- `context-skins.yaml` (118줄) - 컨텍스트별 스킨 예제

### 4.3 테마 시스템
- 테마 관련 설정 UI 개선
- `docs-site/docs/user-guide/themes.md` (258줄) 문서화

---

## 5. MCP (Model Context Protocol) 개선

### 5.1 프로필 시스템
- `pkg/mcp/profiles.go` (283줄) - MCP 프로필 관리
- `pkg/mcp/profiles_test.go` (254줄) - 테스트 코드

### 5.2 MCP 설정 문서
- `deploy/mcp-examples/` 디렉토리 신규
  - `README.md` (366줄)
  - `kubernetes-mcp-config.yaml` (127줄)
  - `kubernetes-mcp-setup.md` (468줄)
  - `profiles-quickstart.md` (338줄)

---

## 6. 기타 개선

### 6.1 Chat History 개선
- 새 세션 생성 시 자동으로 히스토리 패널 닫힘
- AI 토글 버튼 정리

### 6.2 Namespace 체크박스 수정
- AI 패널 토글 버튼과 함께 namespace 체크박스 동작 개선

### 6.3 Provider 추가
- OpenRouter 프로바이더 추가
- CLI 빌드 충돌 해결

### 6.4 문서화
- `docs-site/docs/concepts/cli-mode.md` (356줄) - CLI 모드 문서
- `docs-site/docs/user-guide/themes.md` (258줄) - 테마 가이드
- `README.md` 업데이트 (85줄 변경)

---

## 주요 변경 파일 목록

| 카테고리 | 파일 | 변경량 |
|---------|------|--------|
| **Web UI** | `pkg/web/static/index.html` | 1,322줄 |
| **Web UI** | `pkg/web/static/js/app.js` | 271줄 |
| **Web UI** | `pkg/web/static/js/modules/i18n.js` | 666줄 |
| **Web UI** | `pkg/web/static/css/layout.css` | 402줄 |
| **Cluster Viz** | `pkg/web/static/css/views/cluster-visualizer.css` | 873줄 |
| **Cluster Viz** | `pkg/web/static/js/features/custom-views/cluster-visualizer.js` | 527줄 |
| **CLI** | `pkg/cli/tui_repl.go` | 469줄 |
| **CLI** | `pkg/cli/repl.go` | 227줄 |
| **TUI** | `pkg/ui/welcome.go` | 360줄 |
| **TUI** | `pkg/ui/render_diff.go` | 311줄 |
| **MCP** | `pkg/mcp/profiles.go` | 283줄 |

---

## 커밋 목록 (시간순)

| 해시 | 타입 | 설명 |
|------|------|------|
| `7852ea18` | feat | Ollama 미니멀리스트 테마 추가 |
| `1e66536e` | fix | TUI 화면 깜빡임 개선 |
| `353d4f3e` | fix | 스마트 레이아웃 재구성 |
| `ad4d83a3` | fix | navigateTo 이중 표시 문제 해결 |
| `5aa52bfc` | feat | Cluster Visualizer 추가 |
| `21abcb93` | feat | CLI 기능 추가 |
| `b5ead6b3` | fix | CLI 한글 입력 지원 |
| `dfcc6d24` | feat | CLI 로고 변경 |
| `44307b46` | fix | OpenRouter 프로바이더 추가 |
| `b1cd5d77` | feat | AI 패널 토글 버튼 추가 |
| `68f7e909` | feat | Diff 렌더링 구현 |
| `d2e99572` | feat | 사용자 뱃지 → Settings 연결 |
| `faea78c8` | fix | MCP 패키지명 및 safety analyzer 수정 |
| `a2857842` | feat | Lucide 아이콘 전체 교체 |
| `9d6619a0` | feat | 언어 설정 제한 (영어/한국어) |
| `0fb153d9` | fix | 언어 설정 옵션 텍스트 수정 |
| `ef4acd3e` | feat | Chat History 자동 닫힘 |
| `301d7937` | feat | 로그인 화면 브랜딩 |
| `a8c730fc` | feat | 헤더 버튼 아이콘만 표시 |
| `d56e878e` | feat | 사이드바 TailAdmin 스타일 |
| `e92ecf48` | feat | 한글 i18n 전체 적용 |
| `9843e2c9` | feat | 다크 모드 토글 + 웰컴 스크린 |

---

## 통계 요약

- **Web UI 개선**: 12개 커밋
- **CLI 신규**: 4개 커밋  
- **TUI 개선**: 4개 커밋
- **버그 수정**: 5개 커밋
- **총 코드 변경**: +10,260줄 / -1,694줄 = **net +8,566줄**

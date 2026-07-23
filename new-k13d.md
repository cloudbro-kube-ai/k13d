# k13d 기능 개선 사항 (feature/chkwak vs main)

> **기간**: 2024년 ~ 2026년 7월
> **커밋 수**: 57개 커밋
> **변경 파일**: 89개 파일 (추가 16,377줄, 삭제 1,725줄)

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
- Decision Required 팝업 아이콘도 Lucide로 통일

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
- `i18n.js` 대폭 개선 (734줄 변경)
- **번역된 주요 항목:**
  - AI 어시스턴트 패널 전체 (환영 메시지, 버튼, 상태 표시)
  - AI 모델 테스트 결과 메시지
  - Decision Required 팝업 (제목, 설명, 버튼)
  - Cluster Overview 헤더

### 1.6 로그인 화면 개선
- 브랜딩 정리
- 미니멀리스트 디자인 적용

### 1.7 AI 어시스턴트 패널 개선
- **세션 지우기 버튼** 추가 (휴지통 아이콘)
- **AI 패널 확장 시 헤더 영역 표시** (top: 50px)
- **도구 실행 정보 세션 간 유지** (sessionStorage 활용)
- **Decision Required 팝업 한글 번역** 및 아이콘 스타일 통일

---

## 2. CLI 기능 (신규)

### 2.1 CLI REPL 구현
- `pkg/cli/` 패키지 신규 생성 (총 1,554줄)
  - `repl.go` - 메인 REPL 루프 (462줄)
  - `tui_repl.go` - TUI 기반 REPL (469줄)
  - `input.go` - 입력 처리 (241줄)
  - `output.go` - 출력 포맷팅 (156줄)
  - `history.go` - 히스토리 관리 (100줄)
  - `splash.go` - 스플래시 스크린 (154줄)
  - `commands.go` - 명령어 처리 (26줄)
  - `version.go` - 버전 정보 (7줄)

### 2.2 한글 입력 지원
- readLine에서 **한국어(UTF-8) 입력** 지원
- `b5ead6b3` 커밋으로 수정

### 2.3 CLI 로고 변경
- 새로운 ASCII 아트 로고 적용

### 2.4 커서 네비게이션 (신규)
- **Left/Right 화살표** 커서 이동 지원
- 다중바이트 UTF-8 문자 안전 처리
- Up/Down 히스토리 복원 시 커서가 끝에 위치하지만 자유롭게 편집 가능

### 2.5 개행 입력 지원 (신규)
- **Ctrl+Enter**: 버퍼에 `\n` 삽입
- **Alt+Enter**: 버퍼에 `\n` 삽입
- `refreshLine()`에서 `\n`을 `↵` 시각적 표시기로 대체
- Enter 제출 시 `\n`이 문자열에 보존되어 AI 질문에 개행 포함 가능

### 2.6 시작 시 프롬프트 및 도움말 (신규)
- `readLine()`이 첫 키 입력 전에 프롬프트를 즉시 표시하여 커서 깜빡임 방지
- CLI 시작 시 입력 영역 위에 도움말 메시지 표시

### 2.7 도구 승인 프롬프트 개선 (신규)
- `(Y/A/n/q)` 프롬프트에 **A(all)** 옵션 추가
- `sessionAutoApprove` bool 필드로 세션 전체 자동 승인 지원
- A를 누르면 이후 모든 도구 호출이 자동 승인됨

### 2.8 K8s 컨텍스트 인식 AI 명령어 (신규)
- `:ai` 명령어에 **Kubernetes 컨텍스트 인식** 기능 추가
- **3단계 처리**:
  1. `gatherClusterContext()` - 네임스페이스/리소스 현황 조회
  2. `parseAIArgs()` + `getResourceDetailedContext()` - 리소스 지정 시 YAML/Events/Logs 제공
  3. `toolApprovalCallback()` - safety.PolicyEnforcer 기반 stdin 승인 프롬프트
- `SupportsTools`에 따른 분기 처리

### 2.9 CLI :model 명령어 (신규)
- `:model` 명령어로 AI 모델 프로필 전환 지원
- `repl.go`에서 `handleModelCommand()` 처리 (48줄 추가)
- `output.go`에 모델 목록 포맷팅 추가

### 2.10 CLI MCP 지원 및 ESC 취소 (신규)
- CLI REPL에서 **MCP 서버 연동** 지원
- **ESC 키**로 현재 입력/작업 취소 기능
- `repl.go` 221줄 대폭 개선, `input.go` 17줄 추가
- ESC 키 처리 로직 및 MCP 툴 호출 연동

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

### 6.5 프레젠테이션 자료 (신규)
- k13d 소개 PPT 제작 (20슬라이드, HTML/PDF/PPTX)
- 프로젝트 개요, 경쟁 비교, 아키텍처, UI/CLI 메뉴 구조 포함
- LLM Provider 비교, CLI 참조 스크린샷 등

---

## 주요 변경 파일 목록

| 카테고리 | 파일 | 변경량 |
|---------|------|--------|
| **Web UI** | `pkg/web/static/index.html` | 1,342줄 |
| **Web UI** | `pkg/web/static/js/app.js` | 354줄 |
| **Web UI** | `pkg/web/static/js/modules/i18n.js` | 734줄 |
| **Web UI** | `pkg/web/static/css/layout.css` | 402줄 |
| **Cluster Viz** | `pkg/web/static/css/views/cluster-visualizer.css` | 873줄 |
| **Cluster Viz** | `pkg/web/static/js/features/custom-views/cluster-visualizer.js` | 527줄 |
| **CLI** | `pkg/cli/tui_repl.go` | 469줄 |
| **CLI** | `pkg/cli/repl.go` | 462줄 |
| **CLI** | `pkg/cli/input.go` | 241줄 |
| **TUI** | `pkg/ui/welcome.go` | 360줄 |
| **TUI** | `pkg/ui/render_diff.go` | 311줄 |
| **MCP** | `pkg/mcp/profiles.go` | 283줄 |

---

## 커밋 목록 (시간순)

| 해시 | 타입 | 설명 |
| `363e00e1` | feat | PPTX 생성 스크립트 및 프레젠테이션 추가 |
| `75f3514e` | docs | 20슬라이드 문서 업데이트 |
| `0c3cf699` | feat | GitHub 저장소 주소 슬라이드 추가 |
| `c57dc5ba` | feat | LLM Provider 장표 추가 |
| `f940c010` | docs | 폰트 크기 조정, 슬라이드 재구성 |
| `1b43ae82` | docs | about 슬라이드 텍스트 간소화 |
| `b43020bf` | docs | 슬라이드 재구성 - feature map 추가 |
| `1de54d50` | docs | CLI 참조 이미지 및 PDF 업데이트 |
| `cd2ce214` | docs | CLI 참조 슬라이드 2x2 그리드 변경 |
| `0d76a153` | docs | CLI 참조 UI 슬라이드 4개 스크린샷 추가 |
| `505b4036` | docs | 폰트 크기 통일 |
| `f9bbb6d2` | docs | 아키텍처 다이어그램 슬라이드 추가 |
| `d64b1b2f` | docs | 슬라이드 3 폰트 확대, Lucide 아이콘 추가 |
| `bc022c30` | docs | 슬라이드 2 이름 유래 및 경쟁 비교 |
| `94c15bf3` | docs | 프로젝트 타임라인 날짜 수정 |
| `ccce1a6a` | docs | Web UI 메뉴 구조 및 CLI 명령어 슬라이드 |
| `b8594296` | docs | presentation PDF 추가 |
| `7d0c73cb` | docs | presentation 스크린샷 추가 |
| `3dfb5b07` | docs | 멀티슬라이드 HTML로 변환 |
| `4678ff09` | feat | CLI MCP 지원 및 ESC 취소 기능 |
| `e5e7e43a` | docs | change log 업데이트 및 presentation 추가 |
| `b3c34964` | feat | CLI :model 명령어로 프로필 전환 |
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
| `6b142abc` | docs | 브랜치 변경사항 정리 및 CLI 실행 스크립트 |
| `d90e9355` | fix | Cluster Overview 한글 번역 누락 수정 |
| `ff848016` | fix | AI 패널 확장 시 헤더 영역 표시 |
| `6c9f3f5f` | feat | AI 어시스턴트 패널 한글 번역 추가 |
| `1271138a` | feat | AI 모델 테스트 결과 메시지 한글 번역 |
| `a9a33e65` | feat | AI 채팅 도구 실행 정보 세션 간 유지 |
| `f7217d8c` | feat | AI 어시스턴트 패널 세션 지우기 버튼 |
| `2b6524b7` | feat | Decision Required 팝업 한글 번역 및 아이콘 스타일 통일 |
| `2c7d4bc5` | feat | :ai 명령어 K8s 컨텍스트 인식 및 도구 호출 지원 |
| `00b0c697` | feat | Ctrl+Enter/Alt+Enter로 개행 입력 지원 |
| `07eed852` | feat | 시작 시 프롬프트 커서 표시 및 도움말 텍스트 추가 |
| `baf49fd3` | feat | 도구 승인 프롬프트에 A(all) 옵션 추가 |
| `c287a0f9` | feat | Left/Right 화살표 커서 이동 지원 |

---

## 통계 요약

- **Web UI 개선**: 14개 커밋
- **CLI 신규**: 11개 커밋  
- **TUI 개선**: 4개 커밋
- **프레젠테이션**: 19개 커밋
- **버그 수정**: 6개 커밋
- **문서화**: 5개 커밋
- **총 코드 변경**: +16,377줄 / -1,725줄 = **net +14,652줄**

---

## 롤백 이력

| 날짜 | 원인 | 롤백 전 | 롤백 후 |
|------|------|---------|---------|
| 2026-06-28 | Decision Required 자동 승인 기능 버그 | `942939d9` | `2b6524b7` |

**삭제된 커밋:**
- `942939d9` debug(web): 승인 자동 승인 디버깅을 위한 로깅 추가
- `028350a2` fix(web): 승인 건너뛰기 시 pendingApproval 미설정 문제 수정
- `17cb154b` feat(web): 설정에서 승인 건너뛰기 초기화 기능 추가
- `4defd35e` feat(web): Decision Required 팝업에 '다음에 표시하지 않기' 기능 추가

# TUI 기능

k13d TUI는 k9s 스타일의 터미널 대시보드이며, 리소스 탐색과 AI Assistant를 한 화면에서 함께 다룹니다.

![TUI Help](../../images/tui_help.png)

## 주요 특징

| 기능 | 설명 |
|------|------|
| Vim 스타일 이동 | `j/k`, `g/G`, `Tab`, `Esc` 등 익숙한 키로 조작 |
| Command Mode | `:pods`, `:svc`, `:deploy`, `:model` 같은 명령 입력 |
| Filter Mode | `/`로 현재 테이블을 빠르게 필터링 |
| Resource Actions | YAML, Describe, Logs, Scale, Restart, Delete |
| AI Panel | 선택 리소스 컨텍스트를 포함한 자연어 질문 가능 |
| Prompt History | AI 입력에서 이전 질문을 화살표로 다시 불러오기 |

## 자주 쓰는 키

| 키 | 동작 |
|----|------|
| `:` | Command 모드 |
| `/` | Filter 모드 |
| `Tab` | 패널 전환 |
| `y` | YAML 보기 |
| `d` | Describe |
| `l` | Logs |
| `Shift+O` | 설정 모달 |
| `:model` | 모델 전환 |

## 안정성 관련 메모

- UI 갱신은 `QueueUpdateDraw()` 기반으로 동작합니다
- 승인 모달, AI 패널 토글, 네임스페이스 전환 같은 흐름은 E2E/회귀 테스트로 고정돼 있습니다

## 함께 보면 좋은 문서

- [TUI Dashboard](../user-guide/tui.md)
- [AI Assistant](ai-assistant.md)
- [모델 설정 및 저장](../ai-llm/model-settings-storage.md)

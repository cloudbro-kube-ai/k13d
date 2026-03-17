# TUI Dashboard

TUI Dashboard는 터미널 안에서 리소스 탐색과 AI Assistant를 함께 쓰는 k13d의 기본 인터페이스입니다.

## 시작하기

```bash
./k13d
```

필요하면 다음처럼 범위를 좁혀 시작할 수 있습니다.

```bash
./k13d -n kube-system
./k13d -A
```

## 핵심 조작

| 키 | 설명 |
|----|------|
| `j/k` | 위/아래 이동 |
| `Tab` | 패널 전환 |
| `:` | Command 모드 |
| `/` | Filter 모드 |
| `Esc` | 닫기 / 뒤로 가기 |
| `Shift+O` | 설정 모달 |

## AI Assistant

- `Tab`으로 AI 패널 포커스 이동
- 이전 프롬프트는 화살표로 다시 불러오기 가능
- 승인 정책은 Web UI와 같은 `config.yaml` 설정을 사용

## 모델 전환

```text
:model
:model gpt-oss-local
```

프로필 전환은 `config.yaml`의 `active_model` 과 활성 `llm` 값을 함께 갱신합니다.

## 함께 보면 좋은 문서

- [TUI 기능](../features/tui.md)
- [모델 설정 및 저장](../ai-llm/model-settings-storage.md)
- [설정](../getting-started/configuration.md)

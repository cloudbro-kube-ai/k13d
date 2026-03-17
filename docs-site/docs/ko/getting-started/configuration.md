# 설정

k13d의 설정은 기본적으로 config 디렉터리 아래 `config.yaml`에 저장됩니다. Web UI와 TUI가 같은 파일을 읽고 같은 파일을 다시 저장합니다.

## `config.yaml` 기본 경로

| 플랫폼 | 기본 경로 |
|--------|-----------|
| Linux | `${XDG_CONFIG_HOME:-~/.config}/k13d/config.yaml` |
| macOS | `~/.config/k13d/config.yaml` |
| Windows | `%AppData%\\k13d\\config.yaml` |

다음 우선순위로 경로를 바꿀 수 있습니다.

1. `--config /path/to/config.yaml`
2. `K13D_CONFIG=/path/to/config.yaml`
3. `XDG_CONFIG_HOME=/custom/config-home`

## 파일이 없을 때

- k13d는 내장 기본값으로 먼저 시작합니다.
- Web UI나 TUI에서 실제로 저장하기 전까지는 파일을 만들지 않습니다.
- 명시적으로 `--config`를 지정한 경로가 비어 있어도 부팅 자체는 정상이어야 합니다.

## 저장되는 주요 항목

```yaml
llm:
  provider: openai
  model: gpt-4o
  endpoint: https://api.openai.com/v1

models:
  - name: gpt-4o
    provider: openai
    model: gpt-4o

active_model: gpt-4o
language: ko
beginner_mode: true
enable_audit: true
```

- `llm`: 지금 실제로 사용하는 활성 LLM 연결 정보
- `models`: 저장된 모델 프로필 목록
- `active_model`: 현재 선택된 프로필 이름

권장 방식은 `api_key`를 파일에 평문으로 넣지 않고 환경 변수 placeholder를 쓰는 것입니다.

```yaml
llm:
  provider: anthropic
  model: claude-sonnet-4-6
  endpoint: https://api.anthropic.com
  api_key: ${ANTHROPIC_API_KEY}
```

```bash
export ANTHROPIC_API_KEY=sk-ant-...
./k13d --web --auth-mode local
```

Anthropic 모델 이름은 길고 자주 바뀔 수 있으므로, 축약형 별칭보다 정확한 model ID를 그대로 쓰는 편이 안전합니다. 현재 ID가 헷갈리면 Anthropic `GET /v1/models` 응답의 `id`를 확인하세요.

## 실제 적용 파일 확인 방법

Web UI를 시작하면 터미널 로그에 다음 항목이 표시됩니다.

- `Config File`
- `Config Path Source`
- `Env Overrides`
- `LLM Settings`

설정이 기대와 다르게 보이면 이 로그를 먼저 확인하는 것이 가장 빠릅니다.

## 함께 보면 좋은 문서

- [모델 설정 및 저장](../ai-llm/model-settings-storage.md)
- [LLM 프로바이더](../ai-llm/providers.md)
- [Web Dashboard](../user-guide/web.md)
- [TUI Dashboard](../user-guide/tui.md)

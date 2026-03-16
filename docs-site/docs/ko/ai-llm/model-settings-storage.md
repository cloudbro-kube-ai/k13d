# 모델 설정 및 저장

이 문서는 k13d에서 AI 모델 설정이 어디에 저장되는지, **Web UI**와 **TUI**가 그 값을 어떻게 바꾸는지, 그리고 **Save** 또는 프로필 전환 시 파일에 무엇이 기록되는지를 정리합니다.

## 한눈에 보기

- 모델 설정의 단일 source of truth는 `config.yaml` 입니다.
- 기본 경로는 `<XDG config home>/k13d/config.yaml` 입니다.
- `--config /path/to/config.yaml` 또는 `K13D_CONFIG=/path/to/config.yaml` 으로 경로를 바꿀 수 있습니다.
- Web UI와 TUI는 모두 이 YAML 파일을 다시 써서 저장합니다.
- 현재 빌드는 활성 모델 설정의 권위 있는 저장소로 SQLite를 사용하지 않습니다.
- Web UI/TUI에서 저장하거나 프로필을 바꾸면 즉시 AI client가 다시 만들어지며, 재시작은 필요하지 않습니다.

## Source Of Truth

k13d는 기본적으로 플랫폼 XDG config 디렉터리 아래의 파일에서 모델 설정을 읽습니다.

| 플랫폼 | 기본 경로 |
|--------|-----------|
| Linux | `${XDG_CONFIG_HOME:-~/.config}/k13d/config.yaml` |
| macOS | `~/Library/Application Support/k13d/config.yaml` |
| Windows | `%AppData%\\k13d\\config.yaml` |

아래 예시에는 가독성을 위해 Linux 스타일 `~/.config/k13d/...` 경로를 사용합니다. macOS에서는 `~/Library/Application Support/k13d/...` 로 보면 됩니다.

기본 파일은 다음과 같습니다.

```text
~/.config/k13d/config.yaml
```

다음처럼 경로를 바꿀 수도 있습니다.

```bash
k13d --config /path/to/config.yaml
```

또는:

```bash
export K13D_CONFIG=/path/to/config.yaml
```

저장할 때는 상위 디렉터리를 자동으로 만들고, 파일 권한 `0600` 으로 씁니다.

또한 Web UI 모드로 시작하면 터미널에 `Config File`, `Config Path Source`, `Env Overrides`가 함께 출력됩니다. Web UI가 다른 파일을 읽는 것처럼 보일 때는 이 로그를 먼저 확인하는 것이 가장 빠릅니다.

!!! note "SQLite는 현재 활성 설정의 source가 아닙니다"
    k13d는 `web_settings`, `model_profiles` 같은 SQLite 테이블을 만들 수 있지만, 현재 Web UI/TUI의 실제 LLM 설정은 `config.yaml`에서 읽습니다. SQLite 값이 런타임 LLM 설정을 덮어쓰지는 않습니다.

## `config.yaml`에서 중요한 부분

모델 설정에서 핵심은 세 부분입니다.

```yaml title="~/.config/k13d/config.yaml"
llm:
  provider: upstage
  model: solar-pro2
  endpoint: https://api.upstage.ai/v1
  api_key: ${UPSTAGE_API_KEY}
  reasoning_effort: minimal
  max_iterations: 10

models:
  - name: solar-pro2
    provider: upstage
    model: solar-pro2
    endpoint: https://api.upstage.ai/v1
    api_key: ${UPSTAGE_API_KEY}
    description: "Upstage Solar Pro2"

  - name: gpt-oss-local
    provider: ollama
    model: gpt-oss:20b
    endpoint: http://localhost:11434
    description: "Local Ollama"

active_model: solar-pro2
```

### 각 섹션의 의미

| 섹션 | 의미 |
|------|------|
| `llm` | 현재 런타임에서 실제로 사용하는 활성 LLM 연결 정보 |
| `models` | Web UI 또는 `:model` 로 전환할 수 있는 이름 붙은 저장 프로필 목록 |
| `active_model` | 현재 선택된 `models[]` 프로필 이름 |

## 전역 필드와 프로필 필드의 차이

이 부분이 중요합니다.

### `llm`에만 저장되는 전역 필드

다음 값들은 **프로필별 값이 아니라 전역 런타임 설정**입니다.

- `reasoning_effort`
- `use_json_mode`
- `retry_enabled`
- `max_retries`
- `max_backoff`
- `temperature`
- `max_tokens`
- `max_iterations`

즉, 저장된 프로필을 전환해도 이 값들은 별도로 바꾸지 않는 한 `llm` 섹션 값이 계속 유지됩니다.

### `models[]`에 저장되는 필드

저장 프로필에는 다음 값이 들어갑니다.

- `name`
- `provider`
- `model`
- `endpoint`
- `api_key`
- `region`
- `azure_deployment`
- `skip_tls_verify`
- `description`

!!! note "일부 고급 필드는 현재 YAML 직접 수정이 가장 정확합니다"
    현재 Web UI와 TUI는 provider, model, endpoint, API key 같은 공통 필드 중심으로 노출합니다. `region`, `azure_deployment` 같은 고급 provider 필드는 지금은 `config.yaml`을 직접 수정하는 방식이 가장 정확합니다.

## 액션별로 무엇이 바뀌는가

| 액션 | `llm` 변경 | `models[]` 변경 | `active_model` 변경 | 재시작 필요 |
|------|------------|-----------------|---------------------|-------------|
| Web UI: 현재 LLM 설정 저장 | 예 | 활성 프로필이 있으면 sync | 아니오 | 아니오 |
| Web UI: Add Model Profile | 아니오 | 예 | 아니오 | 아니오 |
| Web UI: Use profile | 예 | 아니오 | 예 | 아니오 |
| Web UI: Delete profile | 경우에 따라 | 예 | 경우에 따라 | 아니오 |
| TUI: `Shift+O` 저장 | 예 | 활성 프로필이 있으면 sync | 아니오 | 아니오 |
| TUI: `:model`, `:model <name>` | 예 | 아니오 | 예 | 아니오 |

## Web UI 동작

### 현재 LLM 연결 수정

경로:

```text
Settings -> AI
```

메인 LLM 폼은 현재 활성 연결을 수정합니다.

- provider
- model
- endpoint
- API key
- 지원되는 경우 reasoning effort

**Save Settings** 를 누르면 Web UI는 다음을 호출합니다.

- `PUT /api/settings` : language/timezone 같은 일반 설정
- `PUT /api/settings/llm` : 활성 LLM 연결

그 다음 내부적으로는 다음 순서로 동작합니다.

1. 메모리의 `llm.*` 값을 갱신합니다.
2. `active_model` 이 실제 프로필을 가리키면, 활성 연결의 일부 필드를 그 프로필에도 다시 반영합니다.
3. AI client를 즉시 다시 생성합니다.
4. `config.yaml` 을 다시 저장합니다.

### API key 입력칸이 비어 있어도 기존 키가 유지됩니다

Web UI는 보안상 기존 API key를 폼에 다시 보여주지 않습니다. 그래서 화면을 열면 API key 칸은 비어 있습니다.

이 상태에서 다른 필드만 바꾸고 저장하면:

- 기존 메모리상의 API key는 유지되고
- key가 지워지지 않으며
- 저장 시 그 값이 `config.yaml`에 써질 수 있습니다

특히 원래 값이 `${ENV_VAR}` 또는 `K13D_LLM_*` override에서 왔다면 이 점이 중요합니다.

### 저장 프로필 추가

경로:

```text
Settings -> AI -> Add Model Profile
```

이 동작은 `models[]` 에 새 항목을 추가합니다.

다만 자동으로 하지는 않는 것:

- `llm` 변경
- `active_model` 변경
- 즉시 활성화

추가 후 실제로 쓰려면 **Use** 버튼을 눌러야 합니다.

### 활성 프로필 전환

경로:

```text
Settings -> AI -> Use
```

이 동작은 `PUT /api/models/active` 를 호출하고:

1. `active_model` 을 바꾸고
2. 선택한 프로필 값을 `llm.provider`, `llm.model`, `llm.endpoint`, `llm.api_key` 등으로 복사하고
3. AI client를 다시 만들고
4. `config.yaml` 을 저장합니다

### 프로필 삭제

경로:

```text
Settings -> AI -> Delete
```

이 동작은 `models[]` 에서 항목을 제거하고 파일을 다시 저장합니다.

삭제한 프로필이 활성 프로필이었다면:

- 다른 프로필이 남아 있으면 첫 번째 남은 프로필이 활성화되고
- 더 이상 프로필이 없으면 `active_model` 은 빈 값이 됩니다

마지막 프로필을 지워도 `llm` 섹션 자체는 그대로 남을 수 있으므로, 이름 붙은 프로필이 하나도 없어도 현재 연결 정보는 계속 유지될 수 있습니다.

## TUI 동작

### 현재 LLM 연결 수정

TUI 설정 모달은 다음 키로 엽니다.

```text
Shift+O
```

현재 TUI 설정 화면에서 수정하는 공통 필드는 다음입니다.

- provider
- model
- endpoint
- API key

**Save** 를 누르면:

1. `llm.provider`, `llm.model`, `llm.endpoint`, 필요하면 `llm.api_key` 가 갱신되고
2. `active_model` 이 실제 프로필과 매칭되면 그 프로필에도 같은 값이 반영되고
3. `config.yaml` 이 저장되고
4. AI client가 즉시 다시 만들어집니다

TUI 설정 모달은 저장 프로필의 추가/삭제 기능은 제공하지 않습니다.

### TUI에서 프로필 전환

다음 명령을 사용합니다.

```text
:model
```

또는:

```text
:model <name>
```

TUI는 프로필 목록을 보여주거나 전환하기 전에 디스크에서 `config.yaml` 을 다시 읽습니다. 그래서 Web UI에서 최근에 바꾼 프로필 정보를 비교적 잘 따라옵니다.

전환 시에는:

1. `active_model` 이 바뀌고
2. 선택한 프로필이 `llm` 으로 복사되고
3. 파일이 저장되고
4. AI client가 다시 만들어집니다

## 환경 변수와 placeholder 동작

로드 순서는 다음과 같습니다.

1. `config.yaml` 로드
2. `${OPENAI_API_KEY}` 같은 placeholder 확장
3. `K13D_*` 환경 변수 override 적용

즉 Web UI나 TUI 설정 화면을 열 때쯤에는 이미 메모리 안에는 실제 값이 들어와 있을 수 있습니다.

!!! warning "저장하면 현재 메모리 값을 그대로 직렬화합니다"
    원래 파일에 `${OPENAI_API_KEY}` 를 써두었거나, 실행 시 `K13D_LLM_API_KEY` 같은 override를 줬더라도, Web UI/TUI에서 저장하면 확장된 실제 값이 `config.yaml` 에 기록될 수 있습니다.

파일 안에 계속 `${ENV_VAR}` 형태를 유지하고 싶다면:

- `config.yaml` 을 직접 수정하고
- secret은 `${ENV_VAR}` 로 유지하고
- 실행 후 Web UI/TUI에서 LLM 설정 저장은 되도록 피하는 것이 안전합니다

## 실전 예시

### 예시 1: 프로필 전환

전환 전:

```yaml
llm:
  provider: upstage
  model: solar-pro2
  endpoint: https://api.upstage.ai/v1

models:
  - name: solar-pro2
    provider: upstage
    model: solar-pro2
    endpoint: https://api.upstage.ai/v1

  - name: gpt-oss-local
    provider: ollama
    model: gpt-oss:20b
    endpoint: http://localhost:11434

active_model: solar-pro2
```

`gpt-oss-local` 로 바꾼 뒤:

```yaml
llm:
  provider: ollama
  model: gpt-oss:20b
  endpoint: http://localhost:11434

models:
  - name: solar-pro2
    provider: upstage
    model: solar-pro2
    endpoint: https://api.upstage.ai/v1

  - name: gpt-oss-local
    provider: ollama
    model: gpt-oss:20b
    endpoint: http://localhost:11434

active_model: gpt-oss-local
```

### 예시 2: Settings에서 활성 연결 저장

이미 `active_model: gpt-oss-local` 이 있고, Settings에서 모델을 `gpt-oss:20b` 에서 `gpt-oss:120b` 로 바꾸면 저장 시 다음이 함께 바뀝니다.

- `llm.model`
- `models[]` 안의 `gpt-oss-local` 프로필의 `model`

반대로 이 동작이 새 이름의 프로필을 자동 생성하지는 않습니다.

## 추천 워크플로우

- 자주 쓰는 모델은 `models[]` 에 이름 붙은 프로필로 정리합니다.
- 프로필 카탈로그를 만들 때는 Web UI의 **Add Model Profile** 을 사용합니다.
- 일상적인 전환은 Web UI의 **Use** 나 TUI의 `:model` 을 사용합니다.
- 현재 활성 연결의 세부값 조정은 LLM Settings 폼에서 합니다.
- secret을 환경 변수 placeholder로 유지하고 싶다면 UI 저장보다 수동 YAML 편집을 우선합니다.

## 관련 문서

- [Configuration](../../getting-started/configuration.md)
- [LLM Providers](../../ai-llm/providers.md)
- [Web Dashboard](../../user-guide/web.md)
- [TUI Dashboard](../../user-guide/tui.md)

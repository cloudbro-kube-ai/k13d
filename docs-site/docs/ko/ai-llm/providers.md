# LLM 프로바이더

k13d는 여러 LLM 프로바이더를 지원합니다. 기본 연결 정보는 `config.yaml`의 `llm` 섹션에 저장되고, 전환 가능한 프로필 목록은 `models`에 저장됩니다.

자세한 저장/전환 방식은 [모델 설정 및 저장](model-settings-storage.md) 문서를 참고하세요.

## 지원 프로바이더

| Provider | 특징 | API Key |
|----------|------|---------|
| OpenAI | GPT-4o, o3 계열 | 필요 |
| Anthropic | Claude Sonnet 4.6, Opus 4.6, Haiku 4.5 | 필요 |
| Gemini | Google Gemini 계열 | 필요 |
| Upstage | Solar 계열 | 필요 |
| Ollama | 로컬 실행 | 보통 불필요 |
| Azure OpenAI | Azure 리소스 기반 | 필요 |
| AWS Bedrock | AWS 자격 증명 사용 | 환경에 따라 필요 |

## OpenAI 예시

```yaml
llm:
  provider: openai
  model: gpt-4o
  endpoint: https://api.openai.com/v1
  api_key: ${OPENAI_API_KEY}
```

## Anthropic 예시

```yaml
llm:
  provider: anthropic
  model: claude-sonnet-4-6
  endpoint: https://api.anthropic.com
  api_key: ${ANTHROPIC_API_KEY}
```

Anthropic 모델 ID는 제품명보다 길고 자주 바뀔 수 있습니다. 축약형 이름 대신, Anthropic `GET /v1/models` 응답의 `id` 값을 그대로 사용하는 것이 가장 안전합니다.

2026-03-17 기준으로 확인된 예시는 다음과 같습니다.

- `claude-sonnet-4-6`
- `claude-opus-4-6`
- `claude-opus-4-5-20251101`
- `claude-haiku-4-5-20251001`
- `claude-sonnet-4-5-20250929`

## Ollama 예시

```yaml
llm:
  provider: ollama
  model: gpt-oss:20b
  endpoint: http://localhost:11434
```

중요: k13d AI Assistant는 tool calling에 의존하므로, **Ollama 모델은 tools/function calling 지원 모델**이어야 합니다. 텍스트 응답만 가능한 모델은 연결은 되어도 agentic 기능이 제대로 동작하지 않을 수 있습니다.

## 로컬에서 빠르게 시작하기

```bash
ollama serve
ollama pull gpt-oss:20b
./k13d --web --auth-mode local
```

그다음 Web UI `Settings > AI` 또는 TUI `Shift+O`, `:model`에서 전환하면 됩니다.

보안상 API key는 `config.yaml`에 직접 적기보다 아래처럼 환경 변수로 넣는 방식을 권장합니다.

```bash
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...
```

## 함께 보면 좋은 문서

- [모델 설정 및 저장](model-settings-storage.md)
- [Web Dashboard](../user-guide/web.md)
- [TUI Dashboard](../user-guide/tui.md)

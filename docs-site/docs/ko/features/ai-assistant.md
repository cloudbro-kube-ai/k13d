# AI Assistant 기능

k13d의 AI Assistant는 클러스터 컨텍스트를 바탕으로 질문에 답하고, 필요하면 도구를 실행하는 agentic assistant입니다.

![AI Assistant](../../images/webui-assistant-pannel.png)

## 핵심 동작

| 항목 | 설명 |
|------|------|
| Natural Language | 자연어 질문 지원 |
| Context Awareness | YAML, Events, Logs, 선택 리소스 컨텍스트 활용 |
| Tool Calling | `kubectl` 우선, `bash`는 마지막 수단 |
| Approval Workflow | 실행 전 승인/거부 흐름 제공 |
| Multi-Provider | OpenAI, Ollama, Anthropic, Gemini 등 지원 |

## 승인 정책

- read-only 명령도 기본적으로 승인 모달 대상입니다
- write 명령은 승인 필요
- dangerous 명령은 설정에 따라 차단 또는 승인 요구
- unknown command도 정책에 따라 승인 요구

정책은 Web UI `Settings > AI` 와 TUI `Shift+O`에서 같은 설정을 공유합니다.

## Ollama 사용 시 주의

Ollama 모델은 반드시 **tools/function calling 지원 모델**이어야 합니다. 텍스트 생성만 가능한 모델은 k13d의 agentic 루프와 맞지 않을 수 있습니다.

## 함께 보면 좋은 문서

- [LLM 프로바이더](../ai-llm/providers.md)
- [모델 설정 및 저장](../ai-llm/model-settings-storage.md)
- [Web Dashboard](../user-guide/web.md)
- [TUI Dashboard](../user-guide/tui.md)

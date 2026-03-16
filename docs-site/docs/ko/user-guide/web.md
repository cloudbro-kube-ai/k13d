# Web Dashboard

Web Dashboard는 브라우저 기반의 k13d 운영 화면입니다. 리소스 탐색, 상세 보기, AI Assistant, 설정, 보고서 기능을 한곳에서 사용할 수 있습니다.

## 시작하기

```bash
./k13d --web --auth-mode local
```

브라우저에서 `http://localhost:8080` 또는 지정한 포트로 접속합니다.

## 자주 사용하는 영역

| 영역 | 설명 |
|------|------|
| 상단 바 | Context, Namespace, Settings, Help |
| 사이드바 | 리소스/뷰 전환 |
| 중앙 패널 | 리소스 테이블과 상세 모달 |
| AI 패널 | 질의 입력, 승인 모달, 실행 결과 |

## AI 입력 팁

- `Enter` 로 전송
- `Shift+Enter` 로 줄바꿈
- `ArrowUp` / `ArrowDown` 으로 이전 질문 히스토리 탐색
- 히스토리는 브라우저 localStorage에 저장됩니다

## 설정 저장

Settings에서 바꾼 항목은 `config.yaml` 또는 해당 저장소(localStorage, DB)에 반영됩니다. LLM/provider/model은 `config.yaml`의 `llm`, `models`, `active_model`에 저장됩니다.

## 함께 보면 좋은 문서

- [Web UI 기능](../features/web-ui.md)
- [설정](../getting-started/configuration.md)
- [모델 설정 및 저장](../ai-llm/model-settings-storage.md)

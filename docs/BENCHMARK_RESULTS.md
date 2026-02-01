# k13d AI Model Benchmark Report

**생성일**: 2026-01-23
**테스트 환경**: Open WebUI + Ollama / Upstage Solar API
**테스트 방법**: k8s-ai-bench 기반 지시 따르기 평가

---

## 요약 (Summary)

| Model | Provider | Size | Pass Rate | Avg Response | 추천 용도 |
|-------|----------|------|-----------|--------------|-----------|
| **gemma3:4b** | Ollama | 4.3B | 80% | **2.5s** | 빠른 응답, 경량 환경 |
| **gpt-oss:latest** | Ollama | 20.9B | 80% | **2.3s** | 균형잡힌 성능 |
| **qwen3:8b** | Ollama | 8.2B | 80% | 3.3s | 한국어 지원, 중간 규모 |
| **solar-pro2** | Upstage | - | 80% | 3.3s | 프로덕션 API, 안정성 |
| **gemma3:27b** | Ollama | 27.4B | 80% | 6.9s | 고품질 응답 필요시 |
| **deepseek-r1:32b** | Ollama | 32.8B | 80% | **11.8s** | 복잡한 추론, 느림 |

---

## 테스트 항목

| Test ID | 설명 | 평가 기준 |
|---------|------|----------|
| greeting-korean | 한국어 인사 응답 | "안녕" 포함 여부 |
| kubectl-basic | kubectl 명령어 지식 | "kubectl", "get", "pod" 포함 |
| k8s-concept | K8s 개념 설명 | "컨테이너" 또는 "container" 포함 |
| troubleshoot | 문제 해결 능력 | "logs", "describe" 포함 |
| yaml-generate | YAML 생성 능력 | "apiVersion", "kind", "nginx" 포함 |

---

## 상세 결과

### qwen3:8b (Qwen3 8B)
- **크기**: 8.2B 파라미터
- **평균 응답**: 3.3초
- **특징**: 한국어 지원 우수, Tool Calling 지원

| Test | Result | Time |
|------|--------|------|
| greeting-korean | ✓ | 2.1s |
| kubectl-basic | ✓ | 2.5s |
| k8s-concept | ✗ | 1.1s |
| troubleshoot | ✓ | 8.1s |
| yaml-generate | ✓ | 2.7s |

---

### gemma3:4b (Gemma3 4B)
- **크기**: 4.3B 파라미터
- **평균 응답**: 2.5초 (가장 빠름)
- **특징**: 경량, 빠른 응답, 리소스 효율적

| Test | Result | Time |
|------|--------|------|
| greeting-korean | ✓ | 2.6s |
| kubectl-basic | ✓ | 1.9s |
| k8s-concept | ✗ | 0.6s |
| troubleshoot | ✓ | 4.4s |
| yaml-generate | ✓ | 3.1s |

---

### gemma3:27b (Gemma3 27B)
- **크기**: 27.4B 파라미터
- **평균 응답**: 6.9초
- **특징**: 고품질 응답, 복잡한 쿼리에 적합

| Test | Result | Time |
|------|--------|------|
| greeting-korean | ✓ | 11.9s |
| kubectl-basic | ✓ | 3.4s |
| k8s-concept | ✗ | 0.6s |
| troubleshoot | ✓ | 11.6s |
| yaml-generate | ✓ | 7.2s |

---

### gpt-oss:latest (GPT-OSS 20.9B)
- **크기**: 20.9B 파라미터
- **평균 응답**: 2.3초 (최고 효율)
- **특징**: 빠른 응답과 큰 모델의 균형

| Test | Result | Time |
|------|--------|------|
| greeting-korean | ✓ | 3.1s |
| kubectl-basic | ✓ | 1.0s |
| k8s-concept | ✗ | 1.0s |
| troubleshoot | ✓ | 4.4s |
| yaml-generate | ✓ | 2.0s |

---

### deepseek-r1:32b (DeepSeek R1 32B)
- **크기**: 32.8B 파라미터
- **평균 응답**: 11.8초 (가장 느림)
- **특징**: 추론 능력 강화, 복잡한 문제 해결

| Test | Result | Time |
|------|--------|------|
| greeting-korean | ✓ | 23.9s |
| kubectl-basic | ✓ | 3.6s |
| k8s-concept | ✗ | 6.7s |
| troubleshoot | ✓ | 13.9s |
| yaml-generate | ✓ | 11.0s |

---

### solar-pro2 (Upstage Solar Pro2)
- **Provider**: Upstage API
- **평균 응답**: 3.3초
- **특징**: 클라우드 API, 안정성, 한국어 최적화

| Test | Result | Time |
|------|--------|------|
| greeting-korean | ✓ | 1.4s |
| kubectl-basic | ✓ | 2.6s |
| k8s-concept | ✗ | 0.8s |
| troubleshoot | ✓ | 8.4s |
| yaml-generate | ✓ | 3.1s |

---

## 권장 사항

### 사용 시나리오별 추천

| 시나리오 | 추천 모델 | 이유 |
|---------|----------|------|
| **빠른 응답 필요** | gemma3:4b, gpt-oss | 2~3초 응답 |
| **한국어 중심** | solar-pro2, qwen3:8b | 한국어 최적화 |
| **복잡한 분석** | gemma3:27b, deepseek-r1 | 높은 추론 능력 |
| **리소스 제한** | gemma3:4b | 4.3B, 메모리 효율 |
| **프로덕션 안정성** | solar-pro2 | 클라우드 API, SLA |

### k13d 기본 설정

현재 k13d 기본값:
- **Provider**: solar (Upstage)
- **Model**: solar-pro2
- **Language**: ko (한국어)

로컬 환경 권장:
```yaml
llm:
  provider: ollama
  model: qwen3:8b  # 또는 gemma3:4b
  endpoint: http://localhost:11434
```

---

## 테스트 방법론

### 평가 기준
1. **지시 따르기**: 프롬프트의 요구사항 충족 여부
2. **한국어 응답**: 한국어로 적절히 응답하는지
3. **Kubernetes 지식**: 정확한 명령어/개념 제시
4. **응답 속도**: 사용자 경험에 영향

### 한계점
- k8s-concept 테스트에서 모든 모델이 실패 (키워드 체크 방식의 한계)
- 실제 kubectl 실행 없이 응답 내용만 평가
- 단일 시도 (Pass@1) 기준

---

## 결론

모든 테스트 모델이 **80% Pass Rate**를 달성했으며, 기본적인 Kubernetes 지식과 한국어 응답 능력을 갖추고 있습니다.

- **속도 우선**: `gemma3:4b` 또는 `gpt-oss:latest`
- **품질 우선**: `gemma3:27b` 또는 `deepseek-r1:32b`
- **균형 (권장)**: `solar-pro2` 또는 `qwen3:8b`

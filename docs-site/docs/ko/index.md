# k13d 한국어 가이드

k13d는 Kubernetes 클러스터를 관리하기 위한 올인원 대시보드입니다.
Terminal UI(TUI)와 Web UI를 모두 제공하며, AI Assistant가 내장되어 있습니다.

---

## 30초 만에 시작하기

### 1. 다운로드

[Releases](https://github.com/cloudbro-kube-ai/k13d/releases) 페이지에서 플랫폼에 맞는 바이너리를 다운로드하세요.

```bash
tar xzf k13d_*.tar.gz && chmod +x k13d
```

!!! tip "macOS 사용자"
    Gatekeeper가 실행을 차단하면 다음 명령을 실행하세요:
    ```bash
    xattr -d com.apple.quarantine ./k13d
    ```

### 2. 실행

```bash
# Web UI (권장)
./k13d -web -auth-mode local
# 브라우저에서 http://localhost:8080 접속 (admin / admin)

# TUI
./k13d
```

kubeconfig는 자동으로 감지됩니다. 별도의 설정 없이 바로 사용할 수 있습니다.

---

## Web UI 주요 기능

Web UI는 브라우저에서 Kubernetes 클러스터의 모든 것을 관리할 수 있는 대시보드입니다.

![Web UI Dashboard](../images/webui-full-screen.png)

### Dashboard & Resource 관리

| 기능 | 설명 |
|------|------|
| **Resource Table** | Pods, Deployments, Services 등 모든 리소스를 실시간으로 확인 |
| **Namespace / Context 전환** | 상단 바에서 namespace와 cluster context를 빠르게 전환 |
| **Detail Modal** | 리소스 클릭 시 Overview / YAML / Events 탭으로 상세 정보 확인 |
| **Custom Resource** | CRD 리소스도 동일한 상세 화면 (status badge, conditions, spec 자동 분석) |

### AI Assistant

![AI Assistant](../images/webui-assistant-pannel.png)

AI Assistant는 자연어로 질문하면 kubectl 명령을 직접 실행해주는 agentic AI입니다.

```
사용자: "nginx pod이 왜 crash 하나요?"
AI: Pod의 YAML, Events, Logs를 분석하여 원인을 진단하고 해결 방법을 제시합니다.
```

- 위험한 명령은 실행 전 **승인(Approve/Reject)** 을 요청합니다
- OpenAI, Ollama, Anthropic, Gemini 등 다양한 LLM provider를 지원합니다
- Settings > AI에서 provider와 model을 설정할 수 있습니다

### Topology & Visualization

![Topology View](../images/webui-topology-view-all.png)

| 기능 | 설명 |
|------|------|
| **Topology Graph** | Deployment → ReplicaSet → Pod 관계를 interactive 그래프로 시각화 |
| **Topology Tree** | 리소스 소유 관계를 트리 형태로 표시 |
| **RBAC Viewer** | Subject → Role binding 관계를 시각적으로 확인 |
| **Network Policy Map** | Ingress/Egress 네트워크 정책을 시각화 |
| **Event Timeline** | 클러스터 이벤트를 시간대별로 그룹화하여 표시 |

### Reports & Metrics

![Cluster Report](../images/web_ui_cluster_report_preview.png)

| 기능 | 설명 |
|------|------|
| **Cluster Report** | 노드 상태, 워크로드, 보안, FinOps 비용 분석 등 선택적 섹션 구성 |
| **Metrics Charts** | CPU, Memory, Pod/Node 수를 시계열 차트로 표시 (SQLite 저장) |
| **Collect Now** | 즉시 메트릭 수집 트리거 |

### 기타 기능

- **Helm Manager** — Release 관리, history 확인, rollback
- **Pod Terminal** — xterm.js 기반 브라우저 터미널로 Pod에 접속
- **Log Viewer** — 실시간 로그 스트리밍, ANSI 컬러, 검색, 다운로드
- **Port Forward** — 컨테이너 포트를 로컬로 포워딩
- **Resource Templates** — Nginx, Redis, PostgreSQL 등 원클릭 배포
- **Notifications** — Slack, Discord, Teams, Email 알림
- **5가지 Theme** — Tokyo Night, Production, Staging, Development, Light

---

## TUI 주요 기능

TUI는 k9s 스타일의 터미널 대시보드로, Vim 키바인딩을 지원합니다.

![TUI Help](../images/tui_help.png)

### Navigation

| 키 | 동작 | 키 | 동작 |
|---|---|---|---|
| `j` / `k` | 위/아래 이동 | `g` / `G` | 맨 위/맨 아래 |
| `/` | Filter 모드 | `Esc` | 뒤로 가기 |
| `:` | Command 모드 | `?` | Help 보기 |
| `Tab` | AI Panel 포커스 | `q` | 종료 |

### Resource Actions

| 키 | 동작 | 키 | 동작 |
|---|---|---|---|
| `d` | Describe | `y` | YAML 보기 |
| `l` | Logs | `s` | Shell 접속 |
| `S` | Scale | `R` | Restart |
| `Ctrl+D` | Delete | `e` | Edit |

### Sorting

리소스를 이름, 상태, 최신 순 등 다양한 기준으로 정렬할 수 있습니다.

| 키 | 동작 |
|---|---|
| `Shift+N` | NAME 순 정렬 |
| `Shift+T` | STATUS 순 정렬 |
| `Shift+A` | AGE 순 정렬 (최신/오래된 순) |
| `:sort` | Column 선택 모달 |

같은 키를 다시 누르면 정렬 방향이 토글됩니다 (ascending ↔ descending).

### Autocomplete

![TUI Autocomplete](../images/tui_auto_complete.png)

`:` command 모드에서 타이핑하면 자동완성 드롭다운이 표시됩니다.
`aliases.yaml`에 정의한 커스텀 alias도 자동완성에 포함됩니다.

### AI Assistant

`Tab` 키로 AI Panel에 포커스를 맞추고 자연어로 질문하면 됩니다.

```
"kube-system namespace에서 running 중인 pod 목록 보여줘"
"이 deployment를 3개로 scale 해줘"
"왜 이 pod이 CrashLoopBackOff 인지 분석해줘"
```

---

## AI 설정 (Optional)

Web UI 실행 후 **Settings > AI** 에서 LLM provider를 설정하세요.

```bash
# OpenAI 사용
export OPENAI_API_KEY=sk-...
./k13d -web -auth-mode local

# Ollama 사용 (로컬, 무료)
ollama pull qwen2.5:3b && ollama serve
./k13d -web -auth-mode local
# Settings > AI에서 Provider를 "ollama"로 변경
```

지원하는 LLM providers:

| Provider | 특징 |
|----------|------|
| **OpenAI** | GPT-4, GPT-4o 등 |
| **Ollama** | 로컬 실행, API key 불필요 |
| **Anthropic** | Claude 시리즈 |
| **Gemini** | Google AI |
| **Upstage/Solar** | Solar 모델 |

---

## CLI 옵션

```bash
./k13d                              # TUI 모드
./k13d -web                         # Web UI (port 8080)
./k13d -web -port 3000              # 커스텀 포트
./k13d -web -auth-mode local        # 인증 모드
./k13d -web --no-auth               # 인증 없음 (개발용)
./k13d --kubeconfig ~/.kube/prod    # kubeconfig 경로 지정
./k13d --context prod-cluster       # 특정 context 사용
./k13d --debug                      # 디버그 로깅
```

---

## Docker로 실행

```bash
docker run -d -p 8080:8080 \
  -v ~/.kube/config:/home/k13d/.kube/config:ro \
  cloudbro/k13d:latest \
  -web -auth-mode local
```

---

## 설정 파일

```
~/.config/k13d/
├── config.yaml       # LLM, language, model profiles 설정
├── hotkeys.yaml      # 커스텀 단축키
├── plugins.yaml      # 외부 플러그인
├── aliases.yaml      # 리소스 command alias (예: pp → pods)
└── views.yaml        # 리소스별 기본 정렬 설정
```

---

## 더 알아보기

전체 문서: **[https://cloudbro-kube-ai.github.io/k13d](https://cloudbro-kube-ai.github.io/k13d)**

| 주제 | 링크 |
|------|------|
| Web UI 기능 상세 | [Web UI Features](../features/web-ui.md) |
| TUI 기능 상세 | [TUI Features](../features/tui.md) |
| AI Assistant | [AI Guide](../features/ai-assistant.md) |
| Configuration | [Config Guide](../getting-started/configuration.md) |
| Docker 배포 | [Docker Guide](../deployment/docker.md) |
| Kubernetes 배포 | [K8s Guide](../deployment/kubernetes.md) |

# K13D Architecture Documentation

## Overview

k13d (Kubernetes AI Dashboard CLI)는 TUI와 Web UI를 제공하는 Kubernetes 관리 도구입니다.
k9s의 TUI 경험과 kubectl-ai의 AI 기능을 결합하여 지능적인 클러스터 관리를 제공합니다.

## System Requirements & Dependencies

### Required Dependencies

| 종류 | 필요 여부 | 설명 |
|------|-----------|------|
| **k13d binary** | 필수 | 단일 실행 바이너리로 배포됨 |
| **Kubernetes Cluster** | 필수 | kubeconfig 필요 (~/.kube/config) |
| **LLM Provider** | 선택 | AI 기능 사용 시 필요 (OpenAI, Ollama 등) |
| **SQLite** | 자동 생성 | 감사 로그용 (modernc.org/sqlite - CGO-free) |
| **External RDB** | 불필요 | 외부 DB 없이 동작 |

### 요약: 단일 바이너리로 동작 가능
- **RDB 불필요**: 내장 SQLite (CGO-free) 사용
- **외부 서비스 불필요**: 모든 기능이 바이너리에 포함
- **선택적 LLM**: AI 기능 없이도 기본 K8s 대시보드로 동작

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         k13d Binary                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │   TUI Mode   │    │   Web Mode   │    │  CLI Mode    │       │
│  │   (tview)    │    │   (HTTP)     │    │  (direct)    │       │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘       │
│         │                   │                    │               │
│         └───────────────────┼────────────────────┘               │
│                             │                                    │
│                    ┌────────▼────────┐                          │
│                    │   Shared Core    │                          │
│                    ├──────────────────┤                          │
│                    │ • AI Agent       │                          │
│                    │ • K8s Client     │                          │
│                    │ • Tool Registry  │                          │
│                    │ • Safety Analyzer│                          │
│                    │ • Session Store  │                          │
│                    │ • Audit Logger   │                          │
│                    └────────┬─────────┘                          │
│                             │                                    │
│         ┌───────────────────┼───────────────────┐               │
│         │                   │                   │               │
│  ┌──────▼──────┐   ┌───────▼───────┐   ┌───────▼──────┐        │
│  │ LLM Provider│   │ Kubernetes API│   │ SQLite (Audit)│        │
│  │ (OpenAI,    │   │   (client-go) │   │              │        │
│  │  Ollama, ..)│   │               │   │              │        │
│  └─────────────┘   └───────────────┘   └──────────────┘        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Module Structure

### Entry Point
```
cmd/kube-ai-dashboard-cli/main.go
├── runTUI()      → TUI 모드 시작
└── runWebServer() → Web 서버 시작
```

### Core Packages

| Package | 경로 | 역할 |
|---------|------|------|
| **ui** | `pkg/ui/` | TUI 컴포넌트 (tview 기반) |
| **web** | `pkg/web/` | Web 서버 & API 핸들러 |
| **ai/agent** | `pkg/ai/agent/` | AI 에이전트 상태 머신 |
| **ai/providers** | `pkg/ai/providers/` | LLM 프로바이더 구현 |
| **ai/safety** | `pkg/ai/safety/` | 명령어 안전성 분석 |
| **ai/tools** | `pkg/ai/tools/` | 도구 레지스트리 & 실행 |
| **ai/sessions** | `pkg/ai/sessions/` | 대화 세션 관리 |
| **k8s** | `pkg/k8s/` | Kubernetes 클라이언트 래퍼 |
| **db** | `pkg/db/` | SQLite 감사 로그 |
| **config** | `pkg/config/` | 설정 관리 |
| **i18n** | `pkg/i18n/` | 다국어 지원 |

---

## AI Agent Architecture

### State Machine

```
    ┌─────────┐
    │  Idle   │◄────────────────────────┐
    └────┬────┘                         │
         │ User Message                 │
         ▼                              │
    ┌─────────┐                         │
    │ Running │◄─────────────────┐      │
    └────┬────┘                  │      │
         │ LLM Response          │      │
         ▼                       │      │
    ┌──────────────┐             │      │
    │ToolAnalysis  │             │      │
    └────┬─────────┘             │      │
         │                       │      │
         ├─ Auto-approve ────────┘      │
         │                              │
         ▼                              │
    ┌──────────────────┐                │
    │WaitingForApproval│                │
    └────┬─────────────┘                │
         │                              │
         ├─ Approved ──► Execute ───────┤
         │                              │
         ├─ Rejected ───────────────────┤
         │                              │
         └─ Timeout ────────────────────┤
                                        │
    ┌─────────┐                         │
    │  Done   │─────────────────────────┤
    └─────────┘                         │
                                        │
    ┌─────────┐                         │
    │  Error  │─────────────────────────┘
    └─────────┘
```

### Agent Communication Patterns

```go
// 1. Listener Pattern (k9s style) - 권장
type AgentListener interface {
    AgentTextReceived(text string)
    AgentStreamChunk(chunk string)
    AgentStreamEnd()
    AgentError(err error)
    AgentStateChanged(state State)
    AgentToolCallRequested(tc *ToolCallInfo)
    AgentToolCallCompleted(tc *ToolCallInfo)
    AgentApprovalRequested(choice *ChoiceRequest)
    AgentApprovalTimeout(choiceID string)
}

// 2. Channel Pattern (async)
type Agent struct {
    Input  chan *Message  // UI → Agent
    Output chan *Message  // Agent → UI
}
```

---

## TUI Architecture

### Component Hierarchy

```
App (tview.Application)
├── Header          # 클러스터/네임스페이스 정보
├── Dashboard       # 리소스 테이블
│   └── ResourceView
│       ├── PodView
│       ├── DeploymentView
│       ├── ServiceView
│       └── ... (20+ views)
├── AIPanel         # AI 어시스턴트
│   ├── OutputView  # 응답 스트리밍
│   ├── InputField  # 질문 입력
│   └── StatusBar   # 상태 표시
├── CommandBar      # 명령어 입력 (:pods, /filter)
└── HelpModal       # 도움말

AIPanel implements:
  - agent.AgentListener (이벤트 수신)
  - agent.AgentApprovalHandler (승인 처리)
```

### Key Bindings

| 키 | 기능 |
|----|------|
| `j/k` | 위/아래 이동 |
| `Tab` | 패널 전환 |
| `:` | 명령어 모드 |
| `/` | 필터 검색 |
| `y` | YAML 보기 |
| `l` | 로그 보기 |
| `d` | Describe |
| `L` | AI 분석 |
| `s` | Scale |
| `r` | Restart |
| `Ctrl+D` | Delete |
| `?` | 도움말 |

---

## Web UI Architecture

### HTTP API Endpoints

```
/                           # 정적 페이지 (embedded)
/api/health                 # Health check

# Authentication
/api/auth/login             # 로그인
/api/auth/logout            # 로그아웃

# AI Chat (SSE)
/api/chat/agentic           # SSE 스트리밍 채팅
/api/tool/approve           # 도구 승인/거부

# Kubernetes Resources
/api/k8s/pods               # Pod 목록
/api/k8s/deployments        # Deployment 목록
/api/k8s/services           # Service 목록
/api/k8s/{resource}         # 기타 리소스

# Operations
/api/deployment/scale       # Scale
/api/deployment/restart     # Restart
/api/node/cordon            # Node Cordon
/api/portforward            # Port Forwarding

# Helm
/api/helm/releases          # Helm releases
/api/helm/install           # Install chart

# Monitoring
/api/metrics/pod            # Pod 메트릭
/api/metrics/node           # Node 메트릭
/api/audit                  # 감사 로그
```

### SSE Event Flow

```
Browser                          Server
   │                               │
   │ POST /api/chat/agentic        │
   │──────────────────────────────►│
   │                               │
   │◄── SSE: event: chunk ─────────│
   │◄── SSE: event: chunk ─────────│
   │                               │
   │◄── SSE: event: tool_request ──│ (승인 필요)
   │                               │
   │ POST /api/tool/approve        │
   │──────────────────────────────►│
   │                               │
   │◄── SSE: event: tool_execution │
   │◄── SSE: event: chunk ─────────│
   │◄── SSE: event: stream_end ────│
   │                               │
```

---

## Safety Analysis

### Command Classification

```go
type CommandType int
const (
    CommandTypeUnknown CommandType = iota
    CommandTypeRead        // get, describe, logs
    CommandTypeWrite       // apply, create, patch
    CommandTypeDangerous   // delete, drain, taint
    CommandTypeInteractive // exec, attach, edit
)
```

### Safety Analysis Flow

```
User Command
     │
     ▼
┌─────────────────┐
│  Shell Parser   │  mvdan.cc/sh/v3
│  (AST Parsing)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Safety Analyzer │
└────────┬────────┘
         │
         ├── ReadOnly? ──► Auto-approve (configurable)
         │
         ├── Write? ──► Require approval
         │
         └── Dangerous? ──► Warning + Require approval
                            (delete ns, rm -rf, etc.)
```

---

## Data Storage

### SQLite (Audit Log)

위치: `~/.config/k13d/audit.db`

```sql
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    user TEXT,
    action TEXT,           -- query, approve, reject, execute
    resource TEXT,         -- pod/nginx, deployment/app
    details TEXT,          -- JSON 상세 정보
    llm_request TEXT,      -- LLM 요청 (optional)
    llm_response TEXT      -- LLM 응답 (optional)
);
```

### Session Storage

- **Memory Store**: 기본값, 프로세스 종료 시 삭제
- **Filesystem Store**: `~/.config/k13d/sessions/`

```
sessions/
├── session_20240115_123456.json
├── session_20240115_143020.json
└── ...
```

---

## Configuration

### Config File Location

```
$XDG_CONFIG_HOME/k13d/config.yaml
# 기본값: ~/.config/k13d/config.yaml
```

### Config Structure

```yaml
# LLM 설정
llm:
  provider: openai          # openai, ollama, gemini, bedrock, azopenai
  model: gpt-4
  endpoint: ""              # 커스텀 엔드포인트 (ollama 등)
  api_key: ""               # 또는 환경변수 사용

# 다중 모델 프로필
models:
  - name: gpt-4
    provider: openai
    model: gpt-4
  - name: local-llama
    provider: ollama
    model: llama3.2
    endpoint: http://localhost:11434

# 활성 모델
active_model: gpt-4

# MCP 서버 (선택)
mcp:
  servers:
    - name: filesystem
      command: npx
      args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]

# 기타 설정
language: ko                # en, ko, zh, ja
enable_audit: true
beginner_mode: false
report_path: ~/k13d-reports
```

### Environment Variables

```bash
# LLM API Keys
OPENAI_API_KEY=sk-...
GOOGLE_API_KEY=...
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
AZURE_OPENAI_API_KEY=...

# Kubernetes
KUBECONFIG=~/.kube/config
```

---

## Call Flow Diagrams

### TUI: AI Query Flow

```
1. User types question in AIPanel.InputField
2. AIPanel.onSubmit() called
3. Agent.SendUserMessage(question)
4. Agent.Run() starts agentic loop
5. Agent.callLLM() → Provider.AskWithTools()
6. Provider streams chunks → Agent.emitStreamChunk()
7. Agent notifies listener → AIPanel.AgentStreamChunk()
8. AIPanel.app.QueueUpdateDraw() → UI update
9. If tool call needed:
   a. Agent.emitToolCallRequest()
   b. AIPanel shows approval dialog
   c. User presses Y/N
   d. AIPanel.sendApproval() → Agent.SendApproval()
   e. Agent executes tool
   f. Agent.emitToolCallCompleted()
10. Agent.setState(StateDone)
11. AIPanel shows final response
```

### Web: AI Query Flow

```
1. Browser POSTs to /api/chat/agentic
2. Server creates SSEAgentListener
3. Server.aiClient.AskWithToolsAndExecution()
4. Provider streams → callback → SSE event
5. If tool call:
   a. SSE event: tool_request
   b. Browser shows approval UI
   c. Browser POSTs /api/tool/approve
   d. Server wakes approval goroutine
   e. Tool executed
   f. SSE event: tool_execution
6. SSE event: stream_end
7. Browser displays complete response
```

---

## Function Reference

### Agent (pkg/ai/agent/agent.go)

| 함수 | 설명 |
|------|------|
| `New(cfg)` | 새 Agent 생성 |
| `Run(ctx)` | 에이전트 루프 시작 |
| `Ask(ctx, question)` | 단일 질문 처리 |
| `SendUserMessage(content)` | 사용자 메시지 전송 |
| `SendApproval(approved)` | 승인/거부 전송 |
| `SetListener(l)` | 이벤트 리스너 설정 |
| `SetApprovalHandler(h)` | 승인 핸들러 설정 |
| `State()` | 현재 상태 반환 |
| `StartSession(provider, model)` | 새 세션 시작 |
| `GetMessages()` | 대화 기록 반환 |

### Safety Analyzer (pkg/ai/safety/analyzer.go)

| 함수 | 설명 |
|------|------|
| `NewAnalyzer()` | 분석기 생성 |
| `Analyze(cmd)` | 명령어 분석 → Report |
| `QuickCheck(cmd)` | 빠른 읽기전용/위험 체크 |

### K8s Client (pkg/k8s/client.go)

| 함수 | 설명 |
|------|------|
| `NewClient(kubeconfig)` | 클라이언트 생성 |
| `ListPods(ns)` | Pod 목록 |
| `GetPodLogs(ns, pod, container)` | 로그 조회 |
| `GetPodMetrics(ns, pod)` | 메트릭 조회 |
| `StartPortForward(...)` | 포트포워딩 |

---

## Test Coverage

| 패키지 | 커버리지 |
|--------|----------|
| pkg/ai/agent | 91.6% |
| pkg/ai/safety | 76.7% |
| pkg/ai/sessions | 67.5% |
| pkg/ai | 49.1% |
| pkg/config | 80%+ |

테스트 실행:
```bash
go test -cover ./...
go test -v ./pkg/ai/agent/...
```

---

## Summary

k13d는 **단일 바이너리**로 배포되며, 외부 데이터베이스 없이 동작합니다:

- **SQLite**: 내장 (modernc.org/sqlite - CGO-free)
- **세션**: 메모리 또는 파일시스템
- **설정**: YAML 파일 (~/.config/k13d/)

필요한 것:
1. k13d 바이너리
2. Kubernetes 클러스터 접근 (kubeconfig)
3. (선택) LLM API 키

불필요한 것:
- 외부 RDB (MySQL, PostgreSQL 등)
- Redis 등 캐시 서버
- 별도 백엔드 서비스

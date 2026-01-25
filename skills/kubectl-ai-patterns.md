# kubectl-ai Patterns Skill

> kubectl-ai에서 배울 수 있는 AI 에이전트 아키텍처와 안전한 명령 실행 패턴

## Overview

kubectl-ai는 Kubernetes 운영을 위한 AI 에이전트로, 안전하고 확장 가능한 도구 실행 시스템을 제공합니다.
이 Skill은 kubectl-ai의 핵심 패턴들을 참조하여 kube-ai-dashboard-cli의 AI 기능 개발에 활용합니다.

**참조 경로**: `/kubectl-ai/`

---

## 1. AI Agent Loop 아키텍처

### Pattern: State Machine 기반 Agent Loop

```
Idle → Running → ToolAnalysis → WaitingForInput/Done → Exited
```

### 적용 방법

```go
type AgentState int

const (
    AgentStateIdle AgentState = iota
    AgentStateRunning
    AgentStateToolAnalysis
    AgentStateWaitingForInput
    AgentStateDone
    AgentStateExited
)

type Conversation struct {
    state         AgentState
    Input         chan *Message      // 사용자 입력 채널
    Output        chan *Message      // AI 출력 채널
    llmChat       gollm.Chat
    tools         *Tools
    maxIterations int
}

func (c *Conversation) Run(ctx context.Context) error {
    for c.iterations < c.maxIterations {
        switch c.state {
        case AgentStateIdle:
            // 사용자 입력 대기
            msg := <-c.Input
            c.processInput(msg)
            c.state = AgentStateRunning

        case AgentStateRunning:
            // LLM 호출 및 응답 처리
            response, err := c.llmChat.SendStreaming(ctx, c.messages...)
            if err != nil {
                return err
            }
            c.processResponse(response)

        case AgentStateToolAnalysis:
            // 도구 실행 분석 및 승인 요청
            if c.requiresApproval() {
                c.state = AgentStateWaitingForInput
                c.requestApproval()
            } else {
                c.executeTools(ctx)
                c.state = AgentStateRunning
            }

        case AgentStateWaitingForInput:
            // 사용자 승인 대기
            approval := <-c.Input
            if approval.Approved {
                c.executeTools(ctx)
                c.state = AgentStateRunning
            } else {
                c.state = AgentStateDone
            }

        case AgentStateDone:
            c.state = AgentStateIdle
            return nil
        }
        c.iterations++
    }
    return ErrMaxIterationsReached
}
```

### Channel 기반 I/O

```go
// 비동기 메시지 처리
type Message struct {
    Type    MessageType
    Content string
    ToolCalls []ToolCall
    Choice  *ChoiceRequest
}

type MessageType int

const (
    MessageTypeText MessageType = iota
    MessageTypeToolCall
    MessageTypeToolResult
    MessageTypeChoiceRequest
    MessageTypeChoiceResponse
)

// UI에서 사용
go func() {
    for msg := range agent.Output {
        switch msg.Type {
        case MessageTypeText:
            ui.DisplayText(msg.Content)
        case MessageTypeChoiceRequest:
            ui.ShowChoiceDialog(msg.Choice)
        }
    }
}()
```

### 장점
- 상태 전이가 명확하여 디버깅 용이
- 채널 기반으로 UI와 분리
- MaxIterations로 무한 루프 방지

---

## 2. Tool 정의 및 등록 시스템

### Pattern: Plugin Architecture with Interface

```go
type Tool interface {
    Name() string
    Description() string
    FunctionDefinition() *gollm.FunctionDefinition
    Run(ctx context.Context, args map[string]any) (any, error)
    IsInteractive(args map[string]any) (bool, error)
    CheckModifiesResource(args map[string]any) string  // "yes", "no", "unknown"
}
```

### Built-in Tools 구현

```go
// Kubectl Tool
type KubectlTool struct {
    kubeconfig string
}

func (t *KubectlTool) Name() string { return "kubectl" }

func (t *KubectlTool) FunctionDefinition() *gollm.FunctionDefinition {
    return &gollm.FunctionDefinition{
        Name:        "kubectl",
        Description: "Execute kubectl commands against the Kubernetes cluster",
        Parameters: gollm.Schema{
            Type: gollm.Object,
            Properties: map[string]*gollm.Schema{
                "command": {
                    Type:        gollm.String,
                    Description: "The kubectl command to execute (without 'kubectl' prefix)",
                },
            },
            Required: []string{"command"},
        },
    }
}

func (t *KubectlTool) Run(ctx context.Context, args map[string]any) (any, error) {
    command := args["command"].(string)

    // 인터랙티브 명령 검사
    if interactive, _ := t.IsInteractive(args); interactive {
        return nil, ErrInteractiveCommand
    }

    // 명령 실행
    cmd := exec.CommandContext(ctx, "kubectl", strings.Fields(command)...)
    cmd.Env = append(os.Environ(), "KUBECONFIG="+t.kubeconfig)

    output, err := cmd.CombinedOutput()
    return string(output), err
}

func (t *KubectlTool) CheckModifiesResource(args map[string]any) string {
    command := args["command"].(string)

    readOnlyVerbs := []string{"get", "describe", "logs", "top", "explain"}
    for _, verb := range readOnlyVerbs {
        if strings.HasPrefix(command, verb+" ") {
            return "no"
        }
    }

    modifyVerbs := []string{"apply", "create", "delete", "patch", "scale", "edit"}
    for _, verb := range modifyVerbs {
        if strings.HasPrefix(command, verb+" ") {
            return "yes"
        }
    }

    return "unknown"
}
```

### Tool Registry

```go
type Tools struct {
    tools map[string]Tool
    mx    sync.RWMutex
}

func (t *Tools) Register(tool Tool) {
    t.mx.Lock()
    defer t.mx.Unlock()
    t.tools[tool.Name()] = tool
}

func (t *Tools) Lookup(name string) Tool {
    t.mx.RLock()
    defer t.mx.RUnlock()
    return t.tools[name]
}

func (t *Tools) AllDefinitions() []*gollm.FunctionDefinition {
    t.mx.RLock()
    defer t.mx.RUnlock()

    var defs []*gollm.FunctionDefinition
    for _, tool := range t.tools {
        defs = append(defs, tool.FunctionDefinition())
    }
    return defs
}
```

---

## 3. LLM Provider 추상화 (gollm)

### Pattern: Provider-Agnostic Client

```go
// 인터페이스 정의
type Chat interface {
    Send(ctx context.Context, messages ...*Message) (*Message, error)
    SendStreaming(ctx context.Context, messages ...*Message) ChatResponseIterator
}

type Client interface {
    StartChat(opts ChatOptions) Chat
    ListModels(ctx context.Context) ([]string, error)
}

// Factory 패턴
type FactoryFunc func(ctx context.Context, opts ClientOptions) (Client, error)

var globalRegistry = &Registry{
    providers: make(map[string]FactoryFunc),
}

func RegisterProvider(id string, factory FactoryFunc) {
    globalRegistry.providers[id] = factory
}

func NewClient(ctx context.Context, opts ClientOptions) (Client, error) {
    factory, ok := globalRegistry.providers[opts.Provider]
    if !ok {
        return nil, fmt.Errorf("unknown provider: %s", opts.Provider)
    }
    return factory(ctx, opts)
}
```

### Streaming Iterator

```go
type ChatResponseIterator func(yield func(*ChatResponse, error) bool)

type ChatResponse struct {
    Text         string
    FunctionCall *FunctionCall
    FinishReason string
}

// 사용 예시
for response, err := range chat.SendStreaming(ctx, messages...) {
    if err != nil {
        return err
    }

    if response.Text != "" {
        // 텍스트 스트리밍 출력
        fmt.Print(response.Text)
    }

    if response.FunctionCall != nil {
        // 도구 호출 처리
        toolCalls = append(toolCalls, response.FunctionCall)
    }
}
```

### Retry Decorator

```go
type RetryConfig struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

func WithRetry(chat Chat, config RetryConfig) Chat {
    return &retryChat{
        underlying: chat,
        config:     config,
    }
}

type retryChat struct {
    underlying Chat
    config     RetryConfig
}

func (r *retryChat) Send(ctx context.Context, messages ...*Message) (*Message, error) {
    var lastErr error
    delay := r.config.InitialDelay

    for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
        msg, err := r.underlying.Send(ctx, messages...)
        if err == nil {
            return msg, nil
        }

        if !isRetryable(err) {
            return nil, err
        }

        lastErr = err

        // Exponential backoff with jitter
        jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
        time.Sleep(delay + jitter)

        delay = time.Duration(float64(delay) * r.config.Multiplier)
        if delay > r.config.MaxDelay {
            delay = r.config.MaxDelay
        }
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### 지원 Provider

| Provider | 환경변수 | 모델 예시 |
|----------|----------|-----------|
| OpenAI | `OPENAI_API_KEY` | gpt-4, gpt-4-turbo |
| Gemini | `GEMINI_API_KEY` | gemini-2.5-pro |
| Bedrock | AWS credentials | claude-3-sonnet |
| Ollama | `OLLAMA_HOST` | gemma3:12b |
| Grok | `GROK_API_KEY` | grok-3-beta |
| Azure OpenAI | `AZURE_OPENAI_*` | deployment name |

---

## 4. MCP (Model Context Protocol) 통합

### Pattern: Adapter for External Tools

```go
// MCP Tool을 kubectl-ai Tool로 변환
type MCPTool struct {
    serverName string
    toolInfo   mcp.Tool
    manager    *MCPManager
}

func (t *MCPTool) Name() string {
    return fmt.Sprintf("%s_%s", t.serverName, t.toolInfo.Name)
}

func (t *MCPTool) FunctionDefinition() *gollm.FunctionDefinition {
    return &gollm.FunctionDefinition{
        Name:        t.Name(),
        Description: t.toolInfo.Description,
        Parameters:  convertMCPSchema(t.toolInfo.InputSchema),
    }
}

func (t *MCPTool) Run(ctx context.Context, args map[string]any) (any, error) {
    return t.manager.CallTool(ctx, t.serverName, t.toolInfo.Name, args)
}
```

### MCP Manager

```go
type MCPManager struct {
    servers map[string]*MCPConnection
    mx      sync.RWMutex
}

type MCPConnection struct {
    Name      string
    Status    ConnectionStatus
    Tools     []mcp.Tool
    client    *mcp.Client
}

func (m *MCPManager) RegisterWithToolSystem(ctx context.Context, tools *Tools) error {
    m.mx.RLock()
    defer m.mx.RUnlock()

    for serverName, conn := range m.servers {
        if conn.Status != Connected {
            continue
        }

        for _, toolInfo := range conn.Tools {
            mcpTool := &MCPTool{
                serverName: serverName,
                toolInfo:   toolInfo,
                manager:    m,
            }
            tools.Register(mcpTool)
        }
    }

    return nil
}
```

### MCP 설정

```yaml
# ~/.config/kubectl-ai/mcp.yaml
servers:
  # 로컬 MCP 서버 (stdio)
  - name: sequential-thinking
    command: npx
    args:
      - -y
      - "@modelcontextprotocol/server-sequential-thinking"

  # 원격 MCP 서버 (HTTP)
  - name: cloudflare-documentation
    url: https://docs.mcp.cloudflare.com/mcp

  # 인증이 필요한 서버
  - name: custom-api
    url: https://api.example.com/mcp
    auth:
      type: "bearer"
      token: "${MCP_TOKEN}"
```

---

## 5. 안전한 명령 실행

### Pattern: Defense in Depth

```
Layer 1: Tool Validation (인터랙티브 명령 차단)
    ↓
Layer 2: Permission Check (리소스 수정 여부)
    ↓
Layer 3: User Approval (명시적 승인)
    ↓
Layer 4: Sandbox Execution (격리 환경)
    ↓
Layer 5: Output Sanitization (결과 검증)
```

### Layer 1: 인터랙티브 명령 검사

```go
func IsInteractiveCommand(command string) (bool, error) {
    patterns := []struct {
        pattern string
        reason  string
    }{
        {` exec .* -it`, "interactive exec not supported"},
        {` exec .* -ti`, "interactive exec not supported"},
        {` port-forward `, "port-forwarding not allowed"},
        {` edit `, "interactive edit not supported"},
        {` run .* -it`, "interactive run not supported"},
    }

    for _, p := range patterns {
        matched, _ := regexp.MatchString(p.pattern, command)
        if matched {
            return true, fmt.Errorf(p.reason)
        }
    }
    return false, nil
}
```

### Layer 2: 리소스 수정 검사

```go
type ToolCallAnalysis struct {
    ToolCalls           []*ToolCall
    ModifiesResources   bool
    InteractiveCommands []string
    Errors              []error
}

func (t *Tools) Analyze(calls []*ToolCall) *ToolCallAnalysis {
    analysis := &ToolCallAnalysis{
        ToolCalls: calls,
    }

    for _, call := range calls {
        // 인터랙티브 검사
        if interactive, err := call.tool.IsInteractive(call.arguments); interactive {
            analysis.InteractiveCommands = append(
                analysis.InteractiveCommands,
                call.name,
            )
            analysis.Errors = append(analysis.Errors, err)
        }

        // 리소스 수정 검사
        modifies := call.tool.CheckModifiesResource(call.arguments)
        if modifies == "yes" || modifies == "unknown" {
            analysis.ModifiesResources = true
        }
    }

    return analysis
}
```

### Layer 3: 사용자 승인 요청

```go
type ChoiceRequest struct {
    ID          string
    Title       string
    Description string
    Options     []ChoiceOption
}

type ChoiceOption struct {
    ID          string
    Label       string
    Description string
    Dangerous   bool
}

func (c *Conversation) requestApproval(analysis *ToolCallAnalysis) {
    options := []ChoiceOption{
        {ID: "approve", Label: "Execute", Description: "Proceed with the commands"},
        {ID: "reject", Label: "Cancel", Description: "Cancel the operation"},
    }

    c.Output <- &Message{
        Type: MessageTypeChoiceRequest,
        Choice: &ChoiceRequest{
            ID:          uuid.New().String(),
            Title:       "Permission Required",
            Description: formatToolCallsDescription(analysis.ToolCalls),
            Options:     options,
        },
    }

    c.state = AgentStateWaitingForInput
}
```

### Layer 4: Sandbox 실행

```go
type Executor interface {
    Execute(ctx context.Context, command string) (string, error)
}

// 로컬 실행
type LocalExecutor struct{}

func (e *LocalExecutor) Execute(ctx context.Context, command string) (string, error) {
    cmd := exec.CommandContext(ctx, "sh", "-c", command)
    output, err := cmd.CombinedOutput()
    return string(output), err
}

// Kubernetes Pod 내 실행
type KubernetesSandbox struct {
    kubeconfig string
    namespace  string
    image      string
}

func (s *KubernetesSandbox) Execute(ctx context.Context, command string) (string, error) {
    // 임시 Pod 생성
    pod := s.createSandboxPod(command)

    // Pod 실행 및 결과 수집
    defer s.cleanup(pod)

    return s.waitAndGetLogs(ctx, pod)
}
```

---

## 6. Prompt Engineering

### Pattern: Dual-Mode Prompting

```go
type PromptData struct {
    Tools               []*gollm.FunctionDefinition
    EnableToolUseShim   bool
    SessionIsInteractive bool
}

const systemPromptTemplate = `You are an AI assistant for Kubernetes operations.

{{if .EnableToolUseShim}}
## Tool Use Format (JSON)
When you need to use a tool, respond with a JSON object:
{
    "thought": "your reasoning",
    "action": {
        "name": "tool_name",
        "reason": "why this tool",
        "command": "the command",
        "modifies_resource": "yes/no/unknown"
    }
}
{{end}}

## Available Tools
{{range .Tools}}
- {{.Name}}: {{.Description}}
{{end}}

{{if .SessionIsInteractive}}
## Interactive Mode Guidelines
1. ALWAYS gather cluster state before creating resources
2. Ask clarifying questions when requirements are ambiguous
3. Present options when multiple solutions exist
4. Require explicit confirmation for destructive operations
{{else}}
## Non-Interactive Mode Guidelines
1. Execute autonomously without user prompts
2. Make reasonable assumptions based on context
3. Stop and report on any errors
{{end}}

## Safety Rules
1. Never execute interactive commands (exec -it, port-forward, edit)
2. Always validate commands before execution
3. Resource-modifying operations require approval
4. Use kubectl get/describe for information gathering
`
```

### ReAct 패턴 (Tool Use Shim)

```go
// Tool Use를 지원하지 않는 모델용
type ReActResponse struct {
    Thought string `json:"thought"`
    Action  struct {
        Name             string         `json:"name"`
        Reason           string         `json:"reason"`
        Command          string         `json:"command"`
        ModifiesResource string         `json:"modifies_resource"`
        Arguments        map[string]any `json:"arguments,omitempty"`
    } `json:"action"`
}

func (c *Conversation) parseReActResponse(text string) (*ReActResponse, error) {
    // JSON 블록 추출
    start := strings.Index(text, "{")
    end := strings.LastIndex(text, "}")
    if start == -1 || end == -1 {
        return nil, ErrNoJSONBlock
    }

    jsonStr := text[start : end+1]

    var response ReActResponse
    if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
        return nil, err
    }

    return &response, nil
}
```

---

## 7. Session 관리

### Pattern: Pluggable Store

```go
type ChatMessageStore interface {
    AddChatMessage(record *Message) error
    SetChatMessages(newHistory []*Message) error
    ChatMessages() []*Message
    ClearChatMessages() error
}

// In-Memory Store
type InMemoryChatStore struct {
    messages []*Message
    mx       sync.RWMutex
}

// File System Store
type FileSystemStore struct {
    basePath string
}

func (f *FileSystemStore) SaveSession(session *Session) error {
    path := filepath.Join(f.basePath, session.ID+".json")
    data, err := json.MarshalIndent(session, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}

func (f *FileSystemStore) LoadSession(id string) (*Session, error) {
    path := filepath.Join(f.basePath, id+".json")
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var session Session
    if err := json.Unmarshal(data, &session); err != nil {
        return nil, err
    }

    return &session, nil
}
```

### Session 구조

```go
type Session struct {
    ID           string            `json:"id"`
    CreatedAt    time.Time         `json:"created_at"`
    LastAccessAt time.Time         `json:"last_access_at"`
    Model        string            `json:"model"`
    Provider     string            `json:"provider"`
    Messages     []*Message        `json:"messages"`
    AgentState   AgentState        `json:"agent_state"`
    MCPStatus    map[string]string `json:"mcp_status"`
}
```

---

## Quick Reference

### 핵심 파일 위치

| 패턴 | 파일 |
|------|------|
| Agent Loop | `/kubectl-ai/pkg/agent/conversation.go` |
| Tool System | `/kubectl-ai/pkg/tools/tools.go` |
| Tool Interface | `/kubectl-ai/pkg/tools/interfaces.go` |
| LLM Abstraction | `/kubectl-ai/gollm/interfaces.go` |
| MCP Integration | `/kubectl-ai/pkg/agent/mcp_client.go` |
| System Prompt | `/kubectl-ai/pkg/agent/systemprompt_template_default.txt` |
| Sandbox | `/kubectl-ai/pkg/sandbox/` |
| Session | `/kubectl-ai/pkg/api/models.go` |

### 적용 우선순위

1. **Tool Interface** - AI 도구의 핵심 추상화
2. **Safety Layers** - 명령 실행 안전성
3. **Agent Loop** - 상태 관리
4. **LLM Provider** - 다중 프로바이더 지원
5. **Session Management** - 대화 영속성
6. **MCP Integration** - 확장성

---

## 활용 예시

### kube-ai-dashboard-cli에서의 적용

```go
// pkg/ai/client.go - kubectl-ai 패턴 적용
type AIAssistant struct {
    conversation *Conversation
    tools        *Tools
    ui           UICallback
}

func NewAIAssistant(config Config) *AIAssistant {
    tools := NewTools()

    // Built-in tools 등록
    tools.Register(NewKubectlTool(config.Kubeconfig))
    tools.Register(NewBashTool())

    // MCP tools 등록
    if config.MCPEnabled {
        mcpManager := NewMCPManager(config.MCPConfig)
        mcpManager.RegisterWithToolSystem(context.Background(), tools)
    }

    return &AIAssistant{
        conversation: NewConversation(config.LLM, tools),
        tools:        tools,
    }
}

func (a *AIAssistant) AnalyzeResource(ctx context.Context, yaml string) {
    prompt := fmt.Sprintf(
        "Analyze this Kubernetes resource and identify any issues:\n\n%s",
        yaml,
    )

    a.conversation.Input <- &Message{
        Type:    MessageTypeText,
        Content: prompt,
    }
}
```

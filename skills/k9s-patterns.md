# k9s Patterns Skill

> k9s에서 배울 수 있는 TUI 아키텍처 패턴과 베스트 프랙티스

## Overview

k9s는 Kubernetes CLI 관리 도구의 정석으로, 뛰어난 TUI 아키텍처와 사용자 경험을 제공합니다.
이 Skill은 k9s의 핵심 패턴들을 참조하여 kube-ai-dashboard-cli 개발에 활용합니다.

**참조 경로**: `/k9s/`

---

## 1. MVC-Style 3계층 아키텍처

### Pattern: Model-Render-View 분리

```
internal/model/   → 데이터 관리 및 리스너
internal/render/  → 리소스별 렌더링 로직
internal/view/    → UI 컴포넌트 및 사용자 상호작용
```

### 적용 방법

```go
// Model Layer - 데이터와 리스너 분리
type TableModel struct {
    data      *TableData
    listeners []TableListener
    mx        sync.RWMutex
}

type TableListener interface {
    TableDataChanged(*TableData)
    TableLoadFailed(error)
}

// Render Layer - 리소스별 렌더러
type PodRenderer struct {
    *BaseRenderer
}

func (r *PodRenderer) Render(data interface{}) []Row {
    // Pod 특화 렌더링 로직
}

// View Layer - UI 컴포넌트
type PodView struct {
    *tview.Table
    model    *TableModel
    renderer *PodRenderer
}
```

### 장점
- 각 계층의 독립적 테스트 가능
- 리소스 타입 추가 시 Renderer만 구현
- UI 변경이 데이터 로직에 영향 없음

---

## 2. Observer 패턴 기반 이벤트 시스템

### Pattern: Listener Interface

```go
// 다양한 리스너 인터페이스 정의
type TableListener interface {
    TableNoData(*TableData)
    TableDataChanged(*TableData)
    TableLoadFailed(error)
}

type StyleListener interface {
    StylesChanged(*Styles)
}

type StackListener interface {
    StackPushed(Component)
    StackPopped(Component, Component)
}
```

### 적용 방법

```go
// 리스너 등록 및 알림
type Model struct {
    listeners []Listener
    mx        sync.RWMutex
}

func (m *Model) AddListener(l Listener) {
    m.mx.Lock()
    defer m.mx.Unlock()
    m.listeners = append(m.listeners, l)
}

func (m *Model) notify(data *Data) {
    m.mx.RLock()
    defer m.mx.RUnlock()
    for _, l := range m.listeners {
        l.DataChanged(data)
    }
}
```

### 장점
- 컴포넌트 간 느슨한 결합
- 다중 구독자 지원
- 타입 안전한 이벤트 처리

---

## 3. Stack 기반 네비게이션

### Pattern: Pages + Stack

```go
type Pages struct {
    *tview.Pages
    *model.Stack
}

type Stack struct {
    items     []Component
    listeners []StackListener
}

func (s *Stack) Push(c Component) {
    s.items = append(s.items, c)
    s.notifyPush(c)
}

func (s *Stack) Pop() Component {
    if len(s.items) == 0 {
        return nil
    }
    top := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    s.notifyPop(top, s.Top())
    return top
}
```

### 적용 방법

```go
// 네비게이션 통합
type App struct {
    pages *Pages
}

func (a *App) Navigate(view View) {
    a.pages.Push(view)
    a.pages.SwitchToPage(view.Name())
}

func (a *App) Back() {
    a.pages.Pop()
    if top := a.pages.Top(); top != nil {
        a.pages.SwitchToPage(top.Name())
    }
}
```

### 장점
- 브라우저 같은 뒤로가기 경험
- Breadcrumb 자동 생성
- 뷰 히스토리 관리

---

## 4. Action 시스템

### Pattern: Thread-Safe KeyActions

```go
type KeyAction struct {
    Key         tcell.Key
    Rune        rune
    Description string
    Action      ActionHandler
    Opts        ActionOpts
}

type ActionOpts struct {
    Visible   bool  // 메뉴에 표시
    Shared    bool  // 하위 뷰에서 상속
    Plugin    bool  // 플러그인 여부
    HotKey    bool  // 핫키 여부
    Dangerous bool  // 위험한 작업
}

type KeyActions struct {
    actions map[tcell.Key]*KeyAction
    mx      sync.RWMutex
}
```

### 적용 방법

```go
func (v *PodView) bindKeys() {
    v.actions.Add(KeyActions{
        tcell.KeyEnter: {
            Description: "View YAML",
            Action:      v.viewYAML,
            Opts:        ActionOpts{Visible: true},
        },
        'l': {
            Description: "View Logs",
            Action:      v.viewLogs,
            Opts:        ActionOpts{Visible: true},
        },
        tcell.KeyCtrlD: {
            Description: "Delete",
            Action:      v.delete,
            Opts:        ActionOpts{Visible: true, Dangerous: true},
        },
    })
}
```

### 장점
- 키바인딩의 중앙 관리
- 자동 메뉴 생성
- 위험 작업 표시

---

## 5. Plugin/HotKey 시스템

### Pattern: Scope-Based Plugin

```yaml
# plugins.yaml
plugins:
  stern-logs:
    shortCut: Shift-L
    description: "Stern multi-pod logs"
    scopes:
      - pods
      - deployments
    command: stern
    args:
      - --context
      - $CONTEXT
      - -n
      - $NAMESPACE
      - $NAME
    background: true
    confirm: false
    dangerous: false
```

### 적용 방법

```go
type Plugin struct {
    ShortCut    string   `yaml:"shortCut"`
    Description string   `yaml:"description"`
    Scopes      []string `yaml:"scopes"`
    Command     string   `yaml:"command"`
    Args        []string `yaml:"args"`
    Background  bool     `yaml:"background"`
    Confirm     bool     `yaml:"confirm"`
    Dangerous   bool     `yaml:"dangerous"`
}

func (p *Plugin) Matches(resource string) bool {
    for _, scope := range p.Scopes {
        if scope == resource || scope == "all" {
            return true
        }
    }
    return false
}
```

### 환경 변수

| 변수 | 설명 |
|------|------|
| `$NAMESPACE` | 선택된 리소스의 네임스페이스 |
| `$NAME` | 선택된 리소스 이름 |
| `$CONTEXT` | 현재 컨텍스트 |
| `$CLUSTER` | 현재 클러스터 |
| `$CONTAINER` | 선택된 컨테이너 (해당 시) |
| `$COL-<NAME>` | 테이블 컬럼 값 |

---

## 6. Skin/Theme 시스템

### Pattern: Hierarchical Style Structure

```yaml
# skins/dracula.yaml
k9s:
  body:
    fgColor: "#f8f8f2"
    bgColor: "#282a36"
    logoColor: "#bd93f9"

  frame:
    border:
      fgColor: "#6272a4"
      focusColor: "#ff79c6"
    menu:
      fgColor: "#f8f8f2"
      keyColor: "#8be9fd"
    crumbs:
      fgColor: "#282a36"
      bgColor: "#bd93f9"
    status:
      newColor: "#50fa7b"
      modifyColor: "#ffb86c"
      errorColor: "#ff5555"

  views:
    table:
      fgColor: "#f8f8f2"
      bgColor: "#282a36"
      cursorColor: "#44475a"
      header:
        fgColor: "#bd93f9"
        bgColor: "#282a36"
    logs:
      fgColor: "#f8f8f2"
      bgColor: "#282a36"
```

### 적용 방법

```go
type Styles struct {
    Body   Body
    Frame  Frame
    Views  Views
    Dialog Dialog
}

type Frame struct {
    Border Border
    Menu   Menu
    Crumbs Crumbs
    Status Status
    Title  Title
}

// 컬러 변환
func (c *Color) ToTcell() tcell.Color {
    if strings.HasPrefix(string(*c), "#") {
        return tcell.GetColor(string(*c))
    }
    return tcell.ColorNames[strings.ToLower(string(*c))]
}
```

### 장점
- YAML 기반 커스터마이징
- 런타임 핫리로드
- 컨텍스트별 스킨 지정

---

## 7. XDG 설정 관리

### Pattern: Multi-Level Configuration

```
~/.config/k9s/
├── config.yaml           # 전역 설정
├── skins/               # 스킨 디렉토리
├── plugins.yaml         # 전역 플러그인
├── hotkeys.yaml         # 전역 핫키
├── aliases.yaml         # 전역 별칭
└── views.yaml           # 커스텀 뷰

~/.local/share/k9s/clusters/
└── <cluster>-<context>/
    ├── config.yaml      # 컨텍스트별 설정
    ├── plugins.yaml     # 컨텍스트별 플러그인
    └── hotkeys.yaml     # 컨텍스트별 핫키
```

### 적용 방법

```go
import "github.com/adrg/xdg"

func ConfigDir() string {
    if dir := os.Getenv("K9S_CONFIG_DIR"); dir != "" {
        return dir
    }
    return filepath.Join(xdg.ConfigHome, "k9s")
}

func ContextConfigDir(cluster, context string) string {
    return filepath.Join(
        xdg.DataHome, "k9s", "clusters",
        fmt.Sprintf("%s-%s", cluster, context),
    )
}
```

---

## 8. File Watching & Hot Reload

### Pattern: fsnotify Watcher

```go
func (c *Config) Watch(ctx context.Context) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }

    go func() {
        defer watcher.Close()
        for {
            select {
            case <-ctx.Done():
                return
            case event := <-watcher.Events:
                if event.Op&fsnotify.Chmod != 0 {
                    continue // chmod 이벤트 무시
                }
                c.reload()
                c.notifyListeners()
            case err := <-watcher.Errors:
                log.Error("Watcher error", "error", err)
            }
        }
    }()

    return watcher.Add(c.configPath)
}
```

### 장점
- 재시작 없이 설정 변경 적용
- 스킨 실시간 변경
- 플러그인 핫로드

---

## 9. FishBuff (Auto-Suggestion)

### Pattern: Command Buffer with Suggestions

```go
type FishBuff struct {
    *CmdBuff
    suggestionFn    SuggestionFunc
    suggestions     []string
    suggestionIndex int
}

type SuggestionFunc func(text string) []string

func (f *FishBuff) AutoSuggest() {
    if f.suggestionFn == nil {
        return
    }
    f.suggestions = f.suggestionFn(f.String())
    f.suggestionIndex = 0
}

func (f *FishBuff) NextSuggestion() string {
    if len(f.suggestions) == 0 {
        return ""
    }
    f.suggestionIndex = (f.suggestionIndex + 1) % len(f.suggestions)
    return f.suggestions[f.suggestionIndex]
}
```

### 적용 방법

```go
// 리소스별 자동완성 제공
cmdBuff.SetSuggestionFunc(func(text string) []string {
    resources := []string{"pods", "deployments", "services", "nodes"}
    var matches []string
    for _, r := range resources {
        if strings.HasPrefix(r, text) {
            matches = append(matches, r)
        }
    }
    return matches
})
```

---

## Quick Reference

### 핵심 파일 위치

| 패턴 | 파일 |
|------|------|
| Model Layer | `/k9s/internal/model/` |
| Render Layer | `/k9s/internal/render/` |
| View Layer | `/k9s/internal/view/` |
| Action System | `/k9s/internal/ui/action.go` |
| Plugin Config | `/k9s/internal/config/plugin.go` |
| Style System | `/k9s/internal/config/styles.go` |
| File Paths | `/k9s/internal/config/files.go` |
| Navigation | `/k9s/internal/ui/pages.go` |

### 적용 우선순위

1. **MVC 분리** - 가장 먼저 적용할 아키텍처 패턴
2. **Action 시스템** - 키바인딩 관리의 핵심
3. **Listener 패턴** - 컴포넌트 간 통신
4. **XDG 설정** - 설정 파일 관리
5. **Skin 시스템** - 테마 커스터마이징
6. **Plugin 시스템** - 확장성

---

## 활용 예시

### kube-ai-dashboard-cli에서의 적용

```go
// pkg/ui/resources/pods.go - k9s 패턴 적용
type PodView struct {
    *BaseResourceView
    renderer *PodRenderer
}

func (v *PodView) bindKeys() {
    v.actions.Add(KeyActions{
        'L': {
            Description: "AI Analyze",
            Action:      v.aiAnalyze,
            Opts:        ActionOpts{Visible: true},
        },
        'h': {
            Description: "Explain This",
            Action:      v.explainResource,
            Opts:        ActionOpts{Visible: true},
        },
    })
}
```

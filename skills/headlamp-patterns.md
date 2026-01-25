# Headlamp Patterns Skill

> Headlamp에서 배울 수 있는 Plugin 아키텍처, 멀티 클러스터 지원, i18n 패턴

## Overview

Headlamp은 Kubernetes 웹 대시보드로, 뛰어난 플러그인 시스템과 멀티 클러스터 아키텍처를 제공합니다.
이 Skill은 Headlamp의 핵심 패턴들을 참조하여 kube-ai-dashboard-cli의 확장성 개발에 활용합니다.

**참조 경로**: `/headlamp/`

---

## 1. Plugin System 아키텍처

### Pattern: Registry-Based Plugin API

```typescript
// Frontend Plugin Registration
class Registry {
    registerSidebarEntry(entry: SidebarEntry): void
    registerRoute(route: Route): void
    registerDetailsViewSection(section: DetailsViewSection): void
    registerAppBarAction(action: AppBarAction): void
    registerDetailsViewSectionsProcessor(processor: Processor): void
    registerResourceTableColumnsProcessor(processor: Processor): void
    registerPluginSettings(component: SettingsComponent): void
}

// Plugin 진입점
export function initialize(register: Registry) {
    register.registerSidebarEntry({
        name: 'MyPlugin',
        icon: 'extension',
        url: '/my-plugin',
        parent: null,
    });

    register.registerRoute({
        path: '/my-plugin',
        component: () => <MyPluginPage />,
    });
}
```

### Go 버전 적용

```go
// Plugin Registry
type Registry struct {
    sidebarEntries    []SidebarEntry
    routes            []Route
    detailSections    []DetailsSection
    appBarActions     []AppBarAction
    columnProcessors  []ColumnProcessor
    mx                sync.RWMutex
}

type SidebarEntry struct {
    Name   string
    Icon   string
    URL    string
    Parent string
}

type Route struct {
    Path      string
    Component ViewComponent
}

func (r *Registry) RegisterSidebarEntry(entry SidebarEntry) {
    r.mx.Lock()
    defer r.mx.Unlock()
    r.sidebarEntries = append(r.sidebarEntries, entry)
}

func (r *Registry) RegisterRoute(route Route) {
    r.mx.Lock()
    defer r.mx.Unlock()
    r.routes = append(r.routes, route)
}

// Plugin Interface
type Plugin interface {
    Name() string
    Version() string
    Initialize(registry *Registry) error
}
```

### Priority-Based Loading

```go
type PluginPriority int

const (
    PriorityShipped     PluginPriority = iota  // 기본 제공
    PriorityUser                                // 사용자 설치
    PriorityDevelopment                         // 개발 중
)

type PluginInfo struct {
    Name        string
    Version     string
    Priority    PluginPriority
    Enabled     bool
    OverriddenBy string
    IsLoaded    bool
}

func (m *PluginManager) LoadPlugins() error {
    // Priority 순으로 정렬 (높은 것이 우선)
    sort.Slice(m.plugins, func(i, j int) bool {
        return m.plugins[i].Priority > m.plugins[j].Priority
    })

    loaded := make(map[string]bool)

    for _, p := range m.plugins {
        if !p.Enabled {
            continue
        }

        // 이미 같은 이름의 플러그인이 로드됨
        if loaded[p.Name] {
            p.IsLoaded = false
            continue
        }

        if err := p.Initialize(m.registry); err != nil {
            log.Error("Failed to load plugin", "name", p.Name, "error", err)
            continue
        }

        p.IsLoaded = true
        loaded[p.Name] = true
    }

    return nil
}
```

### 장점
- 플러그인 간 충돌 방지 (Priority 기반)
- 버전 호환성 검사
- 동적 활성화/비활성화

---

## 2. 멀티 클러스터 Proxy 아키텍처

### Pattern: Multiplexer

```go
type ClusterMultiplexer struct {
    clusters map[string]*ClusterConnection
    mx       sync.RWMutex
}

type ClusterConnection struct {
    Name       string
    Config     *rest.Config
    Client     kubernetes.Interface
    Status     ConnectionStatus
}

// URL Pattern: /clusters/{cluster}/api/v1/...
func (m *ClusterMultiplexer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 클러스터 이름 추출
    clusterName := extractClusterName(r.URL.Path)

    // 클러스터 연결 조회
    conn, ok := m.getCluster(clusterName)
    if !ok {
        http.Error(w, "Cluster not found", http.StatusNotFound)
        return
    }

    // 인증 토큰 추출 및 검증
    token := extractBearerToken(r)
    if token == "" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // 요청 프록시
    m.proxyRequest(w, r, conn, token)
}

func (m *ClusterMultiplexer) proxyRequest(
    w http.ResponseWriter,
    r *http.Request,
    conn *ClusterConnection,
    token string,
) {
    // URL 경로에서 클러스터 이름 제거
    targetPath := stripClusterPrefix(r.URL.Path)

    // 새 요청 생성
    proxyReq := r.Clone(r.Context())
    proxyReq.URL.Path = targetPath
    proxyReq.URL.Host = conn.Config.Host
    proxyReq.Header.Set("Authorization", "Bearer "+token)

    // 프록시 실행
    proxy := httputil.NewSingleHostReverseProxy(proxyReq.URL)
    proxy.ServeHTTP(w, proxyReq)
}
```

### Context 관리

```go
type ContextManager struct {
    contexts map[string]*KubeContext
    current  string
    mx       sync.RWMutex
}

type KubeContext struct {
    Name       string
    Cluster    string
    User       string
    Namespace  string
    Config     *rest.Config
}

func (m *ContextManager) SwitchContext(name string) error {
    m.mx.Lock()
    defer m.mx.Unlock()

    ctx, ok := m.contexts[name]
    if !ok {
        return fmt.Errorf("context not found: %s", name)
    }

    m.current = name

    // 리스너에게 알림
    m.notifyContextChanged(ctx)

    return nil
}

func (m *ContextManager) CurrentContext() *KubeContext {
    m.mx.RLock()
    defer m.mx.RUnlock()
    return m.contexts[m.current]
}
```

---

## 3. Response Caching

### Pattern: Authorization-Aware Cache

```go
type K8sCache struct {
    cache    *theine.Cache[string, CachedResponse]
    ttl      time.Duration
}

type CachedResponse struct {
    Data       []byte
    Headers    http.Header
    StatusCode int
    CachedAt   time.Time
    UserID     string
}

// Cache Key 생성
func generateCacheKey(r *http.Request, userID string) string {
    return fmt.Sprintf(
        "%s:%s:%s:%s",
        r.Method,
        r.URL.Path,
        r.URL.RawQuery,
        userID,  // 사용자별 캐시 분리
    )
}

// Middleware
func (c *K8sCache) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // GET 요청만 캐시
        if r.Method != http.MethodGet {
            next.ServeHTTP(w, r)
            return
        }

        userID := getUserID(r)
        key := generateCacheKey(r, userID)

        // 캐시 조회
        if cached, ok := c.cache.Get(key); ok {
            if time.Since(cached.CachedAt) < c.ttl {
                // 권한 검사 (SSAR)
                if c.hasPermission(r.Context(), r.URL.Path, userID) {
                    c.writeFromCache(w, cached)
                    return
                }
            }
        }

        // 캐시 미스: 요청 실행 및 캐시 저장
        recorder := httptest.NewRecorder()
        next.ServeHTTP(recorder, r)

        // 성공 응답만 캐시
        if recorder.Code == http.StatusOK {
            c.cache.Set(key, CachedResponse{
                Data:       recorder.Body.Bytes(),
                Headers:    recorder.Header(),
                StatusCode: recorder.Code,
                CachedAt:   time.Now(),
                UserID:     userID,
            }, c.ttl)
        }

        c.copyResponse(w, recorder)
    })
}

// 권한 검사 (SubjectAccessReview)
func (c *K8sCache) hasPermission(ctx context.Context, path, userID string) bool {
    // URL에서 리소스 정보 추출
    resource, namespace, verb := parseResourceFromPath(path)

    sar := &authv1.SubjectAccessReview{
        Spec: authv1.SubjectAccessReviewSpec{
            User: userID,
            ResourceAttributes: &authv1.ResourceAttributes{
                Namespace: namespace,
                Verb:      verb,
                Resource:  resource,
            },
        },
    }

    result, err := c.authClient.Create(ctx, sar, metav1.CreateOptions{})
    if err != nil {
        return false
    }

    return result.Status.Allowed
}
```

### 장점
- API 서버 부하 감소
- 사용자별 권한 존중
- 자동 무효화

---

## 4. OIDC 인증 패턴

### Pattern: Token Flow with Refresh

```go
type OIDCAuthenticator struct {
    provider      *oidc.Provider
    oauth2Config  *oauth2.Config
    tokenStorage  TokenStorage
}

type TokenStorage interface {
    Store(userID string, token *oauth2.Token) error
    Retrieve(userID string) (*oauth2.Token, error)
    Delete(userID string) error
}

// 로그인 핸들러
func (a *OIDCAuthenticator) HandleLogin(w http.ResponseWriter, r *http.Request) {
    state := generateState()

    // 상태 저장 (CSRF 방지)
    http.SetCookie(w, &http.Cookie{
        Name:     "oauth_state",
        Value:    state,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
    })

    // IdP로 리다이렉트
    url := a.oauth2Config.AuthCodeURL(state)
    http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// 콜백 핸들러
func (a *OIDCAuthenticator) HandleCallback(w http.ResponseWriter, r *http.Request) {
    // 상태 검증
    stateCookie, _ := r.Cookie("oauth_state")
    if r.URL.Query().Get("state") != stateCookie.Value {
        http.Error(w, "Invalid state", http.StatusBadRequest)
        return
    }

    // 토큰 교환
    code := r.URL.Query().Get("code")
    token, err := a.oauth2Config.Exchange(r.Context(), code)
    if err != nil {
        http.Error(w, "Token exchange failed", http.StatusInternalServerError)
        return
    }

    // 사용자 정보 추출
    userInfo, err := a.extractUserInfo(r.Context(), token)
    if err != nil {
        http.Error(w, "Failed to get user info", http.StatusInternalServerError)
        return
    }

    // 토큰 저장
    a.tokenStorage.Store(userInfo.UserID, token)

    // 세션 쿠키 설정
    a.setSessionCookie(w, userInfo.UserID, token.AccessToken)

    http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// 토큰 리프레시
func (a *OIDCAuthenticator) RefreshToken(ctx context.Context, userID string) (*oauth2.Token, error) {
    token, err := a.tokenStorage.Retrieve(userID)
    if err != nil {
        return nil, err
    }

    // 토큰 만료 확인
    if token.Valid() {
        return token, nil
    }

    // 리프레시
    source := a.oauth2Config.TokenSource(ctx, token)
    newToken, err := source.Token()
    if err != nil {
        return nil, err
    }

    // 새 토큰 저장
    a.tokenStorage.Store(userID, newToken)

    return newToken, nil
}
```

### JMESPath 기반 Claim 매핑

```go
type ClaimMapping struct {
    Username string `yaml:"username"`  // JMESPath expression
    Email    string `yaml:"email"`
    Groups   string `yaml:"groups"`
}

func (a *OIDCAuthenticator) extractUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
    // ID 토큰 검증
    rawIDToken := token.Extra("id_token").(string)
    idToken, err := a.provider.Verifier(&oidc.Config{
        ClientID: a.oauth2Config.ClientID,
    }).Verify(ctx, rawIDToken)
    if err != nil {
        return nil, err
    }

    // Claims 추출
    var claims map[string]interface{}
    if err := idToken.Claims(&claims); err != nil {
        return nil, err
    }

    // JMESPath로 필드 추출
    username, _ := jmespath.Search(a.claimMapping.Username, claims)
    email, _ := jmespath.Search(a.claimMapping.Email, claims)
    groups, _ := jmespath.Search(a.claimMapping.Groups, claims)

    return &UserInfo{
        UserID:   username.(string),
        Email:    email.(string),
        Groups:   toStringSlice(groups),
    }, nil
}
```

---

## 5. i18n 시스템

### Pattern: Dual-System (Main + Plugin)

```go
// Main App i18n
type I18n struct {
    bundles   map[string]*i18n.Bundle
    current   string
    listeners []LanguageListener
}

type LanguageListener interface {
    LanguageChanged(lang string)
}

func (i *I18n) LoadBundle(lang string) error {
    bundle := i18n.NewBundle(language.Make(lang))
    bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

    // 번역 파일 로드
    path := filepath.Join("locales", lang, "translation.json")
    if _, err := bundle.LoadMessageFile(path); err != nil {
        return err
    }

    i.bundles[lang] = bundle
    return nil
}

func (i *I18n) SetLanguage(lang string) {
    i.current = lang

    // 리스너 알림
    for _, l := range i.listeners {
        l.LanguageChanged(lang)
    }
}

func (i *I18n) T(key string, args ...interface{}) string {
    bundle := i.bundles[i.current]
    if bundle == nil {
        bundle = i.bundles["en"]  // 폴백
    }

    localizer := i18n.NewLocalizer(bundle)
    msg, _ := localizer.Localize(&i18n.LocalizeConfig{
        MessageID: key,
    })

    if len(args) > 0 {
        return fmt.Sprintf(msg, args...)
    }
    return msg
}
```

### Plugin i18n 격리

```go
// Plugin별 i18n 인스턴스
type PluginI18n struct {
    pluginName string
    bundles    map[string]*i18n.Bundle
    mainI18n   *I18n
}

func NewPluginI18n(pluginName string, mainI18n *I18n) *PluginI18n {
    pi := &PluginI18n{
        pluginName: pluginName,
        bundles:    make(map[string]*i18n.Bundle),
        mainI18n:   mainI18n,
    }

    // Main i18n 언어 변경 구독
    mainI18n.AddListener(pi)

    return pi
}

func (pi *PluginI18n) LanguageChanged(lang string) {
    // Plugin 번역 파일 로드
    path := filepath.Join(
        "plugins", pi.pluginName, "locales", lang, "translation.json",
    )

    bundle := i18n.NewBundle(language.Make(lang))
    if _, err := bundle.LoadMessageFile(path); err != nil {
        // 영어로 폴백
        path = filepath.Join(
            "plugins", pi.pluginName, "locales", "en", "translation.json",
        )
        bundle.LoadMessageFile(path)
    }

    pi.bundles[lang] = bundle
}
```

### 지원 언어

| 코드 | 언어 |
|------|------|
| en | English |
| ko | 한국어 |
| zh | 简体中文 |
| ja | 日本語 |
| es | Español |
| fr | Français |
| de | Deutsch |
| pt | Português |
| it | Italiano |
| hi | हिन्दी |
| ta | தமிழ் |

---

## 6. Processor 패턴 (확장성)

### Pattern: Pipeline Processors

```go
// Column Processor - 테이블 컬럼 수정
type ColumnProcessor interface {
    ProcessColumns(resource string, columns []Column) []Column
}

type Column struct {
    Name      string
    Label     string
    Getter    func(interface{}) string
    Width     int
    Sortable  bool
}

// Detail Section Processor - 상세 보기 섹션 수정
type DetailSectionProcessor interface {
    ProcessSections(resource string, sections []DetailSection) []DetailSection
}

type DetailSection struct {
    Name      string
    Title     string
    Component ViewComponent
    Priority  int
}

// Processor 체인
type ProcessorChain struct {
    columnProcessors  []ColumnProcessor
    sectionProcessors []DetailSectionProcessor
}

func (c *ProcessorChain) ProcessColumns(resource string, columns []Column) []Column {
    result := columns

    for _, p := range c.columnProcessors {
        result = p.ProcessColumns(resource, result)
    }

    return result
}
```

### 플러그인에서 사용

```go
// 커스텀 컬럼 추가 플러그인
type CustomColumnPlugin struct{}

func (p *CustomColumnPlugin) Initialize(registry *Registry) error {
    registry.RegisterColumnProcessor(p)
    return nil
}

func (p *CustomColumnPlugin) ProcessColumns(resource string, columns []Column) []Column {
    if resource != "pods" {
        return columns
    }

    // 커스텀 컬럼 추가
    return append(columns, Column{
        Name:  "custom",
        Label: "Custom Field",
        Getter: func(obj interface{}) string {
            pod := obj.(*corev1.Pod)
            return pod.Annotations["custom-annotation"]
        },
    })
}
```

---

## 7. WebSocket Multiplexing

### Pattern: Single Connection, Multiple Subscriptions

```go
type WSMultiplexer struct {
    conn        *websocket.Conn
    subscribers map[string][]Subscriber
    mx          sync.RWMutex
}

type Subscriber struct {
    ID       string
    Filter   ResourceFilter
    Callback func(Event)
}

type ResourceFilter struct {
    Kind      string
    Namespace string
    Name      string
}

type Event struct {
    Type     string  // ADDED, MODIFIED, DELETED
    Object   interface{}
}

func (m *WSMultiplexer) Subscribe(filter ResourceFilter, callback func(Event)) string {
    m.mx.Lock()
    defer m.mx.Unlock()

    id := uuid.New().String()
    key := m.filterKey(filter)

    m.subscribers[key] = append(m.subscribers[key], Subscriber{
        ID:       id,
        Filter:   filter,
        Callback: callback,
    })

    // 첫 번째 구독자면 Watch 시작
    if len(m.subscribers[key]) == 1 {
        go m.startWatch(filter)
    }

    return id
}

func (m *WSMultiplexer) Unsubscribe(id string) {
    m.mx.Lock()
    defer m.mx.Unlock()

    for key, subs := range m.subscribers {
        for i, sub := range subs {
            if sub.ID == id {
                m.subscribers[key] = append(subs[:i], subs[i+1:]...)

                // 마지막 구독자면 Watch 중지
                if len(m.subscribers[key]) == 0 {
                    m.stopWatch(key)
                    delete(m.subscribers, key)
                }
                return
            }
        }
    }
}

func (m *WSMultiplexer) dispatch(key string, event Event) {
    m.mx.RLock()
    defer m.mx.RUnlock()

    for _, sub := range m.subscribers[key] {
        go sub.Callback(event)
    }
}
```

---

## Quick Reference

### 핵심 파일 위치

| 패턴 | 파일 |
|------|------|
| Plugin Registry | `/headlamp/frontend/src/plugin/` |
| Backend Proxy | `/headlamp/backend/pkg/` |
| K8s Cache | `/headlamp/backend/pkg/k8cache/` |
| Auth | `/headlamp/backend/pkg/auth/` |
| i18n | `/headlamp/frontend/src/i18n/` |
| Plugin Examples | `/headlamp/plugins/examples/` |

### 적용 우선순위

1. **Plugin Registry** - 확장성의 핵심
2. **Multi-Cluster Proxy** - 멀티 클러스터 지원
3. **Response Caching** - 성능 최적화
4. **i18n System** - 다국어 지원
5. **OIDC Auth** - 엔터프라이즈 인증
6. **Processor Pattern** - 플러그인 확장점

---

## 활용 예시

### kube-ai-dashboard-cli에서의 적용

```go
// pkg/plugin/registry.go - Headlamp 패턴 적용
type PluginRegistry struct {
    sidebarEntries []SidebarEntry
    aiTools        []AITool
    resourceViews  []ResourceView
    mx             sync.RWMutex
}

// AI Tool 등록
func (r *PluginRegistry) RegisterAITool(tool AITool) {
    r.mx.Lock()
    defer r.mx.Unlock()
    r.aiTools = append(r.aiTools, tool)
}

// 플러그인 예시
func InitializeClusterAnalyzerPlugin(registry *PluginRegistry) error {
    registry.RegisterAITool(AITool{
        Name:        "cluster-analyzer",
        Description: "Analyze cluster health",
        Handler: func(ctx context.Context, params map[string]any) (string, error) {
            // AI 기반 클러스터 분석
            return analyzeCluster(ctx, params)
        },
    })

    return nil
}
```

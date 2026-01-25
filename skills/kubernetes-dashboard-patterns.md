# Kubernetes Dashboard Patterns Skill

> Kubernetes Dashboard에서 배울 수 있는 API 설계, 데이터 처리, 메트릭 통합 패턴

## Overview

Kubernetes Dashboard는 공식 웹 대시보드로, 깔끔한 모듈 구조와 효율적인 API 설계를 제공합니다.
이 Skill은 Kubernetes Dashboard의 핵심 패턴들을 참조하여 kube-ai-dashboard-cli의 백엔드 설계에 활용합니다.

**참조 경로**: `/dashboard/`

---

## 1. 멀티 모듈 아키텍처

### Pattern: Independent Microservices

```
dashboard/
├── modules/
│   ├── api/           # Kubernetes API 확장 (집계, 정렬, 필터링)
│   ├── auth/          # 인증 처리
│   ├── web/           # 프론트엔드 및 웹 서버
│   ├── metrics-scraper/  # 메트릭 수집 및 저장
│   └── common/        # 공유 패키지
```

### 적용 방법

```go
// 모듈별 독립 main.go
// modules/api/main.go
func main() {
    // 독립적인 설정 및 라우터
    cfg := config.NewAPIConfig()
    router := api.NewRouter(cfg)

    http.ListenAndServe(cfg.Address, router)
}

// modules/auth/main.go
func main() {
    cfg := config.NewAuthConfig()
    router := auth.NewRouter(cfg)

    http.ListenAndServe(cfg.Address, router)
}
```

### 공유 모듈 (common)

```go
// modules/common/client/client.go
package client

// Per-Request Client 생성
func NewClientFromRequest(r *http.Request) (kubernetes.Interface, error) {
    token := extractBearerToken(r)
    if token == "" {
        return nil, ErrUnauthorized
    }

    config := &rest.Config{
        Host:        getAPIServerHost(),
        BearerToken: token,
    }

    return kubernetes.NewForConfig(config)
}

// modules/common/errors/errors.go
package errors

func NewUnauthorized(reason string) *StatusError {
    return &StatusError{
        Code:    http.StatusUnauthorized,
        Message: reason,
    }
}

func NewNotFound(resource, name string) *StatusError {
    return &StatusError{
        Code:    http.StatusNotFound,
        Message: fmt.Sprintf("%s %q not found", resource, name),
    }
}
```

### 장점
- 모듈별 독립 배포 가능
- 장애 격리
- 수평 확장 용이

---

## 2. DataSelector 패턴

### Pattern: Generic Data Processing

```go
// 범용 데이터 셀 인터페이스
type DataCell interface {
    GetProperty(PropertyName) ComparableValue
}

type ComparableValue interface {
    Compare(ComparableValue) int    // 정렬용
    Contains(ComparableValue) bool  // 필터링용
}

type PropertyName string

const (
    NameProperty      PropertyName = "name"
    NamespaceProperty PropertyName = "namespace"
    AgeProperty       PropertyName = "age"
    StatusProperty    PropertyName = "status"
    CPUProperty       PropertyName = "cpu"
    MemoryProperty    PropertyName = "memory"
)
```

### DataSelector 구현

```go
type DataSelector struct {
    GenericDataList []DataCell
    DataSelectQuery *DataSelectQuery
}

type DataSelectQuery struct {
    Filter      *FilterQuery
    Sort        *SortQuery
    Pagination  *PaginationQuery
    Metrics     *MetricsQuery
}

type FilterQuery struct {
    FilterByProperty []FilterBy
}

type FilterBy struct {
    Property PropertyName
    Value    ComparableValue
}

type SortQuery struct {
    SortByList []SortBy
}

type SortBy struct {
    Property  PropertyName
    Ascending bool
}

type PaginationQuery struct {
    ItemsPerPage int
    Page         int
}

// 필터링 적용
func (ds *DataSelector) Filter() *DataSelector {
    if ds.DataSelectQuery.Filter == nil {
        return ds
    }

    var filtered []DataCell
    for _, cell := range ds.GenericDataList {
        matches := true
        for _, fb := range ds.DataSelectQuery.Filter.FilterByProperty {
            value := cell.GetProperty(fb.Property)
            if !value.Contains(fb.Value) {
                matches = false
                break
            }
        }
        if matches {
            filtered = append(filtered, cell)
        }
    }

    ds.GenericDataList = filtered
    return ds
}

// 정렬 적용
func (ds *DataSelector) Sort() *DataSelector {
    if ds.DataSelectQuery.Sort == nil {
        return ds
    }

    sort.Slice(ds.GenericDataList, func(i, j int) bool {
        for _, sb := range ds.DataSelectQuery.Sort.SortByList {
            a := ds.GenericDataList[i].GetProperty(sb.Property)
            b := ds.GenericDataList[j].GetProperty(sb.Property)

            cmp := a.Compare(b)
            if cmp != 0 {
                if sb.Ascending {
                    return cmp < 0
                }
                return cmp > 0
            }
        }
        return false
    })

    return ds
}

// 페이지네이션 적용
func (ds *DataSelector) Paginate() *DataSelector {
    if ds.DataSelectQuery.Pagination == nil {
        return ds
    }

    pq := ds.DataSelectQuery.Pagination
    start := pq.ItemsPerPage * pq.Page
    end := start + pq.ItemsPerPage

    if start > len(ds.GenericDataList) {
        ds.GenericDataList = []DataCell{}
        return ds
    }

    if end > len(ds.GenericDataList) {
        end = len(ds.GenericDataList)
    }

    ds.GenericDataList = ds.GenericDataList[start:end]
    return ds
}
```

### 리소스별 DataCell 구현

```go
// Pod DataCell
type PodCell struct {
    *corev1.Pod
}

func (c *PodCell) GetProperty(name PropertyName) ComparableValue {
    switch name {
    case NameProperty:
        return StdComparableString(c.Name)
    case NamespaceProperty:
        return StdComparableString(c.Namespace)
    case AgeProperty:
        return StdComparableTime(c.CreationTimestamp.Time)
    case StatusProperty:
        return StdComparableString(string(c.Status.Phase))
    case CPUProperty:
        return StdComparableInt(c.getCPUUsage())
    case MemoryProperty:
        return StdComparableInt(c.getMemoryUsage())
    default:
        return nil
    }
}

// Comparable 타입들
type StdComparableString string

func (s StdComparableString) Compare(other ComparableValue) int {
    return strings.Compare(string(s), string(other.(StdComparableString)))
}

func (s StdComparableString) Contains(other ComparableValue) bool {
    return strings.Contains(
        strings.ToLower(string(s)),
        strings.ToLower(string(other.(StdComparableString))),
    )
}

type StdComparableTime time.Time

func (t StdComparableTime) Compare(other ComparableValue) int {
    a := time.Time(t)
    b := time.Time(other.(StdComparableTime))
    return a.Compare(b)
}
```

### 사용 예시

```go
// API 핸들러에서 사용
func (h *PodHandler) List(w http.ResponseWriter, r *http.Request) {
    // 쿼리 파라미터 파싱
    query := parseDataSelectQuery(r)

    // Pod 목록 조회
    pods, err := h.client.CoreV1().Pods("").List(r.Context(), metav1.ListOptions{})
    if err != nil {
        writeError(w, err)
        return
    }

    // DataCell로 변환
    cells := make([]DataCell, len(pods.Items))
    for i, pod := range pods.Items {
        cells[i] = &PodCell{Pod: &pod}
    }

    // DataSelector 적용
    result := &DataSelector{
        GenericDataList: cells,
        DataSelectQuery: query,
    }

    result.Filter().Sort().Paginate()

    // 응답
    writeJSON(w, result.ToResponse())
}
```

### 장점
- 모든 리소스에 동일한 필터/정렬/페이지네이션
- 타입 안전한 비교 연산
- 메트릭 통합 지원

---

## 3. Init 기반 라우트 등록

### Pattern: Decentralized Route Registration

```go
// 중앙 라우터
// modules/api/pkg/router/router.go
package router

var routes []Route

type Route struct {
    Method  string
    Path    string
    Handler http.HandlerFunc
}

func Register(method, path string, handler http.HandlerFunc) {
    routes = append(routes, Route{
        Method:  method,
        Path:    path,
        Handler: handler,
    })
}

func NewRouter() *mux.Router {
    r := mux.NewRouter()

    for _, route := range routes {
        r.HandleFunc(route.Path, route.Handler).Methods(route.Method)
    }

    return r
}
```

### 각 기능 모듈에서 등록

```go
// modules/api/pkg/handler/pod/handler.go
package pod

import "router"

func init() {
    router.Register("GET", "/api/v1/pods", handleList)
    router.Register("GET", "/api/v1/namespaces/{namespace}/pods", handleListNamespaced)
    router.Register("GET", "/api/v1/namespaces/{namespace}/pods/{name}", handleGet)
    router.Register("DELETE", "/api/v1/namespaces/{namespace}/pods/{name}", handleDelete)
}

func handleList(w http.ResponseWriter, r *http.Request) {
    // 구현
}

// modules/api/pkg/handler/deployment/handler.go
package deployment

func init() {
    router.Register("GET", "/api/v1/deployments", handleList)
    router.Register("POST", "/api/v1/namespaces/{namespace}/deployments/{name}/scale", handleScale)
}
```

### Main에서 import

```go
// modules/api/main.go
package main

import (
    "router"

    // blank import로 init() 실행
    _ "handler/pod"
    _ "handler/deployment"
    _ "handler/service"
    _ "handler/node"
)

func main() {
    r := router.NewRouter()
    http.ListenAndServe(":8080", r)
}
```

### 장점
- 기능별 라우트 분리
- 중앙 라우팅 설정 불필요
- 모듈 추가 시 자동 등록

---

## 4. Request-Scoped Client

### Pattern: Per-Request Kubernetes Client

```go
// 요청별 클라이언트 생성
type ClientFactory struct {
    baseConfig *rest.Config
}

func (f *ClientFactory) FromRequest(r *http.Request) (kubernetes.Interface, error) {
    // 토큰 추출
    token := extractBearerToken(r)
    if token == "" {
        return nil, ErrUnauthorized
    }

    // 요청자 권한으로 클라이언트 생성
    config := rest.CopyConfig(f.baseConfig)
    config.BearerToken = token

    return kubernetes.NewForConfig(config)
}

// 미들웨어로 주입
func WithClient(factory *ClientFactory) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            client, err := factory.FromRequest(r)
            if err != nil {
                writeError(w, err)
                return
            }

            ctx := context.WithValue(r.Context(), clientKey, client)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// 핸들러에서 사용
func handleListPods(w http.ResponseWriter, r *http.Request) {
    client := r.Context().Value(clientKey).(kubernetes.Interface)

    pods, err := client.CoreV1().Pods("").List(r.Context(), metav1.ListOptions{})
    if err != nil {
        writeError(w, err)
        return
    }

    writeJSON(w, pods)
}
```

### 장점
- 요청자 권한 존중 (RBAC)
- 권한 검사 투명화
- 보안 강화

---

## 5. 메트릭 통합 아키텍처

### Pattern: Sidecar Metrics Scraper

```go
// Metrics Scraper 모듈
// modules/metrics-scraper/main.go

type MetricsScraper struct {
    db       *sql.DB
    client   *metricsv1beta1.MetricsV1beta1Client
    interval time.Duration
}

func (s *MetricsScraper) Start(ctx context.Context) error {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            s.scrape(ctx)
        }
    }
}

func (s *MetricsScraper) scrape(ctx context.Context) {
    // Node 메트릭 수집
    nodeMetrics, _ := s.client.NodeMetricses().List(ctx, metav1.ListOptions{})
    for _, nm := range nodeMetrics.Items {
        s.storeNodeMetric(nm)
    }

    // Pod 메트릭 수집
    podMetrics, _ := s.client.PodMetricses("").List(ctx, metav1.ListOptions{})
    for _, pm := range podMetrics.Items {
        s.storePodMetric(pm)
    }

    // 오래된 데이터 정리
    s.cleanup()
}

// SQLite 저장
func (s *MetricsScraper) storeNodeMetric(nm metricsv1beta1.NodeMetrics) {
    query := `
        INSERT INTO node_metrics (name, timestamp, cpu_usage, memory_usage)
        VALUES (?, ?, ?, ?)
    `
    s.db.Exec(query,
        nm.Name,
        nm.Timestamp.Time,
        nm.Usage.Cpu().MilliValue(),
        nm.Usage.Memory().Value(),
    )
}

// 데이터 정리 (보존 기간 초과)
func (s *MetricsScraper) cleanup() {
    cutoff := time.Now().Add(-s.retentionPeriod)
    s.db.Exec("DELETE FROM node_metrics WHERE timestamp < ?", cutoff)
    s.db.Exec("DELETE FROM pod_metrics WHERE timestamp < ?", cutoff)
}
```

### Metrics API

```go
// API 모듈에서 메트릭 제공
type MetricsHandler struct {
    scraperClient *http.Client
    scraperHost   string
}

func (h *MetricsHandler) GetPodMetrics(w http.ResponseWriter, r *http.Request) {
    namespace := mux.Vars(r)["namespace"]
    name := mux.Vars(r)["name"]

    // Sidecar에서 메트릭 조회
    url := fmt.Sprintf("%s/api/v1/pods/%s/%s/metrics", h.scraperHost, namespace, name)
    resp, err := h.scraperClient.Get(url)
    if err != nil {
        writeError(w, err)
        return
    }
    defer resp.Body.Close()

    io.Copy(w, resp.Body)
}
```

### Integration Manager

```go
type IntegrationManager struct {
    integrations map[string]Integration
    mx           sync.RWMutex
}

type Integration interface {
    ID() string
    Health(ctx context.Context) error
    Enable() error
    Disable() error
}

type MetricsIntegration struct {
    enabled bool
    host    string
    client  *http.Client
}

func (m *MetricsIntegration) Health(ctx context.Context) error {
    resp, err := m.client.Get(m.host + "/health")
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unhealthy: status %d", resp.StatusCode)
    }

    return nil
}

func (mgr *IntegrationManager) EnableWithRetry(id string, maxRetries int) {
    go func() {
        for i := 0; i < maxRetries; i++ {
            integration := mgr.Get(id)
            if err := integration.Health(context.Background()); err == nil {
                integration.Enable()
                return
            }
            time.Sleep(time.Second * time.Duration(i+1))
        }
    }()
}
```

---

## 6. 캐시 아키텍처

### Pattern: Cache-and-Network with SSAR Validation

```go
type CacheConfig struct {
    Enabled      bool
    TTL          time.Duration  // 기본 10분
    MaxSize      int            // 최대 항목 수
}

type K8sCache struct {
    cache   *theine.Cache[string, *CachedItem]
    config  CacheConfig
    authz   authz.Interface
}

type CachedItem struct {
    Data       []byte
    ETag       string
    CachedAt   time.Time
    Resource   ResourceInfo
}

type ResourceInfo struct {
    Kind      string
    Namespace string
    Name      string
}

// 캐시 키 생성
func (c *K8sCache) generateKey(r *http.Request, contextID string) string {
    return fmt.Sprintf(
        "%s:%s:%s:%s",
        r.URL.Path,
        r.URL.RawQuery,
        contextID,
        extractUserID(r),
    )
}

// 권한 검증 후 캐시 반환
func (c *K8sCache) Get(ctx context.Context, r *http.Request) (*CachedItem, bool) {
    key := c.generateKey(r, getContextID(r))

    item, ok := c.cache.Get(key)
    if !ok {
        return nil, false
    }

    // TTL 확인
    if time.Since(item.CachedAt) > c.config.TTL {
        c.cache.Delete(key)
        return nil, false
    }

    // SSAR로 권한 검증
    if !c.hasPermission(ctx, r, item.Resource) {
        return nil, false
    }

    return item, true
}

// SubjectAccessReview로 권한 검증
func (c *K8sCache) hasPermission(ctx context.Context, r *http.Request, res ResourceInfo) bool {
    sar := &authv1.SubjectAccessReview{
        Spec: authv1.SubjectAccessReviewSpec{
            User: extractUserID(r),
            ResourceAttributes: &authv1.ResourceAttributes{
                Namespace: res.Namespace,
                Verb:      "get",
                Resource:  strings.ToLower(res.Kind) + "s",
            },
        },
    }

    result, err := c.authz.Create(ctx, sar, metav1.CreateOptions{})
    if err != nil {
        return false
    }

    return result.Status.Allowed
}
```

---

## 7. CSRF 보호

### Pattern: Dual Framework Support

```go
// Gin 미들웨어
func CSRFMiddlewareGin() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Safe methods는 통과
        if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
            c.Next()
            return
        }

        // 토큰 검증
        token := c.GetHeader("X-CSRF-TOKEN")
        if token == "" {
            token = c.PostForm("csrf_token")
        }

        expected := c.GetHeader("Cookie") // 실제로는 세션에서 가져옴

        if !validateCSRFToken(token, expected) {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "CSRF token validation failed",
            })
            return
        }

        c.Next()
    }
}

// go-restful 미들웨어
func CSRFMiddlewareRestful(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
    // Safe methods는 통과
    if req.Request.Method == "GET" || req.Request.Method == "HEAD" || req.Request.Method == "OPTIONS" {
        chain.ProcessFilter(req, resp)
        return
    }

    // 토큰 검증
    token := req.HeaderParameter("X-CSRF-TOKEN")
    if token == "" {
        token = req.Request.FormValue("csrf_token")
    }

    if !validateCSRFToken(token, getExpectedToken(req)) {
        resp.WriteErrorString(http.StatusForbidden, "CSRF token validation failed")
        return
    }

    chain.ProcessFilter(req, resp)
}

// 토큰 생성
func generateCSRFToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.StdEncoding.EncodeToString(b)
}
```

---

## Quick Reference

### 핵심 파일 위치

| 패턴 | 파일 |
|------|------|
| DataSelector | `/dashboard/modules/api/pkg/resource/dataselect/` |
| Common Client | `/dashboard/modules/common/client/` |
| Common Errors | `/dashboard/modules/common/errors/` |
| API Handler | `/dashboard/modules/api/pkg/handler/` |
| Metrics Scraper | `/dashboard/modules/metrics-scraper/` |
| Cache Design | `/dashboard/docs/design/cache.md` |

### 적용 우선순위

1. **DataSelector** - 범용 데이터 처리의 핵심
2. **Request-Scoped Client** - 권한 관리
3. **Multi-Module Architecture** - 확장성
4. **Metrics Integration** - 모니터링
5. **Init-based Registration** - 모듈화
6. **Cache Pattern** - 성능

---

## 활용 예시

### kube-ai-dashboard-cli에서의 적용

```go
// pkg/k8s/dataselector.go - Dashboard 패턴 적용
type ResourceTable struct {
    cells []DataCell
    query *DataSelectQuery
}

func (t *ResourceTable) Filter(filter string) *ResourceTable {
    if filter == "" {
        return t
    }

    var filtered []DataCell
    for _, cell := range t.cells {
        name := cell.GetProperty(NameProperty)
        if name.Contains(StdComparableString(filter)) {
            filtered = append(filtered, cell)
        }
    }

    t.cells = filtered
    return t
}

func (t *ResourceTable) Sort(property PropertyName, ascending bool) *ResourceTable {
    sort.Slice(t.cells, func(i, j int) bool {
        a := t.cells[i].GetProperty(property)
        b := t.cells[j].GetProperty(property)
        cmp := a.Compare(b)
        if ascending {
            return cmp < 0
        }
        return cmp > 0
    })
    return t
}

// TUI에서 사용
func (v *PodView) applyFilter(filter string) {
    v.table.Filter(filter).Sort(NameProperty, true)
    v.refresh()
}
```

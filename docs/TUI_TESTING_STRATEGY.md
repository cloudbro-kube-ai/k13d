# TUI Testing Strategy for kube-ai-dashboard-cli

> AI 에이전트가 TUI 개발 시 효율적으로 테스트할 수 있는 전략 문서

## Overview

TUI(Terminal User Interface) 애플리케이션은 시각적 검증이 필요하지만, AI 에이전트는 화면을 직접 볼 수 없습니다.
이 문서는 AI 에이전트가 TUI 기능을 자동으로 검증할 수 있는 테스트 전략을 정의합니다.

---

## 1. 테스트 인프라 개요

### 핵심 컴포넌트

```
pkg/ui/
├── tui_test_utils.go     # TUITester - SimulationScreen 기반 테스트 유틸리티
├── app_test.go           # 기본 하네스 테스트
└── nav_test.go           # 네비게이션 테스트 예시
```

### TUITester 구조

```go
type TUITester struct {
    App       *tview.Application    // tview 앱 인스턴스
    Screen    tcell.SimulationScreen // 헤드리스 시뮬레이션 스크린
    Assistant *Assistant            // AI 어시스턴트 컴포넌트
}

// 주요 메서드
func NewTUITester() (*TUITester, error)                                    // 테스터 생성
func (t *TUITester) InjectKey(key tcell.Key, r rune, mod tcell.ModMask)   // 키 입력 시뮬레이션
func (t *TUITester) GetContent() string                                    // 화면 내용 추출
func (t *TUITester) Run() func()                                          // 앱 실행 (stop 함수 반환)
func (t *TUITester) AssertPage(tb, pages, expected)                       // 페이지 검증
```

---

## 2. 테스트 레벨 정의

### Level 1: Unit Tests (컴포넌트 단위)

개별 UI 컴포넌트의 로직 검증. 화면 렌더링 없이 순수 로직만 테스트.

**대상:**
- Config 파싱/저장
- i18n 문자열 처리
- K8s 클라이언트 래퍼
- DB/Audit 레이어

**실행 명령:**
```bash
go test -v ./pkg/config/...
go test -v ./pkg/i18n/...
go test -v ./pkg/k8s/...
go test -v ./pkg/db/...
```

### Level 2: Integration Tests (TUI 통합)

SimulationScreen을 사용한 TUI 동작 검증. 헤드리스 환경에서 키 입력과 화면 상태 확인.

**대상:**
- 페이지 네비게이션
- 키바인딩 동작
- 폼 입력 처리
- 모달/다이얼로그 표시

**실행 명령:**
```bash
go test -v ./pkg/ui/...
```

### Level 3: E2E Tests (End-to-End)

실제 K8s 클러스터 연결 없이 fake clientset으로 전체 플로우 검증.

**대상:**
- 리소스 목록 표시
- AI 분석 요청/응답
- YAML/Describe 뷰
- Scale/Delete 확인 다이얼로그

---

## 3. AI 에이전트 친화적 테스트 패턴

### 3.1 화면 내용 검증 (Content Assertion)

```go
func TestDashboardShowsResources(t *testing.T) {
    tester, _ := NewTUITester()
    // ... setup with fake data ...

    stop := tester.Run()
    defer stop()

    content := tester.GetContent()

    // 텍스트 기반 검증 - AI가 결과를 해석하기 쉬움
    if !strings.Contains(content, "nginx-deployment") {
        t.Errorf("Expected 'nginx-deployment' in screen, got:\n%s", content)
    }
    if !strings.Contains(content, "Running") {
        t.Errorf("Expected 'Running' status in screen")
    }
}
```

### 3.2 페이지 상태 검증 (Page State Assertion)

```go
func TestNavigationFlow(t *testing.T) {
    tester, _ := NewTUITester()
    app := InitApp(tester.App, cfg, aiClient, k8sClient)

    stop := tester.Run()
    defer stop()

    // 초기 상태
    tester.AssertPage(t, app.Pages, "main")

    // 's' 키로 설정 페이지 이동
    tester.InjectKey(tcell.KeyRune, 's', tcell.ModNone)
    tester.AssertPage(t, app.Pages, "settings")

    // ESC로 메인 복귀
    tester.InjectKey(tcell.KeyEscape, 0, tcell.ModNone)
    tester.AssertPage(t, app.Pages, "main")
}
```

### 3.3 콜백 트리거 검증 (Callback Verification)

```go
func TestYAMLViewCallback(t *testing.T) {
    tester, _ := NewTUITester()
    app := InitApp(tester.App, cfg, aiClient, k8sClient)

    // 콜백 호출 추적
    var callbackCalled bool
    originalOnYaml := app.Dashboard.OnYaml
    app.Dashboard.OnYaml = func(ns, name string) {
        callbackCalled = true
        if ns != "default" || name != "nginx" {
            t.Errorf("unexpected callback args: %s/%s", ns, name)
        }
        originalOnYaml(ns, name)
    }

    stop := tester.Run()
    defer stop()

    // 'y' 키로 YAML 뷰 트리거
    tester.InjectKey(tcell.KeyRune, 'y', tcell.ModNone)

    if !callbackCalled {
        t.Error("OnYaml callback was not triggered")
    }
}
```

### 3.4 포커스 검증 (Focus Verification)

```go
// 추가할 헬퍼 메서드
func (t *TUITester) GetFocusedPrimitive() tview.Primitive {
    return t.App.GetFocus()
}

func (t *TUITester) AssertFocus(tb testing.TB, expected tview.Primitive) {
    tb.Helper()
    if t.App.GetFocus() != expected {
        tb.Errorf("focus mismatch: expected %T, got %T", expected, t.App.GetFocus())
    }
}
```

---

## 4. 권장 테스트 구조

### 4.1 테스트 파일 구성

```
pkg/ui/
├── tui_test_utils.go        # 공통 테스트 유틸리티
├── tui_test_fixtures.go     # Mock 데이터 및 fake 객체
├── app_test.go              # App 초기화 테스트
├── nav_test.go              # 네비게이션 테스트
├── dashboard_test.go        # Dashboard 컴포넌트 테스트
├── assistant_test.go        # AI 어시스턴트 테스트
├── resource_viewer_test.go  # 리소스 뷰어 테스트
├── log_viewer_test.go       # 로그 뷰어 테스트
├── settings_test.go         # 설정 페이지 테스트
└── command_bar_test.go      # 커맨드 바 테스트
```

### 4.2 Fake K8s 데이터 생성

```go
// tui_test_fixtures.go
func NewFakeK8sClient() *k8s.Client {
    fakeClient := fake.NewSimpleClientset(
        // Pods
        &corev1.Pod{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "nginx-pod",
                Namespace: "default",
            },
            Status: corev1.PodStatus{
                Phase: corev1.PodRunning,
            },
        },
        // Deployments
        &appsv1.Deployment{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "nginx-deployment",
                Namespace: "default",
            },
            Spec: appsv1.DeploymentSpec{
                Replicas: ptr.To(int32(3)),
            },
        },
        // ... more resources
    )
    return &k8s.Client{Clientset: fakeClient}
}
```

### 4.3 테스트 실행 명령어 모음

```bash
# 전체 테스트
go test ./...

# UI 테스트만
go test -v ./pkg/ui/...

# 특정 테스트 함수
go test -v -run TestNavigation ./pkg/ui/...

# 커버리지 포함
go test -cover -coverprofile=coverage.out ./pkg/ui/...
go tool cover -html=coverage.out

# 레이스 감지 (동시성 문제 확인)
go test -race ./pkg/ui/...

# 짧은 타임아웃 (빠른 피드백)
go test -timeout 30s ./pkg/ui/...
```

---

## 5. CI 통합

### 5.1 GitHub Actions 호환

TUITester는 헤드리스로 동작하므로 CI에서 그대로 실행 가능합니다.

```yaml
# .github/workflows/test.yml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: coverage.out
```

### 5.2 테스트 출력 해석

AI 에이전트가 테스트 결과를 해석하기 쉽도록:

```go
// 명확한 에러 메시지
t.Errorf("Page navigation failed:\n  Expected: %s\n  Got: %s\n  After key: %c",
    expected, actual, keyPressed)

// 화면 상태 덤프
t.Errorf("Content assertion failed:\nScreen content:\n---\n%s\n---\nMissing: %s",
    tester.GetContent(), missingText)
```

---

## 6. 테스트 개발 워크플로우

### AI 에이전트 권장 워크플로우

1. **기능 구현 전**: 테스트 케이스 먼저 작성 (TDD)
2. **기능 구현**: 코드 작성
3. **테스트 실행**: `go test -v ./pkg/ui/...`
4. **결과 확인**: 테스트 출력 분석
5. **반복**: 실패 시 수정 후 재실행

### 예시 세션

```bash
# 1. 현재 테스트 상태 확인
go test -v ./pkg/ui/... 2>&1 | head -50

# 2. 특정 테스트 디버깅
go test -v -run TestNavigation ./pkg/ui/... 2>&1

# 3. 새 기능 추가 후 전체 검증
go test -race ./... && echo "All tests passed!"
```

---

## 7. 확장 가능한 테스트 헬퍼

### 7.1 Enhanced TUITester (권장 추가 메서드)

```go
// pkg/ui/tui_test_utils.go에 추가 권장

// WaitForCondition waits for a condition to be true with timeout
func (t *TUITester) WaitForCondition(timeout time.Duration, condition func() bool) bool {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if condition() {
            return true
        }
        time.Sleep(10 * time.Millisecond)
    }
    return false
}

// AssertContentContains checks if screen contains text
func (t *TUITester) AssertContentContains(tb testing.TB, expected string) {
    tb.Helper()
    content := t.GetContent()
    if !strings.Contains(content, expected) {
        tb.Errorf("Screen does not contain '%s'.\nActual content:\n%s", expected, content)
    }
}

// AssertContentNotContains checks if screen doesn't contain text
func (t *TUITester) AssertContentNotContains(tb testing.TB, unexpected string) {
    tb.Helper()
    content := t.GetContent()
    if strings.Contains(content, unexpected) {
        tb.Errorf("Screen unexpectedly contains '%s'.\nActual content:\n%s", unexpected, content)
    }
}

// TypeString simulates typing a string
func (t *TUITester) TypeString(s string) {
    for _, r := range s {
        t.InjectKey(tcell.KeyRune, r, tcell.ModNone)
    }
}

// PressEnter simulates Enter key
func (t *TUITester) PressEnter() {
    t.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
}

// PressEscape simulates Escape key
func (t *TUITester) PressEscape() {
    t.InjectKey(tcell.KeyEscape, 0, tcell.ModNone)
}
```

---

## 8. 테스트 체크리스트

### 새 기능 추가 시 검증 항목

- [ ] 키바인딩 동작 확인
- [ ] 페이지 전환 확인
- [ ] 화면 내용에 예상 텍스트 포함
- [ ] ESC로 이전 상태 복귀 가능
- [ ] 에러 상황 처리 (빈 데이터, 네트워크 오류 등)
- [ ] 동시성 문제 없음 (`go test -race`)

### 커밋 전 필수 실행

```bash
# 1. 포맷팅
gofmt -s -w .

# 2. 정적 분석
go vet ./...

# 3. 린트 (설치된 경우)
golangci-lint run

# 4. 테스트
go test -race ./...

# 5. 빌드
go build ./...
```

---

## Quick Reference

| 목적 | 명령어 |
|------|--------|
| 전체 테스트 | `go test ./...` |
| UI 테스트만 | `go test -v ./pkg/ui/...` |
| 특정 테스트 | `go test -v -run TestName ./pkg/ui/...` |
| 커버리지 | `go test -cover ./...` |
| 레이스 감지 | `go test -race ./...` |
| 빠른 피드백 | `go test -timeout 30s ./...` |

---

## 다음 단계

1. `tui_test_utils.go`에 헬퍼 메서드 추가
2. `tui_test_fixtures.go` 파일 생성하여 fake 데이터 정리
3. 각 주요 컴포넌트별 `*_test.go` 파일 작성
4. CI 파이프라인에 테스트 단계 추가

# k13d Technology Stack

이 문서는 k13d 프로젝트에 사용된 주요 라이브러리와 기술 스택을 정리한 문서입니다.

## 목차

- [Core Technologies](#core-technologies)
- [TUI Framework](#tui-framework)
- [Kubernetes Integration](#kubernetes-integration)
- [Database](#database)
- [Web & Networking](#web--networking)
- [Utilities](#utilities)
- [Frontend Libraries](#frontend-libraries)

---

## Core Technologies

### Go 1.25.0+

프로젝트의 기본 언어입니다.

```go
// 기본 Go 프로그램 구조
package main

import "fmt"

func main() {
    fmt.Println("Hello, k13d!")
}
```

---

## TUI Framework

### tview (v0.42.0)

터미널 기반 사용자 인터페이스를 구축하기 위한 고급 프레임워크입니다. k9s 스타일의 대시보드를 구현하는 핵심 라이브러리입니다.

**GitHub**: https://github.com/rivo/tview

```go
package main

import (
    "github.com/rivo/tview"
)

func main() {
    app := tview.NewApplication()

    // 테이블 생성
    table := tview.NewTable().
        SetBorders(true).
        SetSelectable(true, false)

    // 헤더 추가
    headers := []string{"NAME", "NAMESPACE", "STATUS", "AGE"}
    for col, header := range headers {
        table.SetCell(0, col,
            tview.NewTableCell(header).
                SetTextColor(tview.Styles.SecondaryTextColor).
                SetSelectable(false))
    }

    // 데이터 행 추가
    table.SetCell(1, 0, tview.NewTableCell("nginx-pod"))
    table.SetCell(1, 1, tview.NewTableCell("default"))
    table.SetCell(1, 2, tview.NewTableCell("Running"))
    table.SetCell(1, 3, tview.NewTableCell("2d"))

    // 키 바인딩
    table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        switch event.Key() {
        case tcell.KeyEnter:
            row, _ := table.GetSelection()
            // 선택된 행 처리
            _ = row
        }
        return event
    })

    if err := app.SetRoot(table, true).Run(); err != nil {
        panic(err)
    }
}
```

**주요 컴포넌트:**
- `Table` - 리소스 목록 표시
- `TextView` - 로그, YAML 뷰어
- `Flex` - 레이아웃 관리
- `Modal` - 확인 다이얼로그
- `InputField` - 명령어 입력

---

### tcell (v2.13.6)

저수준 터미널 핸들링 라이브러리입니다. tview의 기반 라이브러리로, 키보드/마우스 이벤트와 화면 렌더링을 담당합니다.

**GitHub**: https://github.com/gdamore/tcell

```go
package main

import (
    "github.com/gdamore/tcell/v2"
)

func main() {
    screen, err := tcell.NewScreen()
    if err != nil {
        panic(err)
    }
    if err := screen.Init(); err != nil {
        panic(err)
    }
    defer screen.Fini()

    // 스타일 정의
    style := tcell.StyleDefault.
        Foreground(tcell.ColorGreen).
        Background(tcell.ColorBlack)

    // 텍스트 출력
    text := "Press 'q' to quit"
    for i, r := range text {
        screen.SetContent(i, 0, r, nil, style)
    }
    screen.Show()

    // 이벤트 루프
    for {
        ev := screen.PollEvent()
        switch ev := ev.(type) {
        case *tcell.EventKey:
            if ev.Rune() == 'q' {
                return
            }
        case *tcell.EventResize:
            screen.Sync()
        }
    }
}
```

**주요 기능:**
- 키보드 이벤트 (`EventKey`)
- 마우스 이벤트 (`EventMouse`)
- 화면 리사이즈 (`EventResize`)
- 스타일링 (색상, 굵기 등)

---

## Kubernetes Integration

### client-go (v0.35.0)

Kubernetes 공식 Go 클라이언트 라이브러리입니다. 클러스터와의 모든 상호작용을 담당합니다.

**GitHub**: https://github.com/kubernetes/client-go

```go
package main

import (
    "context"
    "fmt"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

func main() {
    // kubeconfig 로드
    config, err := clientcmd.BuildConfigFromFlags("",
        clientcmd.RecommendedHomeFile)
    if err != nil {
        panic(err)
    }

    // 클라이언트 생성
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        panic(err)
    }

    // Pod 목록 조회
    pods, err := clientset.CoreV1().Pods("default").
        List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        panic(err)
    }

    for _, pod := range pods.Items {
        fmt.Printf("Pod: %s, Status: %s\n",
            pod.Name, pod.Status.Phase)
    }
}
```

**주요 API:**
- `CoreV1()` - Pods, Services, ConfigMaps, Secrets
- `AppsV1()` - Deployments, StatefulSets, DaemonSets
- `NetworkingV1()` - Ingresses, NetworkPolicies
- `RbacV1()` - Roles, RoleBindings

---

### k8s.io/metrics (v0.35.0)

Kubernetes Metrics API 클라이언트입니다. Pod/Node의 CPU, 메모리 사용량을 조회합니다.

```go
package main

import (
    "context"
    "fmt"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/tools/clientcmd"
    metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
    config, _ := clientcmd.BuildConfigFromFlags("",
        clientcmd.RecommendedHomeFile)

    metricsClient, _ := metricsv.NewForConfig(config)

    // Pod 메트릭 조회
    podMetrics, _ := metricsClient.MetricsV1beta1().
        PodMetricses("default").
        List(context.TODO(), metav1.ListOptions{})

    for _, pm := range podMetrics.Items {
        for _, container := range pm.Containers {
            fmt.Printf("Pod: %s, Container: %s\n", pm.Name, container.Name)
            fmt.Printf("  CPU: %s, Memory: %s\n",
                container.Usage.Cpu().String(),
                container.Usage.Memory().String())
        }
    }
}
```

---

### Helm (v3.19.5)

Kubernetes 패키지 매니저입니다. 차트 설치, 업그레이드, 릴리스 관리 기능을 제공합니다.

**GitHub**: https://github.com/helm/helm

```go
package main

import (
    "fmt"
    "os"

    "helm.sh/helm/v3/pkg/action"
    "helm.sh/helm/v3/pkg/cli"
)

func main() {
    settings := cli.New()

    // Action 설정
    actionConfig := new(action.Configuration)
    if err := actionConfig.Init(settings.RESTClientGetter(),
        "default", os.Getenv("HELM_DRIVER"),
        func(format string, v ...interface{}) {
            fmt.Printf(format, v...)
        }); err != nil {
        panic(err)
    }

    // 릴리스 목록 조회
    client := action.NewList(actionConfig)
    client.Deployed = true

    releases, err := client.Run()
    if err != nil {
        panic(err)
    }

    for _, rel := range releases {
        fmt.Printf("Release: %s, Chart: %s, Status: %s\n",
            rel.Name, rel.Chart.Name(), rel.Info.Status)
    }
}
```

---

## Database

### modernc.org/sqlite (v1.43.0)

CGO 없이 순수 Go로 구현된 SQLite 드라이버입니다. 크로스 컴파일이 용이하고 배포가 간편합니다.

**GitHub**: https://gitlab.com/cznic/sqlite

```go
package main

import (
    "database/sql"
    "fmt"
    "time"

    _ "modernc.org/sqlite"
)

func main() {
    // 데이터베이스 연결
    db, err := sql.Open("sqlite", "./k13d.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 테이블 생성
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS audit_logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
            user TEXT,
            action TEXT,
            resource TEXT,
            details TEXT
        )
    `)
    if err != nil {
        panic(err)
    }

    // 데이터 삽입
    _, err = db.Exec(`
        INSERT INTO audit_logs (user, action, resource, details)
        VALUES (?, ?, ?, ?)`,
        "admin", "DELETE", "pod/nginx", "Deleted pod in default namespace")
    if err != nil {
        panic(err)
    }

    // 데이터 조회
    rows, _ := db.Query(`
        SELECT timestamp, user, action, resource
        FROM audit_logs
        ORDER BY timestamp DESC
        LIMIT 10`)
    defer rows.Close()

    for rows.Next() {
        var ts time.Time
        var user, action, resource string
        rows.Scan(&ts, &user, &action, &resource)
        fmt.Printf("[%s] %s: %s %s\n", ts.Format(time.RFC3339), user, action, resource)
    }
}
```

**k13d에서의 용도:**
- 감사 로그 저장
- 사용자 세션 관리
- 설정 값 저장

---

## Web & Networking

### gorilla/websocket (v1.5.4)

WebSocket 프로토콜 구현체입니다. 실시간 로그 스트리밍, AI 채팅에 사용됩니다.

**GitHub**: https://github.com/gorilla/websocket

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // 개발용, 프로덕션에서는 제한 필요
    },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }
    defer conn.Close()

    for {
        // 메시지 수신
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            break
        }

        fmt.Printf("Received: %s\n", message)

        // 에코 응답
        if err := conn.WriteMessage(messageType, message); err != nil {
            break
        }
    }
}

func main() {
    http.HandleFunc("/ws", wsHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

**k13d에서의 용도:**
- Pod 로그 실시간 스트리밍
- AI 응답 스트리밍
- 터미널 세션 (exec)

---

### go-ldap/ldap (v3.4.12)

LDAP 클라이언트 라이브러리입니다. 엔터프라이즈 환경에서 LDAP/Active Directory 인증에 사용됩니다.

**GitHub**: https://github.com/go-ldap/ldap

```go
package main

import (
    "fmt"
    "log"

    "github.com/go-ldap/ldap/v3"
)

func main() {
    // LDAP 서버 연결
    conn, err := ldap.DialURL("ldap://ldap.example.com:389")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // 바인드 (인증)
    err = conn.Bind("cn=admin,dc=example,dc=com", "password")
    if err != nil {
        log.Fatal(err)
    }

    // 사용자 검색
    searchRequest := ldap.NewSearchRequest(
        "dc=example,dc=com",
        ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
        "(&(objectClass=person)(uid=john))",
        []string{"dn", "cn", "mail"},
        nil,
    )

    sr, err := conn.Search(searchRequest)
    if err != nil {
        log.Fatal(err)
    }

    for _, entry := range sr.Entries {
        fmt.Printf("DN: %s\n", entry.DN)
        fmt.Printf("  CN: %s\n", entry.GetAttributeValue("cn"))
        fmt.Printf("  Email: %s\n", entry.GetAttributeValue("mail"))
    }
}
```

---

## Utilities

### adrg/xdg (v0.5.3)

XDG Base Directory 스펙을 구현한 라이브러리입니다. 설정 파일, 캐시, 데이터 저장 경로를 표준화합니다.

**GitHub**: https://github.com/adrg/xdg

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/adrg/xdg"
)

func main() {
    // 설정 파일 경로 (~/.config/k13d/config.yaml)
    configPath, err := xdg.ConfigFile("k13d/config.yaml")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Config: %s\n", configPath)

    // 데이터 디렉토리 (~/.local/share/k13d/)
    dataDir := filepath.Join(xdg.DataHome, "k13d")
    os.MkdirAll(dataDir, 0755)
    fmt.Printf("Data Dir: %s\n", dataDir)

    // 캐시 디렉토리 (~/.cache/k13d/)
    cacheDir := filepath.Join(xdg.CacheHome, "k13d")
    os.MkdirAll(cacheDir, 0755)
    fmt.Printf("Cache Dir: %s\n", cacheDir)

    // 런타임 디렉토리 (임시 파일용)
    if xdg.RuntimeDir != "" {
        runtimeDir := filepath.Join(xdg.RuntimeDir, "k13d")
        fmt.Printf("Runtime Dir: %s\n", runtimeDir)
    }
}
```

**XDG 경로 매핑:**
| 변수 | macOS | Linux |
|------|-------|-------|
| `ConfigHome` | `~/Library/Application Support` | `~/.config` |
| `DataHome` | `~/Library/Application Support` | `~/.local/share` |
| `CacheHome` | `~/Library/Caches` | `~/.cache` |

---

### cenkalti/backoff (v4.3.0)

지수 백오프(Exponential Backoff) 알고리즘 구현체입니다. 재시도 로직에 사용됩니다.

**GitHub**: https://github.com/cenkalti/backoff

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/cenkalti/backoff/v4"
)

func main() {
    // 재시도할 작업
    attempts := 0
    operation := func() error {
        attempts++
        fmt.Printf("Attempt %d\n", attempts)

        if attempts < 3 {
            return errors.New("temporary error")
        }
        return nil // 성공
    }

    // 지수 백오프 설정
    b := backoff.NewExponentialBackOff()
    b.InitialInterval = 100 * time.Millisecond
    b.MaxInterval = 2 * time.Second
    b.MaxElapsedTime = 10 * time.Second

    // 컨텍스트와 함께 재시도
    ctx := context.Background()
    err := backoff.Retry(operation, backoff.WithContext(b, ctx))

    if err != nil {
        fmt.Printf("Operation failed: %v\n", err)
    } else {
        fmt.Printf("Operation succeeded after %d attempts\n", attempts)
    }
}
```

**k13d에서의 용도:**
- Kubernetes API 재시도
- LLM API 호출 재시도
- WebSocket 재연결

---

### mvdan.cc/sh (v3.12.0)

POSIX 쉘 파서/인터프리터입니다. bash 명령어 파싱 및 안전성 검증에 사용됩니다.

**GitHub**: https://github.com/mvdan/sh

```go
package main

import (
    "fmt"
    "strings"

    "mvdan.cc/sh/v3/syntax"
)

func main() {
    // 쉘 명령어 파싱
    src := "kubectl get pods -n default | grep Running"

    reader := strings.NewReader(src)
    parser := syntax.NewParser()

    file, err := parser.Parse(reader, "")
    if err != nil {
        panic(err)
    }

    // AST 순회
    syntax.Walk(file, func(node syntax.Node) bool {
        switch n := node.(type) {
        case *syntax.CallExpr:
            if len(n.Args) > 0 {
                cmd := n.Args[0]
                fmt.Printf("Command: %s\n", cmd.Lit())
            }
        case *syntax.BinaryCmd:
            fmt.Printf("Pipe operator: %s\n", n.Op)
        }
        return true
    })
}
```

**k13d에서의 용도:**
- AI가 생성한 명령어 검증
- 위험한 명령어 탐지 (rm -rf, kubectl delete 등)
- 명령어 분석 및 로깅

---

### gopkg.in/yaml.v3

YAML 파서/직렬화 라이브러리입니다.

```go
package main

import (
    "fmt"

    "gopkg.in/yaml.v3"
)

type Config struct {
    Provider string `yaml:"provider"`
    Model    string `yaml:"model"`
    APIKey   string `yaml:"api_key"`
}

func main() {
    // YAML 파싱
    data := []byte(`
provider: openai
model: gpt-4
api_key: sk-xxx
`)

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        panic(err)
    }
    fmt.Printf("Provider: %s, Model: %s\n", config.Provider, config.Model)

    // YAML 생성
    newConfig := Config{
        Provider: "ollama",
        Model:    "llama3",
    }

    output, _ := yaml.Marshal(newConfig)
    fmt.Printf("\n%s", output)
}
```

---

## Frontend Libraries

Web UI에서 사용되는 JavaScript 라이브러리들입니다.

### xterm.js

브라우저 기반 터미널 에뮬레이터입니다. Pod exec, 로그 뷰어에 사용됩니다.

**GitHub**: https://github.com/xtermjs/xterm.js

```html
<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="xterm.css">
</head>
<body>
    <div id="terminal"></div>

    <script src="xterm.min.js"></script>
    <script src="xterm-addon-fit.min.js"></script>
    <script>
        const term = new Terminal({
            cursorBlink: true,
            fontSize: 14,
            fontFamily: 'monospace',
            theme: {
                background: '#1e1e1e',
                foreground: '#d4d4d4'
            }
        });

        const fitAddon = new FitAddon.FitAddon();
        term.loadAddon(fitAddon);

        term.open(document.getElementById('terminal'));
        fitAddon.fit();

        // WebSocket으로 서버와 연결
        const ws = new WebSocket('ws://localhost:8080/exec');

        ws.onmessage = (event) => {
            term.write(event.data);
        };

        term.onData((data) => {
            ws.send(data);
        });
    </script>
</body>
</html>
```

---

### Chart.js

차트 라이브러리입니다. 리소스 사용량 그래프에 사용됩니다.

**GitHub**: https://github.com/chartjs/Chart.js

```html
<canvas id="cpuChart"></canvas>

<script src="chart.umd.min.js"></script>
<script>
    const ctx = document.getElementById('cpuChart').getContext('2d');

    new Chart(ctx, {
        type: 'line',
        data: {
            labels: ['1m', '2m', '3m', '4m', '5m'],
            datasets: [{
                label: 'CPU Usage (%)',
                data: [25, 30, 28, 45, 42],
                borderColor: '#4CAF50',
                backgroundColor: 'rgba(76, 175, 80, 0.1)',
                tension: 0.3,
                fill: true
            }]
        },
        options: {
            responsive: true,
            scales: {
                y: {
                    beginAtZero: true,
                    max: 100
                }
            }
        }
    });
</script>
```

---

### marked.js

Markdown 파서입니다. AI 응답 렌더링에 사용됩니다.

**GitHub**: https://github.com/markedjs/marked

```html
<div id="content"></div>

<script src="marked.min.js"></script>
<script>
    const markdown = `
# Pod 분석 결과

## 상태
- **이름**: nginx-deployment-abc123
- **상태**: Running
- **재시작 횟수**: 0

## 권장 사항
1. 리소스 limit 설정을 권장합니다
2. liveness probe 추가를 고려하세요

\`\`\`yaml
resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
\`\`\`
`;

    document.getElementById('content').innerHTML = marked.parse(markdown);
</script>
```

---

### ansi_up.js

ANSI 이스케이프 코드를 HTML로 변환합니다. 터미널 출력 렌더링에 사용됩니다.

**GitHub**: https://github.com/drudru/ansi_up

```html
<pre id="output"></pre>

<script src="ansi_up.min.js"></script>
<script>
    const ansi_up = new AnsiUp();

    // ANSI 컬러 코드가 포함된 텍스트
    const ansiText = '\x1b[32m✓ Pod is running\x1b[0m\n' +
                     '\x1b[31m✗ Service not found\x1b[0m\n' +
                     '\x1b[33m⚠ Warning: High memory usage\x1b[0m';

    // HTML로 변환
    const html = ansi_up.ansi_to_html(ansiText);
    document.getElementById('output').innerHTML = html;
</script>
```

---

## 의존성 요약

| 카테고리 | 라이브러리 | 버전 | 용도 |
|---------|-----------|------|------|
| **TUI** | tview | v0.42.0 | 터미널 UI 프레임워크 |
| | tcell | v2.13.6 | 저수준 터미널 핸들링 |
| **Kubernetes** | client-go | v0.35.0 | K8s API 클라이언트 |
| | metrics | v0.35.0 | 메트릭 조회 |
| | helm | v3.19.5 | 차트 관리 |
| **Database** | sqlite | v1.43.0 | CGO-free SQLite |
| **Web** | websocket | v1.5.4 | WebSocket 통신 |
| | ldap | v3.4.12 | LDAP 인증 |
| **Utilities** | xdg | v0.5.3 | 경로 관리 |
| | backoff | v4.3.0 | 재시도 로직 |
| | sh | v3.12.0 | 쉘 명령어 파싱 |
| | yaml.v3 | v3.0.1 | YAML 처리 |
| **Frontend** | xterm.js | - | 터미널 에뮬레이터 |
| | Chart.js | - | 차트 시각화 |
| | marked.js | - | Markdown 렌더링 |
| | ansi_up.js | - | ANSI 코드 변환 |

---

## 참고 자료

- [tview Wiki](https://github.com/rivo/tview/wiki)
- [client-go Examples](https://github.com/kubernetes/client-go/tree/master/examples)
- [Helm Go SDK](https://helm.sh/docs/topics/advanced/#go-sdk)
- [xterm.js Documentation](https://xtermjs.org/docs/)

# Skills - Reference Patterns for k13d

> k13d (kube-ai-dashboard-cli) 개발을 위한 참조 패턴 및 디자인 가이드라인

## Overview

이 디렉토리는 k13d 개발에 필요한 아키텍처 패턴, 베스트 프랙티스, 디자인 가이드라인을 제공합니다.

---

## Skills 목록

### Code Quality & Security

| Skill | Description |
|-------|-------------|
| [Code Review](./code-review/SKILL.md) | PR 리뷰 및 코드 품질 검토 (Sentry 기반) |
| [Security Review](./security-review/SKILL.md) | 보안 취약점 탐지 및 감사 (Trail of Bits 기반) |
| [Go Best Practices](./go-best-practices/SKILL.md) | Go 언어 및 client-go 베스트 프랙티스 |

---

### 1. [k9s Patterns](./k9s-patterns.md)

**핵심**: TUI 아키텍처 및 사용자 경험

| 패턴 | 설명 | 활용 |
|------|------|------|
| MVC 3계층 | Model-Render-View 분리 | 리소스 뷰 구조화 |
| Observer 패턴 | Listener 기반 이벤트 | 컴포넌트 간 통신 |
| Stack 네비게이션 | Pages + Stack | 뷰 히스토리 관리 |
| Action 시스템 | KeyActions | 키바인딩 관리 |
| Plugin/HotKey | Scope 기반 플러그인 | 확장성 |
| Skin 시스템 | 계층적 스타일 | 테마 커스터마이징 |
| XDG 설정 | 멀티레벨 설정 | 설정 파일 관리 |

**우선순위**: TUI 개발 시 1순위 참조

---

### 2. [kubectl-ai Patterns](./kubectl-ai-patterns.md)

**핵심**: AI 에이전트 설계 및 안전한 명령 실행

| 패턴 | 설명 | 활용 |
|------|------|------|
| Agent Loop | State Machine | AI 상태 관리 |
| Tool System | Plugin Architecture | 도구 등록/실행 |
| LLM Provider | Provider-Agnostic | 다중 LLM 지원 |
| MCP 통합 | Adapter Pattern | 외부 도구 연동 |
| Safety Layers | Defense in Depth | 명령 실행 안전성 |
| Session 관리 | Pluggable Store | 대화 영속성 |

**우선순위**: AI 기능 개발 시 필수 참조

---

### 3. [Headlamp Patterns](./headlamp-patterns.md)

**핵심**: 플러그인 시스템 및 엔터프라이즈 기능

| 패턴 | 설명 | 활용 |
|------|------|------|
| Plugin Registry | Registration API | 플러그인 확장점 |
| Multi-Cluster | Multiplexer | 클러스터 전환 |
| Response Cache | Authorization-Aware | API 캐싱 |
| OIDC 인증 | Token Flow | 엔터프라이즈 인증 |
| i18n 시스템 | Dual-System | 다국어 지원 |
| WS Multiplexing | Single Connection | 실시간 업데이트 |

**우선순위**: 확장성/엔터프라이즈 기능 개발 시 참조

---

### 4. [Kubernetes Dashboard Patterns](./kubernetes-dashboard-patterns.md)

**핵심**: API 설계 및 데이터 처리

| 패턴 | 설명 | 활용 |
|------|------|------|
| Multi-Module | Microservices | 모듈 분리 |
| DataSelector | Generic Processing | 필터/정렬/페이지네이션 |
| Init Registration | Decentralized Routes | 라우트 분산 등록 |
| Request-Scoped | Per-Request Client | 권한 관리 |
| Metrics Integration | Sidecar Scraper | 메트릭 수집 |
| CSRF 보호 | Dual Framework | 보안 |

**우선순위**: 백엔드/API 설계 시 참조

---

### 5. [Frontend Design](./frontend-design/SKILL.md)

**핵심**: Web UI 디자인 품질

Production-grade 프론트엔드 인터페이스 가이드라인:
- 타이포그래피 선택 및 조합
- 컬러 팔레트 및 테마
- 모션 및 마이크로 인터랙션
- 공간 구성 및 레이아웃
- 백그라운드 텍스처 및 비주얼 디테일

**우선순위**: Web UI 개발 시 참조

---

### 6. [Web Design Guidelines](./web-design-guidelines/SKILL.md)

**핵심**: 접근성, 성능 및 UX

Vercel Web Interface Guidelines 기반 100+ 규칙:
- 접근성 (WCAG 준수, ARIA, 키보드 네비게이션)
- 포커스 상태 및 폼 디자인
- 애니메이션 및 모션 환경설정
- 타이포그래피 및 콘텐츠 처리
- 성능 최적화
- 터치 인터랙션 및 모바일 지원
- 다크 모드 및 국제화

**우선순위**: UI 코드 품질 검토 시 참조

---

## 개발 단계별 참조 가이드

```
┌─────────────────────────────────────────────────────────────┐
│                         k13d                                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │     TUI      │    │      AI      │    │   Web UI     │  │
│  │              │    │   Assistant  │    │              │  │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘  │
│         │                   │                   │          │
│         ▼                   ▼                   ▼          │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  k9s         │    │  kubectl-ai  │    │  Frontend +  │  │
│  │  Patterns    │    │  Patterns    │    │  Web Design  │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐│
│  │      Headlamp + K8s Dashboard Patterns                 ││
│  │        (Plugin, i18n, Multi-Cluster, API)              ││
│  └────────────────────────────────────────────────────────┘│
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Phase 1: 기본 TUI
```
참조: k9s-patterns.md
- MVC 3계층 아키텍처 적용
- Action 시스템으로 키바인딩 관리
- Stack 네비게이션 구현
```

### Phase 2: AI 통합
```
참조: kubectl-ai-patterns.md
- Tool Interface 정의
- Agent Loop 구현
- Safety Layers 적용
```

### Phase 3: Web UI
```
참조: frontend-design/, web-design-guidelines/
- 타이포그래피 및 컬러 가이드라인 적용
- 접근성 준수 확인
- 반응형 디자인 구현
```

### Phase 4: 확장성
```
참조: headlamp-patterns.md, kubernetes-dashboard-patterns.md
- Plugin Registry 구현
- i18n 시스템 통합
- Multi-Cluster 지원
```

---

## 핵심 인터페이스

```go
// k9s: Listener Pattern
type TableListener interface {
    TableDataChanged(*TableData)
}

// kubectl-ai: Tool Interface
type Tool interface {
    Run(ctx context.Context, args map[string]any) (any, error)
    CheckModifiesResource(args map[string]any) string
}

// Headlamp: Plugin Registry
type Registry interface {
    RegisterSidebarEntry(entry SidebarEntry)
    RegisterRoute(route Route)
}

// Dashboard: DataCell Interface
type DataCell interface {
    GetProperty(PropertyName) ComparableValue
}
```

---

## 참조 프로젝트

| 프로젝트 | URL | 주요 영역 |
|----------|-----|-----------|
| k9s | https://github.com/derailed/k9s | TUI 패턴 |
| kubectl-ai | https://github.com/GoogleCloudPlatform/kubectl-ai | AI 에이전트 |
| Headlamp | https://github.com/headlamp-k8s/headlamp | 플러그인 시스템 |
| K8s Dashboard | https://github.com/kubernetes/dashboard | API 설계 |
| Vercel Guidelines | https://github.com/vercel-labs/web-interface-guidelines | Web UI |

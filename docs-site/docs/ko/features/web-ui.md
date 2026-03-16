# Web UI 기능

k13d Web UI는 브라우저에서 Kubernetes 리소스 관리와 AI Assistant를 함께 사용할 수 있는 대시보드입니다.

![Web UI Dashboard](../../images/webui-full-screen.png)

## 주요 화면 구성

- 왼쪽: 리소스 내비게이션 사이드바
- 가운데: 리소스 테이블과 상세 보기
- 오른쪽: AI Assistant 패널

## 핵심 기능

| 기능 | 설명 |
|------|------|
| Dashboard | Pods, Deployments, Services 등 리소스 상태를 한눈에 확인 |
| Detail Modal | YAML, Events, Labels, 상태 정보를 탭으로 확인 |
| Overview | 클러스터 건강 상태, 빠른 이동 카드, 최근 이벤트 제공 |
| Applications | `app.kubernetes.io/name` 기준 앱 중심 그룹 보기 |
| Topology | 리소스 관계를 그래프/트리로 시각화 |
| Reports | 클러스터 리포트 생성 |
| Metrics | CPU, Memory, 리소스 수 추이 확인 |

## AI Assistant와 승인 흐름

- 자연어로 진단, 설명, kubectl 실행 요청 가능
- 기본적으로 read-only 명령도 승인 모달을 거칩니다
- write/dangerous 명령은 더 보수적으로 다뤄집니다
- `bash`는 마지막 수단으로만 사용하도록 설계되어 있습니다

## 함께 보면 좋은 문서

- [Web Dashboard](../user-guide/web.md)
- [AI Assistant](ai-assistant.md)
- [설정](../getting-started/configuration.md)

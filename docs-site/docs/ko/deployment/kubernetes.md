# Kubernetes 배포

!!! warning "Beta / 아직 공식 지원 아님"
    Kubernetes 배포는 아직 준비 중이며, 현재는 일반 사용자 대상 공식 지원 경로가 아닙니다.

## 현재 상태

- `deploy/kubernetes/` 아래 manifest는 작업 중인 초안입니다
- 안정적인 in-cluster 배포 경로는 아직 준비되지 않았습니다
- 공식 퍼블릭 Docker 이미지 저장소도 아직 없습니다

## 현재 권장 사용 방법

클러스터 안에 설치하기보다 로컬에서 단일 바이너리로 실행하는 흐름이 현재 공식 지원 경로입니다.

```bash
./k13d
./k13d --web --auth-mode local
```

배포 지원이 준비되면 이 문서에 다음이 추가될 예정입니다.

- 지원되는 manifest
- 이미지 배포/버전 정책
- 인증 및 RBAC 가이드
- 업그레이드/운영 가이드

# Docker 배포

!!! warning "Beta / 아직 공식 지원 아님"
    Docker와 Docker Compose 배포는 아직 준비 중이며, 현재 릴리스에서는 공식 지원 경로가 아닙니다.

## 현재 상태

- 공식 퍼블릭 Docker 이미지 저장소가 아직 없습니다
- 저장소 안의 Docker 관련 파일은 작업 중인 참고 자료입니다
- 실제 사용은 로컬 single-binary 방식이 권장됩니다

## 현재 권장 사용 방법

```bash
./k13d
./k13d --web --auth-mode local
```

지원 준비가 끝나면 이 문서에 다음이 추가될 예정입니다.

- 공식 이미지 경로
- `docker run` 예시
- Docker Compose 예시
- 영속성/보안/업그레이드 가이드

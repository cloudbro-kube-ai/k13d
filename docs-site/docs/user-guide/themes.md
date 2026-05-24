# 테마 (Themes)

k13d는 다양한 색상 테마를 지원하여 사용자의 선호도와 작업 환경에 맞게 UI를 커스터마이징할 수 있습니다.

## 기본 제공 테마

k13d는 다음과 같은 내장 테마를 제공합니다:

### 1. Default (Dracula)

기본 다크 테마로, 눈의 피로를 줄이고 장시간 작업에 적합합니다.

- **배경**: 어두운 보라색 계열 (#282a36)
- **텍스트**: 밝은 회색 (#f8f8f2)
- **강조**: 보라색, 청록색, 녹색 등 다채로운 색상

### 2. Ollama (Minimalist)

Ollama 디자인 시스템을 기반으로 한 미니멀리스트 라이트 테마입니다.

- **배경**: 순백색 (#ffffff) - 종이 같은 깨끗한 배경
- **텍스트**: 순수 검정 (#000000) - 최고의 가독성
- **강조**: 검정색 중심의 단순한 색상 팔레트
- **특징**:
  - 문서 스타일의 깔끔한 인터페이스
  - macOS 터미널 신호등 색상 (빨강/노랑/초록)
  - 최소한의 장식으로 콘텐츠에 집중

**디자인 철학**: Ollama 테마는 "시스템이 곧 문서이고, 문서가 곧 시스템"이라는 철학을 따릅니다. 모든 요소가 Markdown 문서처럼 깔끔하고 읽기 쉽게 디자인되었습니다.

### 3. Production (Red Alert)

프로덕션 환경용 경고 테마입니다.

- **테두리**: 빨간색 (#ff5555)
- **상태바**: 빨간색 배경
- **용도**: 프로덕션 클러스터 작업 시 시각적 경고

### 4. Staging (Orange Caution)

스테이징 환경용 주의 테마입니다.

- **테두리**: 주황색 (#ffb86c)
- **상태바**: 주황색 배경
- **용도**: 스테이징 클러스터 식별

### 5. Development (Green Safe)

개발 환경용 안전 테마입니다.

- **테두리**: 녹색 (#50fa7b)
- **상태바**: 녹색 배경
- **용도**: 개발 클러스터에서 안전하게 작업

## 컨텍스트별 테마 자동 전환

Kubernetes 컨텍스트에 따라 자동으로 테마를 전환할 수 있습니다. 이를 통해 실수로 프로덕션 환경에서 작업하는 것을 방지할 수 있습니다.

### 설정 방법

`~/.config/k13d/context-skins.yaml` 파일을 생성하고 다음과 같이 설정합니다:

```yaml
mappings:
  # 정확한 컨텍스트 이름 매칭
  production: production
  prod-cluster: production
  staging: staging
  development: development

  # 와일드카드 패턴 매칭
  prod-*: production # prod-us-east, prod-eu-west 등
  stg-*: staging # stg-us-east, stg-eu-west 등
  dev-*: development # dev-local, dev-test 등

  # Ollama 테마 사용 예시
  local: ollama # 로컬 개발 환경
  minikube: ollama # Minikube 클러스터
  kind-*: ollama # Kind 클러스터들
```

### 매칭 우선순위

1. **정확한 이름 매칭**: 컨텍스트 이름이 정확히 일치하는 경우
2. **와일드카드 패턴**: `*`를 사용한 패턴 매칭
3. **기본 테마**: 매칭되지 않으면 `default` 테마 사용

### 예제 시나리오

```yaml
mappings:
  # 프로덕션 클러스터들 - 빨간색 경고
  prod-us-east-1: production
  prod-eu-west-1: production
  prod-*: production

  # 스테이징 - 주황색 주의
  staging-*: staging

  # 로컬 개발 - Ollama 미니멀 테마
  docker-desktop: ollama
  minikube: ollama
  kind-*: ollama

  # 개발 클러스터 - 녹색 안전
  dev-*: development
```

## 커스텀 테마 생성

자신만의 테마를 만들 수 있습니다.

### 1. 테마 파일 생성

`~/.config/k13d/skins/mytheme.yaml` 파일을 생성합니다:

```yaml
k13d:
  body:
    fgColor: "#ffffff"
    bgColor: "#1a1a1a"

  frame:
    borderColor: "#444444"
    focusBorderColor: "#00ff00"
    titleColor: "#cccccc"
    focusTitleColor: "#ffffff"

  views:
    table:
      header:
        fgColor: "#00ff00"
        bgColor: "#1a1a1a"
        bold: true
      rowOdd:
        fgColor: "#ffffff"
        bgColor: "#1a1a1a"
      rowEven:
        fgColor: "#ffffff"
        bgColor: "#222222"
      rowSelected:
        fgColor: "#000000"
        bgColor: "#00ff00"
      rowHover:
        fgColor: "#ffffff"
        bgColor: "#333333"

    log:
      fgColor: "#ffffff"
      bgColor: "#1a1a1a"
      errorColor: "#ff0000"
      warningColor: "#ffaa00"
      infoColor: "#00aaff"

    charts:
      default: "#00ff00"
      cpu: "#00aaff"
      memory: "#ff00ff"
      network: "#ffaa00"

  dialog:
    fgColor: "#ffffff"
    bgColor: "#2a2a2a"
    buttonFgColor: "#ffffff"
    buttonBgColor: "#444444"
    buttonFocusFgColor: "#000000"
    buttonFocusBgColor: "#00ff00"

  statusBar:
    fgColor: "#ffffff"
    bgColor: "#00ff00"
    errorColor: "#ff0000"
```

### 2. 테마 적용

context-skins.yaml에서 커스텀 테마를 참조합니다:

```yaml
mappings:
  my-cluster: mytheme
```

## 색상 참조

### Ollama 테마 색상 팔레트

Ollama 테마는 다음 색상 토큰을 사용합니다:

| 토큰                     | 색상    | 용도                      |
| ------------------------ | ------- | ------------------------- |
| `colors.canvas`          | #ffffff | 배경 (순백색)             |
| `colors.ink`             | #000000 | 주요 텍스트 (순수 검정)   |
| `colors.primary`         | #000000 | 강조 요소 (순수 검정)     |
| `colors.body`            | #737373 | 본문 텍스트 (회색)        |
| `colors.charcoal`        | #525252 | 보조 텍스트 (진한 회색)   |
| `colors.mute`            | #a3a3a3 | 비활성 텍스트 (연한 회색) |
| `colors.surface-soft`    | #fafafa | 부드러운 표면             |
| `colors.surface-dark`    | #171717 | 어두운 표면 (반전용)      |
| `colors.hairline`        | #e5e5e5 | 1px 테두리                |
| `colors.hairline-strong` | #d4d4d4 | 강조 테두리               |
| `colors.on-dark`         | #ffffff | 어두운 배경의 텍스트      |
| `colors.terminal-red`    | #ff5f56 | 에러 (macOS 빨강)         |
| `colors.terminal-yellow` | #ffbd2e | 경고 (macOS 노랑)         |
| `colors.terminal-green`  | #27c93f | 성공 (macOS 초록)         |

### 색상 형식

색상은 다음 형식을 지원합니다:

- **Hex 색상**: `#ff5555`, `#ffffff`
- **명명된 색상**: `red`, `blue`, `green` 등 (tcell 지원 색상)

## 테마 목록 확인

사용 가능한 테마 목록을 확인하려면:

```bash
# TUI에서 :alias 명령어로 확인
:alias

# 또는 설정 디렉토리 확인
ls ~/.config/k13d/skins/
```

## 모범 사례

1. **프로덕션 환경**: 항상 `production` 테마를 사용하여 시각적으로 경고
2. **로컬 개발**: `ollama` 테마로 깔끔한 작업 환경 유지
3. **다크 모드 선호**: 기본 `default` 테마 사용
4. **라이트 모드 선호**: `ollama` 테마 사용
5. **컨텍스트 자동 전환**: context-skins.yaml 설정으로 실수 방지

## 문제 해결

### 테마가 적용되지 않음

1. 파일 경로 확인: `~/.config/k13d/skins/테마이름.yaml`
2. YAML 문법 확인: 들여쓰기와 구조가 올바른지 확인
3. 파일 권한 확인: 읽기 권한이 있는지 확인

### 색상이 이상하게 보임

1. 터미널이 트루컬러를 지원하는지 확인
2. `TERM` 환경 변수 확인: `echo $TERM`
3. 권장 터미널: iTerm2, Alacritty, WezTerm

### 컨텍스트 자동 전환이 작동하지 않음

1. context-skins.yaml 파일 위치 확인
2. 컨텍스트 이름 정확히 확인: `kubectl config current-context`
3. 와일드카드 패턴 확인: `prod-*`는 `prod-`로 시작하는 모든 이름 매칭

## 참고 자료

- [Ollama Design System](https://ollama.com) - Ollama 테마의 디자인 철학
- [k9s Skins](https://k9scli.io/topics/skins/) - k9s 테마 시스템 (호환 가능)
- [tcell Colors](https://github.com/gdamore/tcell) - 지원되는 색상 목록

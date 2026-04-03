# 빠른 시작

내일 실습 기준으로 가장 쉬운 방법만 안내합니다.

1. `kubectl` 이 먼저 되는지 확인
2. 운영체제에 맞는 Release Asset 다운로드
3. 압축 해제
4. Web UI 실행

!!! success "실습 권장 시작 방법"
    [Release v1.0.1](https://github.com/cloudbro-kube-ai/k13d/releases/tag/v1.0.1) 의 **단일 바이너리**를 바로 실행하세요.
    전체 릴리즈와 자산 목록은 [GitHub Releases 페이지](https://github.com/cloudbro-kube-ai/k13d/releases) 에서 한 번에 볼 수 있습니다.

    - 접속 주소: `http://localhost:9090`
    - 아이디: `admin`
    - 비밀번호: k13d 실행 후 터미널에 출력됨

!!! info "무엇을 받으면 되나요?"
    실습은 `k13d_v1.0.1_<os>_<arch>` 자산을 받으면 됩니다.
    `k13d-plugin_v1.0.1_<os>_<arch>` 는 `kubectl k13d` 형태로 쓰고 싶을 때만 선택하세요.

## 가장 많이 쓰는 실행 명령

### Web UI

```bash
./k13d --web --port 9090 --auth-mode local
```

### TUI

```bash
./k13d
```

## 시작 전 확인

아래 명령이 먼저 되어야 합니다.

```bash
kubectl get nodes
```

이 명령이 정상 동작하면 바로 진행하면 됩니다.

## 운영체제별 설치 및 실행

=== "macOS (Apple Silicon)"

    ```bash
    curl -L -o k13d_v1.0.1_darwin_arm64.tar.gz \
      https://github.com/cloudbro-kube-ai/k13d/releases/download/v1.0.1/k13d_v1.0.1_darwin_arm64.tar.gz

    tar -zxvf k13d_v1.0.1_darwin_arm64.tar.gz
    cd k13d_v1.0.1_darwin_arm64
    chmod +x ./k13d

    # Remove quarantine and provenance attributes
    xattr -d com.apple.quarantine ./k13d
    xattr -d com.apple.provenance ./k13d

    ./k13d --web --port 9090 --auth-mode local
    ```

=== "macOS (Intel)"

    ```bash
    curl -L -o k13d_v1.0.1_darwin_amd64.tar.gz \
      https://github.com/cloudbro-kube-ai/k13d/releases/download/v1.0.1/k13d_v1.0.1_darwin_amd64.tar.gz

    tar -zxvf k13d_v1.0.1_darwin_amd64.tar.gz
    cd k13d_v1.0.1_darwin_amd64
    chmod +x ./k13d

    # Remove quarantine and provenance attributes
    xattr -d com.apple.quarantine ./k13d
    xattr -d com.apple.provenance ./k13d

    ./k13d --web --port 9090 --auth-mode local
    ```

=== "Linux (amd64)"

    ```bash
    curl -L -o k13d_v1.0.1_linux_amd64.tar.gz \
      https://github.com/cloudbro-kube-ai/k13d/releases/download/v1.0.1/k13d_v1.0.1_linux_amd64.tar.gz

    tar -zxvf k13d_v1.0.1_linux_amd64.tar.gz
    cd k13d_v1.0.1_linux_amd64
    chmod +x ./k13d

    ./k13d --web --port 9090 --auth-mode local
    ```

=== "Linux (arm64)"

    ```bash
    curl -L -o k13d_v1.0.1_linux_arm64.tar.gz \
      https://github.com/cloudbro-kube-ai/k13d/releases/download/v1.0.1/k13d_v1.0.1_linux_arm64.tar.gz

    tar -zxvf k13d_v1.0.1_linux_arm64.tar.gz
    cd k13d_v1.0.1_linux_arm64
    chmod +x ./k13d

    ./k13d --web --port 9090 --auth-mode local
    ```

=== "Windows (amd64)"

    ```powershell
    curl.exe -L -o k13d_v1.0.1_windows_amd64.zip `
      https://github.com/cloudbro-kube-ai/k13d/releases/download/v1.0.1/k13d_v1.0.1_windows_amd64.zip

    Expand-Archive .\k13d_v1.0.1_windows_amd64.zip -DestinationPath .
    cd .\k13d_v1.0.1_windows_amd64

    .\k13d.exe --web --port 9090 --auth-mode local
    ```

실행 후에는 다음 순서로 들어가면 됩니다.

1. 브라우저에서 `http://localhost:9090` 접속
2. 아이디 `admin` 입력
3. 터미널에 나온 비밀번호 입력

!!! tip "macOS 사용자"
    압축 해제 후에는 아래 두 줄을 꼭 실행하세요.

    ```bash
    xattr -d com.apple.quarantine ./k13d
    xattr -d com.apple.provenance ./k13d
    ```

    `No such xattr` 메시지가 나오면 이미 제거된 상태이므로 무시해도 됩니다.

!!! tip "Windows 사용자"
    SmartScreen 경고가 뜨면 **More info** → **Run anyway** 를 선택하세요.

## TUI로 바로 시작하고 싶다면

기본 TUI 실행 명령은 아래 한 줄입니다.

=== "macOS / Linux"

    ```bash
    ./k13d
    ```

=== "Windows"

    ```powershell
    .\k13d.exe
    ```

## 선택 사항: `kubectl k13d`

`kubectl k13d` 형태로 쓰고 싶다면 같은 릴리즈의 `k13d-plugin` 자산을 받거나 [GitHub Releases 페이지](https://github.com/cloudbro-kube-ai/k13d/releases) 에서 전체 목록을 확인하면 됩니다.

예시: **macOS Apple Silicon**

```bash
curl -L -o k13d-plugin_v1.0.1_darwin_arm64.tar.gz \
  https://github.com/cloudbro-kube-ai/k13d/releases/download/v1.0.1/k13d-plugin_v1.0.1_darwin_arm64.tar.gz

tar -zxvf k13d-plugin_v1.0.1_darwin_arm64.tar.gz
chmod +x ./kubectl-k13d
sudo mv ./kubectl-k13d /usr/local/bin/

kubectl k13d
```

`k13d-plugin` 자산은 **macOS** 와 **Linux** 에서 제공됩니다.

## 다음 문서

- [설정](configuration.md)
- [Web Dashboard](../user-guide/web.md)
- [TUI Dashboard](../user-guide/tui.md)

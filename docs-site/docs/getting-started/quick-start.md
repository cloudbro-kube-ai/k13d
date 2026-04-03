# Quick Start

This is the easiest workshop path for `k13d`:

1. Make sure `kubectl` already works
2. Download the release asset for your OS and CPU
3. Extract it
4. Run the Web UI locally

!!! success "Workshop default"
    Start with the **single binary** from [Release v1.0.1](https://github.com/cloudbro-kube-ai/k13d/releases/tag/v1.0.1).
    If you want to browse every asset first, open the full [GitHub Releases page](https://github.com/cloudbro-kube-ai/k13d/releases).

    - Open `http://localhost:9090`
    - Username: `admin`
    - Password: printed in the terminal when k13d starts

!!! info "What to download"
    Use the main `k13d_v1.0.1_<os>_<arch>` asset for the workshop.
    The `k13d-plugin_v1.0.1_<os>_<arch>` assets are optional and are only needed if you specifically want `kubectl k13d`.

## Most Common Commands

### Web UI

```bash
./k13d --web --port 9090 --auth-mode local
```

### TUI

```bash
./k13d
```

## Before You Start

You only need three things:

- A running Kubernetes cluster
- `kubectl` installed
- A working kubeconfig

Check it first:

```bash
kubectl get nodes
```

If that command works, you are ready.

## Download And Run

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

After the command starts:

1. Open `http://localhost:9090`
2. Log in with username `admin`
3. Copy the password from the terminal

!!! tip "macOS note"
    After extracting the archive, run both `xattr` commands before opening `k13d`.
    If macOS says an attribute does not exist, you can ignore that message.

!!! tip "Windows note"
    If SmartScreen blocks the app, click **More info** and then **Run anyway**.

## Want The TUI Instead?

If you want the terminal dashboard instead of the Web UI, run:

=== "macOS / Linux"

    ```bash
    ./k13d
    ```

=== "Windows"

    ```powershell
    .\k13d.exe
    ```

## Optional: `kubectl k13d`

If you specifically want the kubectl plugin, download the matching `k13d-plugin` asset from the same release or browse the full [GitHub Releases page](https://github.com/cloudbro-kube-ai/k13d/releases).

Example for **macOS Apple Silicon**:

```bash
curl -L -o k13d-plugin_v1.0.1_darwin_arm64.tar.gz \
  https://github.com/cloudbro-kube-ai/k13d/releases/download/v1.0.1/k13d-plugin_v1.0.1_darwin_arm64.tar.gz

tar -zxvf k13d-plugin_v1.0.1_darwin_arm64.tar.gz
chmod +x ./kubectl-k13d
sudo mv ./kubectl-k13d /usr/local/bin/

kubectl k13d
```

`k13d-plugin` assets are available for **macOS** and **Linux**.

## Next Steps

- [Configuration](configuration.md)
- [Web Dashboard](../user-guide/web.md)
- [TUI Dashboard](../user-guide/tui.md)

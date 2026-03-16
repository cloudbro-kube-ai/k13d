# Installation

k13d can be installed in multiple ways depending on your environment and requirements.

!!! warning "Supported today: local binary for TUI and Web UI"
    The only officially supported install path today is the **single binary** running locally with your kubeconfig.
    **Docker, Docker Compose, Kubernetes, Helm, and containerized air-gapped deployment flows are still Beta / in preparation and are not officially supported yet.**
    There is **no official public Docker image repository** for end users yet.

## Prerequisites

- **Go 1.25+** (for building from source)
- **Kubernetes 1.29+** cluster with a valid kubeconfig
- **kubectl** installed and configured

---

## Binary Installation

### Build from Source

The recommended way to install k13d is to build from source:

```bash
# Clone the repository
git clone https://github.com/cloudbro-kube-ai/k13d.git
cd k13d

# Build the binary
make build

# Or build with Go directly
go build -o k13d ./cmd/kube-ai-dashboard-cli/main.go

# Verify installation
./k13d --version
```

### Cross-Platform Builds

Build binaries for multiple platforms:

```bash
# Build for all platforms
make build-all

# Build for specific platforms
make build-linux    # Linux (amd64, arm64, arm)
make build-darwin   # macOS (Intel, Apple Silicon)
make build-windows  # Windows (amd64)
```

### Install to PATH

```bash
# Move binary to a directory in your PATH
sudo mv k13d /usr/local/bin/

# Or add to your PATH in ~/.bashrc or ~/.zshrc
export PATH="$PATH:/path/to/k13d"
```

---

## Docker / Docker Compose

!!! warning "Beta / not supported yet"
    Docker and Docker Compose deployment are still being prepared.
    Do not treat the current repository files as a supported end-user install path yet.
    Use the local binary instead: `./k13d` for TUI or `./k13d --web --auth-mode local` for the Web UI.

---

## Kubernetes / Helm

!!! warning "Beta / not supported yet"
    Kubernetes and Helm deployment are not officially supported in the current release line.
    The manifests and chart-related material in this repository should be treated as **work in progress**.
    For now, please evaluate k13d as a local binary with the TUI or Web UI.

---

## Air-Gapped Installation

!!! info "Current status"
    The **supported** offline story today is still the **single binary** copied into the target environment and run locally.
    Container-image-based air-gapped deployment remains Beta / in preparation.

---

## macOS Gatekeeper

!!! warning "macOS Security Warning"
    macOS may block the binary with *"Apple could not verify k13d is free of malware"*.

### Option 1: Remove Quarantine Attribute

```bash
# Remove quarantine and provenance attributes
xattr -d com.apple.quarantine ./k13d
xattr -d com.apple.provenance ./k13d
```

### Option 2: Allow in System Settings

1. Open **System Settings** → **Privacy & Security**
2. Scroll down to find the blocked app message
3. Click **"Allow Anyway"**
4. Run the binary again and click **"Open"** when prompted

---

## Verifying Installation

```bash
# Check version
./k13d --version

# Run TUI mode (requires valid kubeconfig)
./k13d

# Run web mode with local auth (recommended for desktop use)
./k13d --web --auth-mode local
# Open http://localhost:8080 — Username: admin / Password: printed in terminal

# Token/cluster-oriented auth modes exist in the binary, but their deployment story is still preview-only
./k13d --web --auth-mode token
```

### Authentication Modes

When running the Web server (`--web`), choose an authentication mode:

| Mode | Flag | Description |
|------|------|-------------|
| **Local** | `--auth-mode local` | Supported and recommended for local desktop Web UI use. |
| **Token** | `--auth-mode token` | Preview only. The broader deployment/in-cluster usage story is not officially supported yet. |
| **LDAP** | `--auth-mode ldap` | Preview only. Provider-specific LDAP wiring is still incomplete. |
| **OIDC** | `--auth-mode oidc` | Preview only. Provider-specific OIDC wiring is still incomplete. |
| **No Auth** | `--no-auth` | Disables authentication entirely. **Not recommended** — use only for local testing. |

For LDAP, OIDC, MFA, and SAML guidance, see the [Security guide](../features/security.md).

For local/desktop usage, `--auth-mode local` is the simplest option:

```bash
# With auto-generated password (printed in terminal)
./k13d --web --auth-mode local

# With custom credentials
./k13d --web --auth-mode local --admin-user myadmin --admin-password mysecurepassword

# Or via environment variables
export K13D_USERNAME=myadmin
export K13D_PASSWORD=mysecurepassword
./k13d --web --auth-mode local
```

---

## Next Steps

<div class="grid cards" markdown>

-   :material-rocket-launch:{ .lg .middle } __Quick Start__

    ---

    Learn the basics and run your first commands

    [:octicons-arrow-right-24: Quick Start](quick-start.md)

-   :material-cog:{ .lg .middle } __Configuration__

    ---

    Configure LLM providers and customize settings

    [:octicons-arrow-right-24: Configuration](configuration.md)

-   :material-console:{ .lg .middle } __TUI Guide__

    ---

    Master the terminal interface

    [:octicons-arrow-right-24: TUI Dashboard](../user-guide/tui.md)

</div>

# Homebrew formula for k13d - AI-Powered Kubernetes Dashboard
# Install: brew install cloudbro-kube-ai/tap/k13d
class K13d < Formula
  desc "AI-Powered Kubernetes Dashboard CLI with TUI and Web UI"
  homepage "https://github.com/cloudbro-kube-ai/k13d"
  version "0.7.7"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/cloudbro-kube-ai/k13d/releases/download/v#{version}/k13d_v#{version}_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_DARWIN_AMD64_SHA256"
    end
    on_arm do
      url "https://github.com/cloudbro-kube-ai/k13d/releases/download/v#{version}/k13d_v#{version}_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_DARWIN_ARM64_SHA256"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/cloudbro-kube-ai/k13d/releases/download/v#{version}/k13d_v#{version}_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_AMD64_SHA256"
    end
    on_arm do
      url "https://github.com/cloudbro-kube-ai/k13d/releases/download/v#{version}/k13d_v#{version}_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"
    end
  end

  depends_on "kubectl" => :recommended

  def install
    bin.install "k13d"

    # Install kubectl plugin
    (bin/"kubectl-k13d").write <<~EOS
      #!/bin/sh
      exec "#{bin}/k13d" "$@"
    EOS

    # Generate shell completions
    generate_completions_from_executable(bin/"k13d", "--completion")
  end

  test do
    assert_match "k13d version", shell_output("#{bin}/k13d --version")
  end
end

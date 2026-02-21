#!/bin/bash
# k13d 빌드 후 서비스 배포
set -e

echo "📦 Building k13d..."
cd /Users/youngjukim/Desktop/k13d
make build 2>/dev/null || go build -o build/k13d ./cmd/kubectl-k13d/

echo "📋 Copying to Services..."
cp build/k13d /Users/youngjukim/Services/k13d

echo "🔄 Restarting service..."
launchctl stop com.youngjukim.k13d
sleep 1
launchctl start com.youngjukim.k13d
sleep 3

echo "✅ Done! Status:"
curl -s -o /dev/null -w "HTTP %{http_code}" http://localhost:80
echo

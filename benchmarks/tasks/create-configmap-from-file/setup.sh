#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create the nginx.conf file
cat > /tmp/nginx.conf << 'EOF'
server {
    listen 80;
    server_name localhost;

    location / {
        root /usr/share/nginx/html;
        index index.html;
    }
}
EOF

echo "Setup complete: namespace 'benchmark' and /tmp/nginx.conf are ready"

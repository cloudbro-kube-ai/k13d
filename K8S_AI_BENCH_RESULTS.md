# k8s-ai-bench Evaluation Results

**Date**: 2026-01-23
**Tasks**: 23 (Easy: 6, Medium: 14, Hard: 3)
**Methodology**: Based on [k8s-ai-bench](https://github.com/gke-labs/k8s-ai-bench)

## Summary - Cloud Provider Models

| Rank | Model | Easy | Medium | Hard | Total | Avg Time |
|:----:|-------|------|--------|------|-------|----------|
| 🥇 | **gemini-3-flash** | 6/6 | **14/14** | 3/3 | **100%** | **5.9s** |
| 🥈 | gpt-5-mini | 6/6 | 13/14 | 3/3 | 95.7% | 22.2s |
| 🥈 | solar-pro2 (high) | 6/6 | 13/14 | 3/3 | 95.7% | 8.9s |
| 4 | gpt-5 | 6/6 | 12/14 | 3/3 | 91.3% | 35.4s |
| 4 | gemini-3-pro | 6/6 | 12/14 | 3/3 | 91.3% | 19.1s |
| 6 | o3-mini | 6/6 | 11/14 | 2/3 | 82.6% | 6.0s |

## Summary - Local/Self-hosted Models (Ollama)

| Rank | Model | Easy | Medium | Hard | Total | Avg Time |
|:----:|-------|------|--------|------|-------|----------|
| 🥇 | qwen3:8b | 6/6 | 13/14 | 3/3 | 95.7% | 9.0s |
| 🥈 | gpt-oss:latest | 5/6 | 13/14 | 3/3 | 91.3% | 4.1s |
| 🥈 | deepseek-r1:32b | 6/6 | 12/14 | 3/3 | 91.3% | 13.0s |
| 4 | solar-pro2 (low) | 6/6 | 12/14 | 3/3 | 91.3% | 3.3s |
| 5 | gemma3:27b | 3/6 | 12/14 | 3/3 | 78.3% | 4.0s |
| 6 | gemma3:4b | 3/6 | 10/14 | 2/3 | 65.2% | 1.7s |

## Key Findings

- **gemini-3-flash** achieves **100% accuracy** with the fastest response time (5.9s) - best overall choice!
- **gpt-5-mini** and **solar-pro2 (high)** tie at 95.7% - excellent cloud alternatives
- **qwen3:8b** leads local models at 95.7% with only 8B parameters - best for self-hosting
- Most models struggle with `fix-probes` task (liveness/readiness probe configuration)
- Larger models don't always outperform smaller optimized ones (gpt-5-mini > gpt-5)

## Detailed Results

### qwen3:8b

| Task | Difficulty | Result |
|------|------------|--------|
| create-pod | easy | ✓ |
| create-pod-resources-limits | easy | ✓ |
| fix-pending-pod | easy | ✓ |
| fix-rbac-wrong-resource | easy | ✓ |
| deployment-traffic-switch | easy | ✓ |
| resize-pvc | easy | ✓ |
| fix-crashloop | medium | ✓ |
| fix-image-pull | medium | ✓ |
| fix-probes | medium | ✗ |
| fix-service-routing | medium | ✓ |
| fix-service-with-no-endpoints | medium | ✓ |
| scale-deployment | medium | ✓ |
| scale-down-deployment | medium | ✓ |
| rolling-update-deployment | medium | ✓ |
| create-simple-rbac | medium | ✓ |
| create-network-policy | medium | ✓ |
| debug-app-logs | medium | ✓ |
| create-pod-mount-configmaps | medium | ✓ |
| multi-container-pod-communication | medium | ✓ |
| list-images-for-pods | medium | ✓ |
| horizontal-pod-autoscaler | hard | ✓ |
| create-canary-deployment | hard | ✓ |
| statefulset-lifecycle | hard | ✓ |

### gemma3:4b

| Task | Difficulty | Result |
|------|------------|--------|
| create-pod | easy | ✗ |
| create-pod-resources-limits | easy | ✓ |
| fix-pending-pod | easy | ✓ |
| fix-rbac-wrong-resource | easy | ✓ |
| deployment-traffic-switch | easy | ✗ |
| resize-pvc | easy | ✗ |
| fix-crashloop | medium | ✓ |
| fix-image-pull | medium | ✗ |
| fix-probes | medium | ✗ |
| fix-service-routing | medium | ✗ |
| fix-service-with-no-endpoints | medium | ✓ |
| scale-deployment | medium | ✓ |
| scale-down-deployment | medium | ✓ |
| rolling-update-deployment | medium | ✗ |
| create-simple-rbac | medium | ✓ |
| create-network-policy | medium | ✓ |
| debug-app-logs | medium | ✓ |
| create-pod-mount-configmaps | medium | ✓ |
| multi-container-pod-communication | medium | ✓ |
| list-images-for-pods | medium | ✓ |
| horizontal-pod-autoscaler | hard | ✓ |
| create-canary-deployment | hard | ✓ |
| statefulset-lifecycle | hard | ✗ |

### gemma3:27b

| Task | Difficulty | Result |
|------|------------|--------|
| create-pod | easy | ✗ |
| create-pod-resources-limits | easy | ✓ |
| fix-pending-pod | easy | ✗ |
| fix-rbac-wrong-resource | easy | ✓ |
| deployment-traffic-switch | easy | ✗ |
| resize-pvc | easy | ✓ |
| fix-crashloop | medium | ✓ |
| fix-image-pull | medium | ✓ |
| fix-probes | medium | ✗ |
| fix-service-routing | medium | ✓ |
| fix-service-with-no-endpoints | medium | ✓ |
| scale-deployment | medium | ✓ |
| scale-down-deployment | medium | ✓ |
| rolling-update-deployment | medium | ✗ |
| create-simple-rbac | medium | ✓ |
| create-network-policy | medium | ✓ |
| debug-app-logs | medium | ✓ |
| create-pod-mount-configmaps | medium | ✓ |
| multi-container-pod-communication | medium | ✓ |
| list-images-for-pods | medium | ✓ |
| horizontal-pod-autoscaler | hard | ✓ |
| create-canary-deployment | hard | ✓ |
| statefulset-lifecycle | hard | ✓ |

### gpt-oss:latest

| Task | Difficulty | Result |
|------|------------|--------|
| create-pod | easy | ✗ |
| create-pod-resources-limits | easy | ✓ |
| fix-pending-pod | easy | ✓ |
| fix-rbac-wrong-resource | easy | ✓ |
| deployment-traffic-switch | easy | ✓ |
| resize-pvc | easy | ✓ |
| fix-crashloop | medium | ✓ |
| fix-image-pull | medium | ✓ |
| fix-probes | medium | ✗ |
| fix-service-routing | medium | ✓ |
| fix-service-with-no-endpoints | medium | ✓ |
| scale-deployment | medium | ✓ |
| scale-down-deployment | medium | ✓ |
| rolling-update-deployment | medium | ✓ |
| create-simple-rbac | medium | ✓ |
| create-network-policy | medium | ✓ |
| debug-app-logs | medium | ✓ |
| create-pod-mount-configmaps | medium | ✓ |
| multi-container-pod-communication | medium | ✓ |
| list-images-for-pods | medium | ✓ |
| horizontal-pod-autoscaler | hard | ✓ |
| create-canary-deployment | hard | ✓ |
| statefulset-lifecycle | hard | ✓ |

### deepseek-r1:32b

| Task | Difficulty | Result |
|------|------------|--------|
| create-pod | easy | ✓ |
| create-pod-resources-limits | easy | ✓ |
| fix-pending-pod | easy | ✓ |
| fix-rbac-wrong-resource | easy | ✓ |
| deployment-traffic-switch | easy | ✓ |
| resize-pvc | easy | ✓ |
| fix-crashloop | medium | ✓ |
| fix-image-pull | medium | ✗ |
| fix-probes | medium | ✗ |
| fix-service-routing | medium | ✓ |
| fix-service-with-no-endpoints | medium | ✓ |
| scale-deployment | medium | ✓ |
| scale-down-deployment | medium | ✓ |
| rolling-update-deployment | medium | ✓ |
| create-simple-rbac | medium | ✓ |
| create-network-policy | medium | ✓ |
| debug-app-logs | medium | ✓ |
| create-pod-mount-configmaps | medium | ✓ |
| multi-container-pod-communication | medium | ✓ |
| list-images-for-pods | medium | ✓ |
| horizontal-pod-autoscaler | hard | ✓ |
| create-canary-deployment | hard | ✓ |
| statefulset-lifecycle | hard | ✓ |

### solar-pro2

| Task | Difficulty | Result |
|------|------------|--------|
| create-pod | easy | ✓ |
| create-pod-resources-limits | easy | ✓ |
| fix-pending-pod | easy | ✓ |
| fix-rbac-wrong-resource | easy | ✓ |
| deployment-traffic-switch | easy | ✓ |
| resize-pvc | easy | ✓ |
| fix-crashloop | medium | ✗ |
| fix-image-pull | medium | ✓ |
| fix-probes | medium | ✗ |
| fix-service-routing | medium | ✗ |
| fix-service-with-no-endpoints | medium | ✓ |
| scale-deployment | medium | ✓ |
| scale-down-deployment | medium | ✓ |
| rolling-update-deployment | medium | ✓ |
| create-simple-rbac | medium | ✓ |
| create-network-policy | medium | ✓ |
| debug-app-logs | medium | ✓ |
| create-pod-mount-configmaps | medium | ✓ |
| multi-container-pod-communication | medium | ✓ |
| list-images-for-pods | medium | ✓ |
| horizontal-pod-autoscaler | hard | ✓ |
| create-canary-deployment | hard | ✓ |
| statefulset-lifecycle | hard | ✓ |


# Reports

k13d can generate comprehensive cluster analysis reports in multiple formats.

## Overview

Reports provide:

- **Cluster Health** - Overall cluster status
- **Resource Analysis** - Detailed resource information
- **AI Insights** - AI-powered recommendations
- **Cost Analysis** - Resource cost estimates
- **Security Audit** - Security findings

## Report Types

### Cluster Overview Report

General cluster health and status:

- Node status and capacity
- Workload summary
- Resource utilization
- Recent events
- Health indicators

### Security Audit Report

Security analysis:

- RBAC configuration
- Network policies
- Pod security standards
- Secret management
- CVE scanning results

### Resource Optimization Report

Cost and efficiency analysis:

- Over-provisioned resources
- Unused resources
- Right-sizing recommendations
- Cost estimates

### AI Analysis Report

AI-powered insights:

- Anomaly detection
- Performance recommendations
- Best practice violations
- Predicted issues

## Generating Reports

### TUI Mode

```bash
# Open reports menu
:reports

# Generate specific report
:report cluster-overview
:report security-audit
:report optimization
```

### Web Mode

1. Navigate to **Reports** section
2. Select report type
3. Configure options:
   - Namespace filter
   - Time range
   - Output format
4. Click **Generate**

### CLI Mode

```bash
# Generate cluster overview
k13d report --type cluster-overview

# Generate security audit
k13d report --type security-audit --namespace production

# Generate with AI analysis
k13d report --type optimization --ai-analysis

# Specify output format and location
k13d report --type cluster-overview \
            --format pdf \
            --output ~/reports/cluster-$(date +%Y%m%d).pdf
```

## Output Formats

### Markdown

```bash
k13d report --format markdown
```

Output:
```markdown
# Cluster Overview Report

## Executive Summary
- **Cluster**: production-cluster
- **Nodes**: 5 (5 Ready)
- **Pods**: 127 Running, 3 Pending

## Node Status
| Name | Status | CPU | Memory |
|------|--------|-----|--------|
| node-1 | Ready | 45% | 62% |
...
```

### HTML

```bash
k13d report --format html
```

Generates a styled HTML page with interactive elements.

### PDF

```bash
k13d report --format pdf
```

Generates a professional PDF document.

### JSON

```bash
k13d report --format json
```

Structured data for programmatic processing.

## Report Configuration

### Default Settings

```yaml
# ~/.config/k13d/config.yaml

reports:
  default_format: markdown
  output_path: ~/k13d-reports
  include_ai_analysis: true
  ai_model: gpt-4

  # Default options for each report type
  cluster_overview:
    include_events: true
    event_limit: 100

  security_audit:
    check_rbac: true
    check_network_policies: true
    check_pod_security: true

  optimization:
    include_cost_analysis: true
    cost_model: aws  # aws, gcp, azure, custom
```

### Cost Models

Configure cost estimation:

```yaml
reports:
  optimization:
    cost_model: custom
    custom_costs:
      cpu_per_core_hour: 0.05
      memory_per_gb_hour: 0.01
      storage_per_gb_month: 0.10
```

## Report Sections

### Cluster Overview

```
┌─────────────────────────────────────────┐
│           Cluster Overview               │
├─────────────────────────────────────────┤
│ 1. Executive Summary                     │
│ 2. Node Status                          │
│ 3. Workload Summary                     │
│ 4. Resource Utilization                 │
│ 5. Recent Events                        │
│ 6. Health Indicators                    │
│ 7. Recommendations (AI)                 │
└─────────────────────────────────────────┘
```

### Security Audit

```
┌─────────────────────────────────────────┐
│           Security Audit                 │
├─────────────────────────────────────────┤
│ 1. Executive Summary                     │
│ 2. RBAC Analysis                        │
│    - Overly permissive roles            │
│    - Unused roles                       │
│ 3. Network Policy Coverage              │
│ 4. Pod Security Analysis                │
│    - Privileged containers              │
│    - Host network usage                 │
│ 5. Secret Management                    │
│ 6. Image Security                       │
│ 7. Compliance Status                    │
│ 8. Recommendations                      │
└─────────────────────────────────────────┘
```

### Optimization

```
┌─────────────────────────────────────────┐
│        Resource Optimization             │
├─────────────────────────────────────────┤
│ 1. Cost Summary                         │
│ 2. Over-provisioned Resources           │
│    - CPU: 15 pods using < 10%           │
│    - Memory: 8 pods using < 20%         │
│ 3. Unused Resources                     │
│    - 3 ConfigMaps not referenced        │
│    - 2 Secrets not in use               │
│ 4. Right-sizing Recommendations         │
│ 5. Estimated Savings                    │
│ 6. Action Items                         │
└─────────────────────────────────────────┘
```

## Scheduled Reports

### Cron-based Scheduling

```bash
# Add to crontab
0 8 * * 1 k13d report --type cluster-overview --format pdf --output ~/reports/weekly-$(date +\%Y\%m\%d).pdf
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: k13d-weekly-report
spec:
  schedule: "0 8 * * 1"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: k13d
            image: cloudbro/k13d:latest
            command:
              - k13d
              - report
              - --type=cluster-overview
              - --format=pdf
            volumeMounts:
              - name: reports
                mountPath: /reports
          restartPolicy: OnFailure
          volumes:
            - name: reports
              persistentVolumeClaim:
                claimName: k13d-reports
```

## Email Reports

### Configuration

```yaml
reports:
  email:
    enabled: true
    smtp_host: smtp.example.com
    smtp_port: 587
    username: reports@example.com
    password: ${SMTP_PASSWORD}
    from: "k13d Reports <reports@example.com>"
    to:
      - team@example.com
      - alerts@example.com
```

### Usage

```bash
k13d report --type cluster-overview --email
```

## Report API

### REST API

```bash
# Generate report via API
curl -X POST http://localhost:8080/api/reports \
     -H "Content-Type: application/json" \
     -d '{
       "type": "cluster-overview",
       "format": "json",
       "namespace": "production"
     }'

# Get report status
curl http://localhost:8080/api/reports/{report-id}

# Download report
curl http://localhost:8080/api/reports/{report-id}/download
```

### Response

```json
{
  "id": "report-123",
  "type": "cluster-overview",
  "status": "completed",
  "created_at": "2024-01-15T10:30:00Z",
  "download_url": "/api/reports/report-123/download"
}
```

## Best Practices

### 1. Regular Reporting

Schedule weekly cluster overview and monthly security audits.

### 2. Archive Reports

Keep historical reports for trend analysis:

```yaml
reports:
  retention_days: 90
  archive_path: ~/k13d-archive
```

### 3. Share with Stakeholders

Use PDF format for non-technical stakeholders.

### 4. Act on Recommendations

Review AI recommendations and create action items.

## Troubleshooting

### Report Generation Fails

- Check disk space for output
- Verify namespace access
- Check AI provider connection

### Missing Data

- Verify metrics-server is running
- Check RBAC permissions
- Ensure resources exist

### Slow Generation

- Reduce event limit
- Filter to specific namespaces
- Disable AI analysis for quick reports

## Next Steps

- [Configuration](../getting-started/configuration.md) - Report settings
- [AI Assistant](../concepts/ai-assistant.md) - AI analysis
- [Security](../concepts/security.md) - Security features

# Reports

k13d currently generates cluster reports from the **Web UI**.

## What Reports Include

Reports can include these sections:

- **Nodes**: node readiness, cordon state, pressure warnings, taints, capacity and allocatable values
- **Namespaces**: namespace activity and workload counts
- **Workloads**: pods, deployments, services, and top container images
- **Events**: recent warning events
- **Security**: built-in pod / RBAC / network / privilege signals
- **Security Full**: extended scan when the security scanner is available
- **FinOps**: heuristic compute-cost analysis and rightsizing guidance
- **Metrics**: historical cluster metrics when the collector is enabled
- **AI Analysis**: optional narrative summary from the configured LLM

## Generate A Report

1. Open **Reports** in the Web UI.
2. Select the sections you want.
3. Optionally enable **AI Analysis**.
4. Preview in-browser or download the report.

The selected sections now control the exported HTML/CSV output as well. If you do not select a section, it is omitted from the generated report.

## Output Formats

k13d currently supports:

- **HTML**: best for human-readable reports and browser preview
- **CSV**: tabular export for spreadsheets and follow-up analysis
- **JSON**: raw structured data

There is no standalone `k13d report` CLI command and no built-in PDF or Markdown export in the current binary. For PDF, download **HTML** and use your browser's Print → Save as PDF flow.

## FinOps Notes

The FinOps section is intentionally a **heuristic estimate**, not a cloud invoice.

- It focuses on **compute-style cost signals** from running pod requests.
- If live pod metrics are available, k13d uses them to improve usage and efficiency fields.
- If metrics-server is unavailable, k13d falls back to request-derived estimates and labels the result accordingly.
- Direct provider charges such as control-plane fees, storage classes, egress, committed-use discounts, and reserved capacity are not modeled precisely.

Use the FinOps section as a prioritization tool:

- find namespaces driving the largest share of estimated spend
- identify pods missing requests/limits
- spot underutilized workloads when live metrics exist
- review LoadBalancer sprawl for direct savings opportunities

## Node Health Checks

The node section is meant to be operationally useful, not just inventory.

Each node report includes:

- Ready / NotReady state
- Cordoned (`Unschedulable`) state
- pressure conditions such as `MemoryPressure`, `DiskPressure`, and `PIDPressure`
- network availability warnings
- taints
- capacity and allocatable CPU / memory values

This makes the report usable as both a lightweight cluster assessment and a handoff artifact when a node issue is suspected.

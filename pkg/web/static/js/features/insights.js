// Audit logs and reports
let auditFilter = { onlyLLM: false, onlyErrors: false };

async function showAuditLogs() {
    document.getElementById('audit-modal').classList.add('active');
    // Sync filter checkboxes
    document.getElementById('audit-filter-llm').checked = auditFilter.onlyLLM;
    document.getElementById('audit-filter-errors').checked = auditFilter.onlyErrors;
    loadAuditModalData();
}

function closeAuditModal() {
    document.getElementById('audit-modal').classList.remove('active');
}

async function loadAuditModalData() {
    const body = document.getElementById('audit-modal-body');
    body.innerHTML = '<tr><td colspan="8" style="text-align:center;padding:40px;color:var(--text-secondary);">Loading audit logs...</td></tr>';

    try {
        let params = new URLSearchParams();
        if (auditFilter.onlyLLM) params.append('only_llm', 'true');
        if (auditFilter.onlyErrors) params.append('only_errors', 'true');

        const resp = await fetchWithAuth('/api/audit?' + params.toString());
        if (!resp.ok) {
            const errText = await resp.text();
            throw new Error(errText || `HTTP ${resp.status}`);
        }
        const data = await resp.json();

        document.getElementById('audit-entry-count').textContent =
            `Showing ${data.logs ? data.logs.length : 0} entries`;

        if (data.logs && data.logs.length > 0) {
            body.innerHTML = data.logs.map(log => {
                const isLLM = log.action_type === 'llm' || log.llm_tool;
                const statusBadge = log.success
                    ? '<span style="color: var(--accent-green);">✓</span>'
                    : '<span style="color: var(--accent-red);">✗</span>';
                const actionBadge = getActionBadge(log.action, log.action_type);
                const llmDetails = isLLM && log.llm_tool
                    ? `<div style="margin-top:5px;padding:5px;background:var(--bg-tertiary);border-radius:4px;font-size:11px;">
                                <strong>LLM Tool:</strong> ${escapeHtml(log.llm_tool)}<br>
                                <strong>Command:</strong> <code style="color:var(--accent-yellow);">${escapeHtml(log.llm_command || 'N/A')}</code><br>
                                <strong>Approved:</strong> ${log.llm_approved ? 'Yes' : 'No'}
                                ${log.llm_request ? `<br><strong>Question:</strong> ${escapeHtml(truncateText(log.llm_request, 100))}` : ''}
                              </div>`
                    : '';
                const errorInfo = log.error_msg
                    ? `<div style="color:var(--accent-red);margin-top:3px;font-size:11px;">Error: ${escapeHtml(log.error_msg)}</div>`
                    : '';

                return `
                            <tr style="${!log.success ? 'background: rgba(239,68,68,0.1);' : (isLLM ? 'background: rgba(59,130,246,0.05);' : '')}">
                                <td style="white-space:nowrap;padding:8px 12px;">${formatDateTime(log.timestamp)}</td>
                                <td style="padding:8px 12px;">${escapeHtml(log.user || 'anonymous')}</td>
                                <td style="padding:8px 12px;color:var(--accent-cyan);">${escapeHtml(log.k8s_user || '-')}</td>
                                <td style="padding:8px 12px;">${actionBadge}</td>
                                <td style="padding:8px 12px;">${escapeHtml(log.resource)}</td>
                                <td style="padding:8px 12px;"><span style="padding:2px 6px;border-radius:3px;background:var(--bg-tertiary);font-size:11px;">${escapeHtml(log.source || 'unknown')}</span></td>
                                <td style="text-align:center;padding:8px 12px;">${statusBadge}</td>
                                <td style="padding:8px 12px;">
                                    ${escapeHtml(log.details)}
                                    ${llmDetails}
                                    ${errorInfo}
                                </td>
                            </tr>
                        `;
            }).join('');
        } else {
            body.innerHTML =
                '<tr><td colspan="8" style="text-align:center;padding:40px;color:var(--text-secondary);">No audit logs found</td></tr>';
        }
    } catch (e) {
        console.error('Failed to load audit logs:', e);
        body.innerHTML =
            '<tr><td colspan="8" style="text-align:center;padding:40px;color:var(--accent-red);">Failed to load audit logs</td></tr>';
    }
}

function toggleAuditFilter(filterName) {
    auditFilter[filterName] = !auditFilter[filterName];
    loadAuditModalData();
}

function getActionBadge(action, actionType) {
    const colors = {
        'llm': { bg: 'rgba(59,130,246,0.2)', color: 'var(--accent-blue)', icon: '🤖' },
        'mutation': { bg: 'rgba(234,179,8,0.2)', color: 'var(--accent-yellow)', icon: '⚡' },
        'auth': { bg: 'rgba(139,92,246,0.2)', color: 'var(--accent-purple)', icon: '🔐' },
        'config': { bg: 'rgba(34,197,94,0.2)', color: 'var(--status-running)', icon: '⚙️' }
    };
    const style = colors[actionType] || colors['mutation'];
    return `<span style="padding:2px 8px;border-radius:4px;background:${style.bg};color:${style.color};font-size:12px;">${style.icon} ${escapeHtml(action)}</span>`;
}

function truncateText(text, maxLen) {
    if (!text) return '';
    return text.length > maxLen ? text.substring(0, maxLen) + '...' : text;
}

// ==========================================
// Topology View Functions
// ==========================================

let topologyGraph = null;
let topologyData = null;
let topologySelectedNode = null;
let topologyFocusNodeId = null; // When set, show subgraph around this resource

const topologyStatusColors = {
    running: '#9ece6a',
    pending: '#e0af68',
    failed: '#f7768e',
    succeeded: '#7aa2f7',
    unknown: '#a9b1d6',
    active: '#ff9e64',
};

const topologyKindShapes = {
    Deployment: 'rect',
    ReplicaSet: 'rect',
    StatefulSet: 'rect',
    DaemonSet: 'rect',
    Pod: 'circle',
    Service: 'diamond',
    Ingress: 'diamond',
    Job: 'rect',
    CronJob: 'rect',
    ConfigMap: 'triangle',
    Secret: 'triangle',
    PVC: 'rect',
    HPA: 'diamond',
    NetworkPolicy: 'diamond',
    Namespace: 'rect',
    External: 'triangle',
};

const topologyKindLabels = {
    Deployment: 'Deploy',
    ReplicaSet: 'RS',
    StatefulSet: 'STS',
    DaemonSet: 'DS',
    Pod: 'Pod',
    Service: 'Svc',
    Ingress: 'Ing',
    Job: 'Job',
    CronJob: 'CJ',
    ConfigMap: 'CM',
    Secret: 'Sec',
    PVC: 'PVC',
    HPA: 'HPA',
    NetworkPolicy: 'NetPol',
    Namespace: 'NS',
    External: 'Ext',
};

const topologyEdgeStyles = {
    owns: { lineDash: 0, stroke: '#565f89' },
    selects: { lineDash: [5, 5], stroke: '#7aa2f7' },
    mounts: { lineDash: [2, 4], stroke: '#bb9af7' },
    routes: { lineDash: 0, stroke: '#9ece6a' },
    scales: { lineDash: [8, 4], stroke: '#e0af68' },
    'netpol-select': { lineDash: [3, 3], stroke: '#ff9e64' },
    'netpol-ingress': { lineDash: 0, stroke: '#9ece6a' },
    'netpol-egress': { lineDash: 0, stroke: '#f7768e' },
};

function hideTopologyView() {
    const topoContainer = document.getElementById('topology-container');
    const mainPanel = document.querySelector('.main-panel');
    if (topoContainer) topoContainer.style.display = 'none';
    if (mainPanel) mainPanel.style.display = '';
}

function showTopology() {
    currentResource = 'topology';
    document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
    const topoNav = document.querySelector('.nav-item[data-resource="topology"]');
    if (topoNav) topoNav.classList.add('active');

    // Hide main panel, custom views and overview, show topology
    hideOverviewPanel();
    hideAllCustomViews();
    const mainPanel = document.querySelector('.main-panel');
    const topoContainer = document.getElementById('topology-container');
    if (mainPanel) mainPanel.style.display = 'none';
    if (topoContainer) topoContainer.style.display = 'flex';

    // Sync namespace select
    syncTopologyNamespaces();

    loadTopology();
}

function syncTopologyNamespaces() {
    const srcSelect = document.getElementById('namespace-select');
    const topoSelect = document.getElementById('topology-ns-select');
    if (!srcSelect || !topoSelect) return;

    // Copy options from main namespace select
    topoSelect.innerHTML = '';
    for (const opt of srcSelect.options) {
        const newOpt = document.createElement('option');
        newOpt.value = opt.value;
        newOpt.textContent = opt.textContent;
        topoSelect.appendChild(newOpt);
    }
    topoSelect.value = srcSelect.value;
}

function onTopologyNamespaceChange() {
    loadTopology();
}

async function loadTopology() {
    const namespace = document.getElementById('topology-ns-select')?.value || '';
    const kindFilter = document.getElementById('topology-kind-filter')?.value || '';
    const graphContainer = document.getElementById('topology-graph');
    if (!graphContainer) return;

    try {
        const showNetPol = document.getElementById('topology-show-netpol')?.checked;
        let apiUrl = `/api/topology/?namespace=${encodeURIComponent(namespace)}`;
        if (showNetPol) apiUrl += '&include_netpol=true';
        const resp = await fetchWithAuth(apiUrl);
        const data = await resp.json();
        topologyData = data;

        // Filter based on checkboxes
        const showCM = document.getElementById('topology-show-configmaps')?.checked;
        const showSec = document.getElementById('topology-show-secrets')?.checked;

        let filteredNodes = data.nodes || [];
        let filteredEdges = data.edges || [];

        if (!showCM) {
            const cmIds = new Set(filteredNodes.filter(n => n.kind === 'ConfigMap').map(n => n.id));
            filteredNodes = filteredNodes.filter(n => n.kind !== 'ConfigMap');
            filteredEdges = filteredEdges.filter(e => !cmIds.has(e.source) && !cmIds.has(e.target));
        }
        if (!showSec) {
            const secIds = new Set(filteredNodes.filter(n => n.kind === 'Secret').map(n => n.id));
            filteredNodes = filteredNodes.filter(n => n.kind !== 'Secret');
            filteredEdges = filteredEdges.filter(e => !secIds.has(e.source) && !secIds.has(e.target));
        }

        // Kind filter: show selected kind + all connected resources
        if (kindFilter) {
            const kindNodeIds = new Set(filteredNodes.filter(n => n.kind === kindFilter).map(n => n.id));
            const connectedIds = new Set(kindNodeIds);
            filteredEdges.forEach(e => {
                if (kindNodeIds.has(e.source)) connectedIds.add(e.target);
                if (kindNodeIds.has(e.target)) connectedIds.add(e.source);
            });
            filteredNodes = filteredNodes.filter(n => connectedIds.has(n.id));
            filteredEdges = filteredEdges.filter(e => connectedIds.has(e.source) && connectedIds.has(e.target));
        }

        // Resource focus: show subgraph reachable from the focused resource
        if (topologyFocusNodeId) {
            const focusResult = extractSubgraph(filteredNodes, filteredEdges, topologyFocusNodeId);
            filteredNodes = focusResult.nodes;
            filteredEdges = focusResult.edges;
        }

        renderTopologyGraph(filteredNodes, filteredEdges);

        // After rendering, highlight the focused node
        if (topologyFocusNodeId && topologyGraph) {
            try {
                topologyGraph.setElementState(topologyFocusNodeId, ['selected']);
            } catch (e) { /* node may not exist */ }
        }
    } catch (err) {
        graphContainer.innerHTML = `<div style="display:flex;align-items:center;justify-content:center;height:100%;color:var(--accent-red);">Failed to load topology: ${escapeHtml(err.message)}</div>`;
    }
}

// Extract the connected subgraph reachable from a root node (BFS in both directions)
function extractSubgraph(nodes, edges, rootId) {
    const visited = new Set([rootId]);
    const queue = [rootId];
    // Walk up: find ancestors (who owns/selects/routes to this node)
    // Walk down: find descendants (what this node owns/selects/routes)
    while (queue.length > 0) {
        const current = queue.shift();
        edges.forEach(e => {
            if (e.source === current && !visited.has(e.target)) {
                visited.add(e.target);
                queue.push(e.target);
            }
            if (e.target === current && !visited.has(e.source)) {
                visited.add(e.source);
                queue.push(e.source);
            }
        });
    }
    return {
        nodes: nodes.filter(n => visited.has(n.id)),
        edges: edges.filter(e => visited.has(e.source) && visited.has(e.target)),
    };
}

function renderTopologyGraph(nodes, edges) {
    const container = document.getElementById('topology-graph');
    if (!container) return;

    // Destroy previous graph
    if (topologyGraph) {
        topologyGraph.destroy();
        topologyGraph = null;
    }

    if (!nodes || nodes.length === 0) {
        container.innerHTML = '<div style="display:flex;align-items:center;justify-content:center;height:100%;color:var(--text-secondary);">No resources found in this namespace</div>';
        return;
    }

    // Clear container
    container.innerHTML = '';

    // Read theme colors from CSS variables for canvas rendering
    const cs = getComputedStyle(document.documentElement);
    const labelColor = cs.getPropertyValue('--text-primary').trim() || '#c0caf5';
    const edgeDimColor = cs.getPropertyValue('--text-muted').trim() || '#565f89';

    // Transform data for G6
    const g6Nodes = nodes.map(n => ({
        id: n.id,
        data: {
            kind: n.kind,
            name: n.name,
            namespace: n.namespace,
            status: n.status,
            info: n.info || {},
            kindLabel: topologyKindLabels[n.kind] || n.kind,
        },
    }));

    const g6Edges = edges.map((e, i) => ({
        id: `edge-${i}`,
        source: e.source,
        target: e.target,
        data: { type: e.type },
    }));

    const statusColor = (status) => topologyStatusColors[status] || topologyStatusColors.unknown;

    topologyGraph = new G6.Graph({
        container,
        autoFit: 'view',
        padding: [40, 40, 40, 40],
        data: { nodes: g6Nodes, edges: g6Edges },
        node: {
            type: (d) => {
                const shape = topologyKindShapes[d.data?.kind] || 'circle';
                return shape;
            },
            style: {
                size: (d) => {
                    const kind = d.data?.kind;
                    if (kind === 'Pod') return 36;
                    if (kind === 'Service' || kind === 'Ingress' || kind === 'HPA') return 40;
                    if (kind === 'ConfigMap' || kind === 'Secret') return 36;
                    return [110, 44]; // rect: wider for label text
                },
                fill: (d) => {
                    const color = statusColor(d.data?.status);
                    return color + '33'; // 20% opacity
                },
                stroke: (d) => statusColor(d.data?.status),
                lineWidth: 2,
                labelText: (d) => {
                    const label = d.data?.kindLabel || '';
                    const name = d.data?.name || '';
                    const kind = d.data?.kind;
                    // Shorter truncation for non-rect shapes
                    const isCompact = (kind === 'Pod' || kind === 'Service' || kind === 'Ingress' || kind === 'HPA' || kind === 'ConfigMap' || kind === 'Secret');
                    const maxLen = isCompact ? 10 : 14;
                    const shortName = name.length > maxLen ? name.substring(0, maxLen - 2) + '..' : name;
                    return isCompact ? `${label}\n${shortName}` : `${label}: ${shortName}`;
                },
                labelFill: labelColor,
                labelFontSize: (d) => {
                    const kind = d.data?.kind;
                    const isCompact = (kind === 'Pod' || kind === 'ConfigMap' || kind === 'Secret');
                    return isCompact ? 9 : 10;
                },
                labelFontFamily: 'SF Mono, Monaco, Consolas, Liberation Mono, monospace',
                labelPlacement: (d) => {
                    // Place label below for small shapes so text doesn't overflow
                    const kind = d.data?.kind;
                    if (kind === 'Pod' || kind === 'Service' || kind === 'Ingress' || kind === 'HPA' || kind === 'ConfigMap' || kind === 'Secret') return 'bottom';
                    return 'center';
                },
                labelMaxLines: 2,
                labelWordWrap: true,
                labelWordWrapWidth: 100,
                labelOffsetY: (d) => {
                    const kind = d.data?.kind;
                    if (kind === 'Pod' || kind === 'Service' || kind === 'Ingress' || kind === 'HPA' || kind === 'ConfigMap' || kind === 'Secret') return 8;
                    return 0;
                },
            },
            state: {
                highlight: {
                    stroke: '#7aa2f7',
                    lineWidth: 3,
                    shadowColor: '#7aa2f7',
                    shadowBlur: 10,
                },
                dim: {
                    opacity: 0.3,
                },
                selected: {
                    stroke: '#7dcfff',
                    lineWidth: 3,
                    shadowColor: '#7dcfff',
                    shadowBlur: 12,
                },
            },
        },
        edge: {
            type: 'line',
            style: {
                stroke: (d) => {
                    const es = topologyEdgeStyles[d.data?.type] || topologyEdgeStyles.owns;
                    return es.stroke;
                },
                lineDash: (d) => {
                    const es = topologyEdgeStyles[d.data?.type] || topologyEdgeStyles.owns;
                    return es.lineDash || 0;
                },
                lineWidth: 1,
                endArrow: true,
                endArrowSize: 6,
            },
            state: {
                dim: {
                    opacity: 0.15,
                },
            },
        },
        layout: {
            type: 'dagre',
            rankdir: 'TB',
            nodesep: 50,
            ranksep: 70,
        },
        behaviors: [
            'drag-canvas',
            'zoom-canvas',
            'click-select',
        ],
    });

    // Event: node click → show detail
    topologyGraph.on('node:click', (evt) => {
        const nodeId = evt.target.id;
        const nodeData = nodes.find(n => n.id === nodeId);
        if (nodeData) {
            showTopologyDetail(nodeData);
        }
    });

    // Event: double click → navigate to dashboard
    topologyGraph.on('node:dblclick', (evt) => {
        const nodeId = evt.target.id;
        const nodeData = nodes.find(n => n.id === nodeId);
        if (nodeData) {
            topologyNavigateToDashboardForNode(nodeData);
        }
    });

    // Event: canvas click → close detail
    topologyGraph.on('canvas:click', () => {
        closeTopologyDetail();
    });

    topologyGraph.render();

    // Force fit-view after render to reset zoom/pan state.
    // G6 autoFit:'view' may not reliably reset viewport when
    // re-creating graphs in the same container (e.g. switching
    // from focused subgraph back to full topology).
    requestAnimationFrame(() => {
        if (topologyGraph) {
            try { topologyGraph.fitView(); } catch (e) { }
        }
    });
}

function showTopologyDetail(nodeData) {
    topologySelectedNode = nodeData;
    const detail = document.getElementById('topology-detail');
    const title = document.getElementById('topology-detail-title');
    const body = document.getElementById('topology-detail-body');

    if (!detail || !body) return;

    title.textContent = `${nodeData.kind}: ${nodeData.name}`;

    let html = '';
    html += `<div class="topology-detail-field">
                <div class="label">Kind</div>
                <div class="value">${escapeHtml(nodeData.kind)}</div>
            </div>`;
    html += `<div class="topology-detail-field">
                <div class="label">Name</div>
                <div class="value">${escapeHtml(nodeData.name)}</div>
            </div>`;
    html += `<div class="topology-detail-field">
                <div class="label">Namespace</div>
                <div class="value">${escapeHtml(nodeData.namespace)}</div>
            </div>`;
    html += `<div class="topology-detail-field">
                <div class="label">Status</div>
                <div class="value"><span class="topology-status-badge ${nodeData.status}">${escapeHtml(nodeData.status)}</span></div>
            </div>`;

    // Show info fields
    if (nodeData.info) {
        for (const [key, val] of Object.entries(nodeData.info)) {
            html += `<div class="topology-detail-field">
                        <div class="label">${escapeHtml(key)}</div>
                        <div class="value">${escapeHtml(val)}</div>
                    </div>`;
        }
    }

    // Show connections
    if (topologyData) {
        const incoming = (topologyData.edges || []).filter(e => e.target === nodeData.id);
        const outgoing = (topologyData.edges || []).filter(e => e.source === nodeData.id);
        if (incoming.length > 0) {
            html += `<div class="topology-detail-field">
                        <div class="label">Incoming (${incoming.length})</div>
                        <div class="value" style="font-size:11px;">`;
            for (const e of incoming) {
                html += `${escapeHtml(e.source)} <span style="color:var(--text-secondary);">(${e.type})</span><br>`;
            }
            html += `</div></div>`;
        }
        if (outgoing.length > 0) {
            html += `<div class="topology-detail-field">
                        <div class="label">Outgoing (${outgoing.length})</div>
                        <div class="value" style="font-size:11px;">`;
            for (const e of outgoing) {
                html += `${escapeHtml(e.target)} <span style="color:var(--text-secondary);">(${e.type})</span><br>`;
            }
            html += `</div></div>`;
        }
    }

    body.innerHTML = html;
    detail.classList.add('active');
}

function closeTopologyDetail() {
    const detail = document.getElementById('topology-detail');
    if (detail) detail.classList.remove('active');
    topologySelectedNode = null;
}

function topologyNavigateToDashboard() {
    if (!topologySelectedNode) return;
    topologyNavigateToDashboardForNode(topologySelectedNode);
}

function topologyNavigateToDashboardForNode(nodeData) {
    const kindToResource = {
        Pod: 'pods',
        Deployment: 'deployments',
        ReplicaSet: 'replicasets',
        StatefulSet: 'statefulsets',
        DaemonSet: 'daemonsets',
        Service: 'services',
        Ingress: 'ingresses',
        Job: 'jobs',
        CronJob: 'cronjobs',
        ConfigMap: 'configmaps',
        Secret: 'secrets',
        PVC: 'persistentvolumeclaims',
        HPA: 'hpas',
    };
    const resource = kindToResource[nodeData.kind];
    if (resource) {
        // Set namespace if available
        if (nodeData.namespace) {
            const nsSelect = document.getElementById('namespace-select');
            if (nsSelect) nsSelect.value = nodeData.namespace;
            currentNamespace = nodeData.namespace;
        }
        switchResource(resource);
    }
}

function topologyFitView() {
    if (topologyGraph) {
        topologyGraph.fitView();
    }
}

// Show topology focused on a specific resource (called from dashboard Topo button)
function showTopologyForResource(kind, name, namespace) {
    topologyFocusNodeId = `${kind}/${namespace}/${name}`;

    // Set namespace in topology view
    const nsSelect = document.getElementById('topology-ns-select');
    if (nsSelect) nsSelect.value = namespace || '';

    // Clear kind filter when focusing on a specific resource
    const kindFilter = document.getElementById('topology-kind-filter');
    if (kindFilter) kindFilter.value = '';

    showTopology();
}

// Clear the focused resource and show the full topology
function clearTopologyFocus() {
    topologyFocusNodeId = null;
    const kindFilter = document.getElementById('topology-kind-filter');
    if (kindFilter) kindFilter.value = '';
    loadTopology();
}

function filterTopologyGraph(query) {
    if (!topologyGraph || !topologyData) return;

    const q = query.toLowerCase().trim();
    if (!q) {
        // Clear filter: reset all states
        topologyGraph.getNodeData().forEach(n => {
            topologyGraph.setElementState(n.id, []);
        });
        topologyGraph.getEdgeData().forEach(e => {
            topologyGraph.setElementState(e.id, []);
        });
        topologyGraph.draw();
        return;
    }

    const matchedIds = new Set();
    (topologyData.nodes || []).forEach(n => {
        if (n.name.toLowerCase().includes(q) || n.kind.toLowerCase().includes(q)) {
            matchedIds.add(n.id);
        }
    });

    topologyGraph.getNodeData().forEach(n => {
        topologyGraph.setElementState(n.id, matchedIds.has(n.id) ? ['highlight'] : ['dim']);
    });
    topologyGraph.getEdgeData().forEach(e => {
        const connected = matchedIds.has(e.source) || matchedIds.has(e.target);
        topologyGraph.setElementState(e.id, connected ? [] : ['dim']);
    });
    topologyGraph.draw();
}

async function showReports() {
    document.getElementById('reports-modal').classList.add('active');
    // Reset status/preview on open
    document.getElementById('report-status').innerHTML = '';
    document.getElementById('report-preview').innerHTML = '';
}

function closeReportsModal() {
    document.getElementById('reports-modal').classList.remove('active');
}

// Build sections query string from report checkboxes
function getReportSections() {
    const mapping = {
        'report-sec-workloads': 'workloads',
        'report-sec-nodes': 'nodes,namespaces',
        'report-sec-security': 'security',
        'report-sec-trivy': 'security_full',
        'report-sec-finops': 'finops',
        'report-sec-events': 'events',
        'report-sec-metrics': 'metrics',
    };
    const parts = [];
    for (const [id, value] of Object.entries(mapping)) {
        if (document.getElementById(id)?.checked) parts.push(value);
    }
    return parts.join(',');
}

function getReportIncludeAI() {
    return document.getElementById('report-sec-ai')?.checked ?? false;
}

function reportSelectAll() {
    document.querySelectorAll('[id^="report-sec-"]').forEach(cb => cb.checked = true);
}

function reportSelectNone() {
    document.querySelectorAll('[id^="report-sec-"]').forEach(cb => cb.checked = false);
}

// Preview report in new window
async function previewReport() {
    const includeAI = getReportIncludeAI();
    const sections = getReportSections();
    const statusEl = document.getElementById('report-status');

    if (!sections && !includeAI) {
        statusEl.innerHTML = `<div style="color: var(--accent-yellow);">Please select at least one section.</div>`;
        return;
    }

    if (includeAI && !llmConnected) {
        statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    AI is not connected. Please configure LLM settings first, or uncheck "AI Analysis".
                </div>`;
        return;
    }

    statusEl.innerHTML = `<div style="color: var(--accent-blue);">
                <span class="loading-dots"><span></span><span></span><span></span></span>
                Generating report preview... This may take a moment.
            </div>`;

    try {
        const url = `/api/reports/preview?ai=${includeAI}&sections=${encodeURIComponent(sections)}`;
        const resp = await fetchWithAuth(url);

        if (!resp.ok) throw new Error('Failed to generate report');

        const html = await resp.text();

        // Show preview inline using an iframe (avoids popup blocker issues)
        const previewEl = document.getElementById('report-preview');
        const iframe = document.createElement('iframe');
        iframe.style.cssText = 'width:100%;height:600px;border:1px solid var(--border-color);border-radius:8px;background:#fff;';
        iframe.sandbox = 'allow-same-origin';
        previewEl.innerHTML = '';
        previewEl.appendChild(iframe);
        iframe.contentDocument.open();
        iframe.contentDocument.write(html);
        iframe.contentDocument.close();

        statusEl.innerHTML = `<div style="color: var(--accent-green);">
                    Report preview generated below
                </div>`;
    } catch (e) {
        statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    Failed to generate preview: ${e.message}
                </div>`;
    }
}

// Download report
async function downloadReport(format) {
    const includeAI = getReportIncludeAI();
    const sections = getReportSections();
    const statusEl = document.getElementById('report-status');

    if (!sections && !includeAI) {
        statusEl.innerHTML = `<div style="color: var(--accent-yellow);">Please select at least one section.</div>`;
        return;
    }

    if (includeAI && !llmConnected) {
        statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    AI is not connected. Please configure LLM settings first, or uncheck "AI Analysis".
                </div>`;
        return;
    }

    statusEl.innerHTML = `<div style="color: var(--accent-blue);">
                <span class="loading-dots"><span></span><span></span><span></span></span>
                Generating ${format.toUpperCase()} report...
            </div>`;

    try {
        const url = `/api/reports?format=${format}&ai=${includeAI}&download=true&sections=${encodeURIComponent(sections)}`;
        const resp = await fetchWithAuth(url);

        if (!resp.ok) throw new Error('Failed to generate report');

        const blob = await resp.blob();
        const filename = resp.headers.get('Content-Disposition')?.match(/filename=(.+)/)?.[1]
            || `k13d-report-${new Date().toISOString().slice(0, 10)}.${format}`;

        // Trigger download
        const downloadUrl = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = downloadUrl;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(downloadUrl);

        statusEl.innerHTML = `<div style="color: var(--accent-green);">
                    ✓ Report downloaded: ${filename}
                </div>`;

        if (format === 'html') {
            document.getElementById('report-preview').innerHTML = `
                        <p style="color: var(--text-secondary); margin-top: 10px;">
                            💡 <strong>Tip:</strong> Open the HTML file in your browser and use Print → Save as PDF to create a PDF version.
                        </p>
                    `;
        }
    } catch (e) {
        statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    ✕ Failed to download report: ${e.message}
                </div>`;
    }
}

async function generateReport(format) {
    const includeAI = getReportIncludeAI();
    const sections = getReportSections();
    const statusEl = document.getElementById('report-status');
    const previewEl = document.getElementById('report-preview');

    if (!sections && !includeAI) {
        statusEl.innerHTML = `<div style="color: var(--accent-yellow);">Please select at least one section.</div>`;
        return;
    }

    if (includeAI && !llmConnected) {
        statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    AI is not connected. Please configure LLM settings first, or uncheck "AI Analysis".
                </div>`;
        return;
    }

    statusEl.innerHTML = `<div style="color: var(--accent-blue);">
                <span class="loading-dots"><span></span><span></span><span></span></span>
                Generating report... This may take a moment.
            </div>`;
    previewEl.innerHTML = '';

    try {
        const url = `/api/reports?format=${format}&ai=${includeAI}&sections=${encodeURIComponent(sections)}`;

        if (format === 'json') {
            // View JSON in preview
            const resp = await fetchWithAuth(url);
            const report = await resp.json();

            statusEl.innerHTML = `<div style="color: var(--accent-green);">
                        ✓ Report generated successfully at ${formatDateTime(report.generated_at)}
                    </div>`;

            // Calculate total potential savings
            const totalSavings = (report.finops_analysis?.cost_optimizations || [])
                .reduce((sum, opt) => sum + (opt.estimated_saving || 0), 0);

            // Show summary with FinOps
            previewEl.innerHTML = `
                        <div style="background: var(--bg-tertiary); padding: 20px; border-radius: 8px; margin-top: 20px;">
                            <h3 style="margin-bottom: 15px;">📈 Report Summary</h3>
                            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px;">
                                <div style="background: var(--bg-secondary); padding: 15px; border-radius: 6px; text-align: center;">
                                    <div style="font-size: 22px; font-weight: bold; color: var(--accent-blue);">${report.node_summary?.total || 0}</div>
                                    <div style="font-size: 11px; color: var(--text-secondary);">Nodes (${report.node_summary?.ready || 0} Ready)</div>
                                </div>
                                <div style="background: var(--bg-secondary); padding: 15px; border-radius: 6px; text-align: center;">
                                    <div style="font-size: 22px; font-weight: bold; color: var(--accent-green);">${report.workloads?.total_pods || 0}</div>
                                    <div style="font-size: 11px; color: var(--text-secondary);">Pods (${report.workloads?.running_pods || 0} Running)</div>
                                </div>
                                <div style="background: var(--bg-secondary); padding: 15px; border-radius: 6px; text-align: center;">
                                    <div style="font-size: 22px; font-weight: bold; color: var(--accent-purple);">${report.workloads?.total_deployments || 0}</div>
                                    <div style="font-size: 11px; color: var(--text-secondary);">Deployments</div>
                                </div>
                                <div style="background: var(--bg-secondary); padding: 15px; border-radius: 6px; text-align: center;">
                                    <div style="font-size: 22px; font-weight: bold; color: ${report.health_score >= 90 ? 'var(--accent-green)' : report.health_score >= 70 ? 'var(--accent-yellow)' : 'var(--accent-red)'};">${Math.round(report.health_score || 0)}%</div>
                                    <div style="font-size: 11px; color: var(--text-secondary);">Health Score</div>
                                </div>
                            </div>

                            <!-- FinOps Section -->
                            <div style="margin-top: 25px; background: linear-gradient(135deg, #1a472a 0%, #2d5a3d 100%); padding: 20px; border-radius: 8px; border: 1px solid #4caf50;">
                                <h3 style="margin-bottom: 15px; color: #9ece6a;">💰 FinOps Cost Analysis</h3>
                                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin-bottom: 15px;">
                                    <div style="background: rgba(0,0,0,0.3); padding: 15px; border-radius: 6px; text-align: center;">
                                        <div style="font-size: 24px; font-weight: bold; color: #9ece6a;">$${(report.finops_analysis?.total_estimated_monthly_cost || 0).toFixed(2)}</div>
                                        <div style="font-size: 11px; color: var(--text-secondary);">Est. Monthly Cost</div>
                                    </div>
                                    <div style="background: rgba(0,0,0,0.3); padding: 15px; border-radius: 6px; text-align: center;">
                                        <div style="font-size: 24px; font-weight: bold; color: #7dcfff;">${(report.finops_analysis?.resource_efficiency?.cpu_requests_vs_capacity || 0).toFixed(1)}%</div>
                                        <div style="font-size: 11px; color: var(--text-secondary);">CPU Utilization</div>
                                    </div>
                                    <div style="background: rgba(0,0,0,0.3); padding: 15px; border-radius: 6px; text-align: center;">
                                        <div style="font-size: 24px; font-weight: bold; color: #bb9af7;">${(report.finops_analysis?.resource_efficiency?.memory_requests_vs_capacity || 0).toFixed(1)}%</div>
                                        <div style="font-size: 11px; color: var(--text-secondary);">Memory Utilization</div>
                                    </div>
                                    <div style="background: rgba(0,0,0,0.3); padding: 15px; border-radius: 6px; text-align: center;">
                                        <div style="font-size: 24px; font-weight: bold; color: #f7768e;">$${totalSavings.toFixed(2)}</div>
                                        <div style="font-size: 11px; color: var(--text-secondary);">Potential Savings/mo</div>
                                    </div>
                                </div>

                                ${(report.finops_analysis?.cost_optimizations || []).length > 0 ? `
                                    <h4 style="margin: 15px 0 10px 0; color: #e0af68;">⚡ Cost Optimization Recommendations</h4>
                                    <div style="max-height: 200px; overflow-y: auto;">
                                        ${(report.finops_analysis?.cost_optimizations || []).slice(0, 5).map(opt => `
                                            <div style="background: rgba(0,0,0,0.2); padding: 10px; border-radius: 4px; margin-bottom: 8px; border-left: 3px solid ${opt.priority === 'high' ? '#f7768e' : opt.priority === 'medium' ? '#e0af68' : '#9ece6a'};">
                                                <div style="display: flex; justify-content: space-between; align-items: center;">
                                                    <span style="font-weight: bold; color: ${opt.priority === 'high' ? '#f7768e' : opt.priority === 'medium' ? '#e0af68' : '#9ece6a'};">[${opt.priority.toUpperCase()}] ${escapeHtml(opt.category)}</span>
                                                    <span style="color: #9ece6a; font-weight: bold;">Save $${(opt.estimated_saving || 0).toFixed(2)}/mo</span>
                                                </div>
                                                <div style="font-size: 12px; margin-top: 5px; color: var(--text-secondary);">${escapeHtml(opt.description)}</div>
                                            </div>
                                        `).join('')}
                                    </div>
                                ` : '<p style="color: var(--text-secondary);">No optimization recommendations at this time.</p>'}
                            </div>

                            ${report.ai_analysis ? `
                                <div style="margin-top: 20px;">
                                    <h4 style="margin-bottom: 10px;">🤖 AI Analysis with FinOps Insights</h4>
                                    <div style="background: var(--bg-primary); padding: 15px; border-radius: 6px; white-space: pre-wrap; font-size: 13px; max-height: 300px; overflow-y: auto; border-left: 3px solid var(--accent-blue);">
                                        ${escapeHtml(report.ai_analysis)}
                                    </div>
                                </div>
                            ` : ''}

                            <div style="margin-top: 20px;">
                                <h4 style="margin-bottom: 10px;">📊 Cost by Namespace (Top 5)</h4>
                                <table style="width: 100%; font-size: 12px;">
                                    <tr style="background: var(--bg-secondary);"><th style="padding: 8px;">Namespace</th><th style="padding: 8px;">Pods</th><th style="padding: 8px;">CPU</th><th style="padding: 8px;">Memory</th><th style="padding: 8px;">Est. Cost</th></tr>
                                    ${(report.finops_analysis?.cost_by_namespace || []).slice(0, 5).map(ns => `
                                        <tr><td style="padding: 8px;">${escapeHtml(ns.namespace)}</td><td style="padding: 8px;">${ns.pod_count}</td><td style="padding: 8px;">${escapeHtml(ns.cpu_requests)}</td><td style="padding: 8px;">${escapeHtml(ns.memory_requests)}</td><td style="padding: 8px;">$${(ns.estimated_cost || 0).toFixed(2)}</td></tr>
                                    `).join('')}
                                </table>
                            </div>

                            <div style="margin-top: 20px;">
                                <h4 style="margin-bottom: 10px;">🐳 Top Container Images</h4>
                                <table style="width: 100%; font-size: 12px;">
                                    <tr style="background: var(--bg-secondary);"><th style="padding: 8px;">Image</th><th style="padding: 8px;">Tag</th><th style="padding: 8px;">Pods</th></tr>
                                    ${(report.images || []).slice(0, 8).map(img => `
                                        <tr><td style="padding: 8px;">${escapeHtml(img.repository)}</td><td style="padding: 8px;">${escapeHtml(img.tag)}</td><td style="padding: 8px;">${img.pod_count}</td></tr>
                                    `).join('')}
                                </table>
                            </div>
                        </div>
                    `;
        }
    } catch (e) {
        statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    ✕ Failed to generate report: ${e.message}
                </div>`;
    }
}

// Note: Auto-refresh is now handled by startAutoRefresh() in init()
// with user-configurable interval settings

// Global search across all resources
let searchTimeout = null;
let searchSelectedIndex = -1;
let searchResults = [];


// ============================
// Cluster Visualizer View
// k8s-diagram.netlify.app Style
// ============================

let cvData = { nodes: [], pods: [], services: [], deployments: [], statefulsets: [], daemonsets: [], namespaces: [], ingresses: [] };
let cvTrafficActive = true;
let cvSelectedId = null;
let cvNsFilter = new Set();
let cvSearchTerm = '';

function showClusterVisualizer() {
    showCustomView('cluster-viz-container', 'cluster-viz');
    loadClusterVizData();
}

async function loadClusterVizData() {
    const body = document.getElementById('cv-canvas-body');
    if (!body) return;
    body.innerHTML = '<div style="display:flex;align-items:center;justify-content:center;height:100%;color:var(--cv-text-muted);">Loading cluster data...</div>';
    try {
        const [nodesRes, podsRes, svcsRes, depsRes, stsRes, dsRes, ingsRes] = await Promise.all([
            fetchWithAuth('/api/k8s/nodes'),
            fetchWithAuth('/api/k8s/pods'),
            fetchWithAuth('/api/k8s/services'),
            fetchWithAuth('/api/k8s/deployments'),
            fetchWithAuth('/api/k8s/statefulsets'),
            fetchWithAuth('/api/k8s/daemonsets'),
            fetchWithAuth('/api/k8s/ingresses')
        ]);
        cvData.nodes = (await nodesRes.json()).items || [];
        cvData.pods = (await podsRes.json()).items || [];
        cvData.services = (await svcsRes.json()).items || [];
        cvData.deployments = (await depsRes.json()).items || [];
        cvData.statefulsets = (await stsRes.json()).items || [];
        cvData.daemonsets = (await dsRes.json()).items || [];
        cvData.ingresses = (await ingsRes.json()).items || [];

        const nsSet = new Set();
        cvData.pods.forEach(p => nsSet.add(p.namespace));
        cvData.services.forEach(s => nsSet.add(s.namespace));
        cvData.deployments.forEach(d => nsSet.add(d.namespace));
        cvData.statefulsets.forEach(s => nsSet.add(s.namespace));
        cvData.daemonsets.forEach(d => nsSet.add(d.namespace));
        cvData.namespaces = [...nsSet].sort();
        cvNsFilter = new Set(cvData.namespaces);

        renderCvNsFilters();
        renderCvMetrics();
        renderCvDiagram();
    } catch (e) {
        body.innerHTML = `<div style="display:flex;align-items:center;justify-content:center;height:100%;color:#f87171;">Failed to load: ${escapeHtml(e.message)}</div>`;
    }
}

function renderCvNsFilters() {
    const el = document.getElementById('cv-ns-filters');
    if (!el) return;
    el.innerHTML = cvData.namespaces.map(ns => `
        <label class="cv-namespace-toggle">
            <input type="checkbox" checked onchange="cvToggleNs('${escapeHtml(ns)}')">
            <span>${escapeHtml(ns)}</span>
        </label>
    `).join('');
}

function cvToggleNs(ns) {
    if (cvNsFilter.has(ns)) cvNsFilter.delete(ns);
    else cvNsFilter.add(ns);
    renderCvDiagram();
}

function renderCvMetrics() {
    const cpuEl = document.getElementById('cv-cpu-val');
    const memEl = document.getElementById('cv-mem-val');
    const cpuBar = document.getElementById('cv-cpu-bar');
    const memBar = document.getElementById('cv-mem-bar');
    if (cpuEl) cpuEl.textContent = cvData.nodes.length + ' node(s)';
    if (memEl) memEl.textContent = cvData.pods.length + ' pod(s)';
    if (cpuBar) cpuBar.style.width = Math.min(100, cvData.nodes.length * 33) + '%';
    if (memBar) memBar.style.width = Math.min(100, cvData.pods.length * 5) + '%';
}

function renderCvDiagram() {
    const body = document.getElementById('cv-canvas-body');
    if (!body) return;

    const filteredPods = cvData.pods.filter(p => cvNsFilter.has(p.namespace));
    const filteredSvcs = cvData.services.filter(s => cvNsFilter.has(s.namespace));
    const filteredDeps = cvData.deployments.filter(d => cvNsFilter.has(d.namespace));
    const filteredSts = cvData.statefulsets.filter(s => cvNsFilter.has(s.namespace));
    const filteredDs = cvData.daemonsets.filter(d => cvNsFilter.has(d.namespace));
    const filteredIngs = cvData.ingresses.filter(i => cvNsFilter.has(i.namespace));

    const externalSvcs = filteredSvcs.filter(s => s.type === 'LoadBalancer' || s.type === 'NodePort');
    const internalSvcs = filteredSvcs.filter(s => s.type === 'ClusterIP');

    const nsGroups = {};
    filteredPods.forEach(p => {
        if (!nsGroups[p.namespace]) nsGroups[p.namespace] = { pods: [], deps: [], sts: [], ds: [] };
        nsGroups[p.namespace].pods.push(p);
    });
    filteredDeps.forEach(d => { if (nsGroups[d.namespace]) nsGroups[d.namespace].deps.push(d); });
    filteredSts.forEach(s => { if (nsGroups[s.namespace]) nsGroups[s.namespace].sts.push(s); });
    filteredDs.forEach(d => { if (nsGroups[d.namespace]) nsGroups[d.namespace].ds.push(d); });

    let html = `<div class="cv-workspace">
        <aside class="cv-sidebar">
            <div class="cv-control-group">
                <div class="cv-sidebar-title">
                    <svg class="cv-icon" viewBox="0 0 24 24"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
                    Search
                </div>
                <div class="cv-control-item">
                    <input type="text" class="cv-search-input" placeholder="Search resources..." oninput="cvSearch(this.value)">
                </div>
            </div>
            <div class="cv-control-group">
                <div class="cv-sidebar-title">
                    <svg class="cv-icon" viewBox="0 0 24 24"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>
                    Namespaces
                </div>
                <div id="cv-ns-filters"></div>
            </div>
            <div class="cv-control-group">
                <div class="cv-sidebar-title">
                    <svg class="cv-icon" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
                    Controls
                </div>
                <button id="cv-traffic-btn" class="cv-toggle-btn active" onclick="cvToggleTraffic()">
                    <svg class="cv-icon" viewBox="0 0 24 24"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>
                    <span>Traffic: ON</span>
                </button>
            </div>
            <div class="cv-control-group">
                <div class="cv-sidebar-title">
                    <svg class="cv-icon" viewBox="0 0 24 24"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"></rect><rect x="2" y="14" width="20" height="8" rx="2" ry="2"></rect><line x1="6" y1="6" x2="6.01" y2="6"></line><line x1="6" y1="18" x2="6.01" y2="18"></line></svg>
                    Cluster Metrics
                </div>
                <div class="cv-metric-card">
                    <div class="cv-metric-header"><span>Nodes</span><span id="cv-cpu-val">0</span></div>
                    <div class="cv-metric-bar-bg"><div id="cv-cpu-bar" class="cv-metric-bar-fill"></div></div>
                </div>
                <div class="cv-metric-card">
                    <div class="cv-metric-header"><span>Pods</span><span id="cv-mem-val">0</span></div>
                    <div class="cv-metric-bar-bg"><div id="cv-mem-bar" class="cv-metric-bar-fill"></div></div>
                </div>
            </div>
            <div class="cv-control-group" style="margin-top:auto;">
                <div class="cv-sidebar-title">Legend</div>
                <div class="cv-legend-list">
                    <div class="cv-legend-item"><div class="cv-legend-symbol node"><svg class="cv-icon" style="color:var(--cv-primary)" viewBox="0 0 24 24"><rect x="2" y="2" width="20" height="20" rx="2" ry="2"></rect><rect x="9" y="9" width="6" height="6"></rect></svg></div><span>Node</span></div>
                    <div class="cv-legend-item"><div class="cv-legend-symbol service"><svg class="cv-icon" style="color:var(--cv-info)" viewBox="0 0 24 24"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></div><span>Service</span></div>
                    <div class="cv-legend-item"><div class="cv-legend-symbol pod"><svg class="cv-icon" style="color:var(--cv-success)" viewBox="0 0 24 24"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path></svg></div><span>Pod</span></div>
                    <div class="cv-legend-item"><div class="cv-legend-symbol daemonset"><svg class="cv-icon" style="color:var(--cv-warning)" viewBox="0 0 24 24"><polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2"></polygon></svg></div><span>Controller</span></div>
                </div>
            </div>
        </aside>

        <main class="cv-canvas-area">
            <div class="cv-diagram-container" id="cv-diagram">
                <svg id="cv-connections-svg" class="cv-connections-svg"></svg>`;

    // Column 1: Internet
    const firstExtSvcId = externalSvcs.length > 0 ? 'card-svc-ext-0' : 'cv-boundary';
    html += `<div class="cv-diagram-column">
        <div class="cv-external-client-card" id="card-internet" data-id="internet" data-connects-to="${firstExtSvcId}">
            <div class="cv-logo-icon" style="background:linear-gradient(135deg,#60a5fa,#3b82f6);box-shadow:0 0 12px rgba(59,130,246,0.3);">
                <svg class="cv-icon cv-icon-lg" viewBox="0 0 24 24"><path d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2zm0 18a8 8 0 1 1 8-8 8 8 0 0 1-8 8z"></path><path d="M12 6a6 6 0 1 0 6 6 6 6 0 0 0-6-6zm0 10a4 4 0 1 1 4-4 4 4 0 0 1-4 4z"></path></svg>
            </div>
            <div style="font-weight:600;font-size:14px;">Internet</div>
            <div style="font-size:11px;color:var(--cv-text-muted);">HTTP / HTTPS</div>
        </div>
    </div>`;

    // Column 2: External Services
    if (externalSvcs.length > 0) {
        html += `<div class="cv-diagram-column">
            <div class="cv-column-header">External Services</div>`;
        externalSvcs.forEach((svc, i) => {
            const ip = svc.externalIP || svc.clusterIP || '-';
            const nextId = i + 1 < externalSvcs.length ? 'card-svc-ext-' + (i + 1) : 'cv-boundary';
            html += renderCvCard('service', svc.name, svc.namespace, 'Service (' + escapeHtml(svc.type) + ')',
                `IP: ${escapeHtml(ip)}`, `Port: ${escapeHtml(svc.ports || '-')}`, 'svc-ext-' + i, null, false, nextId);
        });
        html += `</div>`;
    }

    // Column 3: Cluster Boundary
    html += `<div class="cv-cluster-boundary" id="cv-boundary">
        <div class="cv-boundary-label">Kubernetes Cluster</div>`;

    if (cvData.nodes.length === 0) {
        html += `<div style="color:var(--cv-text-muted);text-align:center;padding:20px;">No nodes found</div>`;
    }

    cvData.nodes.forEach(node => {
        const nodeRole = node.roles || 'worker';
        const nodeReady = node.status === 'Ready';
        html += `<div class="cv-k8s-node-container" id="card-node-${escapeHtml(node.name)}" data-id="node-${escapeHtml(node.name)}" onclick="cvSelectResource('node-${escapeHtml(node.name)}', event)">
            <div class="cv-node-info-bar">
                <div class="cv-node-title">
                    <svg class="cv-icon" viewBox="0 0 24 24"><rect x="2" y="2" width="20" height="20" rx="2" ry="2"></rect><rect x="9" y="9" width="6" height="6"></rect></svg>
                    <span class="cv-node-title-text">Node: ${escapeHtml(node.name)}</span>
                    <span class="cv-node-role-badge">${escapeHtml(nodeRole)}</span>
                </div>
                <span class="status-pill"><span class="status-pill-dot"></span>${escapeHtml(node.status || 'Unknown')}</span>
            </div>
            <div class="cv-namespaces-wrapper">`;

        const nodePods = filteredPods.filter(p => p.node === node.name);
        const nodeNs = nodePods.length > 0 ? [...new Set(nodePods.map(p => p.namespace))] : Object.keys(nsGroups);

        nodeNs.forEach(ns => {
            if (!nsGroups[ns]) return;
            const g = nsGroups[ns];
            const nsSvcs = internalSvcs.filter(s => s.namespace === ns);
            const nsClass = ns === 'kube-system' ? 'ns-kube-system' : 'ns-default';

            html += `<div class="cv-namespace-box ${nsClass}" id="cv-ns-${escapeHtml(ns)}">
                <div class="cv-namespace-label">Namespace: ${escapeHtml(ns)}</div>
                <div class="cv-namespace-content">
                    <div class="cv-ns-column services">
                        <div class="cv-ns-column-header">Services</div>`;
            if (nsSvcs.length === 0) {
                html += `<div class="cv-empty-slot">No internal services</div>`;
            } else {
                nsSvcs.forEach((svc, i) => {
                    html += renderCvCard('service', svc.name, svc.namespace, 'Service (ClusterIP)',
                        `IP: ${escapeHtml(svc.clusterIP || '-')}`, `Port: ${escapeHtml(svc.ports || '-')}`, 'svc-int-' + ns + '-' + i);
                });
            }
            html += `</div><div class="cv-ns-column workloads">
                <div class="cv-ns-column-header">Controllers</div>`;
            g.deps.forEach(dep => {
                const depPods = g.pods.filter(p => p.name.includes(dep.name)).map(p => 'card-pod-' + p.name);
                const connectsTo = depPods.length > 0 ? depPods.join(',') : 'cv-boundary';
                html += renderCvCard('deployment', dep.name, dep.namespace, 'Deployment',
                    `Replicas: ${escapeHtml(dep.ready || '-')}`, '', 'deploy-' + dep.name, null, false, connectsTo);
            });
            g.sts.forEach(st => {
                const stsPods = g.pods.filter(p => p.name.includes(st.name)).map(p => 'card-pod-' + p.name);
                const connectsTo = stsPods.length > 0 ? stsPods.join(',') : 'cv-boundary';
                html += renderCvCard('statefulset', st.name, st.namespace, 'StatefulSet',
                    `Replicas: ${escapeHtml(st.ready || '-')}`, '', 'sts-' + st.name, null, false, connectsTo);
            });
            g.ds.forEach(d => {
                const dsPods = g.pods.filter(p => p.name.includes(d.name)).map(p => 'card-pod-' + p.name);
                const connectsTo = dsPods.length > 0 ? dsPods.join(',') : 'cv-boundary';
                html += renderCvCard('daemonset', d.name, d.namespace, 'DaemonSet',
                    `Ready: ${d.ready || 0}/${d.desired || 0}`, '', 'ds-' + d.name, null, false, connectsTo);
            });
            html += `</div><div class="cv-ns-column pods">
                <div class="cv-ns-column-header">Pods</div>`;
            g.pods.forEach(pod => {
                const podKind = pod.name.split('-').slice(0, -2).join('-') || pod.name;
                const isWarn = pod.restarts > 10;
                html += renderCvCard('pod', pod.name, pod.namespace, 'Pod (' + escapeHtml(podKind) + ')',
                    `IP: ${escapeHtml(pod.ip || '-')}`,
                    isWarn ? `Restarts: ${pod.restarts}` : escapeHtml(pod.ready || ''),
                    'pod-' + pod.name, pod.status, isWarn);
            });
            html += `</div></div></div>`;
        });

        html += `</div></div>`;
    });

    html += `</div></div></div></div>`;

    // Detail Drawer
    html += `<div class="cv-details-drawer" id="cv-drawer">
        <div class="cv-drawer-header">
            <div class="cv-drawer-title-area">
                <div class="cv-drawer-subtitle" id="cv-drawer-type"></div>
                <div class="cv-drawer-title" id="cv-drawer-name"></div>
            </div>
            <button class="cv-close-drawer-btn" onclick="cvCloseDrawer()">
                <svg class="cv-icon cv-icon-lg" viewBox="0 0 24 24"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
            </button>
        </div>
        <div class="cv-drawer-tabs">
            <button class="cv-drawer-tab active" onclick="cvSwitchTab('overview')">Details</button>
            <button class="cv-drawer-tab" onclick="cvSwitchTab('yaml')">YAML</button>
            <button class="cv-drawer-tab" onclick="cvSwitchTab('logs')">Logs</button>
        </div>
        <div class="cv-drawer-body">
            <div class="cv-tab-content active" id="cv-tab-overview">
                <div class="cv-info-grid" id="cv-drawer-grid"></div>
                <div>
                    <div class="cv-info-label" style="margin-bottom:8px;">Labels</div>
                    <div class="cv-labels-container" id="cv-drawer-labels"></div>
                </div>
            </div>
            <div class="cv-tab-content" id="cv-tab-yaml"><pre class="cv-pre"><code id="cv-yaml-code"></code></pre></div>
            <div class="cv-tab-content" id="cv-tab-logs"><pre class="cv-pre" style="background-color:#020408;max-height:450px;"><code id="cv-logs-code"></code></pre></div>
        </div>
    </div>`;

    body.innerHTML = html;

    renderCvNsFilters();
    renderCvMetrics();
    setTimeout(() => {
        cvUpdateConnections();
    }, 100);
}

function renderCvCard(type, name, ns, typeLabel, meta1, meta2, cardId, status, isWarn, connectsTo) {
    const cssClass = type === 'service' ? 'card-service' : type === 'pod' ? 'card-pod' : type === 'daemonset' ? 'card-daemonset' : 'card-workload';
    const statusClass = isWarn ? 'warning' : '';
    const displayName = name.length > 22 ? name.substring(0, 20) + '..' : name;
    const connectsAttr = connectsTo ? ` data-connects-to="${connectsTo}"` : '';
    return `<div class="resource-card ${cssClass}" id="card-${cardId}" data-id="${cardId}" data-type="${type}" data-namespace="${escapeHtml(ns)}"${connectsAttr} onclick="cvSelectResource('${cardId}', event)">
        <div class="resource-header">
            <span class="resource-type">${typeLabel}</span>
            ${status ? `<span class="status-pill ${statusClass}"><span class="status-pill-dot"></span>${escapeHtml(status)}</span>` : ''}
        </div>
        <div class="resource-name" title="${escapeHtml(name)}">${escapeHtml(displayName)}</div>
        <div class="resource-meta"><span>${meta1}</span>${meta2 ? '<span>' + meta2 + '</span>' : ''}</div>
    </div>`;
}

// === SVG Connections (matching reference site logic) ===
function cvUpdateConnections() {
    const svg = document.getElementById('cv-connections-svg');
    const container = document.getElementById('cv-diagram');
    if (!svg || !container) return;
    svg.innerHTML = '';

    const containerRect = container.getBoundingClientRect();
    svg.setAttribute('width', container.scrollWidth);
    svg.setAttribute('height', container.scrollHeight);

    const sources = container.querySelectorAll('[data-connects-to]');

    sources.forEach(source => {
        const targetsCsv = source.getAttribute('data-connects-to');
        if (!targetsCsv) return;
        if (source.offsetParent === null) return;

        const sourceRect = source.getBoundingClientRect();
        if (sourceRect.width === 0 || sourceRect.height === 0) return;

        const startX = sourceRect.right - containerRect.left + container.scrollLeft;
        const startY = sourceRect.top + (sourceRect.height / 2) - containerRect.top + container.scrollTop;

        targetsCsv.split(',').forEach(targetId => {
            const target = document.getElementById(targetId.trim());
            if (!target || target.offsetParent === null) return;

            const targetRect = target.getBoundingClientRect();
            if (targetRect.width === 0 || targetRect.height === 0) return;

            const endX = targetRect.left - containerRect.left + container.scrollLeft;
            const endY = targetRect.top + (targetRect.height / 2) - containerRect.top + container.scrollTop;

            const controlOffset = Math.max(50, Math.abs(endX - startX) * 0.4);
            const d = `M ${startX} ${startY} C ${startX + controlOffset} ${startY}, ${endX - controlOffset} ${endY}, ${endX} ${endY}`;

            const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
            path.setAttribute('d', d);
            path.setAttribute('class', 'connection-path');
            path.setAttribute('id', `conn-${source.getAttribute('data-id')}-${targetId}`);
            path.setAttribute('data-source', source.getAttribute('data-id'));
            path.setAttribute('data-target', targetId);
            svg.appendChild(path);

            if (cvTrafficActive) {
                const dot = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
                const isPurple = source.getAttribute('data-id').startsWith('deploy-') || source.getAttribute('data-id').startsWith('ds-') || source.getAttribute('data-id').startsWith('sts-');
                dot.setAttribute('class', `flow-dot animate-dot ${isPurple ? 'purple' : ''}`);

                const animateMotion = document.createElementNS('http://www.w3.org/2000/svg', 'animateMotion');
                animateMotion.setAttribute('dur', isPurple ? '4s' : '3s');
                animateMotion.setAttribute('repeatCount', 'indefinite');
                animateMotion.setAttribute('path', d);
                animateMotion.setAttribute('begin', `${Math.random() * 2}s`);

                dot.appendChild(animateMotion);
                svg.appendChild(dot);
            }
        });
    });
}

// === Traffic Animation Toggle ===
function cvToggleTraffic() {
    cvTrafficActive = !cvTrafficActive;
    const btn = document.getElementById('cv-traffic-btn');
    const dots = document.querySelectorAll('.animate-dot');
    if (cvTrafficActive) {
        if (btn) { btn.classList.add('active'); btn.querySelector('span').textContent = 'Traffic: ON'; }
        dots.forEach(d => d.style.display = 'block');
        cvUpdateConnections();
    } else {
        if (btn) { btn.classList.remove('active'); btn.querySelector('span').textContent = 'Traffic: OFF'; }
        dots.forEach(d => d.style.display = 'none');
    }
}

function cvStartAnimation() {
    // Animation is handled by SVG animateMotion elements
}

// === Search ===
function cvSearch(term) {
    cvSearchTerm = term.toLowerCase();
    document.querySelectorAll('.resource-card').forEach(card => {
        const name = (card.getAttribute('data-id') || '').toLowerCase();
        const ns = (card.getAttribute('data-namespace') || '').toLowerCase();
        if (!cvSearchTerm || name.includes(cvSearchTerm) || ns.includes(cvSearchTerm)) {
            card.classList.remove('dimmed');
        } else {
            card.classList.add('dimmed');
        }
    });
}

// === Resource Selection & Drawer ===
function cvSelectResource(id, event) {
    if (event) event.stopPropagation();
    document.querySelectorAll('.resource-card').forEach(c => { c.classList.remove('selected', 'selected-purple'); });

    const card = document.getElementById(`card-${id}`);
    if (!card) return;
    const isPurpleKind = card.getAttribute('data-type') === 'Deployment' || card.getAttribute('data-type') === 'StatefulSet' || id.startsWith('node-');
    card.classList.add(isPurpleKind ? 'selected-purple' : 'selected');
    cvSelectedId = id;
    cvPopulateDrawer(id);
    document.getElementById('cv-drawer').classList.add('open');
    cvHighlightConnections(id);
}

function cvCloseDrawer() {
    document.getElementById('cv-drawer').classList.remove('open');
    document.querySelectorAll('.resource-card').forEach(c => { c.classList.remove('selected', 'selected-purple'); });
    cvSelectedId = null;
    cvResetConnections();
}

function cvSwitchTab(tabId) {
    document.querySelectorAll('#cv-drawer .cv-drawer-tab').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('#cv-drawer .cv-tab-content').forEach(c => c.classList.remove('active'));
    const tabs = ['overview', 'yaml', 'logs'];
    const idx = tabs.indexOf(tabId);
    if (idx >= 0) document.querySelectorAll('#cv-drawer .cv-drawer-tab')[idx].classList.add('active');
    const tabEl = document.getElementById(`cv-tab-${tabId}`);
    if (tabEl) tabEl.classList.add('active');
}

function cvPopulateDrawer(id) {
    const card = document.getElementById(`card-${id}`);
    if (!card) return;
    const type = card.getAttribute('data-type') || 'Unknown';
    const ns = card.getAttribute('data-namespace') || '';
    const name = card.getAttribute('data-id') || id;

    document.getElementById('cv-drawer-type').textContent = `${ns} / ${type}`;
    document.getElementById('cv-drawer-name').textContent = name.replace(/^(pod-|deploy-|svc-|sts-|ds-|node-)/, '');

    // Find data
    let data = null;
    if (type === 'pod') data = cvData.pods.find(p => 'pod-' + p.name === id || p.name === name);
    else if (type === 'service') data = cvData.services.find(s => s.name === name);
    else if (type === 'deployment') data = cvData.deployments.find(d => 'deploy-' + d.name === id || d.name === name);
    else if (type === 'statefulset') data = cvData.statefulsets.find(s => 'sts-' + s.name === id || s.name === name);
    else if (type === 'daemonset') data = cvData.daemonsets.find(d => 'ds-' + d.name === id || d.name === name);
    else if (id.startsWith('node-')) data = cvData.nodes.find(n => 'node-' + n.name === id);

    const grid = document.getElementById('cv-drawer-grid');
    const labels = document.getElementById('cv-drawer-labels');

    if (!data) {
        grid.innerHTML = '<div style="color:var(--cv-text-muted);grid-column:1/-1;">No data available</div>';
        labels.innerHTML = '';
        return;
    }

    let gridHtml = '';
    Object.entries(data).forEach(([key, val]) => {
        if (key === 'security' || key === 'containers') return;
        const statusColor = (String(val) === 'Ready' || String(val) === 'Running' || String(val) === 'Active') ? 'var(--cv-success)' : '';
        gridHtml += `<div class="cv-info-item"><span class="cv-info-label">${escapeHtml(key)}</span><span class="cv-info-value" style="${statusColor ? 'color:' + statusColor : ''}">${escapeHtml(String(val ?? '-'))}</span></div>`;
    });
    grid.innerHTML = gridHtml;

    if (data.containers && data.containers.length > 0) {
        labels.innerHTML = data.containers.map(c => `<span class="cv-label-tag">${escapeHtml(c)}</span>`).join('');
    } else {
        labels.innerHTML = '<span style="font-size:12px;color:var(--cv-text-muted);">No containers</span>';
    }

    // YAML
    const yamlCode = document.getElementById('cv-yaml-code');
    if (yamlCode) yamlCode.textContent = JSON.stringify(data, null, 2);

    // Logs
    const logsCode = document.getElementById('cv-logs-code');
    if (logsCode) logsCode.textContent = `Log data for ${name}...`;
}

// === Connection Highlighting ===
function cvHighlightConnections(id) {
    cvResetConnections();
    document.querySelectorAll('.connection-path').forEach(path => {
        const src = path.getAttribute('data-source');
        const target = path.getAttribute('data-target');
        if (src === id || target === id) {
            const isPurple = src.startsWith('deploy-') || target.startsWith('deploy-') || id.startsWith('node-');
            path.classList.add(isPurple ? 'active-purple' : 'active');
        }
    });
}

function cvResetConnections() {
    document.querySelectorAll('.connection-path').forEach(p => { p.classList.remove('active', 'active-purple'); });
}

// === Window Resize ===
let cvResizeTimer = null;
window.addEventListener('resize', () => {
    if (document.getElementById('cv-diagram')) {
        clearTimeout(cvResizeTimer);
        cvResizeTimer = setTimeout(cvUpdateConnections, 150);
    }
});

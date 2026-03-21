const customViewIds = [
    'overview-container',
    'metrics-dashboard-container',
    'topology-tree-container',
    'applications-container',
    'helm-container',
    'rbac-viz-container',
    'netpol-viz-container',
    'timeline-container',
    'gitops-container'
];

const customViewNamespaceSelectIds = [
    'metrics-dash-ns-select',
    'topo-tree-ns-select',
    'apps-ns-select',
    'rbac-viz-ns-select',
    'netpol-viz-ns-select',
    'timeline-ns-select'
];

function showOverviewPanel() {
    closeMobileSidebar();
    currentResource = 'overview';
    document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
    const nav = document.querySelector('.nav-item[data-resource="overview"]');
    if (nav) nav.classList.add('active');
    hideTopologyView();
    hideAllCustomViews();
    const mainPanel = document.querySelector('.main-panel');
    if (mainPanel) mainPanel.style.display = 'none';

    const aiPanel = document.getElementById('ai-panel');
    const resizeHandle = document.getElementById('resize-handle');
    if (aiPanel) aiPanel.style.display = 'none';
    if (resizeHandle) resizeHandle.style.display = 'none';

    const btn = document.getElementById('ai-toggle-btn');
    if (btn) btn.classList.remove('active');

    const container = document.getElementById('overview-container');
    if (container) container.style.display = 'flex';
    loadOverviewData();
}

function hideOverviewPanel() {
    const container = document.getElementById('overview-container');
    if (container) container.style.display = 'none';

    const saved = localStorage.getItem('k13d_ai_panel');
    if (saved !== 'closed') {
        const aiPanel = document.getElementById('ai-panel');
        const resizeHandle = document.getElementById('resize-handle');
        if (aiPanel) aiPanel.style.display = 'flex';
        if (resizeHandle) resizeHandle.style.display = 'block';

        const btn = document.getElementById('ai-toggle-btn');
        if (btn) btn.classList.add('active');
    }
}

function hideAllCustomViews() {
    customViewIds.forEach((id) => {
        const el = document.getElementById(id);
        if (el) el.style.display = 'none';
    });
}

function showCustomView(containerId, resource) {
    currentResource = resource;
    document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
    const nav = document.querySelector(`.nav-item[data-resource="${resource}"]`);
    if (nav) nav.classList.add('active');

    hideOverviewPanel();
    hideTopologyView();
    hideAllCustomViews();

    const mainPanel = document.querySelector('.main-panel');
    if (mainPanel) mainPanel.style.display = 'none';

    const container = document.getElementById(containerId);
    if (container) container.style.display = 'flex';

    syncCustomViewNamespaces();
}

function syncCustomViewNamespaces() {
    const src = document.getElementById('namespace-select');
    if (!src) return;

    customViewNamespaceSelectIds.forEach((id) => {
        const sel = document.getElementById(id);
        if (!sel) return;

        const prev = sel.value;
        sel.innerHTML = '';

        for (const opt of src.options) {
            const next = document.createElement('option');
            next.value = opt.value;
            next.textContent = opt.textContent;
            sel.appendChild(next);
        }

        if (prev) {
            sel.value = prev;
        } else if (currentNamespace) {
            sel.value = currentNamespace;
        } else {
            sel.value = '';
        }
    });
}

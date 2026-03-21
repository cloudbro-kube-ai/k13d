function showTopologyTreeView() {
    showCustomView('topology-tree-container', 'topology-tree');
    loadTopologyTreeData();
}

async function loadTopologyTreeData() {
    const body = document.getElementById('topo-tree-body');
    const type = document.getElementById('topo-tree-type-select')?.value || 'deploy';
    const ns = document.getElementById('topo-tree-ns-select')?.value || '';
    body.innerHTML = '<div class="loading-placeholder">Loading topology tree...</div>';

    try {
        let params = `?type=${encodeURIComponent(type)}`;
        if (ns) params += `&namespace=${encodeURIComponent(ns)}`;

        const resp = await fetchWithAuth(`/api/xray${params}`);
        const data = await resp.json();
        if (!data.nodes || data.nodes.length === 0) {
            body.innerHTML = '<div class="loading-placeholder">No resources found for this type/namespace.</div>';
            return;
        }

        body.innerHTML = `<div class="xray-tree">${data.nodes.map((node) => renderXRayNode(node, 0)).join('')}</div>`;
    } catch (e) {
        body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load topology tree: ${escapeHtml(e.message)}</div>`;
    }
}

function renderXRayNode(node, depth) {
    const hasChildren = node.children && node.children.length > 0;
    const statusClass = (node.status || '').toLowerCase().replace(/\s+/g, '');
    const kindIcons = {
        Deployment: '⊞',
        StatefulSet: '⊟',
        DaemonSet: '⊠',
        ReplicaSet: '◫',
        Pod: '◉',
        Job: '⧫',
        CronJob: '⏱',
        Service: '◎',
        ConfigMap: '⊡',
        Secret: '⊗'
    };
    const icon = kindIcons[node.kind] || '◇';
    const id = `xray-${depth}-${(node.name || '').replace(/[^a-z0-9]/gi, '-')}`;

    return `
                <div class="xray-node">
                    <div class="xray-node-header" onclick="toggleXRayNode('${id}')">
                        <span class="xray-toggle">${hasChildren ? '▼' : '·'}</span>
                        <span class="xray-icon">${icon}</span>
                        <span class="xray-kind">${escapeHtml(node.kind)}</span>
                        <span class="xray-name">${escapeHtml(node.name)}</span>
                        <span class="xray-status ${statusClass}">${escapeHtml(node.status || '')}</span>
                    </div>
                    ${hasChildren ? `<div class="xray-children" id="${id}">${node.children.map((child) => renderXRayNode(child, depth + 1)).join('')}</div>` : ''}
                </div>`;
}

function toggleXRayNode(id) {
    const el = document.getElementById(id);
    if (!el) return;

    const isHidden = el.style.display === 'none';
    el.style.display = isHidden ? '' : 'none';

    const header = el.previousElementSibling;
    if (header) {
        const toggle = header.querySelector('.xray-toggle');
        if (toggle) toggle.textContent = isHidden ? '▼' : '▶';
    }
}

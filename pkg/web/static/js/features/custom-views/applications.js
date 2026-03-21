function showApplicationsView() {
    showCustomView('applications-container', 'applications');
    loadApplicationsData();
}

async function loadApplicationsData() {
    const body = document.getElementById('applications-body');
    const ns = document.getElementById('apps-ns-select')?.value || '';
    body.innerHTML = `<div class="loading-placeholder">${t('msg_loading_applications')}</div>`;

    try {
        const params = ns ? `?namespace=${encodeURIComponent(ns)}` : '';
        const resp = await fetchWithAuth(`/api/applications${params}`);
        const apps = await resp.json();
        if (!apps || apps.length === 0) {
            body.innerHTML = `<div class="loading-placeholder">${t('msg_no_apps')}</div>`;
            return;
        }

        window.appsData = apps;
        body.innerHTML = `<div class="apps-grid">${apps.map((app, idx) => {
            const resourceChips = Object.entries(app.resources || {}).map(([kind, items]) =>
                `<span class="app-resource-chip">${escapeHtml(kind)} (${items.length})</span>`
            ).join('');

            return `
                        <div class="app-card" onclick="showAppDetail(${idx})">
                            <div class="app-card-header">
                                <span class="app-card-name">${escapeHtml(app.name)}</span>
                                <span class="app-card-badge ${app.status || 'healthy'}">${escapeHtml(app.status || 'healthy')}</span>
                            </div>
                            <div class="app-card-meta">
                                ${app.version ? `<span>v${escapeHtml(app.version)}</span>` : ''}
                                ${app.component ? `<span>${escapeHtml(app.component)}</span>` : ''}
                                ${app.podCount !== undefined ? `<span>Pods: ${app.readyPods || 0}/${app.podCount}</span>` : ''}
                            </div>
                            <div class="app-card-resources">${resourceChips}</div>
                        </div>`;
        }).join('')}</div>`;
    } catch (e) {
        body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load applications: ${escapeHtml(e.message)}</div>`;
    }
}

function showAppDetail(index) {
    const app = (window.appsData || [])[index];
    if (!app) return;

    document.getElementById('app-detail-title').textContent = app.name;

    let html = `<div style="display:flex;align-items:center;gap:10px;margin-bottom:16px;">
        <span class="app-card-badge ${app.status || 'healthy'}" style="font-size:13px;">${escapeHtml(app.status || 'healthy')}</span>
        ${app.version ? `<span style="color:var(--text-secondary);font-size:13px;">v${escapeHtml(app.version)}</span>` : ''}
        ${app.component ? `<span style="color:var(--text-secondary);font-size:13px;">${escapeHtml(app.component)}</span>` : ''}
    </div>`;

    if (app.podCount !== undefined) {
        html += `<div style="margin-bottom:16px;padding:12px;background:var(--bg-tertiary);border-radius:8px;">
            <div style="font-weight:600;margin-bottom:6px;color:var(--text-primary);">Pods</div>
            <div style="font-size:24px;font-weight:700;color:var(--accent-blue);">${app.readyPods || 0} / ${app.podCount}</div>
            <div style="font-size:12px;color:var(--text-secondary);">ready</div>
        </div>`;
    }

    const resources = app.resources || {};
    const kinds = Object.keys(resources);
    if (kinds.length > 0) {
        html += `<div style="font-weight:600;margin-bottom:8px;color:var(--text-primary);">${t('header_resources')}</div>`;
        kinds.forEach((kind) => {
            const items = resources[kind] || [];
            html += `<div style="margin-bottom:12px;">
                <div style="font-size:13px;font-weight:600;color:var(--accent-blue);margin-bottom:4px;">${escapeHtml(kind)} (${items.length})</div>
                <table style="width:100%;border-collapse:collapse;font-size:12px;">
                    <thead><tr style="border-bottom:1px solid var(--border-color);">
                        <th style="text-align:left;padding:4px 8px;color:var(--text-secondary);">${t('th_name')}</th>
                        <th style="text-align:left;padding:4px 8px;color:var(--text-secondary);">${t('th_namespace')}</th>
                        <th style="text-align:left;padding:4px 8px;color:var(--text-secondary);">${t('th_status')}</th>
                    </tr></thead><tbody>`;
            items.forEach((item) => {
                const status = (item.status || '').toLowerCase();
                const statusStyle = status === 'running' || status === 'active' || status === 'ready'
                    ? 'color:var(--accent-green)'
                    : status === 'failed'
                        ? 'color:var(--accent-red)'
                        : 'color:var(--text-secondary)';
                html += `<tr style="border-bottom:1px solid var(--border-subtle);">
                    <td style="padding:4px 8px;color:var(--text-primary);">${escapeHtml(item.name || '')}</td>
                    <td style="padding:4px 8px;color:var(--text-secondary);">${escapeHtml(item.namespace || '')}</td>
                    <td style="padding:4px 8px;${statusStyle};">${escapeHtml(item.status || '-')}</td>
                </tr>`;
            });
            html += '</tbody></table></div>';
        });
    }

    document.getElementById('app-detail-body').innerHTML = html;
    document.getElementById('app-detail-overlay').classList.add('active');
}

function closeAppDetail() {
    document.getElementById('app-detail-overlay').classList.remove('active');
}

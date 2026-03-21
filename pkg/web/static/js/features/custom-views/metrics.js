function showMetricsDashboard() {
    showCustomView('metrics-dashboard-container', 'metrics');
    loadMetricsDashData();
}

async function loadMetricsDashData() {
    const body = document.getElementById('metrics-dash-body');
    const ns = document.getElementById('metrics-dash-ns-select')?.value || '';
    body.innerHTML = '<div class="loading-placeholder">Loading metrics...</div>';

    try {
        const params = ns ? `?namespace=${encodeURIComponent(ns)}` : '';
        const resp = await fetchWithAuth(`/api/pulse${params}`);
        const data = await resp.json();
        const cpuPct = data.cpu_avail && data.cpu_capacity_milli > 0
            ? Math.round(data.cpu_used_milli / data.cpu_capacity_milli * 100)
            : 0;
        const memPct = data.mem_avail && data.mem_capacity_mib > 0
            ? Math.round(data.mem_used_mib / data.mem_capacity_mib * 100)
            : 0;
        const barColor = (pct) => {
            if (pct > 80) return 'var(--accent-red)';
            if (pct > 60) return 'var(--accent-yellow)';
            return 'var(--accent-green)';
        };

        body.innerHTML = `
                    <div class="pulse-grid">
                        <div class="pulse-card">
                            <div class="pulse-card-title">Pods</div>
                            <div class="pulse-card-value">${data.pods_running}<span style="font-size:14px;color:var(--text-secondary);">/${data.pods_total}</span></div>
                            <div class="pulse-card-sub" style="color:var(--accent-green);">${data.pods_running} Running</div>
                            ${data.pods_pending > 0 ? `<div class="pulse-card-sub" style="color:var(--accent-yellow);">${data.pods_pending} Pending</div>` : ''}
                            ${data.pods_failed > 0 ? `<div class="pulse-card-sub" style="color:var(--accent-red);">${data.pods_failed} Failed</div>` : ''}
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">Deployments</div>
                            <div class="pulse-card-value">${data.deploys_ready}<span style="font-size:14px;color:var(--text-secondary);">/${data.deploys_total}</span></div>
                            <div class="pulse-card-sub">${data.deploys_ready} Ready${data.deploys_updating > 0 ? `, ${data.deploys_updating} Updating` : ''}</div>
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">StatefulSets</div>
                            <div class="pulse-card-value">${data.sts_ready}<span style="font-size:14px;color:var(--text-secondary);">/${data.sts_total}</span></div>
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">DaemonSets</div>
                            <div class="pulse-card-value">${data.ds_ready}<span style="font-size:14px;color:var(--text-secondary);">/${data.ds_total}</span></div>
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">Jobs</div>
                            <div class="pulse-card-value">${data.jobs_complete}<span style="font-size:14px;color:var(--text-secondary);">/${data.jobs_total}</span></div>
                            <div class="pulse-card-sub">${data.jobs_active || 0} Active, ${data.jobs_failed || 0} Failed</div>
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">Nodes</div>
                            <div class="pulse-card-value">${data.nodes_ready}<span style="font-size:14px;color:var(--text-secondary);">/${data.nodes_total}</span></div>
                            ${data.nodes_not_ready > 0 ? `<div class="pulse-card-sub" style="color:var(--accent-red);">${data.nodes_not_ready} Not Ready</div>` : '<div class="pulse-card-sub" style="color:var(--accent-green);">All Ready</div>'}
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">CPU Usage</div>
                            ${data.cpu_avail ? `
                                <div class="pulse-card-value">${cpuPct}%</div>
                                <div class="pulse-bar"><div class="pulse-bar-fill" style="width:${cpuPct}%;background:${barColor(cpuPct)};"></div></div>
                                <div class="pulse-card-sub">${data.cpu_used_milli}m / ${data.cpu_capacity_milli}m</div>
                            ` : `
                                <div class="pulse-card-value" style="font-size:14px;color:var(--text-secondary);">N/A</div>
                                <div class="pulse-card-sub" style="color:var(--accent-yellow);">metrics-server not available</div>
                            `}
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">Memory Usage</div>
                            ${data.mem_avail ? `
                                <div class="pulse-card-value">${memPct}%</div>
                                <div class="pulse-bar"><div class="pulse-bar-fill" style="width:${memPct}%;background:${barColor(memPct)};"></div></div>
                                <div class="pulse-card-sub">${data.mem_used_mib}Mi / ${data.mem_capacity_mib}Mi</div>
                            ` : `
                                <div class="pulse-card-value" style="font-size:14px;color:var(--text-secondary);">N/A</div>
                                <div class="pulse-card-sub" style="color:var(--accent-yellow);">metrics-server not available</div>
                            `}
                        </div>
                    </div>
                    <div style="display:flex;gap:10px;margin-bottom:16px;">
                        <button onclick="showMetrics()" style="padding:8px 16px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-secondary);color:var(--accent-blue);cursor:pointer;font-size:12px;">Historical Charts</button>
                        <button onclick="showApplicationsView()" style="padding:8px 16px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-secondary);color:var(--accent-purple);cursor:pointer;font-size:12px;">Applications</button>
                    </div>
                    ${data.events && data.events.length > 0 ? `
                    <div class="pulse-events">
                        <h3>Recent Events</h3>
                        ${data.events.map((event) => `
                            <div class="pulse-event-item">
                                <span class="pulse-event-badge ${event.type === 'Warning' ? 'warning' : 'normal'}">${escapeHtml(event.type || 'Normal')}</span>
                                <span style="color:var(--accent-cyan);font-family:monospace;">${escapeHtml(event.reason || '')}</span>
                                <span style="flex:1;">${escapeHtml(event.message || '')}</span>
                                <span style="color:var(--text-muted);flex-shrink:0;">${escapeHtml(event.age || '')}</span>
                            </div>
                        `).join('')}
                    </div>` : ''}
                `;
    } catch (e) {
        body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load metrics: ${escapeHtml(e.message)}</div>`;
    }
}

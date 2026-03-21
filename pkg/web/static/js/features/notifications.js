function setNotificationResultState(background, color, message) {
    const result = document.getElementById('notif-test-result');
    result.style.display = 'block';
    result.style.background = background;
    result.style.color = color;
    result.textContent = message;
}

async function loadNotificationSettings() {
    try {
        const resp = await fetchWithAuth('/api/notifications/config');
        const data = await resp.json();
        if (data.enabled !== undefined) document.getElementById('notif-enabled').checked = data.enabled;
        if (data.provider) document.getElementById('notif-platform').value = data.provider;
        if (data.webhook_url) {
            const webhookInput = document.getElementById('notif-webhook-url');
            webhookInput.value = data.webhook_url;
            webhookInput.dataset.maskedValue = data.webhook_url;
        }
        if (data.channel) document.getElementById('notif-channel').value = data.channel;
        const evts = data.events || [];
        const evtMap = { 'pod_crash': 'notif-evt-crash', 'oom_killed': 'notif-evt-oom', 'node_not_ready': 'notif-evt-node', 'deploy_fail': 'notif-evt-deploy', 'image_pull_fail': 'notif-evt-imagepull' };
        Object.entries(evtMap).forEach(([key, id]) => {
            const el = document.getElementById(id);
            if (el) el.checked = evts.includes(key);
        });
        if (data.smtp) {
            const s = data.smtp;
            if (s.host) document.getElementById('notif-smtp-host').value = s.host;
            if (s.port) document.getElementById('notif-smtp-port').value = s.port;
            if (s.username) document.getElementById('notif-smtp-username').value = s.username;
            if (s.from) document.getElementById('notif-smtp-from').value = s.from;
            if (s.to) document.getElementById('notif-smtp-to').value = s.to.join(', ');
            document.getElementById('notif-smtp-tls').checked = s.use_tls !== false;
        }
        updateNotifPlaceholder();
        loadNotificationHistory();
    } catch (e) {
        console.warn('Failed to load notification settings:', e);
    }
}

async function saveNotificationSettings() {
    const provider = document.getElementById('notif-platform').value;
    const webhookInput = document.getElementById('notif-webhook-url');
    const webhookValue = webhookInput.value;
    const maskedValue = webhookInput.dataset.maskedValue || '';
    const preserveWebhookURL = provider !== 'email' && webhookValue !== '' && webhookValue === maskedValue;
    const payload = {
        enabled: document.getElementById('notif-enabled').checked,
        provider: provider,
        webhook_url: preserveWebhookURL ? '' : webhookValue,
        preserve_webhook_url: preserveWebhookURL,
        channel: document.getElementById('notif-channel').value,
        events: [
            document.getElementById('notif-evt-crash')?.checked ? 'pod_crash' : '',
            document.getElementById('notif-evt-oom')?.checked ? 'oom_killed' : '',
            document.getElementById('notif-evt-node')?.checked ? 'node_not_ready' : '',
            document.getElementById('notif-evt-deploy')?.checked ? 'deploy_fail' : '',
            document.getElementById('notif-evt-imagepull')?.checked ? 'image_pull_fail' : ''
        ].filter(Boolean)
    };
    if (provider === 'email') {
        const toStr = document.getElementById('notif-smtp-to').value;
        payload.smtp = {
            host: document.getElementById('notif-smtp-host').value,
            port: parseInt(document.getElementById('notif-smtp-port').value, 10) || 587,
            username: document.getElementById('notif-smtp-username').value,
            password: document.getElementById('notif-smtp-password').value,
            from: document.getElementById('notif-smtp-from').value,
            to: toStr ? toStr.split(',').map(s => s.trim()).filter(Boolean) : [],
            use_tls: document.getElementById('notif-smtp-tls').checked
        };
    }
    try {
        await fetchWithAuth('/api/notifications/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        setNotificationResultState('rgba(46,160,67,0.15)', 'var(--accent-green)', 'Settings saved!');
        setTimeout(() => {
            document.getElementById('notif-test-result').style.display = 'none';
        }, 3000);
    } catch (e) {
        alert('Failed to save: ' + e.message);
    }
}

async function testNotification() {
    setNotificationResultState('rgba(56,132,244,0.15)', 'var(--accent-blue)', 'Sending test notification...');
    try {
        const resp = await fetchWithAuth('/api/notifications/test', { method: 'POST' });
        if (!resp.ok) {
            const text = await resp.text();
            throw new Error(text || resp.statusText);
        }
        setNotificationResultState('rgba(46,160,67,0.15)', 'var(--accent-green)', 'Test notification sent!');
    } catch (e) {
        setNotificationResultState('rgba(248,81,73,0.15)', 'var(--accent-red)', 'Failed: ' + e.message);
    }
}

function updateNotifPlaceholder() {
    const platform = document.getElementById('notif-platform').value;
    const webhookSection = document.getElementById('notif-webhook-section');
    const smtpSection = document.getElementById('notif-smtp-section');
    if (platform === 'email') {
        webhookSection.style.display = 'none';
        smtpSection.style.display = 'block';
        return;
    }

    webhookSection.style.display = 'block';
    smtpSection.style.display = 'none';
    const urlInput = document.getElementById('notif-webhook-url');
    const placeholders = {
        slack: 'https://hooks.slack.com/services/...',
        discord: 'https://discord.com/api/webhooks/...',
        teams: 'https://outlook.office.com/webhook/...',
        custom: 'https://your-webhook-url.com/hook'
    };
    urlInput.placeholder = placeholders[platform] || placeholders.custom;
}

async function loadNotificationHistory() {
    const body = document.getElementById('notif-history-body');
    try {
        const resp = await fetchWithAuth('/api/notifications/history');
        const items = await resp.json();
        if (!items || items.length === 0) {
            body.innerHTML = '<div class="loading-placeholder" style="font-size:12px;">No notifications sent yet.</div>';
            return;
        }
        body.innerHTML = items.map(h => {
            const time = h.timestamp ? formatDateTime(h.timestamp) : '';
            const icon = h.success ? '<span style="color:var(--accent-green);">&#10003;</span>' : '<span style="color:var(--accent-red);">&#10007;</span>';
            return `<div style="display:flex;align-items:center;gap:8px;padding:6px 0;border-bottom:1px solid var(--border-color);font-size:12px;">
                        ${icon}
                        <span style="color:var(--text-secondary);min-width:140px;">${escapeHtml(time)}</span>
                        <span style="color:var(--accent-blue);min-width:80px;">${escapeHtml(h.event_type || '')}</span>
                        <span style="flex:1;color:var(--text-primary);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">${escapeHtml(h.message || '')}</span>
                    </div>`;
        }).join('');
    } catch (e) {
        body.innerHTML = '<div class="loading-placeholder" style="font-size:12px;color:var(--accent-red);">Failed to load history</div>';
    }
}

// Settings
function showSettings() {
    document.getElementById('settings-modal').classList.add('active');
    loadSettings();
    loadVersionInfo();
    // Show Admin tab only for admin users
    const adminTab = document.getElementById('admin-tab');
    if (adminTab) {
        adminTab.style.display = (currentUser && currentUser.role === 'admin') ? 'block' : 'none';
    }
}

function closeSettings() {
    document.getElementById('settings-modal').classList.remove('active');
}

function switchSettingsTab(tab) {
    document.querySelectorAll('.tabs .tab').forEach((t, i) => {
        const isActive = t.textContent.toLowerCase().includes(tab);
        t.classList.toggle('active', isActive);
        t.setAttribute('aria-selected', String(isActive));
    });
    document.querySelectorAll('.settings-content').forEach(c => c.style.display = 'none');
    document.getElementById(`settings-${tab}`).style.display = 'block';

    // Load data for specific tabs
    if (tab === 'ai') {
        loadModelProfiles();
        updateEndpointPlaceholder();
        loadLLMStatus();
        onLLMTabOpened();
        loadToolApprovalSettings();
        loadAgentSettings();
    } else if (tab === 'mcp') {
        loadMCPServers();
        loadMCPTools();
    } else if (tab === 'admin') {
        loadAdminUsers();
        loadAuthStatus();
        loadRoles();
    } else if (tab === 'security') {
        checkTrivyStatus();
        loadTrivyInstructions();
        loadSecurityPreferences();
    } else if (tab === 'metrics') {
        loadPrometheusSettings();
    } else if (tab === 'notifications') {
        loadNotificationSettings();
    } else if (tab === 'general') {
        // Load saved theme
        const saved = localStorage.getItem('k13d_theme') || 'light';
        const sel = document.getElementById('setting-theme');
        if (sel) sel.value = saved;
    }
}

const SECURITY_PREFERENCES_STORAGE_KEY = 'k13d-security-preferences';

function loadSecurityPreferences() {
    let prefs = {
        scan_images: true,
        min_severity: 'HIGH'
    };

    try {
        const saved = localStorage.getItem(SECURITY_PREFERENCES_STORAGE_KEY);
        if (saved) {
            prefs = { ...prefs, ...JSON.parse(saved) };
        }
    } catch (e) {
        console.warn('Failed to load security preferences:', e);
    }

    const scanImages = document.getElementById('security-scan-images');
    const minSeverity = document.getElementById('security-min-severity');
    if (scanImages) scanImages.checked = prefs.scan_images !== false;
    if (minSeverity) minSeverity.value = prefs.min_severity || 'HIGH';
}

function saveSecurityPreferences() {
    const prefs = {
        scan_images: document.getElementById('security-scan-images')?.checked ?? true,
        min_severity: document.getElementById('security-min-severity')?.value || 'HIGH'
    };
    localStorage.setItem(SECURITY_PREFERENCES_STORAGE_KEY, JSON.stringify(prefs));
    showToast('Security preferences saved');
}

// Theme / Skin support
function applyTheme(theme) {
    const html = document.documentElement;
    html.setAttribute('data-theme', theme);
    localStorage.setItem('k13d_theme', theme);
    updateThemeIcon();
    // Sync settings dropdown
    const sel = document.getElementById('setting-theme');
    if (sel) sel.value = theme;
}

// Apply saved theme on load
(function initSettingsTheme() {
    const saved = localStorage.getItem('k13d_theme') || 'light';
    applyTheme(saved);
})();

// ==========================================
// Trivy/Security Functions
// ==========================================
async function checkTrivyStatus() {
    const indicator = document.getElementById('trivy-status-indicator');
    const statusText = document.getElementById('trivy-status-text');
    const versionEl = document.getElementById('trivy-version');
    const pathEl = document.getElementById('trivy-path');
    const installBtn = document.getElementById('trivy-install-btn');
    const instructionsDiv = document.getElementById('trivy-instructions');

    try {
        const resp = await fetchWithAuth('/api/security/trivy/status');
        const status = await resp.json();

        if (status.installed) {
            indicator.style.background = 'var(--accent-green)';
            indicator.style.boxShadow = '0 0 8px var(--accent-green)';
            statusText.textContent = 'Installed';
            versionEl.textContent = status.version ? `Version: ${status.version}` : '';
            pathEl.textContent = status.path || '';
            installBtn.style.display = 'none';
            instructionsDiv.style.display = 'none';

            if (status.update_available) {
                versionEl.innerHTML += ` <span style="color:var(--accent-yellow);">(Update available: ${status.latest_version})</span>`;
            }
        } else {
            indicator.style.background = 'var(--accent-red)';
            indicator.style.boxShadow = '0 0 8px var(--accent-red)';
            statusText.textContent = 'Not Installed';
            versionEl.textContent = status.latest_version ? `Latest: ${status.latest_version}` : '';
            pathEl.textContent = '';
            installBtn.style.display = 'inline-block';
            instructionsDiv.style.display = 'block';
        }
    } catch (e) {
        indicator.style.background = 'var(--text-secondary)';
        statusText.textContent = 'Unknown';
        versionEl.textContent = '';
        pathEl.textContent = '';
        console.error('Failed to check Trivy status:', e);
    }
}

async function loadTrivyInstructions() {
    try {
        const resp = await fetchWithAuth('/api/security/trivy/instructions');
        const data = await resp.json();
        document.getElementById('trivy-install-commands').textContent = data.instructions || '';
    } catch (e) {
        console.error('Failed to load Trivy instructions:', e);
    }
}

async function installTrivy() {
    const btn = document.getElementById('trivy-install-btn');
    const progressDiv = document.getElementById('trivy-install-progress');
    const progressBar = document.getElementById('trivy-progress-bar');
    const progressText = document.getElementById('trivy-progress-text');

    btn.disabled = true;
    btn.textContent = 'Installing...';
    progressDiv.style.display = 'block';
    progressBar.style.width = '10%';
    progressText.textContent = 'Starting download...';

    try {
        const resp = await fetchWithAuth('/api/security/trivy/install', { method: 'POST' });
        const result = await resp.json();

        if (result.success) {
            progressBar.style.width = '100%';
            progressText.textContent = result.message;
            showToast('Trivy installed successfully', 'success');
            setTimeout(() => {
                checkTrivyStatus();
                progressDiv.style.display = 'none';
            }, 1500);
        } else {
            progressBar.style.background = 'var(--accent-red)';
            progressText.textContent = 'Error: ' + result.error;
            showToast('Failed to install Trivy: ' + result.error, 'error');
        }
    } catch (e) {
        progressBar.style.background = 'var(--accent-red)';
        progressText.textContent = 'Installation failed';
        showToast('Failed to install Trivy', 'error');
    } finally {
        btn.disabled = false;
        btn.textContent = 'Install Trivy';
    }
}

async function runSecurityScan() {
    const resultDiv = document.getElementById('security-scan-result');
    resultDiv.style.display = 'block';
    resultDiv.innerHTML = '<div style="color:var(--text-secondary);"><span class="loading-spinner"></span> Running full security scan...</div>';

    try {
        const resp = await fetchWithAuth('/api/security/scan', { method: 'POST' });
        const result = await resp.json();

        if (result.error) {
            resultDiv.innerHTML = `<div style="color:var(--accent-red);">Error: ${escapeHtml(result.error)}</div>`;
            return;
        }

        // Display summary
        const critical = result.image_vulns?.severity_counts?.CRITICAL || 0;
        const high = result.image_vulns?.severity_counts?.HIGH || 0;
        const podIssues = result.pod_security_issues?.length || 0;
        const rbacIssues = result.rbac_issues?.length || 0;

        resultDiv.innerHTML = `
                    <div style="padding:12px;background:var(--bg-primary);border-radius:8px;border:1px solid var(--border-color);">
                        <div style="font-weight:600;margin-bottom:8px;">Scan Complete</div>
                        <div style="display:grid;grid-template-columns:repeat(4,1fr);gap:8px;">
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-red);">${critical}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">Critical CVEs</div>
                            </div>
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-yellow);">${high}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">High CVEs</div>
                            </div>
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-purple);">${podIssues}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">Pod Issues</div>
                            </div>
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-cyan);">${rbacIssues}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">RBAC Issues</div>
                            </div>
                        </div>
                        <div style="margin-top:8px;font-size:11px;color:var(--text-secondary);">
                            Duration: ${result.duration || 'N/A'} | Score: ${(result.overall_score || 0).toFixed(1)}/100
                        </div>
                    </div>
                `;
        showToast('Security scan completed', 'success');
    } catch (e) {
        resultDiv.innerHTML = `<div style="color:var(--accent-red);">Failed to run security scan</div>`;
        showToast('Security scan failed', 'error');
    }
}

async function runQuickSecurityScan() {
    const resultDiv = document.getElementById('security-scan-result');
    resultDiv.style.display = 'block';
    resultDiv.innerHTML = '<div style="color:var(--text-secondary);"><span class="loading-spinner"></span> Running quick scan...</div>';

    try {
        const resp = await fetchWithAuth('/api/security/scan/quick', { method: 'POST' });
        const result = await resp.json();

        if (result.error) {
            resultDiv.innerHTML = `<div style="color:var(--accent-red);">Error: ${escapeHtml(result.error)}</div>`;
            return;
        }

        const podIssues = result.pod_security_issues?.length || 0;
        const rbacIssues = result.rbac_issues?.length || 0;
        const networkIssues = result.network_issues?.length || 0;

        resultDiv.innerHTML = `
                    <div style="padding:12px;background:var(--bg-primary);border-radius:8px;border:1px solid var(--border-color);">
                        <div style="font-weight:600;margin-bottom:8px;">Quick Scan Complete</div>
                        <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:8px;">
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-purple);">${podIssues}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">Pod Issues</div>
                            </div>
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-cyan);">${rbacIssues}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">RBAC Issues</div>
                            </div>
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-yellow);">${networkIssues}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">Network Issues</div>
                            </div>
                        </div>
                        <div style="margin-top:8px;font-size:11px;color:var(--text-secondary);">
                            Score: ${(result.overall_score || 0).toFixed(1)}/100
                        </div>
                    </div>
                `;
        showToast('Quick scan completed', 'success');
    } catch (e) {
        resultDiv.innerHTML = `<div style="color:var(--accent-red);">Failed to run quick scan</div>`;
        showToast('Quick scan failed', 'error');
    }
}

// LLM Connection Test Functions
async function testLLMConnection() {
    const btn = document.getElementById('llm-test-btn');
    const btnText = document.getElementById('llm-test-btn-text');
    const indicator = document.getElementById('llm-status-indicator');
    const statusText = document.getElementById('llm-status-text');
    const statusDetail = document.getElementById('llm-status-detail');

    // Show testing state
    btn.disabled = true;
    btnText.textContent = 'Testing...';
    indicator.style.background = '#888';
    indicator.style.boxShadow = '0 0 8px rgba(136,136,136,0.5)';
    indicator.style.animation = 'pulse 1s infinite';
    statusText.textContent = 'Testing Connection...';
    statusDetail.textContent = 'Please wait...';

    // Get current form values to test
    const testConfig = {
        provider: document.getElementById('setting-llm-provider').value,
        model: document.getElementById('setting-llm-model').value,
        endpoint: document.getElementById('setting-llm-endpoint').value,
        api_key: document.getElementById('setting-llm-apikey').value
    };

    try {
        const resp = await fetchWithAuth('/api/llm/test', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(testConfig)
        });
        if (!resp.ok) {
            throw new Error(`Server error (${resp.status})`);
        }
        const status = await resp.json();

        if (status.connected) {
            // Success - green light
            indicator.style.background = '#10b981';
            indicator.style.boxShadow = '0 0 12px rgba(16,185,129,0.8)';
            indicator.style.animation = '';
            statusText.textContent = 'Connection Successful';
            statusText.style.color = 'var(--accent-green)';
            statusDetail.textContent = `${status.provider} / ${status.model} - Response time: ${status.response_time_ms}ms`;
        } else {
            // Failure - red light
            indicator.style.background = '#ef4444';
            indicator.style.boxShadow = '0 0 12px rgba(239,68,68,0.8)';
            indicator.style.animation = '';
            statusText.textContent = 'Connection Failed';
            statusText.style.color = 'var(--accent-red)';

            if (status.error === "tool calling 모델이 필요함") {
                statusText.textContent = testConfig.provider === 'ollama'
                    ? 'Ollama Tools Support Required'
                    : 'Tool Calling Not Supported';

                const extraNote = testConfig.provider === 'ollama'
                    ? `<br>${escapeHtml(getOllamaToolSupportWarning(testConfig.model))}`
                    : '';

                statusDetail.innerHTML = `<strong>tool calling 모델이 필요함</strong><br>${escapeHtml(status.message || 'Please use a model that supports functions/tools.')}${extraNote}`;
            } else {
                statusDetail.textContent = status.error || 'Unknown error';
                if (status.message) {
                    statusDetail.textContent += ' - ' + status.message;
                }
            }
        }
    } catch (e) {
        // Error - red light
        indicator.style.background = '#ef4444';
        indicator.style.boxShadow = '0 0 12px rgba(239,68,68,0.8)';
        indicator.style.animation = '';
        statusText.textContent = 'Connection Error';
        statusText.style.color = 'var(--accent-red)';
        statusDetail.textContent = e.message || 'Failed to test connection';
    } finally {
        btn.disabled = false;
        btnText.textContent = 'Test Connection';
    }
}

async function loadLLMStatus() {
    try {
        const resp = await fetchWithAuth('/api/llm/status');
        const status = await resp.json();

        const indicator = document.getElementById('llm-status-indicator');
        const statusText = document.getElementById('llm-status-text');
        const statusDetail = document.getElementById('llm-status-detail');

        if (status.configured && status.ready) {
            indicator.style.background = '#f59e0b';
            indicator.style.boxShadow = '0 0 8px rgba(245,158,11,0.5)';
            statusText.textContent = 'LLM Configured';
            statusText.style.color = 'var(--accent-yellow)';
            statusDetail.textContent = `${status.provider} / ${status.model} - Click 'Test Connection' to verify`;
        } else if (!status.configured) {
            indicator.style.background = '#888';
            indicator.style.boxShadow = '0 0 8px rgba(136,136,136,0.5)';
            statusText.textContent = 'LLM Not Configured';
            statusText.style.color = 'var(--text-secondary)';
            statusDetail.textContent = 'Configure provider, model, and API key below';
        } else {
            indicator.style.background = '#888';
            indicator.style.boxShadow = '0 0 8px rgba(136,136,136,0.5)';
            statusText.textContent = 'Configuration Incomplete';
            statusText.style.color = 'var(--text-secondary)';
            const missing = [];
            if (!status.has_api_key) missing.push('API key');
            if (!status.endpoint && !status.default_endpoint) missing.push('endpoint');
            statusDetail.textContent = missing.length > 0 ? 'Missing: ' + missing.join(', ') : 'Check configuration';
        }
    } catch (e) {
        console.error('Failed to load LLM status:', e);
    }
}

function updateEndpointPlaceholder(setDefaults = true) {
    const provider = document.getElementById('setting-llm-provider').value;
    const endpointInput = document.getElementById('setting-llm-endpoint');
    const hint = document.getElementById('endpoint-hint');

    const defaults = {
        'upstage': { placeholder: 'https://api.upstage.ai/v1', hint: '(Default: Upstage Solar API)', model: 'solar-pro2', apiKeyHint: 'up_...' },
        'openai': { placeholder: 'https://api.openai.com/v1', hint: '(Default: OpenAI API)', model: 'gpt-4o', apiKeyHint: 'sk-...' },
        'ollama': { placeholder: 'http://localhost:11434', hint: '(Required for Ollama)', model: 'gpt-oss:20b', apiKeyHint: '' },
        'gemini': { placeholder: 'https://generativelanguage.googleapis.com/v1beta', hint: '(Default: Gemini API)', model: 'gemini-2.5-flash', apiKeyHint: 'AIza...' },
        'anthropic': { placeholder: 'https://api.anthropic.com', hint: '(Default: Anthropic API)', model: 'claude-3-opus', apiKeyHint: 'sk-ant-...' },
        'bedrock': { placeholder: '', hint: '(Uses AWS credentials)', model: '', apiKeyHint: '' },
        'azopenai': { placeholder: 'https://your-resource.openai.azure.com', hint: '(Azure resource endpoint required)', model: '', apiKeyHint: '' }
    };

    const config = defaults[provider] || { placeholder: '', hint: '', model: '', apiKeyHint: '' };
    endpointInput.placeholder = config.placeholder;
    hint.textContent = config.hint;

    // Update model placeholder; only overwrite value when user switches provider
    const modelInput = document.getElementById('setting-llm-model');
    if (modelInput) {
        modelInput.placeholder = config.model || '';
        if (setDefaults && config.model) {
            modelInput.value = config.model;
        }
    }

    // Only overwrite endpoint value when user switches provider
    if (setDefaults && config.placeholder) {
        endpointInput.value = config.placeholder;
    }

    // Update API key placeholder
    const apiKeyInput = document.getElementById('setting-llm-apikey');
    if (apiKeyInput && config.apiKeyHint) {
        apiKeyInput.placeholder = config.apiKeyHint;
    }

    // Update API key link based on provider
    const apiKeyLabel = apiKeyInput?.parentElement?.querySelector('label');
    if (apiKeyLabel) {
        const existingLink = apiKeyLabel.querySelector('a');
        if (existingLink) existingLink.remove();

        const links = {
            'upstage': { url: 'https://console.upstage.ai/api-keys', text: 'Get API Key →' },
            'openai': { url: 'https://platform.openai.com/api-keys', text: 'Get API Key →' },
            'anthropic': { url: 'https://console.anthropic.com/settings/keys', text: 'Get API Key →' },
            'gemini': { url: 'https://aistudio.google.com/app/apikey', text: 'Get API Key →' }
        };

        if (links[provider]) {
            const link = document.createElement('a');
            link.href = links[provider].url;
            link.target = '_blank';
            link.style.cssText = 'font-size:11px;color:var(--accent-blue);margin-left:8px;';
            link.textContent = links[provider].text;
            apiKeyLabel.appendChild(link);
        }
    }

    // Update reasoning effort UI visibility (only for Solar)
    updateReasoningEffortUI();

    // Always show "Fetch Models" button; backend handles provider-specific logic.
    const fetchBtn = document.getElementById('fetch-models-btn');
    if (fetchBtn) {
        fetchBtn.style.display = 'inline';
    }
    // Hide model select and clear when switching providers
    const modelSelect2 = document.getElementById('setting-llm-model-select');
    if (modelSelect2) {
        modelSelect2.style.display = 'none';
        modelSelect2.innerHTML = '';
    }

    updateLLMToolSupportWarning();
}

function getOllamaToolSupportWarning(model) {
    const trimmed = (model || '').trim();
    const subject = trimmed
        ? `Ollama model "${trimmed}"`
        : 'This Ollama model';

    return `${subject} must support tools/function calling. Text-only models may connect, but the k13d AI Assistant will not work correctly. Recommended: gpt-oss:20b or another Ollama model whose card explicitly lists tools support.`;
}

function updateOllamaToolSupportWarning(targetId, provider, model) {
    const notice = document.getElementById(targetId);
    if (!notice) return;

    const show = provider === 'ollama';
    notice.style.display = show ? 'block' : 'none';
    notice.textContent = show ? getOllamaToolSupportWarning(model) : '';
}

function updateLLMToolSupportWarning() {
    updateOllamaToolSupportWarning(
        'llm-tool-support-warning',
        document.getElementById('setting-llm-provider')?.value,
        document.getElementById('setting-llm-model')?.value
    );
}

function updateNewModelToolSupportWarning() {
    updateOllamaToolSupportWarning(
        'new-model-tool-support-warning',
        document.getElementById('new-model-provider')?.value,
        document.getElementById('new-model-model')?.value
    );
}

async function fetchAvailableModels() {
    const provider = document.getElementById('setting-llm-provider').value;
    const apiKey = document.getElementById('setting-llm-apikey').value;
    const endpoint = document.getElementById('setting-llm-endpoint').value;
    const status = document.getElementById('fetch-models-status');
    const modelSelect = document.getElementById('setting-llm-model-select');
    const modelInput = document.getElementById('setting-llm-model');
    const btn = document.getElementById('fetch-models-btn');

    if (!apiKey && provider !== 'ollama') {
        status.textContent = 'API key required';
        status.style.color = 'var(--accent-red)';
        return;
    }

    btn.disabled = true;
    status.textContent = 'Fetching...';
    status.style.color = 'var(--text-secondary)';

    try {
        const resp = await fetchWithAuth('/api/llm/available-models', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ provider, api_key: apiKey, endpoint })
        });
        const data = await resp.json();

        if (data.error) {
            status.textContent = data.error;
            status.style.color = 'var(--accent-red)';
            return;
        }

        const models = data.models || [];
        if (models.length === 0) {
            status.textContent = 'No models found';
            status.style.color = 'var(--accent-yellow)';
            return;
        }

        // Populate select box with fetched models
        const currentModel = modelInput.value;
        modelSelect.innerHTML = models.map(m => {
            const escaped = escapeHtml(m);
            const selected = m === currentModel ? ' selected' : '';
            return `<option value="${escaped}"${selected}>${escaped}</option>`;
        }).join('');
        modelSelect.style.display = 'block';

        // Auto-select: keep current model if it exists in the list, otherwise use first model
        if (models.includes(currentModel)) {
            modelSelect.value = currentModel;
        } else {
            modelSelect.value = models[0];
            modelInput.value = models[0];
        }

        status.textContent = `${models.length} models available`;
        status.style.color = 'var(--accent-green)';
    } catch (e) {
        status.textContent = 'Failed to fetch';
        status.style.color = 'var(--accent-red)';
    } finally {
        btn.disabled = false;
    }
}

// Model Management Functions
async function loadModelProfiles() {
    try {
        const resp = await fetchWithAuth('/api/models');
        const data = await resp.json();
        const container = document.getElementById('models-list');

        if (!data.models || data.models.length === 0) {
            container.innerHTML = '<p style="color:var(--text-secondary);">No model profiles configured.</p>';
            return;
        }

        container.innerHTML = data.models.map(m => `
                    <div class="settings-row" style="background:var(--bg-primary);padding:12px;border-radius:8px;margin-bottom:8px;">
                        <div style="flex:1;">
                            <div style="font-weight:bold;display:flex;align-items:center;gap:8px;">
                                ${escapeHtml(m.name)}
                                ${m.is_active ? '<span style="background:var(--accent-green);color:var(--bg-primary);padding:2px 8px;border-radius:4px;font-size:10px;">ACTIVE</span>' : ''}
                                ${m.skip_tls_verify ? '<span style="background:var(--accent-yellow);color:var(--bg-primary);padding:2px 6px;border-radius:4px;font-size:10px;">TLS Skip</span>' : ''}
                            </div>
                            <div style="font-size:12px;color:var(--text-secondary);margin-top:4px;">
                                ${escapeHtml(m.provider)} / ${escapeHtml(m.model)} ${m.description ? '- ' + escapeHtml(m.description) : ''}
                            </div>
                            ${m.warning ? `<div style="margin-top:8px;font-size:12px;color:var(--accent-yellow);line-height:1.5;">${escapeHtml(m.warning)}</div>` : ''}
                        </div>
                        <div style="display:flex;gap:8px;">
                            ${!m.is_active ? `<button class="btn btn-secondary" onclick="switchModel('${escapeHtml(m.name)}')" style="padding:4px 12px;font-size:12px;">Use</button>` : ''}
                            <button class="btn btn-secondary" onclick="deleteModel('${escapeHtml(m.name)}')" style="padding:4px 12px;font-size:12px;color:var(--accent-red);">Delete</button>
                        </div>
                    </div>
                `).join('');
    } catch (e) {
        console.error('Failed to load models:', e);
    }
}

async function switchModel(name) {
    try {
        const resp = await fetchWithAuth('/api/models/active', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name })
        });
        if (!resp.ok) {
            const errData = await resp.json().catch(() => ({}));
            showToast(errData.error || 'Failed to switch model', 'error');
            return;
        }
        const data = await resp.json().catch(() => ({}));
        // Reload both profile list AND LLM form fields to stay in sync
        loadModelProfiles();
        loadSettings();
        showToast('Switched to model: ' + name, 'success');
        if (data.warning) {
            showToast(data.warning, 'warning');
        }
    } catch (e) {
        showToast('Failed to switch model: ' + e.message, 'error');
    }
}

async function deleteModel(name) {
    if (!confirm('Delete model profile "' + name + '"?')) return;
    try {
        const resp = await fetchWithAuth('/api/models?name=' + encodeURIComponent(name), {
            method: 'DELETE'
        });
        if (!resp.ok) {
            const errData = await resp.json().catch(() => ({}));
            showToast(errData.error || 'Failed to delete model', 'error');
            return;
        }
        // Reload profiles and settings (active model may have changed)
        loadModelProfiles();
        loadSettings();
        showToast('Deleted model: ' + name, 'success');
    } catch (e) {
        showToast('Failed to delete model: ' + e.message, 'error');
    }
}

function showAddModelForm() {
    syncNewModelProviderOptions();
    prefillNewModelFormFromCurrentLLM();
    document.getElementById('add-model-form').style.display = 'block';
    updateNewModelToolSupportWarning();
}

function hideAddModelForm() {
    document.getElementById('add-model-form').style.display = 'none';
    // Clear form
    document.getElementById('new-model-name').value = '';
    document.getElementById('new-model-model').value = '';
    document.getElementById('new-model-endpoint').value = '';
    document.getElementById('new-model-apikey').value = '';
    document.getElementById('new-model-description').value = '';
    document.getElementById('new-model-skip-tls').checked = false;
    updateNewModelToolSupportWarning();
}

function syncNewModelProviderOptions() {
    const currentProviderSelect = document.getElementById('setting-llm-provider');
    const newProviderSelect = document.getElementById('new-model-provider');
    if (!currentProviderSelect || !newProviderSelect) return;

    newProviderSelect.innerHTML = currentProviderSelect.innerHTML;
}

function prefillNewModelFormFromCurrentLLM() {
    const currentProvider = document.getElementById('setting-llm-provider');
    const currentModel = document.getElementById('setting-llm-model');
    const currentEndpoint = document.getElementById('setting-llm-endpoint');
    const currentAPIKey = document.getElementById('setting-llm-apikey');
    const newProvider = document.getElementById('new-model-provider');
    const newModel = document.getElementById('new-model-model');
    const newEndpoint = document.getElementById('new-model-endpoint');
    const newAPIKey = document.getElementById('new-model-apikey');

    if (currentProvider && newProvider) {
        newProvider.value = currentProvider.value;
    }
    if (currentModel && newModel) {
        newModel.value = currentModel.value || '';
    }
    if (currentEndpoint && newEndpoint) {
        newEndpoint.value = currentEndpoint.value || '';
    }
    if (currentAPIKey && newAPIKey) {
        newAPIKey.value = currentAPIKey.value || '';
    }
}

async function addModelProfile() {
    const profile = {
        name: document.getElementById('new-model-name').value.trim(),
        provider: document.getElementById('new-model-provider').value,
        model: document.getElementById('new-model-model').value.trim(),
        endpoint: document.getElementById('new-model-endpoint').value.trim(),
        api_key: document.getElementById('new-model-apikey').value,
        description: document.getElementById('new-model-description').value.trim(),
        skip_tls_verify: document.getElementById('new-model-skip-tls').checked
    };

    if (!profile.name || !profile.model) {
        showToast('Name and Model are required', 'error');
        return;
    }

    try {
        const resp = await fetchWithAuth('/api/models', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(profile)
        });
        if (!resp.ok) {
            const errData = await resp.json().catch(() => ({}));
            showToast(errData.error || 'Failed to add model', 'error');
            return;
        }
        const data = await resp.json().catch(() => ({}));
        hideAddModelForm();
        loadModelProfiles();
        showToast('Added model: ' + profile.name, 'success');
        if (data.warning) {
            showToast(data.warning, 'warning');
        }
    } catch (e) {
        showToast('Failed to add model: ' + e.message, 'error');
    }
}

// MCP Management Functions
async function loadMCPServers() {
    try {
        const resp = await fetchWithAuth('/api/mcp/servers');
        const data = await resp.json();
        const container = document.getElementById('mcp-servers-list');

        if (!data.servers || data.servers.length === 0) {
            container.innerHTML = '<p style="color:var(--text-secondary);">No MCP servers configured.</p>';
            return;
        }

        container.innerHTML = data.servers.map(s => `
                    <div class="settings-row" style="background:var(--bg-primary);padding:12px;border-radius:8px;margin-bottom:8px;">
                        <div style="flex:1;">
                            <div style="font-weight:bold;display:flex;align-items:center;gap:8px;">
                                ${escapeHtml(s.name)}
                                ${s.connected ? '<span style="background:var(--accent-green);color:var(--bg-primary);padding:2px 8px;border-radius:4px;font-size:10px;">CONNECTED</span>' : s.enabled ? '<span style="background:var(--accent-yellow);color:var(--bg-primary);padding:2px 8px;border-radius:4px;font-size:10px;">DISCONNECTED</span>' : '<span style="background:var(--bg-tertiary);padding:2px 8px;border-radius:4px;font-size:10px;">DISABLED</span>'}
                            </div>
                            <div style="font-size:12px;color:var(--text-secondary);margin-top:4px;">
                                ${escapeHtml(s.command)} ${s.args ? escapeHtml(s.args.join(' ')) : ''} ${s.description ? '- ' + escapeHtml(s.description) : ''}
                            </div>
                        </div>
                        <div style="display:flex;gap:8px;">
                            ${s.enabled ? `<button class="btn btn-secondary" onclick="toggleMCPServer('${s.name}', 'disable')" style="padding:4px 12px;font-size:12px;">Disable</button>` : `<button class="btn btn-secondary" onclick="toggleMCPServer('${s.name}', 'enable')" style="padding:4px 12px;font-size:12px;">Enable</button>`}
                            ${s.enabled ? `<button class="btn btn-secondary" onclick="toggleMCPServer('${s.name}', 'reconnect')" style="padding:4px 12px;font-size:12px;">Reconnect</button>` : ''}
                            <button class="btn btn-secondary" onclick="deleteMCPServer('${s.name}')" style="padding:4px 12px;font-size:12px;color:var(--accent-red);">Delete</button>
                        </div>
                    </div>
                `).join('');
    } catch (e) {
        console.error('Failed to load MCP servers:', e);
    }
}

async function loadMCPTools() {
    try {
        const resp = await fetchWithAuth('/api/mcp/tools');
        const data = await resp.json();
        const container = document.getElementById('mcp-tools-list');

        let html = '';

        if (data.builtin_tools && data.builtin_tools.length > 0) {
            html += '<div style="margin-bottom:12px;"><strong>Built-in Tools:</strong></div>';
            html += data.builtin_tools.map(t => `
                        <div style="background:var(--bg-primary);padding:8px 12px;border-radius:4px;margin-bottom:4px;font-size:12px;">
                            <span style="color:var(--accent-blue);">${t.name}</span>
                            <span style="color:var(--text-secondary);margin-left:8px;">${t.description || ''}</span>
                        </div>
                    `).join('');
        }

        if (data.mcp_tools && data.mcp_tools.length > 0) {
            html += '<div style="margin:12px 0;"><strong>MCP Tools:</strong></div>';
            html += data.mcp_tools.map(t => `
                        <div style="background:var(--bg-primary);padding:8px 12px;border-radius:4px;margin-bottom:4px;font-size:12px;">
                            <span style="color:var(--accent-purple);">${t.name}</span>
                            <span style="color:var(--text-secondary);margin-left:8px;">${t.description || ''}</span>
                            <span style="color:var(--accent-cyan);margin-left:8px;font-size:10px;">[${t.server}]</span>
                        </div>
                    `).join('');
        }

        if (!html) {
            html = '<p style="color:var(--text-secondary);">No tools available.</p>';
        }

        container.innerHTML = html;
    } catch (e) {
        console.error('Failed to load MCP tools:', e);
    }
}

async function toggleMCPServer(name, action) {
    try {
        await fetchWithAuth('/api/mcp/servers', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, action })
        });
        loadMCPServers();
        loadMCPTools();
    } catch (e) {
        alert('Failed to ' + action + ' MCP server: ' + e.message);
    }
}

async function deleteMCPServer(name) {
    if (!confirm('Delete MCP server "' + name + '"?')) return;
    try {
        await fetchWithAuth('/api/mcp/servers?name=' + encodeURIComponent(name), {
            method: 'DELETE'
        });
        loadMCPServers();
        loadMCPTools();
    } catch (e) {
        alert('Failed to delete MCP server: ' + e.message);
    }
}

function showAddMCPForm() {
    document.getElementById('add-mcp-form').style.display = 'block';
}

function hideAddMCPForm() {
    document.getElementById('add-mcp-form').style.display = 'none';
    // Clear form
    document.getElementById('new-mcp-name').value = '';
    document.getElementById('new-mcp-command').value = '';
    document.getElementById('new-mcp-args').value = '';
    document.getElementById('new-mcp-description').value = '';
    document.getElementById('new-mcp-enabled').checked = true;
}

async function addMCPServer() {
    const argsStr = document.getElementById('new-mcp-args').value.trim();
    const args = argsStr ? argsStr.split(',').map(a => a.trim()).filter(a => a) : [];

    const server = {
        name: document.getElementById('new-mcp-name').value.trim(),
        command: document.getElementById('new-mcp-command').value.trim(),
        args: args,
        description: document.getElementById('new-mcp-description').value.trim(),
        enabled: document.getElementById('new-mcp-enabled').checked
    };

    if (!server.name || !server.command) {
        alert('Name and Command are required');
        return;
    }

    try {
        const resp = await fetchWithAuth('/api/mcp/servers', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(server)
        });
        const data = await resp.json();
        hideAddMCPForm();
        loadMCPServers();
        loadMCPTools();
        if (data.warning) {
            alert(data.warning);
        }
    } catch (e) {
        alert('Failed to add MCP server: ' + e.message);
    }
}

// === Feature Permissions ===
let userPermissions = {};

async function loadUserPermissions() {
    try {
        const resp = await fetchWithAuth('/api/auth/permissions');
        if (resp.ok) {
            const data = await resp.json();
            userPermissions = data.features || {};
            applyFeaturePermissions();
        }
    } catch (e) {
        console.warn('Failed to load permissions:', e);
    }
}

function hasFeature(name) {
    if (!userPermissions || Object.keys(userPermissions).length === 0) return true;
    return userPermissions[name] === true;
}

function applyFeaturePermissions() {
    const featureMap = {
        'topology': 'topology',
        'reports': 'reports',
        'helm': 'helm',
        'security': 'security_scanning',
    };
    document.querySelectorAll('.sidebar-item[data-view]').forEach(item => {
        const view = item.getAttribute('data-view');
        const feature = featureMap[view];
        if (feature && !hasFeature(feature)) {
            item.style.display = 'none';
        }
    });
}

// === Roles Management ===
async function loadRoles() {
    try {
        const resp = await fetchWithAuth('/api/roles');
        if (!resp.ok) return;
        const roles = await resp.json();
        const container = document.getElementById('roles-list-container');
        if (!container) return;

        let html = '<table class="data-table" style="width:100%;"><thead><tr><th>Role</th><th>Type</th><th>Features</th><th>Actions</th></tr></thead><tbody>';
        for (const role of roles) {
            const type = role.is_custom ? '<span style="color:var(--accent-color);">Custom</span>' : 'Built-in';
            const featureCount = role.allowed_features ? (role.allowed_features.includes('*') ? 'All' : role.allowed_features.length) : 0;
            const actions = role.is_custom ? `<button class="btn btn-sm" onclick="editRole('${escapeHtml(role.name)}')">Edit</button> <button class="btn btn-sm btn-danger" onclick="deleteRole('${escapeHtml(role.name)}')">Delete</button>` : '<span style="color:var(--text-secondary);">Protected</span>';
            html += `<tr><td><strong>${escapeHtml(role.name)}</strong></td><td>${type}</td><td>${featureCount}</td><td>${actions}</td></tr>`;
        }
        html += '</tbody></table>';
        container.innerHTML = html;
    } catch (e) {
        console.error('Failed to load roles:', e);
    }
}

function closeRoleModal() {
    document.getElementById('role-editor-modal')?.remove();
}

function buildRoleFeatureCheckboxes(selectedFeatures = []) {
    const allFeatures = ['dashboard', 'topology', 'reports', 'metrics', 'helm', 'terminal', 'rbac_viewer', 'network_policy', 'event_timeline', 'ai_assistant', 'security_scanning', 'audit_logs', 'settings_general', 'settings_ai', 'settings_metrics', 'settings_mcp', 'settings_notifications'];
    const selectAll = selectedFeatures.includes('*');
    return allFeatures.map(f => {
        const checked = selectAll || selectedFeatures.includes(f) ? 'checked' : '';
        return `<label style="display:block;margin:4px 0;"><input type="checkbox" value="${f}" ${checked}> ${f.replace(/_/g, ' ')}</label>`;
    }).join('');
}

function showRoleModal(options = {}) {
    const {
        title = 'Create Custom Role',
        name = '',
        description = '',
        selectedFeatures = [],
        submitLabel = 'Create',
        submitAction = "createRole()"
    } = options;

    closeRoleModal();

    const modal = document.createElement('div');
    modal.id = 'role-editor-modal';
    modal.className = 'modal-overlay';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.innerHTML = `<div class="modal-content" style="max-width:500px;max-height:80vh;overflow-y:auto;">
                <h3>${escapeHtml(title)}</h3>
                <div class="form-group"><label>Role Name</label><input type="text" id="new-role-name" class="form-control" placeholder="e.g., developer" value="${escapeHtml(name)}" ${name ? 'readonly' : ''}></div>
                <div class="form-group"><label>Description</label><input type="text" id="new-role-desc" class="form-control" placeholder="e.g., Developer with limited access" value="${escapeHtml(description)}"></div>
                <div class="form-group"><label>Allowed Features</label><div id="new-role-features" style="max-height:300px;overflow-y:auto;border:1px solid var(--border-color);padding:8px;border-radius:4px;">${buildRoleFeatureCheckboxes(selectedFeatures)}</div></div>
                <div style="display:flex;gap:8px;margin-top:16px;">
                    <button class="btn btn-primary" onclick="${submitAction}">${escapeHtml(submitLabel)}</button>
                    <button class="btn" onclick="closeRoleModal()">Cancel</button>
                </div>
            </div>`;
    document.body.appendChild(modal);
    modal.classList.add('active');
}

async function showCreateRoleModal() {
    showRoleModal({
        title: 'Create Custom Role',
        selectedFeatures: ['dashboard', 'topology', 'reports', 'metrics', 'helm', 'terminal', 'rbac_viewer', 'network_policy', 'event_timeline', 'ai_assistant', 'security_scanning', 'audit_logs', 'settings_general', 'settings_ai', 'settings_metrics', 'settings_mcp', 'settings_notifications'],
        submitLabel: 'Create',
        submitAction: 'createRole()'
    });
}

async function createRole() {
    const name = document.getElementById('new-role-name').value.trim();
    const desc = document.getElementById('new-role-desc').value.trim();
    if (!name) { showToast('Role name is required', 'error'); return; }

    const features = [];
    document.querySelectorAll('#new-role-features input:checked').forEach(cb => features.push(cb.value));

    try {
        const resp = await fetchWithAuth('/api/roles', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, description: desc, allowed_features: features, is_custom: true })
        });
        if (resp.ok) {
            showToast('Role created successfully');
            closeRoleModal();
            loadRoles();
        } else {
            const err = await resp.text();
            showToast(err, 'error');
        }
    } catch (e) {
        showToast('Failed to create role', 'error');
    }
}

async function editRole(name) {
    try {
        const resp = await fetchWithAuth('/api/roles/' + encodeURIComponent(name));
        if (!resp.ok) {
            showToast('Failed to load role', 'error');
            return;
        }
        const role = await resp.json();
        showRoleModal({
            title: `Edit Role: ${name}`,
            name,
            description: role.description || '',
            selectedFeatures: role.allowed_features || [],
            submitLabel: 'Save',
            submitAction: `updateRole(${JSON.stringify(name)})`
        });
    } catch (e) {
        showToast('Failed to load role', 'error');
    }
}

async function updateRole(name) {
    const desc = document.getElementById('new-role-desc').value.trim();
    const features = [];
    document.querySelectorAll('#new-role-features input:checked').forEach(cb => features.push(cb.value));

    try {
        const resp = await fetchWithAuth('/api/roles/' + encodeURIComponent(name), {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ description: desc, allowed_features: features, is_custom: true })
        });
        if (resp.ok) {
            showToast('Role updated successfully');
            closeRoleModal();
            loadRoles();
        } else {
            showToast(await resp.text(), 'error');
        }
    } catch (e) {
        showToast('Failed to update role', 'error');
    }
}

async function deleteRole(name) {
    if (!confirm(`Delete role "${name}"?`)) return;
    try {
        const resp = await fetchWithAuth(`/api/roles/${name}`, { method: 'DELETE' });
        if (resp.ok) {
            showToast('Role deleted');
            loadRoles();
        } else {
            showToast(await resp.text(), 'error');
        }
    } catch (e) {
        showToast('Failed to delete role', 'error');
    }
}

// === Tool Approval Settings ===
async function loadToolApprovalSettings() {
    try {
        const resp = await fetchWithAuth('/api/settings/tool-approval');
        if (!resp.ok) return;
        const policy = await resp.json();

        const setToggle = (id, active) => {
            const el = document.getElementById(id);
            if (el) el.classList.toggle('active', active);
        };
        setToggle('ta-auto-approve-ro', policy.auto_approve_read_only === true);
        setToggle('ta-require-write', policy.require_approval_for_write !== false);
        setToggle('ta-block-dangerous', policy.block_dangerous === true);
        setToggle('ta-require-unknown', policy.require_approval_for_unknown !== false);

        const timeout = document.getElementById('ta-timeout');
        if (timeout) timeout.value = policy.approval_timeout_seconds || 60;

        const patterns = document.getElementById('ta-blocked-patterns');
        if (patterns) patterns.value = (policy.blocked_patterns || []).join('\n');
    } catch (e) {
        console.error('Failed to load tool approval settings:', e);
    }
}

function toggleToolApproval(el) {
    el.classList.toggle('active');
}

async function saveToolApprovalSettings() {
    const policy = {
        auto_approve_read_only: document.getElementById('ta-auto-approve-ro')?.classList.contains('active') ?? false,
        require_approval_for_write: document.getElementById('ta-require-write')?.classList.contains('active') ?? true,
        block_dangerous: document.getElementById('ta-block-dangerous')?.classList.contains('active') ?? false,
        require_approval_for_unknown: document.getElementById('ta-require-unknown')?.classList.contains('active') ?? true,
        approval_timeout_seconds: parseInt(document.getElementById('ta-timeout')?.value) || 60,
        blocked_patterns: (document.getElementById('ta-blocked-patterns')?.value || '').split('\n').filter(l => l.trim()),
    };
    try {
        const resp = await fetchWithAuth('/api/settings/tool-approval', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(policy)
        });
        if (resp.ok) showToast('Tool approval settings saved');
        else showToast('Failed to save settings', 'error');
    } catch (e) {
        showToast('Failed to save settings', 'error');
    }
}

// === Agent Settings ===
async function loadAgentSettings() {
    try {
        const resp = await fetchWithAuth('/api/settings/agent');
        if (!resp.ok) return;
        const data = await resp.json();

        const maxIter = document.getElementById('agent-max-iterations');
        if (maxIter) { maxIter.value = data.max_iterations || 10; document.getElementById('agent-max-iter-val').textContent = maxIter.value; }

        const effort = document.getElementById('agent-reasoning-effort');
        if (effort) effort.value = data.reasoning_effort || 'medium';

        const temp = document.getElementById('agent-temperature');
        if (temp) { temp.value = Math.round((data.temperature || 0.7) * 100); document.getElementById('agent-temp-val').textContent = (temp.value / 100).toFixed(1); }

        const tokens = document.getElementById('agent-max-tokens');
        if (tokens) tokens.value = data.max_tokens || 4096;
    } catch (e) {
        console.error('Failed to load agent settings:', e);
    }
}

async function saveAgentSettings() {
    const settings = {
        max_iterations: parseInt(document.getElementById('agent-max-iterations')?.value) || 10,
        reasoning_effort: document.getElementById('agent-reasoning-effort')?.value || 'medium',
        temperature: parseInt(document.getElementById('agent-temperature')?.value || '70') / 100,
        max_tokens: parseInt(document.getElementById('agent-max-tokens')?.value) || 4096,
    };
    try {
        const resp = await fetchWithAuth('/api/settings/agent', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(settings)
        });
        if (resp.ok) {
            showToast('Agent settings saved');
        } else {
            const errData = await resp.json().catch(() => ({}));
            const errMsg = errData.message || errData.error || `Failed to save settings (${resp.status})`;
            showToast(errMsg, 'error');
        }
    } catch (e) {
        showToast('Failed to save settings', 'error');
    }
}

// Admin User Management Functions
async function loadAdminUsers() {
    try {
        const resp = await fetchWithAuth('/api/admin/users');
        if (!resp.ok) {
            if (resp.status === 403) {
                document.getElementById('admin-users-list').innerHTML = '<p style="color:var(--accent-red);">Access denied. Admin role required.</p>';
                return;
            }
            throw new Error('Failed to load users');
        }
        const data = await resp.json();
        const container = document.getElementById('admin-users-list');

        if (!data.users || data.users.length === 0) {
            container.innerHTML = '<p style="color:var(--text-secondary);">No users found.</p>';
            return;
        }

        container.innerHTML = data.users.map(u => `
                    <div class="settings-row" style="background:var(--bg-primary);padding:12px;border-radius:8px;margin-bottom:8px;">
                        <div style="flex:1;">
                            <div style="font-weight:bold;display:flex;align-items:center;gap:8px;">
                                ${escapeHtml(u.username)}
                                <span style="background:${u.role === 'admin' ? 'var(--accent-red)' : u.role === 'user' ? 'var(--accent-blue)' : 'var(--bg-tertiary)'};color:${u.role === 'admin' || u.role === 'user' ? '#fff' : 'var(--text-primary)'};padding:2px 8px;border-radius:4px;font-size:10px;text-transform:uppercase;">${u.role}</span>
                                <span style="background:var(--bg-tertiary);padding:2px 8px;border-radius:4px;font-size:10px;">${u.source || 'local'}</span>
                            </div>
                            <div style="font-size:12px;color:var(--text-secondary);margin-top:4px;">
                                ${u.email ? escapeHtml(u.email) + ' · ' : ''}Last login: ${u.last_login ? new Date(u.last_login).toLocaleString() : 'Never'}
                            </div>
                        </div>
                        <div style="display:flex;gap:8px;">
                            ${u.source === 'local' ? `
                                <button class="btn btn-secondary" onclick="showResetPasswordModal('${escapeHtml(u.username)}')" style="padding:4px 12px;font-size:12px;">Reset Password</button>
                                <button class="btn btn-secondary" onclick="deleteUser('${escapeHtml(u.username)}')" style="padding:4px 12px;font-size:12px;color:var(--accent-red);">Delete</button>
                            ` : '<span style="font-size:11px;color:var(--text-secondary);">External user</span>'}
                        </div>
                    </div>
                `).join('');
    } catch (e) {
        console.error('Failed to load admin users:', e);
        document.getElementById('admin-users-list').innerHTML = '<p style="color:var(--accent-red);">Failed to load users.</p>';
    }
}

async function loadAuthStatus() {
    try {
        const resp = await fetchWithAuth('/api/admin/status');
        if (!resp.ok) return;
        const data = await resp.json();

        // Update current auth mode display
        const currentModeEl = document.getElementById('current-auth-mode');
        if (currentModeEl) {
            const modeLabels = {
                'local': 'Local (Username/Password)',
                'token': 'Kubernetes Token',
                'oidc': 'OIDC/OAuth SSO',
                'ldap': 'LDAP/Active Directory'
            };
            currentModeEl.textContent = modeLabels[data.auth_mode] || data.auth_mode || 'Unknown';
        }

        // Set the auth mode select to current value
        const authModeSelect = document.getElementById('auth-mode');
        if (authModeSelect && data.auth_mode) {
            authModeSelect.value = data.auth_mode;
            onAuthModeChange(data.auth_mode);
        }

        const redirectURI = document.getElementById('oidc-redirect-uri');
        if (redirectURI && !redirectURI.value) {
            redirectURI.value = `${window.location.origin}/api/auth/oidc/callback`;
        }

        if (data.oidc_configured) {
            try {
                const oidcResp = await fetchWithAuth('/api/auth/oidc/status');
                if (oidcResp.ok) {
                    const oidc = await oidcResp.json();
                    if (document.getElementById('oidc-provider-name')) {
                        document.getElementById('oidc-provider-name').value = oidc.provider_name || '';
                    }
                    if (document.getElementById('oidc-provider-url')) {
                        document.getElementById('oidc-provider-url').value = oidc.provider_url || '';
                    }
                    if (document.getElementById('oidc-client-id')) {
                        document.getElementById('oidc-client-id').value = oidc.client_id || '';
                    }
                    if (document.getElementById('oidc-scopes')) {
                        document.getElementById('oidc-scopes').value = oidc.scopes || 'openid email profile';
                    }
                    if (document.getElementById('oidc-redirect-uri')) {
                        document.getElementById('oidc-redirect-uri').value = oidc.redirect_uri || `${window.location.origin}/api/auth/oidc/callback`;
                    }
                    if (document.getElementById('oauth-admin-roles')) {
                        document.getElementById('oauth-admin-roles').value = (oidc.admin_roles || []).join(', ');
                    }
                    if (document.getElementById('oauth-roles-claim')) {
                        document.getElementById('oauth-roles-claim').value = 'roles';
                    }
                }
            } catch (configErr) {
                console.log('OIDC status not available:', configErr);
            }
        }

        if (data.ldap_enabled) {
            try {
                const ldapResp = await fetchWithAuth('/api/auth/ldap/status');
                if (ldapResp.ok) {
                    const ldapStatus = await ldapResp.json();
                    const ldap = ldapStatus.config || ldapStatus;
                    const ldapScheme = ldap.use_tls ? 'ldaps' : 'ldap';

                    if (document.getElementById('ldap-server-url')) {
                        document.getElementById('ldap-server-url').value = ldap.host ? `${ldapScheme}://${ldap.host}:${ldap.port || ''}` : '';
                    }
                    if (document.getElementById('ldap-bind-dn')) {
                        document.getElementById('ldap-bind-dn').value = ldap.bind_dn || '';
                    }
                    if (document.getElementById('ldap-user-search-base')) {
                        document.getElementById('ldap-user-search-base').value = ldap.user_search_base || ldap.base_dn || '';
                    }
                    if (document.getElementById('ldap-user-search-filter')) {
                        document.getElementById('ldap-user-search-filter').value = ldap.user_search_filter || '(uid=%s)';
                    }
                    if (document.getElementById('ldap-group-search-base')) {
                        document.getElementById('ldap-group-search-base').value = ldap.group_search_base || '';
                    }
                    if (document.getElementById('ldap-group-search-filter')) {
                        document.getElementById('ldap-group-search-filter').value = ldap.group_search_filter || '';
                    }
                    if (document.getElementById('ldap-admin-group')) {
                        document.getElementById('ldap-admin-group').value = (ldap.admin_groups || []).join(', ');
                    }
                }
            } catch (configErr) {
                console.log('LDAP status not available:', configErr);
            }
        }

        setAuthSettingsRuntimeOnly(
            'Authentication provider settings are loaded at server startup and are read-only in the current Web UI. Update startup configuration, then restart k13d.',
            { allowLDAPTest: data.auth_mode === 'ldap' && data.ldap_enabled }
        );
    } catch (e) {
        console.error('Failed to load auth status:', e);
    }
}

function showAddUserForm() {
    document.getElementById('add-user-form').style.display = 'block';
}

function hideAddUserForm() {
    document.getElementById('add-user-form').style.display = 'none';
    document.getElementById('new-user-username').value = '';
    document.getElementById('new-user-password').value = '';
    document.getElementById('new-user-email').value = '';
    document.getElementById('new-user-role').value = 'viewer';
}

async function addUser() {
    const user = {
        username: document.getElementById('new-user-username').value.trim(),
        password: document.getElementById('new-user-password').value,
        email: document.getElementById('new-user-email').value.trim(),
        role: document.getElementById('new-user-role').value
    };

    if (!user.username || !user.password) {
        alert('Username and password are required');
        return;
    }

    try {
        const resp = await fetchWithAuth('/api/admin/users', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(user)
        });

        if (!resp.ok) {
            const error = await resp.text();
            throw new Error(error);
        }

        hideAddUserForm();
        loadAdminUsers();
        alert('User created successfully');
    } catch (e) {
        alert('Failed to create user: ' + e.message);
    }
}

async function deleteUser(username) {
    if (!confirm('Delete user "' + username + '"? This action cannot be undone.')) return;

    try {
        const resp = await fetchWithAuth('/api/admin/users/' + encodeURIComponent(username), {
            method: 'DELETE'
        });

        if (!resp.ok) {
            const error = await resp.text();
            throw new Error(error);
        }

        loadAdminUsers();
        alert('User deleted successfully');
    } catch (e) {
        alert('Failed to delete user: ' + e.message);
    }
}

function showResetPasswordModal(username) {
    const newPassword = prompt('Enter new password for ' + username + ':');
    if (!newPassword) return;

    resetUserPassword(username, newPassword);
}

async function resetUserPassword(username, newPassword) {
    try {
        const resp = await fetchWithAuth('/api/admin/reset-password', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, new_password: newPassword })
        });

        if (!resp.ok) {
            const error = await resp.text();
            throw new Error(error);
        }

        alert('Password reset successfully');
    } catch (e) {
        alert('Failed to reset password: ' + e.message);
    }
}

async function loadSettings() {
    try {
        const resp = await fetchWithAuth('/api/settings');
        const data = await resp.json();
        currentLanguage = data.language || 'ko';
        document.getElementById('setting-language').value = currentLanguage;
        document.getElementById('setting-log-level').value = data.log_level || 'info';
        // Load timezone setting
        if (data.timezone) {
            appTimezone = data.timezone;
            localStorage.setItem('k13d_timezone', appTimezone);
        }
        const tzSelect = document.getElementById('setting-timezone');
        if (tzSelect) tzSelect.value = appTimezone || 'auto';
        if (data.llm) {
            const provider = data.llm.provider || 'upstage';
            document.getElementById('setting-llm-provider').value = provider;

            // Set model and endpoint with defaults based on provider
            const defaults = {
                'upstage': { model: 'solar-pro2', endpoint: 'https://api.upstage.ai/v1' },
                'openai': { model: 'gpt-4o', endpoint: 'https://api.openai.com/v1' },
                'ollama': { model: 'gpt-oss:20b', endpoint: 'http://localhost:11434' },
                'gemini': { model: 'gemini-2.5-flash', endpoint: 'https://generativelanguage.googleapis.com/v1beta' },
                'anthropic': { model: 'claude-3-opus', endpoint: 'https://api.anthropic.com' }
            };
            const providerDefaults = defaults[provider] || { model: '', endpoint: '' };

            document.getElementById('setting-llm-model').value = data.llm.model || providerDefaults.model;
            document.getElementById('setting-llm-endpoint').value = data.llm.endpoint || providerDefaults.endpoint;
            currentLLMModel = data.llm.model || providerDefaults.model;

            // Load reasoning effort setting
            if (data.llm.reasoning_effort) {
                reasoningEffort = data.llm.reasoning_effort;
                localStorage.setItem('k13d_reasoning_effort', reasoningEffort);
            }
        } else {
            // No LLM config from server, set Upstage defaults
            document.getElementById('setting-llm-provider').value = 'upstage';
            document.getElementById('setting-llm-model').value = 'solar-pro2';
            document.getElementById('setting-llm-endpoint').value = 'https://api.upstage.ai/v1';
            currentLLMModel = 'solar-pro2';
        }
        // Update endpoint placeholder/hints without overwriting loaded values
        updateEndpointPlaceholder(false);
        syncNewModelProviderOptions();
        updateLLMToolSupportWarning();
        // Load local settings
        updateSettingsUI();
        // Update AI panel status
        updateAIStatus();
        // Update UI language based on loaded settings
        updateUILanguage();
        // Load Prometheus settings
        loadPrometheusSettings();
    } catch (e) {
        console.error('Failed to load settings:', e);
    }
}

// Prometheus Settings Functions
async function loadPrometheusSettings() {
    try {
        const resp = await fetchWithAuth('/api/prometheus/settings');
        const data = await resp.json();

        document.getElementById('prometheus-expose-metrics').checked = data.expose_metrics || false;
        document.getElementById('prometheus-external-url').value = data.external_url || '';
        document.getElementById('prometheus-collect-k8s').checked = data.collect_k8s_metrics !== false;
        document.getElementById('prometheus-collection-interval').value = data.collection_interval || 60;
        document.getElementById('prometheus-retention-days').value = data.metrics_retention_days || 30;

        updatePrometheusExposeInfo();
        updatePrometheusStatus(data.expose_metrics, data.external_url);
        updateMetricsStorageInfo();
    } catch (e) {
        console.error('Failed to load Prometheus settings:', e);
    }
}

function updateMetricsStorageInfo() {
    const info = document.getElementById('metrics-storage-info');
    const retention = parseInt(document.getElementById('prometheus-retention-days')?.value || '30', 10);
    if (!info) return;
    info.textContent = `Metrics are stored in local SQLite and retained for ${retention} day${retention === 1 ? '' : 's'}.`;
}

function updatePrometheusExposeInfo() {
    const isChecked = document.getElementById('prometheus-expose-metrics').checked;
    document.getElementById('prometheus-expose-info').style.display = isChecked ? 'block' : 'none';
}

function updatePrometheusStatus(exposeEnabled, externalUrl) {
    const statusEl = document.getElementById('prometheus-status');
    if (!statusEl) return;

    if (externalUrl) {
        statusEl.classList.add('connected');
        statusEl.classList.remove('disconnected');
        statusEl.querySelector('span').textContent = 'Prometheus Connected';
    } else if (exposeEnabled) {
        statusEl.classList.remove('connected', 'disconnected');
        statusEl.querySelector('span').textContent = 'Prometheus: Exposing';
    } else {
        statusEl.classList.remove('connected');
        statusEl.classList.add('disconnected');
        statusEl.querySelector('span').textContent = 'Metrics Source';
    }

    // Check metrics-server availability
    fetchWithAuth('/api/metrics/nodes').then(resp => resp.json()).then(data => {
        if (!data.error && data.items && data.items.length > 0) {
            // Check if real CPU/Memory data exists
            const hasMetrics = data.items.some(n => (n.cpu || 0) > 0 || (n.memory || 0) > 0);
            if (hasMetrics) {
                statusEl.classList.add('connected');
                statusEl.classList.remove('disconnected');
                const currentText = statusEl.querySelector('span').textContent;
                if (!currentText.includes('Prometheus')) {
                    statusEl.querySelector('span').textContent = 'metrics-server: Connected';
                }
            } else {
                if (!statusEl.classList.contains('connected')) {
                    statusEl.querySelector('span').textContent = 'metrics-server: N/A';
                }
            }
        }
    }).catch(() => { });
}

async function testPrometheusConnection() {
    const url = document.getElementById('prometheus-external-url').value;
    const username = document.getElementById('prometheus-username').value;
    const password = document.getElementById('prometheus-password').value;
    const resultEl = document.getElementById('prometheus-test-result');

    if (!url) {
        resultEl.style.display = 'block';
        resultEl.style.background = 'rgba(247, 118, 142, 0.1)';
        resultEl.style.color = 'var(--accent-red)';
        resultEl.innerHTML = 'Please enter a Prometheus URL';
        return;
    }

    resultEl.style.display = 'block';
    resultEl.style.background = 'var(--bg-primary)';
    resultEl.style.color = 'var(--text-secondary)';
    resultEl.innerHTML = 'Testing connection...';

    try {
        const resp = await fetchWithAuth('/api/prometheus/test', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url, username, password })
        });
        const data = await resp.json();

        if (data.success) {
            resultEl.style.background = 'rgba(158, 206, 106, 0.1)';
            resultEl.style.color = 'var(--accent-green)';
            resultEl.innerHTML = `✓ Connected successfully! Prometheus version: ${data.version || 'unknown'}`;
        } else {
            resultEl.style.background = 'rgba(247, 118, 142, 0.1)';
            resultEl.style.color = 'var(--accent-red)';
            resultEl.innerHTML = `✗ Connection failed: ${data.error}`;
        }
    } catch (e) {
        resultEl.style.background = 'rgba(247, 118, 142, 0.1)';
        resultEl.style.color = 'var(--accent-red)';
        resultEl.innerHTML = `✗ Error: ${e.message}`;
    }
}

async function savePrometheusSettings() {
    const settings = {
        expose_metrics: document.getElementById('prometheus-expose-metrics').checked,
        external_url: document.getElementById('prometheus-external-url').value,
        username: document.getElementById('prometheus-username').value,
        password: document.getElementById('prometheus-password').value,
        collect_k8s_metrics: document.getElementById('prometheus-collect-k8s').checked,
        collection_interval: parseInt(document.getElementById('prometheus-collection-interval').value),
        metrics_retention_days: parseInt(document.getElementById('prometheus-retention-days').value)
    };

    try {
        const resp = await fetchWithAuth('/api/prometheus/settings', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(settings)
        });
        if (!resp.ok) {
            const errData = await resp.json().catch(() => ({}));
            const errMsg = errData.message || errData.error || `Failed to save Prometheus settings (${resp.status})`;
            showToast(errMsg, 'error');
            return;
        }
        showToast(t('msg_settings_saved') || 'Settings saved');
        updatePrometheusStatus(settings.expose_metrics, settings.external_url);
        updateMetricsStorageInfo();
    } catch (e) {
        showToast('Failed to save Prometheus settings', 'error');
    }
}

async function cleanupOldMetrics() {
    try {
        await fetchWithAuth('/api/metrics/collect', { method: 'POST' });
        showToast('Metrics cleanup initiated');
    } catch (e) {
        showToast('Failed to cleanup metrics', 'error');
    }
}

function toggleMetricsAutoRefresh() {
    const checkbox = document.getElementById('metrics-auto-refresh');
    if (checkbox.checked) {
        if (!metricsInterval) {
            metricsInterval = setInterval(loadMetrics, 30000);
        }
    } else {
        if (metricsInterval) {
            clearInterval(metricsInterval);
            metricsInterval = null;
        }
    }
}

// Load version info for About page
async function loadVersionInfo() {
    try {
        const resp = await fetch('/api/version');
        const data = await resp.json();

        const versionEl = document.getElementById('about-version');
        const buildTimeEl = document.getElementById('about-build-time');
        const gitCommitEl = document.getElementById('about-git-commit');

        if (versionEl) {
            versionEl.textContent = data.version || 'dev';
            // Add badge for dev version
            if (data.version === 'dev') {
                versionEl.innerHTML = '<span style="color: var(--accent-yellow);">dev</span> <span style="font-size: 10px; color: var(--text-muted);">(development build)</span>';
            }
        }
        if (buildTimeEl) {
            if (data.build_time && data.build_time !== 'unknown') {
                // Format the date nicely
                const date = new Date(data.build_time);
                if (!isNaN(date.getTime())) {
                    buildTimeEl.textContent = date.toLocaleString();
                } else {
                    buildTimeEl.textContent = data.build_time;
                }
            } else {
                buildTimeEl.textContent = '-';
            }
        }
        if (gitCommitEl) {
            if (data.git_commit && data.git_commit !== 'unknown') {
                // Show shortened commit hash
                const shortCommit = data.git_commit.substring(0, 7);
                gitCommitEl.textContent = shortCommit;
                gitCommitEl.title = data.git_commit;
            } else {
                gitCommitEl.textContent = '-';
            }
        }
    } catch (e) {
        console.error('Failed to load version info:', e);
    }
}

// Update AI Assistant panel with model name and connection status
async function updateAIStatus() {
    const statusDot = document.getElementById('ai-status-dot');
    const modelBadge = document.getElementById('ai-model-badge');

    if (!statusDot || !modelBadge) return;

    // Show checking state
    statusDot.className = 'ai-status-dot checking';
    statusDot.title = 'Checking connection...';

    try {
        // Get LLM settings to display model name
        const settingsResp = await fetchWithAuth('/api/settings');
        const settings = await settingsResp.json();

        if (settings.llm && settings.llm.model) {
            currentLLMModel = settings.llm.model;
            modelBadge.textContent = currentLLMModel;
            modelBadge.title = `${settings.llm.provider || 'openai'}: ${currentLLMModel}`;
        } else {
            modelBadge.textContent = 'Not configured';
            modelBadge.title = 'AI model not configured';
            statusDot.className = 'ai-status-dot disconnected';
            statusDot.title = 'AI not configured';
            llmConnected = false;
            return;
        }

        // Ping test - try to check LLM connection
        const pingResp = await fetchWithAuth('/api/ai/ping');
        if (pingResp.ok) {
            statusDot.className = 'ai-status-dot connected';
            statusDot.title = 'Connected';
            llmConnected = true;
        } else {
            statusDot.className = 'ai-status-dot disconnected';
            statusDot.title = 'Connection failed';
            llmConnected = false;
        }
    } catch (e) {
        console.error('Failed to check AI status:', e);
        statusDot.className = 'ai-status-dot disconnected';
        statusDot.title = 'Connection error';
        modelBadge.textContent = 'Error';
        llmConnected = false;
    }
}

function updateSettingsUI() {
    const streamingToggle = document.getElementById('setting-streaming');
    const autoRefreshToggle = document.getElementById('setting-auto-refresh');
    const intervalSelect = document.getElementById('setting-refresh-interval');

    if (streamingToggle) {
        streamingToggle.classList.toggle('active', useStreaming);
    }
    if (autoRefreshToggle) {
        autoRefreshToggle.classList.toggle('active', autoRefreshEnabled);
    }
    if (intervalSelect) {
        intervalSelect.value = autoRefreshInterval;
    }
}

function toggleStreamingSetting() {
    useStreaming = !useStreaming;
    localStorage.setItem('k13d_use_streaming', useStreaming);
    updateSettingsUI();
}

function toggleAutoRefreshSetting() {
    toggleAutoRefresh();
    updateSettingsUI();
}

function setAutoRefreshIntervalSetting(value) {
    setAutoRefreshInterval(parseInt(value));
    updateSettingsUI();
}

function toggleReasoningEffort() {
    reasoningEffort = reasoningEffort === 'minimal' ? 'high' : 'minimal';
    localStorage.setItem('k13d_reasoning_effort', reasoningEffort);
    updateReasoningEffortUI();
    // Save to server config
    saveReasoningEffortToServer();
}

function updateReasoningEffortUI() {
    const toggle = document.getElementById('reasoning-effort-toggle');
    const status = document.getElementById('reasoning-effort-status');
    const section = document.getElementById('reasoning-effort-section');
    const provider = document.getElementById('setting-llm-provider')?.value;

    // Show/hide section based on provider (only for upstage)
    if (section) {
        section.style.display = (provider === 'upstage') ? 'block' : 'none';
    }

    if (toggle) {
        toggle.classList.toggle('active', reasoningEffort === 'high');
    }
    if (status) {
        status.textContent = reasoningEffort === 'high'
            ? 'Current: high (deeper reasoning enabled)'
            : 'Current: minimal (default)';
    }
}

async function saveReasoningEffortToServer() {
    const provider = document.getElementById('setting-llm-provider')?.value;
    const model = document.getElementById('setting-llm-model')?.value;
    const endpoint = document.getElementById('setting-llm-endpoint')?.value || '';

    if (!provider || !model) {
        return;
    }

    try {
        const resp = await fetchWithAuth('/api/settings/llm', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                provider,
                model,
                endpoint,
                reasoning_effort: reasoningEffort
            })
        });

        if (!resp.ok) {
            throw new Error(`Failed to save reasoning effort (${resp.status})`);
        }
    } catch (e) {
        console.error('Failed to save reasoning effort:', e);
    }
}

// SSO/Authentication settings handlers
const AUTH_RUNTIME_INPUT_IDS = [
    'auth-mode',
    'oidc-provider-name',
    'oidc-provider-url',
    'oidc-client-id',
    'oidc-client-secret',
    'oidc-scopes',
    'oidc-redirect-uri',
    'oauth-roles-claim',
    'oauth-admin-roles',
    'oauth-allowed-roles',
    'ldap-server-url',
    'ldap-bind-dn',
    'ldap-bind-password',
    'ldap-user-search-base',
    'ldap-user-search-filter',
    'ldap-group-search-base',
    'ldap-group-search-filter',
    'ldap-admin-group'
];

const AUTH_RUNTIME_TOGGLE_IDS = [
    'allow-password-login',
    'enable-signup',
    'oauth-signup',
    'oauth-merge-email',
    'oauth-role-mgmt'
];

function setAuthSettingsRuntimeOnly(note, options = {}) {
    const noteEl = document.getElementById('auth-runtime-note');
    if (noteEl) {
        noteEl.textContent = note;
    }

    AUTH_RUNTIME_INPUT_IDS.forEach(id => {
        const el = document.getElementById(id);
        if (el) {
            el.disabled = true;
        }
    });

    AUTH_RUNTIME_TOGGLE_IDS.forEach(id => {
        const el = document.getElementById(id);
        if (!el) return;
        el.dataset.disabled = 'true';
        el.style.pointerEvents = 'none';
        el.style.opacity = '0.5';
    });

    const saveBtn = document.getElementById('auth-save-btn');
    if (saveBtn) {
        saveBtn.disabled = true;
        saveBtn.title = 'Authentication provider settings are configured at startup in the current build.';
    }

    const testBtn = document.getElementById('auth-ldap-test-btn');
    if (testBtn) {
        const allowLDAPTest = !!options.allowLDAPTest;
        testBtn.disabled = !allowLDAPTest;
        testBtn.title = allowLDAPTest
            ? 'Test the configured runtime LDAP connection'
            : 'LDAP test is only available when the server is already running with LDAP configured';
    }
}

function onAuthModeChange(mode) {
    const oidcSection = document.getElementById('oidc-settings');
    const ldapSection = document.getElementById('ldap-settings');
    const oauthRoleSection = document.getElementById('oauth-role-settings');

    // Hide all sections first
    if (oidcSection) oidcSection.style.display = 'none';
    if (ldapSection) ldapSection.style.display = 'none';
    if (oauthRoleSection) oauthRoleSection.style.display = 'none';

    // Show relevant section based on mode
    if (mode === 'oidc') {
        if (oidcSection) oidcSection.style.display = 'block';
        if (oauthRoleSection) oauthRoleSection.style.display = 'block';
    } else if (mode === 'ldap') {
        if (ldapSection) ldapSection.style.display = 'block';
    }
}

function toggleAllowPasswordLogin() {
    const toggle = document.getElementById('allow-password-login');
    if (toggle && toggle.dataset.disabled !== 'true') {
        toggle.classList.toggle('active');
    }
}

function toggleEnableSignup() {
    const toggle = document.getElementById('enable-signup');
    if (toggle && toggle.dataset.disabled !== 'true') {
        toggle.classList.toggle('active');
    }
}

async function testLDAPConnection() {
    const btn = document.getElementById('auth-ldap-test-btn') || event?.target;
    if (!btn || btn.disabled) return;

    const authMode = document.getElementById('auth-mode')?.value;
    if (authMode !== 'ldap') {
        alert('LDAP test is only available when the server is already running in LDAP mode.');
        return;
    }

    const originalText = btn.textContent;
    btn.textContent = 'Testing...';
    btn.disabled = true;

    try {
        const resp = await fetchWithAuth('/api/auth/ldap/test', {
            method: 'POST'
        });

        const result = await resp.json();
        if (resp.ok && result.status === 'ok') {
            const serverURL = document.getElementById('ldap-server-url')?.value || 'configured server';
            alert('Configured LDAP connection successful.\n\nServer: ' + serverURL);
        } else {
            alert('LDAP connection failed:\n' + (result.error || 'Unknown error'));
        }
    } catch (e) {
        alert('LDAP connection test failed:\n' + e.message);
    } finally {
        btn.textContent = originalText;
        btn.disabled = false;
    }
}

function getAuthSettings() {
    const mode = document.getElementById('auth-mode')?.value || 'local';
    const settings = { mode };

    if (mode === 'oidc') {
        settings.oidc = {
            provider_url: document.getElementById('oidc-provider-url')?.value || '',
            client_id: document.getElementById('oidc-client-id')?.value || '',
            client_secret: document.getElementById('oidc-client-secret')?.value || '',
            scopes: document.getElementById('oidc-scopes')?.value || 'openid profile email',
            redirect_uri: document.getElementById('oidc-redirect-uri')?.value || ''
        };
        settings.oauth_roles = {
            roles_claim: document.getElementById('oauth-roles-claim')?.value || 'roles',
            admin_roles: document.getElementById('oauth-admin-roles')?.value || '',
            allowed_roles: document.getElementById('oauth-allowed-roles')?.value || ''
        };
        settings.allow_password_login = document.getElementById('allow-password-login')?.classList.contains('active') || false;
        settings.enable_signup = document.getElementById('enable-signup')?.classList.contains('active') || false;
    } else if (mode === 'ldap') {
        settings.ldap = {
            server_url: document.getElementById('ldap-server-url')?.value || '',
            bind_dn: document.getElementById('ldap-bind-dn')?.value || '',
            bind_password: document.getElementById('ldap-bind-password')?.value || '',
            user_search_base: document.getElementById('ldap-user-search-base')?.value || '',
            user_search_filter: document.getElementById('ldap-user-search-filter')?.value || '(uid={{username}})',
            group_search_base: document.getElementById('ldap-group-search-base')?.value || '',
            group_search_filter: document.getElementById('ldap-group-search-filter')?.value || '',
            admin_group: document.getElementById('ldap-admin-group')?.value || ''
        };
    }

    return settings;
}

async function saveAuthSettings() {
    alert(
        'Authentication provider settings are startup-configured in the current build.\n\n' +
        'Use --auth-mode plus your startup configuration, then restart k13d.\n' +
        'See the Security / Configuration docs for LDAP, OIDC, and SAML guidance.'
    );
}

async function saveSettings() {
    try {
        // Save general settings (including timezone)
        const newTimezone = document.getElementById('setting-timezone')?.value || 'auto';
        const settingsResp = await fetchWithAuth('/api/settings', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                language: document.getElementById('setting-language').value,
                log_level: document.getElementById('setting-log-level').value,
                timezone: newTimezone
            })
        });

        if (!settingsResp.ok) {
            const errData = await settingsResp.json().catch(() => ({}));
            const errMsg = errData.message || errData.error || `General settings error (${settingsResp.status})`;
            showToast(errMsg, 'error');
            return;
        }
        // Apply timezone immediately
        appTimezone = newTimezone;
        localStorage.setItem('k13d_timezone', appTimezone);

        // Save LLM settings
        const apiKey = document.getElementById('setting-llm-apikey').value;
        const llmResp = await fetchWithAuth('/api/settings/llm', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                provider: document.getElementById('setting-llm-provider').value,
                model: document.getElementById('setting-llm-model').value,
                endpoint: document.getElementById('setting-llm-endpoint').value,
                api_key: apiKey,
                reasoning_effort: reasoningEffort
            })
        });

        if (!llmResp.ok) {
            const errData = await llmResp.json().catch(() => ({}));
            const errMsg = errData.message || errData.error || `LLM settings error (${llmResp.status})`;
            showToast(errMsg, 'error');
            return;
        }

        // Update current language for AI responses
        currentLanguage = document.getElementById('setting-language').value;
        updateUILanguage();

        closeSettings();
        showToast(t('msg_settings_saved'));

        // Update AI status (model name and connection status)
        updateAIStatus();
    } catch (e) {
        alert('Failed to save settings');
    }
}

(function(global) {
    'use strict';

    const GUARDRAILS_STORAGE_KEY = 'k13d-guardrails';
    const RISK_STYLES = {
        safe: { color: 'var(--accent-green)', icon: '✓', label: 'Safe' },
        warning: { color: 'var(--accent-yellow)', icon: '⚠', label: 'Warning' },
        dangerous: { color: 'var(--accent-red)', icon: '⚡', label: 'Dangerous' },
        critical: { color: '#ff4757', icon: '☠', label: 'Critical' }
    };

    let guardrailsConfig = {
        enabled: true,
        strictMode: false,
        autoAnalyze: true,
        currentNamespace: 'default',
        recentAnalysis: null,
        analysisHistory: []
    };

    function loadGuardrailsConfig() {
        try {
            const saved = localStorage.getItem(GUARDRAILS_STORAGE_KEY);
            if (saved) {
                guardrailsConfig = { ...guardrailsConfig, ...JSON.parse(saved) };
            }
        } catch (e) {
            console.error('Failed to load guardrails config:', e);
        }
        updateGuardrailsUI();
    }

    function saveGuardrailsConfig() {
        try {
            localStorage.setItem(GUARDRAILS_STORAGE_KEY, JSON.stringify(guardrailsConfig));
        } catch (e) {
            console.error('Failed to save guardrails config:', e);
        }
    }

    async function analyzeK8sSafety(command, namespace = null) {
        if (!guardrailsConfig.enabled) {
            return { safe: true, riskLevel: 'safe', allowed: true };
        }

        try {
            const response = await fetchWithAuth('/api/safety/analyze', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    command,
                    namespace: namespace || guardrailsConfig.currentNamespace || currentNamespace
                })
            });

            if (!response.ok) {
                console.error('Safety analysis failed:', response.status);
                return { safe: true, riskLevel: 'safe', allowed: true };
            }

            const analysis = await response.json();
            guardrailsConfig.recentAnalysis = analysis;
            guardrailsConfig.analysisHistory.unshift({
                command,
                analysis,
                timestamp: Date.now()
            });
            if (guardrailsConfig.analysisHistory.length > 50) {
                guardrailsConfig.analysisHistory.pop();
            }
            saveGuardrailsConfig();
            updateGuardrailsUI(analysis);

            return {
                ...analysis,
                allowed: !analysis.requires_approval || !guardrailsConfig.strictMode
            };
        } catch (e) {
            console.error('Safety analysis error:', e);
            return { safe: true, riskLevel: 'safe', allowed: true };
        }
    }

    function checkGuardrails(message) {
        if (!guardrailsConfig.enabled) {
            return { allowed: true };
        }

        const lowerMessage = message.toLowerCase();
        const criticalPatterns = [
            { pattern: 'delete namespace', reason: 'Deleting a namespace removes ALL resources within it' },
            { pattern: 'delete ns ', reason: 'Deleting a namespace removes ALL resources within it' },
            { pattern: '--all-namespaces', reason: 'Operation affects ALL namespaces in the cluster' },
            { pattern: 'drain node', reason: 'Draining a node evicts all pods' },
            { pattern: 'delete node', reason: 'Deleting a node removes it from the cluster' },
            { pattern: '--force --grace-period=0', reason: 'Force deletion bypasses graceful termination' },
            { pattern: 'rm -rf', reason: 'Recursive file deletion is dangerous' },
        ];

        for (const { pattern, reason } of criticalPatterns) {
            if (lowerMessage.includes(pattern)) {
                return {
                    allowed: !guardrailsConfig.strictMode,
                    requireConfirmation: true,
                    riskLevel: 'critical',
                    reason,
                    dangerous: true
                };
            }
        }

        const dangerousPatterns = [
            { pattern: 'delete deployment', reason: 'Deleting deployments stops all pods' },
            { pattern: 'delete statefulset', reason: 'StatefulSet deletion can cause data issues' },
            { pattern: 'delete service', reason: 'Deleting services breaks connectivity' },
            { pattern: 'delete pvc', reason: 'PVC deletion can cause data loss' },
            { pattern: 'delete secret', reason: 'Deleting secrets can break applications' },
            { pattern: 'scale --replicas=0', reason: 'Scaling to zero stops all pods' },
        ];

        for (const { pattern, reason } of dangerousPatterns) {
            if (lowerMessage.includes(pattern)) {
                return {
                    allowed: true,
                    requireConfirmation: true,
                    riskLevel: 'dangerous',
                    reason
                };
            }
        }

        const warningPatterns = [
            { pattern: 'delete pod', reason: 'Pod deletion causes temporary unavailability' },
            { pattern: 'scale ', reason: 'Scaling changes running pod count' },
            { pattern: 'rollout restart', reason: 'Restart causes temporary unavailability' },
            { pattern: 'apply ', reason: 'Applying changes modifies cluster state' },
            { pattern: 'patch ', reason: 'Patching modifies resource configuration' },
        ];

        for (const { pattern, reason } of warningPatterns) {
            if (lowerMessage.includes(pattern)) {
                return {
                    allowed: true,
                    requireConfirmation: true,
                    riskLevel: 'warning',
                    reason
                };
            }
        }

        const productionIndicators = ['prod', 'production', 'live', 'main', 'master'];
        for (const indicator of productionIndicators) {
            if (lowerMessage.includes(indicator)) {
                return {
                    allowed: true,
                    requireConfirmation: true,
                    riskLevel: 'warning',
                    reason: 'Possible production environment detected'
                };
            }
        }

        return { allowed: true, riskLevel: 'safe' };
    }

    function showSafetyConfirmation(analysis, onConfirm, onCancel) {
        const style = RISK_STYLES[analysis.riskLevel] || RISK_STYLES.warning;
        const modal = document.createElement('div');
        modal.className = 'modal-overlay';
        modal.id = 'safety-confirmation-modal';
        modal.innerHTML = `
                <div class="modal" style="max-width: 500px;">
                    <div class="modal-header" style="background: ${style.color}20; border-bottom: 2px solid ${style.color};">
                        <h2 style="color: ${style.color};">${style.icon} ${style.label}: Safety Check Required</h2>
                        <button class="modal-close" onclick="closeSafetyConfirmation(false)">&times;</button>
                    </div>
                    <div class="modal-body" style="padding: 20px;">
                        <div style="margin-bottom: 16px;">
                            <strong style="color: ${style.color};">Risk Level:</strong> ${analysis.riskLevel.toUpperCase()}
                        </div>

                        ${analysis.explanation ? `
                        <div style="margin-bottom: 16px; padding: 12px; background: var(--bg-tertiary); border-radius: 8px;">
                            ${analysis.explanation}
                        </div>
                        ` : ''}

                        ${analysis.warnings && analysis.warnings.length > 0 ? `
                        <div style="margin-bottom: 16px;">
                            <strong>Warnings:</strong>
                            <ul style="margin: 8px 0; padding-left: 20px; color: var(--accent-yellow);">
                                ${analysis.warnings.map(w => `<li>${w}</li>`).join('')}
                            </ul>
                        </div>
                        ` : ''}

                        ${analysis.recommendations && analysis.recommendations.length > 0 ? `
                        <div style="margin-bottom: 16px;">
                            <strong>Recommendations:</strong>
                            <ul style="margin: 8px 0; padding-left: 20px; color: var(--text-secondary);">
                                ${analysis.recommendations.map(r => `<li>${r}</li>`).join('')}
                            </ul>
                        </div>
                        ` : ''}

                        <div style="margin-top: 20px; padding: 12px; background: ${style.color}10; border: 1px solid ${style.color}40; border-radius: 8px;">
                            <strong>Do you want to proceed with this action?</strong>
                        </div>
                    </div>
                    <div class="modal-footer" style="display: flex; gap: 12px; justify-content: flex-end;">
                        <button class="btn btn-secondary" onclick="closeSafetyConfirmation(false)">Cancel</button>
                        <button class="btn" style="background: ${style.color};" onclick="closeSafetyConfirmation(true)">
                            Proceed Anyway
                        </button>
                    </div>
                </div>
            `;

        document.body.appendChild(modal);
        global._safetyConfirmCallbacks = { onConfirm, onCancel };
    }

    function closeSafetyConfirmation(confirmed) {
        const modal = document.getElementById('safety-confirmation-modal');
        if (modal) {
            modal.remove();
        }

        const callbacks = global._safetyConfirmCallbacks;
        if (!callbacks) {
            return;
        }

        if (confirmed && callbacks.onConfirm) {
            callbacks.onConfirm();
        } else if (!confirmed && callbacks.onCancel) {
            callbacks.onCancel();
        }
        delete global._safetyConfirmCallbacks;
    }

    function updateGuardrailsUI(analysis = null) {
        const indicator = document.getElementById('guardrails-indicator');
        const limitDisplay = document.getElementById('guardrails-limit');
        if (!indicator || !limitDisplay) {
            return;
        }

        if (!guardrailsConfig.enabled) {
            indicator.className = 'guardrails-indicator warning';
            indicator.innerHTML = '<span class="dot"></span><span>Protection Off</span>';
            limitDisplay.textContent = 'K8s Safety: Disabled';
            return;
        }

        if (analysis) {
            const style = RISK_STYLES[analysis.risk_level] || RISK_STYLES.safe;
            indicator.className = `guardrails-indicator ${analysis.risk_level || 'safe'}`;
            indicator.innerHTML = `<span class="dot" style="background: ${style.color};"></span><span>${style.label}</span>`;
            limitDisplay.textContent = `Last: ${analysis.category || 'read-only'} | ${analysis.affected_scope || 'pod'}`;
            return;
        }

        indicator.className = 'guardrails-indicator safe';
        indicator.innerHTML = '<span class="dot"></span><span>Protected</span>';
        limitDisplay.textContent = 'K8s Safety: Active';
    }

    function toggleGuardrailsSetting() {
        const toggle = document.getElementById('guardrails-toggle');
        if (!toggle) {
            return;
        }
        guardrailsConfig.enabled = !guardrailsConfig.enabled;
        toggle.classList.toggle('active', guardrailsConfig.enabled);
        saveGuardrailsConfig();
        updateGuardrailsUI();
    }

    function toggleStrictMode() {
        const toggle = document.getElementById('guardrails-strict-toggle');
        if (!toggle) {
            return;
        }
        guardrailsConfig.strictMode = !guardrailsConfig.strictMode;
        toggle.classList.toggle('active', guardrailsConfig.strictMode);
        saveGuardrailsConfig();
        showToast(
            guardrailsConfig.strictMode
                ? 'Strict mode enabled - dangerous operations will be blocked'
                : 'Strict mode disabled - dangerous operations will require confirmation',
            guardrailsConfig.strictMode ? 'warning' : 'info'
        );
    }

    function toggleAutoAnalyze() {
        const toggle = document.getElementById('guardrails-auto-analyze');
        if (!toggle) {
            return;
        }
        guardrailsConfig.autoAnalyze = !guardrailsConfig.autoAnalyze;
        toggle.classList.toggle('active', guardrailsConfig.autoAnalyze);
        saveGuardrailsConfig();
    }

    function clearGuardrailsHistory() {
        guardrailsConfig.analysisHistory = [];
        guardrailsConfig.recentAnalysis = null;
        saveGuardrailsConfig();
        updateGuardrailsHistoryUI();
        updateGuardrailsUI();
        showToast('Safety check history cleared', 'success');
    }

    function updateGuardrailsHistoryUI() {
        const historyDiv = document.getElementById('guardrails-history');
        if (!historyDiv) {
            return;
        }

        if (!guardrailsConfig.analysisHistory || guardrailsConfig.analysisHistory.length === 0) {
            historyDiv.innerHTML = '<div style="color:var(--text-secondary); font-size:13px;">No recent checks</div>';
            return;
        }

        historyDiv.innerHTML = guardrailsConfig.analysisHistory.slice(0, 10).map((item) => {
            const style = RISK_STYLES[item.analysis.risk_level] || RISK_STYLES.safe;
            const time = formatTime(item.timestamp);
            const cmd = item.command.length > 50 ? `${item.command.substring(0, 47)}...` : item.command;
            return `
                    <div style="display:flex; align-items:center; gap:8px; padding:6px 0; border-bottom:1px solid var(--border-color);">
                        <span style="color:${style.color}; font-size:14px;">${style.icon}</span>
                        <span style="flex:1; font-size:12px; font-family:monospace; color:var(--text-secondary);" title="${item.command}">${cmd}</span>
                        <span style="font-size:11px; color:var(--text-secondary);">${time}</span>
                    </div>
                `;
        }).join('');
    }

    function loadGuardrailsSettingsUI() {
        document.getElementById('guardrails-toggle')?.classList.toggle('active', guardrailsConfig.enabled);
        document.getElementById('guardrails-strict-toggle')?.classList.toggle('active', guardrailsConfig.strictMode || false);
        document.getElementById('guardrails-auto-analyze')?.classList.toggle('active', guardrailsConfig.autoAnalyze !== false);
        updateGuardrailsHistoryUI();
    }

    function onLLMTabOpened() {
        loadGuardrailsSettingsUI();
    }

    global.analyzeK8sSafety = analyzeK8sSafety;
    global.checkGuardrails = checkGuardrails;
    global.showSafetyConfirmation = showSafetyConfirmation;
    global.closeSafetyConfirmation = closeSafetyConfirmation;
    global.toggleGuardrailsSetting = toggleGuardrailsSetting;
    global.toggleStrictMode = toggleStrictMode;
    global.toggleAutoAnalyze = toggleAutoAnalyze;
    global.clearGuardrailsHistory = clearGuardrailsHistory;
    global.loadGuardrailsSettingsUI = loadGuardrailsSettingsUI;
    global.onLLMTabOpened = onLLMTabOpened;

    loadGuardrailsConfig();
})(window);

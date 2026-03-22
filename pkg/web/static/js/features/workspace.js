function closeAllModals() {
    document.querySelectorAll('.modal-overlay').forEach(m => m.classList.remove('active'));
    closeCommandBar();
    closeYamlEditor();
}

async function initWorkspaceFeatures() {
    loadRecentNamespaces();
    await loadChatHistory();
}

// ==================== Command Bar (TUI-style : mode) ====================
const commandDefinitions = [
    // Resource commands
    { name: 'pods', alias: ['po', 'pod'], desc: 'View Pods', action: () => switchResource('pods') },
    { name: 'deployments', alias: ['deploy', 'dep'], desc: 'View Deployments', action: () => switchResource('deployments') },
    { name: 'services', alias: ['svc', 'service'], desc: 'View Services', action: () => switchResource('services') },
    { name: 'statefulsets', alias: ['sts'], desc: 'View StatefulSets', action: () => switchResource('statefulsets') },
    { name: 'daemonsets', alias: ['ds'], desc: 'View DaemonSets', action: () => switchResource('daemonsets') },
    { name: 'replicasets', alias: ['rs'], desc: 'View ReplicaSets', action: () => switchResource('replicasets') },
    { name: 'configmaps', alias: ['cm'], desc: 'View ConfigMaps', action: () => switchResource('configmaps') },
    { name: 'secrets', alias: ['sec'], desc: 'View Secrets', action: () => switchResource('secrets') },
    { name: 'ingresses', alias: ['ing'], desc: 'View Ingresses', action: () => switchResource('ingresses') },
    { name: 'jobs', alias: ['job'], desc: 'View Jobs', action: () => switchResource('jobs') },
    { name: 'cronjobs', alias: ['cj'], desc: 'View CronJobs', action: () => switchResource('cronjobs') },
    { name: 'nodes', alias: ['no', 'node'], desc: 'View Nodes', action: () => switchResource('nodes') },
    { name: 'namespaces', alias: ['ns'], desc: 'View Namespaces', action: () => switchResource('namespaces') },
    { name: 'pvcs', alias: ['pvc'], desc: 'View PVCs', action: () => switchResource('pvcs') },
    { name: 'pvs', alias: ['pv'], desc: 'View PVs', action: () => switchResource('pvs') },
    { name: 'events', alias: ['ev'], desc: 'View Events', action: () => switchResource('events') },
    { name: 'serviceaccounts', alias: ['sa'], desc: 'View Service Accounts', action: () => switchResource('serviceaccounts') },
    { name: 'roles', alias: ['role'], desc: 'View Roles', action: () => switchResource('roles') },
    { name: 'rolebindings', alias: ['rb'], desc: 'View RoleBindings', action: () => switchResource('rolebindings') },
    { name: 'clusterroles', alias: ['cr'], desc: 'View ClusterRoles', action: () => switchResource('clusterroles') },
    { name: 'clusterrolebindings', alias: ['crb'], desc: 'View ClusterRoleBindings', action: () => switchResource('clusterrolebindings') },
    // Actions
    { name: 'refresh', alias: ['r', 'reload'], desc: 'Refresh current data', action: () => refreshData() },
    { name: 'ai', alias: ['assistant', 'chat'], desc: 'Toggle AI Panel', action: () => toggleAIPanel() },
    { name: 'settings', alias: ['config', 'set'], desc: 'Open Settings', action: () => showSettings() },
    { name: 'help', alias: ['?', 'h'], desc: 'Show Shortcuts', action: () => showShortcuts() },
    { name: 'yaml', alias: ['edit', 'create'], desc: 'Open YAML Editor', action: () => openYamlEditor() },
    { name: 'metrics', alias: ['metric'], desc: 'Show Metrics View', action: () => document.getElementById('metrics-tab')?.click() },
    { name: 'audit', alias: ['log', 'history'], desc: 'Show Audit Logs', action: () => document.getElementById('audit-tab')?.click() },
];

let commandSelectedIndex = 0;
let filteredCommands = [];

function openCommandBar() {
    const overlay = document.getElementById('command-bar-overlay');
    const input = document.getElementById('command-input');
    overlay.classList.add('active');
    input.value = '';
    input.focus();
    commandSelectedIndex = 0;
    updateCommandSuggestions('');
}

function closeCommandBar() {
    document.getElementById('command-bar-overlay').classList.remove('active');
}

function handleCommandBarKeydown(e) {
    const input = document.getElementById('command-input');

    switch (e.key) {
        case 'Escape':
            e.preventDefault();
            closeCommandBar();
            break;
        case 'ArrowDown':
            e.preventDefault();
            commandSelectedIndex = Math.min(commandSelectedIndex + 1, filteredCommands.length - 1);
            renderCommandSuggestions();
            break;
        case 'ArrowUp':
            e.preventDefault();
            commandSelectedIndex = Math.max(commandSelectedIndex - 1, 0);
            renderCommandSuggestions();
            break;
        case 'Tab':
            e.preventDefault();
            if (filteredCommands.length > 0) {
                input.value = filteredCommands[commandSelectedIndex].name;
                updateCommandSuggestions(input.value);
            }
            break;
        case 'Enter':
            e.preventDefault();
            executeSelectedCommand();
            break;
        default:
            // Let input handle it, then update suggestions
            setTimeout(() => updateCommandSuggestions(input.value), 0);
    }
}

function updateCommandSuggestions(query) {
    query = query.toLowerCase().trim();

    if (!query) {
        filteredCommands = commandDefinitions.slice(0, 15);
    } else {
        filteredCommands = commandDefinitions.filter(cmd => {
            if (cmd.name.startsWith(query)) return true;
            if (cmd.alias.some(a => a.startsWith(query))) return true;
            if (cmd.desc.toLowerCase().includes(query)) return true;
            return false;
        }).slice(0, 10);
    }

    commandSelectedIndex = 0;
    renderCommandSuggestions();
}

function renderCommandSuggestions() {
    const container = document.getElementById('command-suggestions');

    if (filteredCommands.length === 0) {
        container.innerHTML = '<div class="command-suggestion" style="color: var(--text-secondary);">No matching commands</div>';
        return;
    }

    container.innerHTML = filteredCommands.map((cmd, i) => `
                <div class="command-suggestion ${i === commandSelectedIndex ? 'selected' : ''}"
                     onclick="executeCommand(${i})"
                     onmouseover="commandSelectedIndex = ${i}; renderCommandSuggestions();">
                    <div>
                        <span class="command-suggestion-name">${cmd.name}</span>
                        <span class="command-suggestion-desc"> - ${cmd.desc}</span>
                    </div>
                    <span class="command-suggestion-shortcut">${cmd.alias[0] || ''}</span>
                </div>
            `).join('');
}

function executeCommand(index) {
    if (filteredCommands[index]) {
        closeCommandBar();
        filteredCommands[index].action();
    }
}

function executeSelectedCommand() {
    const input = document.getElementById('command-input').value.trim().toLowerCase();

    // First try exact match
    const exactMatch = commandDefinitions.find(cmd =>
        cmd.name === input || cmd.alias.includes(input)
    );

    if (exactMatch) {
        closeCommandBar();
        exactMatch.action();
        return;
    }

    // Otherwise execute selected suggestion
    if (filteredCommands[commandSelectedIndex]) {
        closeCommandBar();
        filteredCommands[commandSelectedIndex].action();
    }
}

// Click outside to close command bar
document.getElementById('command-bar-overlay')?.addEventListener('click', (e) => {
    if (e.target.id === 'command-bar-overlay') {
        closeCommandBar();
    }
});

// ==================== Namespace Quick Switcher ====================
let recentNamespaces = [];
let namespaceIndicatorTimeout = null;

function trackNamespaceUsage(ns) {
    // Remove if already exists
    recentNamespaces = recentNamespaces.filter(n => n !== ns);
    // Add to front
    if (ns) {
        recentNamespaces.unshift(ns);
    }
    // Keep max 9
    recentNamespaces = recentNamespaces.slice(0, 9);
    // Save to localStorage
    localStorage.setItem('k13d-recent-namespaces', JSON.stringify(recentNamespaces));
}

function loadRecentNamespaces() {
    try {
        const saved = localStorage.getItem('k13d-recent-namespaces');
        if (saved) {
            recentNamespaces = JSON.parse(saved);
        }
    } catch (e) {
        console.error('Failed to load recent namespaces:', e);
    }
}

function switchToRecentNamespace(index) {
    if (index === 0) {
        // All namespaces
        document.getElementById('namespace-select').value = '';
        currentNamespace = '';
        onNamespaceChange();
        showToast('Switched to All Namespaces');
        return;
    }

    const ns = recentNamespaces[index - 1];
    if (ns) {
        document.getElementById('namespace-select').value = ns;
        currentNamespace = ns;
        onNamespaceChange();
        showToast(`Switched to namespace: ${ns}`);
    }
}

function showNamespaceIndicator() {
    const indicator = document.getElementById('namespace-indicator');

    // Use recent namespaces, or fall back to available namespaces from selector
    let nsList = recentNamespaces.slice(0, 9);
    if (nsList.length === 0) {
        const nsSelect = document.getElementById('namespace-select');
        if (nsSelect) {
            for (const opt of nsSelect.options) {
                if (opt.value && nsList.length < 9) {
                    nsList.push(opt.value);
                }
            }
        }
    }

    // Build namespace keys
    let html = `
                <div class="namespace-key ${!currentNamespace ? 'current' : ''}" onclick="switchToRecentNamespace(0)">
                    <span class="namespace-key-num">0</span>
                    <span class="namespace-key-name">All</span>
                </div>
            `;

    for (let i = 0; i < 9; i++) {
        const ns = nsList[i];
        const isCurrent = ns && ns === currentNamespace;
        html += `
                    <div class="namespace-key ${isCurrent ? 'current' : ''} ${!ns ? 'disabled' : ''}"
                         onclick="${ns ? `switchToNamespaceByName('${ns}')` : ''}"
                         style="${!ns ? 'opacity: 0.3; cursor: default;' : ''}">
                        <span class="namespace-key-num">${i + 1}</span>
                        <span class="namespace-key-name">${ns || '-'}</span>
                    </div>
                `;
    }

    indicator.innerHTML = html;
    indicator.classList.add('active');

    // Auto hide after 3 seconds
    if (namespaceIndicatorTimeout) {
        clearTimeout(namespaceIndicatorTimeout);
    }
    namespaceIndicatorTimeout = setTimeout(hideNamespaceIndicator, 3000);
}

function switchToNamespaceByName(ns) {
    document.getElementById('namespace-select').value = ns;
    currentNamespace = ns;
    trackNamespaceUsage(ns);
    onNamespaceChange();
    showToast(`Switched to namespace: ${ns}`);
    hideNamespaceIndicator();
}

function hideNamespaceIndicator() {
    document.getElementById('namespace-indicator').classList.remove('active');
    if (namespaceIndicatorTimeout) {
        clearTimeout(namespaceIndicatorTimeout);
        namespaceIndicatorTimeout = null;
    }
}

// ==================== YAML Editor ====================
const yamlTemplates = [
    {
        title: 'Pod',
        desc: 'Basic Pod template',
        yaml: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  namespace: default
  labels:
    app: my-app
spec:
  containers:
  - name: main
    image: nginx:latest
    ports:
    - containerPort: 80
    resources:
      limits:
        memory: "128Mi"
        cpu: "500m"`
    },
    {
        title: 'Deployment',
        desc: 'Deployment with replicas',
        yaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: main
        image: nginx:latest
        ports:
        - containerPort: 80`
    },
    {
        title: 'Service',
        desc: 'ClusterIP Service',
        yaml: `apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
spec:
  selector:
    app: my-app
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
  type: ClusterIP`
    },
    {
        title: 'ConfigMap',
        desc: 'Configuration data',
        yaml: `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
data:
  config.json: |
    {
      "key": "value"
    }
  APP_ENV: production`
    },
    {
        title: 'Secret',
        desc: 'Opaque Secret',
        yaml: `apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: default
type: Opaque
stringData:
  username: admin
  password: changeme`
    },
    {
        title: 'Ingress',
        desc: 'HTTP Ingress rule',
        yaml: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  namespace: default
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-service
            port:
              number: 80`
    },
    {
        title: 'CronJob',
        desc: 'Scheduled job',
        yaml: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: my-cronjob
  namespace: default
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: job
            image: busybox
            command: ["echo", "Hello"]
          restartPolicy: OnFailure`
    },
    {
        title: 'PVC',
        desc: 'Persistent Volume Claim',
        yaml: `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
  namespace: default
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi`
    }
];

let yamlEditorMode = 'create'; // 'create' or 'edit'
let yamlEditingResource = null;

function openYamlEditor(existingYaml = null, resourceInfo = null) {
    const modal = document.getElementById('yaml-editor-modal');
    const textarea = document.getElementById('yaml-editor-content');
    const modeLabel = document.getElementById('yaml-editor-mode');
    const nsSelect = document.getElementById('yaml-editor-namespace');

    // Populate namespace select
    nsSelect.innerHTML = document.getElementById('namespace-select').innerHTML;
    nsSelect.value = currentNamespace || '';

    // Render templates
    renderYamlTemplates();

    if (existingYaml) {
        textarea.value = existingYaml;
        yamlEditorMode = 'edit';
        modeLabel.textContent = 'Edit';
        modeLabel.style.background = 'var(--accent-yellow)';
        yamlEditingResource = resourceInfo;
    } else {
        textarea.value = '';
        yamlEditorMode = 'create';
        modeLabel.textContent = 'Create';
        modeLabel.style.background = 'var(--accent-blue)';
        yamlEditingResource = null;
    }

    updateYamlEditorStatus('valid', 'Ready');
    modal.classList.add('active');
    textarea.focus();
}

function closeYamlEditor() {
    document.getElementById('yaml-editor-modal').classList.remove('active');
}

function renderYamlTemplates() {
    const container = document.getElementById('yaml-template-list');
    container.innerHTML = yamlTemplates.map((tpl, i) => `
                <div class="yaml-template-item" onclick="loadYamlTemplate(${i})">
                    <div class="yaml-template-item-title">${tpl.title}</div>
                    <div class="yaml-template-item-desc">${tpl.desc}</div>
                </div>
            `).join('');
}

function loadYamlTemplate(index) {
    const tpl = yamlTemplates[index];
    if (tpl) {
        const textarea = document.getElementById('yaml-editor-content');
        // Replace namespace in template
        const ns = document.getElementById('yaml-editor-namespace').value || 'default';
        let yaml = tpl.yaml.replace(/namespace: default/g, `namespace: ${ns}`);
        textarea.value = yaml;
        updateYamlEditorStatus('valid', 'Template loaded');
    }
}

function validateYaml() {
    const yaml = document.getElementById('yaml-editor-content').value;

    if (!yaml.trim()) {
        updateYamlEditorStatus('invalid', 'YAML is empty');
        return false;
    }

    // Basic validation
    if (!yaml.includes('apiVersion:')) {
        updateYamlEditorStatus('invalid', 'Missing apiVersion');
        return false;
    }
    if (!yaml.includes('kind:')) {
        updateYamlEditorStatus('invalid', 'Missing kind');
        return false;
    }
    if (!yaml.includes('metadata:')) {
        updateYamlEditorStatus('invalid', 'Missing metadata');
        return false;
    }

    updateYamlEditorStatus('valid', 'YAML is valid');
    return true;
}

function formatYaml() {
    // Simple formatting - just normalize indentation
    const textarea = document.getElementById('yaml-editor-content');
    const yaml = textarea.value;

    try {
        // Basic cleanup
        let formatted = yaml
            .replace(/\t/g, '  ')  // Tabs to spaces
            .replace(/  +$/gm, '') // Trailing spaces
            .replace(/\n{3,}/g, '\n\n'); // Multiple blank lines

        textarea.value = formatted;
        updateYamlEditorStatus('valid', 'Formatted');
    } catch (e) {
        updateYamlEditorStatus('invalid', 'Format error: ' + e.message);
    }
}

async function applyYaml() {
    const yaml = document.getElementById('yaml-editor-content').value;
    const dryRun = document.getElementById('yaml-dry-run').checked;
    const namespace = document.getElementById('yaml-editor-namespace').value || 'default';

    if (!validateYaml()) {
        return;
    }

    updateYamlEditorStatus('valid', dryRun ? 'Validating (dry-run)...' : 'Applying...');

    try {
        const resp = await fetchWithAuth('/api/k8s/apply', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                yaml: yaml,
                namespace: namespace,
                dryRun: dryRun
            })
        });

        const result = await resp.json();

        if (result.error) {
            updateYamlEditorStatus('invalid', 'Error: ' + result.error);
            return;
        }

        if (dryRun) {
            updateYamlEditorStatus('valid', 'Dry-run successful! Uncheck "Dry Run" to apply.');
            showToast('Dry-run validation passed', 'success');
        } else {
            updateYamlEditorStatus('valid', 'Applied successfully!');
            showToast('Resource applied successfully', 'success');
            // Refresh data
            refreshData();
            // Close editor after short delay
            setTimeout(closeYamlEditor, 1500);
        }
    } catch (e) {
        updateYamlEditorStatus('invalid', 'Error: ' + e.message);
    }
}

function updateYamlEditorStatus(state, message) {
    const status = document.getElementById('yaml-editor-status');
    status.className = 'yaml-editor-status ' + state;
    status.querySelector('.status-text').textContent = message;
}

function handleYamlEditorKeydown(e) {
    const isMeta = e.metaKey || e.ctrlKey;

    if (e.key === 'Escape') {
        e.preventDefault();
        closeYamlEditor();
        return;
    }

    if (isMeta && e.key === 'Enter') {
        e.preventDefault();
        applyYaml();
        return;
    }

    if (isMeta && e.shiftKey && e.key.toLowerCase() === 'f') {
        e.preventDefault();
        formatYaml();
        return;
    }
}

// Edit existing resource YAML
function editResourceYaml(resource, item) {
    // Get full YAML from API
    const ns = item.namespace || currentNamespace;
    const name = item.name;

    fetchWithAuth(`/api/k8s/${resource}/${name}?namespace=${ns}&format=yaml`)
        .then(resp => resp.text())
        .then(yaml => {
            openYamlEditor(yaml, { resource, name, namespace: ns });
        })
        .catch(e => {
            showToast('Failed to load YAML: ' + e.message, 'error');
        });
}

// ==================== Chat History (SQLite via API) ====================
let chatHistory = [];
let currentChatId = null;

// Clean up legacy localStorage chat data
localStorage.removeItem('k13d-chat-history');

async function loadChatHistory() {
    try {
        const resp = await fetchWithAuth('/api/sessions');
        if (resp.ok) {
            chatHistory = await resp.json();
            if (!chatHistory) chatHistory = [];
        }
    } catch (e) {
        console.error('Failed to load chat history:', e);
        chatHistory = [];
    }
    renderChatHistoryList();

    // Load most recent chat or create new one
    if (chatHistory.length > 0) {
        loadChat(chatHistory[0].id);
    } else {
        createNewChat();
    }
}

async function createNewChat() {
    try {
        const resp = await fetchWithAuth('/api/sessions', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        if (resp.ok) {
            const newSession = await resp.json();
            currentChatId = newSession.id;
            currentSessionId = newSession.id;
            sessionStorage.setItem('k13d_session_id', currentSessionId);

            // Refresh list and show new chat
            await loadChatHistory();
        }
    } catch (e) {
        console.error('Failed to create new chat:', e);
    }
}

async function loadChat(chatId) {
    try {
        const resp = await fetchWithAuth(`/api/sessions/${chatId}`);
        if (!resp.ok) return;
        const session = await resp.json();

        currentChatId = chatId;
        currentSessionId = chatId;
        sessionStorage.setItem('k13d_session_id', currentSessionId);

        // Clear and restore messages
        const container = document.getElementById('ai-messages');
        container.innerHTML = '';

        if (!session.messages || session.messages.length === 0) {
            // Show welcome message for new chats
            container.innerHTML = `
                    <div class="message assistant">
                        <div class="message-content">
                            Welcome to k13d! I can help you manage your Kubernetes cluster.
                            <br><br>
                            Try asking:
                            <br>- "Show me all pods"
                            <br>- "Create an nginx pod"
                            <br>- "Scale deployment to 3 replicas"
                            <br><br>
                            <strong>Tip:</strong> Click any resource row to add it as context for AI analysis!
                        </div>
                    </div>
                `;
        } else {
            // Restore messages from backend
            session.messages.forEach(msg => {
                addMessageToDOM(msg.content, msg.role === 'user', false);
            });
        }

        renderChatHistoryList();
    } catch (e) {
        console.error('Failed to load chat:', e);
    }
}

async function deleteChat(chatId, event) {
    event.stopPropagation();

    if (!confirm('Delete this chat?')) return;

    try {
        await fetchWithAuth(`/api/sessions/${chatId}`, {
            method: 'DELETE'
        });
    } catch (e) {
        console.error('Failed to delete chat:', e);
    }

    // Refresh list
    chatHistory = chatHistory.filter(c => c.id !== chatId);
    if (currentChatId === chatId) {
        if (chatHistory.length > 0) {
            loadChat(chatHistory[0].id);
        } else {
            createNewChat();
        }
    }

    renderChatHistoryList();
}

function renderChatHistoryList(filter = '') {
    const container = document.getElementById('chat-history-list');
    let filtered = chatHistory;

    if (filter) {
        const lowerFilter = filter.toLowerCase();
        filtered = chatHistory.filter(c =>
            (c.title || '').toLowerCase().includes(lowerFilter)
        );
    }

    if (filtered.length === 0) {
        container.innerHTML = `
                    <div class="chat-history-empty">
                        <div class="chat-history-empty-icon">💬</div>
                        <div>${filter ? 'No matching chats' : 'No chat history yet'}</div>
                        <div style="margin-top: 8px; font-size: 11px;">Start a new conversation!</div>
                    </div>
                `;
        return;
    }

    container.innerHTML = filtered.map(chat => {
        const date = new Date(chat.updated_at || chat.created_at);
        const dateStr = formatChatDate(date);
        const msgCount = chat.message_count || 0;
        const isActive = chat.id === currentChatId;

        return `
                    <div class="chat-history-item ${isActive ? 'active' : ''}" onclick="loadChat('${chat.id}')">
                        <div class="chat-history-title">${escapeHtml(chat.title || 'New Chat')}</div>
                        <div class="chat-history-meta">
                            <span>${dateStr}</span>
                            <span>${msgCount} message${msgCount !== 1 ? 's' : ''}</span>
                        </div>
                        <button class="chat-history-edit" onclick="startRenameChat('${chat.id}', event)" title="Rename">✏️</button>
                        <button class="chat-history-delete" onclick="deleteChat('${chat.id}', event)" title="Delete">🗑️</button>
                    </div>
                `;
    }).join('');
}

function formatChatDate(date) {
    const now = new Date();
    const diff = now - date;
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) {
        return formatTimeShort(date);
    } else if (days === 1) {
        return 'Yesterday';
    } else if (days < 7) {
        return date.toLocaleDateString([], { weekday: 'short' });
    } else {
        return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
    }
}

function filterChatHistory(query) {
    renderChatHistoryList(query);
}

// Generate a meaningful chat title from the first message
function generateChatTitle(content) {
    if (typeof stripAIContextFromMessage === 'function') {
        content = stripAIContextFromMessage(content);
    }

    // Remove markdown, code blocks, and extra whitespace
    let title = content
        .replace(/```[\s\S]*?```/g, '')  // Remove code blocks
        .replace(/`[^`]+`/g, '')          // Remove inline code
        .replace(/\*\*([^*]+)\*\*/g, '$1') // Remove bold
        .replace(/\*([^*]+)\*/g, '$1')     // Remove italic
        .replace(/#+\s*/g, '')             // Remove headers
        .replace(/\n/g, ' ')               // Replace newlines
        .replace(/\s+/g, ' ')              // Collapse whitespace
        .trim();

    // If it starts with common question words, keep them
    const questionPatterns = [
        /^(show|list|get|create|delete|scale|restart|describe|explain|why|what|how|can|help|find|check|monitor|deploy|update|patch|edit|fix|debug)/i
    ];

    // Extract the main intent (first meaningful phrase)
    const words = title.split(' ');
    let titleWords = [];
    let charCount = 0;

    for (const word of words) {
        if (charCount + word.length > 35) break;
        titleWords.push(word);
        charCount += word.length + 1;
    }

    title = titleWords.join(' ');

    // Capitalize first letter
    if (title.length > 0) {
        title = title.charAt(0).toUpperCase() + title.slice(1);
    }

    // Add ellipsis if truncated
    if (words.length > titleWords.length) {
        title += '...';
    }

    return title || 'New Chat';
}

// Rename chat functionality
function startRenameChat(chatId, event) {
    event.stopPropagation();
    const chat = chatHistory.find(c => c.id === chatId);
    if (!chat) return;

    const item = event.target.closest('.chat-history-item');
    const titleEl = item.querySelector('.chat-history-title');
    const currentTitle = chat.title;

    // Replace title with input
    titleEl.innerHTML = `<input type="text" class="chat-history-rename-input" value="${escapeHtml(currentTitle)}" />`;
    const input = titleEl.querySelector('input');
    input.focus();
    input.select();

    // Handle save on Enter or blur
    let renameCommitted = false;
    const saveRename = async () => {
        if (renameCommitted) return;
        renameCommitted = true;
        const newTitle = input.value.trim() || 'New Chat';
        try {
            const resp = await fetchWithAuth(`/api/sessions/${chatId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title: newTitle })
            });
            if (!resp.ok) {
                const errorText = await resp.text();
                throw new Error(errorText || `HTTP ${resp.status}`);
            }
            chat.title = newTitle;
            chat.updated_at = new Date().toISOString();
            renderChatHistoryList();
        } catch (e) {
            renameCommitted = false;
            console.error('Failed to rename chat:', e);
            showToast('Failed to rename chat: ' + e.message, 'error');
            renderChatHistoryList();
        }
    };

    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            saveRename();
        } else if (e.key === 'Escape') {
            renderChatHistoryList();
        }
    });

    input.addEventListener('blur', saveRename);
}

function toggleChatHistory() {
    const sidebar = document.getElementById('chat-history-sidebar');
    const panel = document.getElementById('ai-panel');

    sidebar.classList.toggle('open');
    panel.classList.toggle('history-open');
}

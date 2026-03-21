/**
 * k13d Web UI Application
 * Main application JavaScript
 * 
 * Modules:
 *   - State & Config (global state, table headers, resources)
 *   - i18n (translations)
 *   - Core (init, auth, API, refresh, sorting, pagination)
 *   - Dashboard (table rendering, resource views, detail panels)
 *   - AI Chat (messaging, streaming, tool approval, guardrails)
 *   - Settings (settings modal, LLM config, Ollama, security, admin)
 *   - Topology (graph visualization)
 *   - Terminal (WebSocket terminal)
 *   - Log Viewer (pod logs)
 *   - Metrics (cluster metrics, charts)
 *   - YAML Editor
 *   - Search & Command Bar
 *   - Chat History
 *   - Reports
 *   - Port Forwarding
 */

// State
let currentResource = 'pods';
let currentNamespace = '';
let isLoading = false;
var authToken = localStorage.getItem('k13d_token');
let currentUser = null;
let sidebarCollapsed = localStorage.getItem('k13d_sidebar_collapsed') === 'true';
let debugMode = localStorage.getItem('k13d_debug_mode') === 'true';
let aiContextItems = []; // Resources added as context for AI
let currentLanguage = 'ko'; // Default language (Korean)
let currentLLMModel = ''; // Current LLM model name
let llmConnected = false; // LLM connection status
let currentSessionId = sessionStorage.getItem('k13d_session_id') || ''; // AI conversation session ID
let appTimezone = localStorage.getItem('k13d_timezone') || 'auto'; // Timezone setting
let aiAbortController = null; // To cancel AI generation
let currentClusterContext = localStorage.getItem('k13d_current_context') || 'default';
let activeDashboardLoadId = 0;
let dataFreshnessResetTimer = null;

// Timezone formatting helpers
function getTimezoneOptions() {
    if (appTimezone === 'auto' || !appTimezone) return {};
    return { timeZone: appTimezone };
}

function formatTime(isoString) {
    const date = new Date(isoString);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', ...getTimezoneOptions() });
}

function formatDateTime(isoString) {
    const date = new Date(isoString);
    return date.toLocaleString([], getTimezoneOptions());
}

function formatTimeShort(date) {
    if (typeof date === 'string') date = new Date(date);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', ...getTimezoneOptions() });
}

// Auto-refresh settings (default to enabled with 30s interval)
let autoRefreshEnabled = localStorage.getItem('k13d_auto_refresh') !== 'false'; // default true
let autoRefreshInterval = parseInt(localStorage.getItem('k13d_refresh_interval')) || 30; // seconds
let autoRefreshTimer = null;

// SSE streaming settings
let useStreaming = localStorage.getItem('k13d_use_streaming') !== 'false'; // default true
let currentEventSource = null;

// Reasoning effort setting (for Solar Pro2)
let reasoningEffort = localStorage.getItem('k13d_reasoning_effort') || 'minimal'; // default minimal
const resourceCachePolicy = Object.freeze({
    ttlMs: 15 * 1000,
    maxStaleMs: 3 * 60 * 1000,
    persist: 'session',
});
const namespacesCachePolicy = Object.freeze({
    ttlMs: 60 * 1000,
    maxStaleMs: 10 * 60 * 1000,
    persist: 'session',
});
const overviewCachePolicy = Object.freeze({
    ttlMs: 20 * 1000,
    maxStaleMs: 2 * 60 * 1000,
    persist: 'session',
});
const backgroundRefreshConcurrency = 4;

// Table headers for all resource types
const tableHeaders = {
    pods: ['NAME', 'NAMESPACE', 'READY', 'STATUS', 'RESTARTS', 'AGE', 'IP'],
    deployments: ['NAME', 'NAMESPACE', 'READY', 'UP-TO-DATE', 'AVAILABLE', 'AGE'],
    daemonsets: ['NAME', 'NAMESPACE', 'DESIRED', 'CURRENT', 'READY', 'AGE'],
    statefulsets: ['NAME', 'NAMESPACE', 'READY', 'AGE'],
    replicasets: ['NAME', 'NAMESPACE', 'DESIRED', 'CURRENT', 'READY', 'AGE'],
    jobs: ['NAME', 'NAMESPACE', 'COMPLETIONS', 'DURATION', 'AGE'],
    cronjobs: ['NAME', 'NAMESPACE', 'SCHEDULE', 'SUSPEND', 'ACTIVE', 'LAST SCHEDULE'],
    services: ['NAME', 'NAMESPACE', 'TYPE', 'CLUSTER-IP', 'PORTS', 'AGE'],
    ingresses: ['NAME', 'NAMESPACE', 'CLASS', 'HOSTS', 'ADDRESS', 'AGE'],
    networkpolicies: ['NAME', 'NAMESPACE', 'POD-SELECTOR', 'AGE'],
    configmaps: ['NAME', 'NAMESPACE', 'DATA', 'AGE'],
    secrets: ['NAME', 'NAMESPACE', 'TYPE', 'DATA', 'AGE'],
    serviceaccounts: ['NAME', 'NAMESPACE', 'SECRETS', 'AGE'],
    persistentvolumes: ['NAME', 'CAPACITY', 'ACCESS MODES', 'RECLAIM POLICY', 'STATUS', 'CLAIM'],
    persistentvolumeclaims: ['NAME', 'NAMESPACE', 'STATUS', 'VOLUME', 'CAPACITY', 'ACCESS MODES'],
    nodes: ['NAME', 'STATUS', 'ROLES', 'VERSION', 'AGE'],
    namespaces: ['NAME', 'STATUS', 'AGE'],
    events: ['NAME', 'TYPE', 'REASON', 'MESSAGE', 'COUNT', 'LAST SEEN'],
    roles: ['NAME', 'NAMESPACE', 'AGE'],
    rolebindings: ['NAME', 'NAMESPACE', 'ROLE', 'AGE'],
    clusterroles: ['NAME', 'AGE'],
    clusterrolebindings: ['NAME', 'ROLE', 'AGE']
};

// All supported resource types
const allResources = [
    'pods', 'deployments', 'daemonsets', 'statefulsets', 'replicasets', 'jobs', 'cronjobs',
    'services', 'ingresses', 'networkpolicies',
    'configmaps', 'secrets', 'serviceaccounts',
    'persistentvolumes', 'persistentvolumeclaims',
    'nodes', 'namespaces', 'events',
    'roles', 'rolebindings', 'clusterroles', 'clusterrolebindings'
];

// Cluster-scoped resources (no namespace)
const clusterScopedResources = ['nodes', 'namespaces', 'persistentvolumes', 'clusterroles', 'clusterrolebindings'];

// Custom Resource state
let loadedCRDs = []; // List of CRDs with their info
let currentCRD = null; // Currently selected CRD (for viewing instances)

// Sorting and Pagination State
let sortColumn = null;
let sortDirection = 'asc'; // 'asc' or 'desc'
let currentPage = 1;
let pageSize = 50;
let allItems = []; // All items before pagination
let filteredItems = []; // Items after filtering

// Column filter state
let columnFiltersVisible = false;
let columnFilters = {}; // { 'NAME': 'nginx', 'STATUS': 'Running' }

// Field mapping for sorting (header name -> item property)
const fieldMapping = {
    'NAME': 'name',
    'NAMESPACE': 'namespace',
    'READY': 'ready',
    'STATUS': 'status',
    'RESTARTS': 'restarts',
    'AGE': 'age',
    'IP': 'ip',
    'UP-TO-DATE': 'upToDate',
    'AVAILABLE': 'available',
    'DESIRED': 'desired',
    'CURRENT': 'current',
    'COMPLETIONS': 'completions',
    'DURATION': 'duration',
    'SCHEDULE': 'schedule',
    'SUSPEND': 'suspend',
    'ACTIVE': 'active',
    'LAST SCHEDULE': 'lastSchedule',
    'TYPE': 'type',
    'CLUSTER-IP': 'clusterIP',
    'PORTS': 'ports',
    'CLASS': 'class',
    'HOSTS': 'hosts',
    'ADDRESS': 'address',
    'POD-SELECTOR': 'podSelector',
    'DATA': 'data',
    'SECRETS': 'secrets',
    'CAPACITY': 'capacity',
    'ACCESS MODES': 'accessModes',
    'RECLAIM POLICY': 'reclaimPolicy',
    'CLAIM': 'claim',
    'VOLUME': 'volume',
    'ROLES': 'roles',
    'VERSION': 'version',
    'REASON': 'reason',
    'MESSAGE': 'message',
    'COUNT': 'count',
    'LAST SEEN': 'lastSeen',
    'ROLE': 'role'
};

// Sort items by column
function sortItems(items, column, direction) {
    const field = fieldMapping[column] || column.toLowerCase().replace(/[- ]/g, '');
    return [...items].sort((a, b) => {
        let valA = a[field];
        let valB = b[field];

        // Handle age sorting (convert to comparable values)
        if (column === 'AGE' || column === 'LAST SEEN' || column === 'DURATION') {
            valA = parseAgeToSeconds(valA);
            valB = parseAgeToSeconds(valB);
        }
        // Handle numeric fields
        else if (column === 'RESTARTS' || column === 'COUNT' || column === 'DESIRED' ||
            column === 'CURRENT' || column === 'AVAILABLE' || column === 'ACTIVE' ||
            column === 'DATA' || column === 'SECRETS') {
            valA = parseInt(valA) || 0;
            valB = parseInt(valB) || 0;
        }
        // Handle ready format (e.g., "1/1")
        else if (column === 'READY' || column === 'COMPLETIONS') {
            valA = parseReadyValue(valA);
            valB = parseReadyValue(valB);
        }
        // Handle strings (case-insensitive)
        else {
            valA = (valA || '').toString().toLowerCase();
            valB = (valB || '').toString().toLowerCase();
        }

        if (valA < valB) return direction === 'asc' ? -1 : 1;
        if (valA > valB) return direction === 'asc' ? 1 : -1;
        return 0;
    });
}

// Parse age string to seconds for sorting
function parseAgeToSeconds(age) {
    if (!age || age === '-') return 0;
    const str = age.toString();
    const match = str.match(/(\d+)([smhd])/);
    if (!match) return 0;
    const value = parseInt(match[1]);
    const unit = match[2];
    switch (unit) {
        case 's': return value;
        case 'm': return value * 60;
        case 'h': return value * 3600;
        case 'd': return value * 86400;
        default: return value;
    }
}

// Parse ready value (e.g., "1/1" -> 1)
function parseReadyValue(ready) {
    if (!ready || ready === '-') return 0;
    const parts = ready.toString().split('/');
    return parseInt(parts[0]) || 0;
}

// Handle column header click for sorting
function onColumnSort(column, headerElement) {
    // Toggle direction if same column, otherwise default to asc
    if (sortColumn === column) {
        sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
    } else {
        sortColumn = column;
        sortDirection = 'asc';
    }

    // Update header styling
    document.querySelectorAll('#table-header th').forEach(th => {
        th.classList.remove('sort-asc', 'sort-desc');
    });
    headerElement.classList.add(sortDirection === 'asc' ? 'sort-asc' : 'sort-desc');

    // Re-render with sorted data
    currentPage = 1;
    applyFilterAndSort();
}

// Apply filter and sort to items
function applyFilterAndSort() {
    const filterInput = document.getElementById('filter-input');
    const filterText = filterInput ? filterInput.value.toLowerCase() : '';

    // Filter items by global filter
    filteredItems = allItems.filter(item => {
        if (!filterText) return true;
        // Search in all values of the item
        return Object.values(item).some(val =>
            val && val.toString().toLowerCase().includes(filterText)
        );
    });

    // Apply column-specific filters
    const activeColumnFilters = Object.entries(columnFilters).filter(([_, v]) => v && v.trim());
    if (activeColumnFilters.length > 0) {
        filteredItems = filteredItems.filter(item => {
            return activeColumnFilters.every(([column, filterVal]) => {
                const field = fieldMapping[column] || column.toLowerCase().replace(/[- ]/g, '');
                const itemValue = item[field];
                if (itemValue === undefined || itemValue === null) return false;
                return itemValue.toString().toLowerCase().includes(filterVal.toLowerCase());
            });
        });
    }

    // Sort items
    if (sortColumn) {
        filteredItems = sortItems(filteredItems, sortColumn, sortDirection);
    }

    // Render current page
    renderCurrentPage();

    // Update active column filters display (chips)
    updateActiveColumnFiltersDisplay();
}

// Render table headers and column filter row
function renderTableHeaders(resource) {
    const headers = tableHeaders[resource] || ['NAME', 'NAMESPACE', 'STATUS', 'AGE'];
    const headerRow = `<tr>${headers.map(h => {
        const sortClass = sortColumn === h ? (sortDirection === 'asc' ? 'sort-asc' : 'sort-desc') : '';
        return `<th class="${sortClass}" onclick="onColumnSort('${h}', this)">${h}<span class="sort-icon"></span></th>`;
    }).join('')}</tr>`;

    const filterRow = `<tr class="column-filter-row ${columnFiltersVisible ? 'active' : ''}" id="column-filter-row">
                ${headers.map(h => {
        const filterValue = columnFilters[h] || '';
        const placeholder = h === 'ACTIONS' ? '' : `Filter ${h.toLowerCase()}...`;
        const disabled = h === 'ACTIONS' ? 'disabled style="visibility:hidden"' : '';
        return `<th><input type="text" class="column-filter-input" placeholder="${placeholder}"
                        value="${filterValue}"
                        data-column="${h}"
                        ${disabled}
                        onkeyup="onColumnFilterChange(event, '${h}')"
                        onclick="event.stopPropagation()"></th>`;
    }).join('')}
            </tr>`;

    const headerEl = document.getElementById('table-header');
    if (headerEl) {
        headerEl.innerHTML = headerRow + filterRow;
    }
}

// Toggle column filters visibility
function toggleColumnFilters() {
    columnFiltersVisible = !columnFiltersVisible;
    const filterRow = document.getElementById('column-filter-row');
    const toggleBtn = document.getElementById('column-filter-toggle');

    if (filterRow) {
        filterRow.classList.toggle('active', columnFiltersVisible);
    }
    if (toggleBtn) {
        toggleBtn.classList.toggle('active', columnFiltersVisible);
    }

    // Focus first filter input when showing
    if (columnFiltersVisible && filterRow) {
        const firstInput = filterRow.querySelector('.column-filter-input');
        if (firstInput) {
            setTimeout(() => firstInput.focus(), 50);
        }
    }
}

// Handle column filter input change
function onColumnFilterChange(event, column) {
    const value = event.target.value;
    columnFilters[column] = value;

    // Debounce the filter application
    clearTimeout(window.columnFilterTimeout);
    window.columnFilterTimeout = setTimeout(() => {
        currentPage = 1;
        applyFilterAndSort();
    }, 200);
}

// Update the active column filters chips display
function updateActiveColumnFiltersDisplay() {
    const container = document.getElementById('active-column-filters');
    if (!container) return;

    const activeFilters = Object.entries(columnFilters).filter(([_, v]) => v && v.trim());

    if (activeFilters.length === 0) {
        container.innerHTML = '';
        return;
    }

    container.innerHTML = activeFilters.map(([col, val]) =>
        `<span class="column-filter-chip">
                    <span class="col-name">${col}:</span>
                    <span>${val}</span>
                    <span class="remove-col-filter" onclick="clearColumnFilter('${col}')">&times;</span>
                </span>`
    ).join('');
}

// Clear a specific column filter
function clearColumnFilter(column) {
    delete columnFilters[column];

    // Update the input field if visible
    const input = document.querySelector(`.column-filter-input[data-column="${column}"]`);
    if (input) {
        input.value = '';
    }

    currentPage = 1;
    applyFilterAndSort();
}

// Clear all column filters
function clearAllColumnFilters() {
    columnFilters = {};

    // Clear all input fields
    document.querySelectorAll('.column-filter-input').forEach(input => {
        input.value = '';
    });

    currentPage = 1;
    applyFilterAndSort();
}

// Render current page of items (Virtual Scrolling)
function renderCurrentPage() {
    const totalItems = filteredItems.length;

    // Hide pagination UI since we are virtual scrolling
    const paginationContainer = document.getElementById('pagination-container');
    if (paginationContainer) paginationContainer.style.display = 'none';

    if (totalItems === 0) {
        const headers = tableHeaders[currentResource] || [];
        document.getElementById('table-body').innerHTML =
            `<tr><td colspan="${headers.length}" style="text-align:center;padding:40px;">No ${currentResource} found</td></tr>`;
        return;
    }

    if (window.virtualScroller) {
        window.virtualScroller.setItems(filteredItems, (item, index) => {
            return generateRowHTML(currentResource, item, index);
        });
    } else {
        // Fallback
        document.getElementById('table-body').innerHTML = filteredItems.slice(0, 100).map((item, index) => generateRowHTML(currentResource, item, index)).join('');
        addRowClickHandlers();
    }
}

// Update pagination UI
function updatePaginationUI(totalItems, totalPages) {
    const startItem = totalItems === 0 ? 0 : (currentPage - 1) * (pageSize === -1 ? totalItems : pageSize) + 1;
    const endItem = pageSize === -1 ? totalItems : Math.min(currentPage * pageSize, totalItems);

    document.getElementById('pagination-info').textContent =
        `Showing ${startItem}-${endItem} of ${totalItems} items`;
    document.getElementById('page-indicator').textContent = `${currentPage} / ${totalPages || 1}`;

    document.getElementById('prev-page-btn').disabled = currentPage <= 1;
    document.getElementById('next-page-btn').disabled = currentPage >= totalPages;
}

// Pagination controls
function goToNextPage() {
    currentPage++;
    renderCurrentPage();
}

function goToPrevPage() {
    currentPage--;
    renderCurrentPage();
}

function onPageSizeChange() {
    pageSize = parseInt(document.getElementById('page-size-select').value);
    currentPage = 1;
    renderCurrentPage();
}

// Theme toggle (dark/light)
function initTheme() {
    const saved = localStorage.getItem('k13d_theme') || 'light';
    document.documentElement.setAttribute('data-theme', saved);
    updateThemeIcon();
}

function toggleTheme() {
    const current = document.documentElement.getAttribute('data-theme');
    if (!current || current === 'light') {
        // Switch to Tokyo Night
        applyTheme('tokyo-night');
    } else {
        // Switch to Light
        applyTheme('light');
    }
}

function updateThemeIcon() {
    const btn = document.getElementById('theme-toggle');
    if (!btn) return;
    const theme = document.documentElement.getAttribute('data-theme');
    const isLight = !theme || theme === 'light';
    btn.textContent = isLight ? '☀️' : '🌙';
    btn.title = isLight ? 'Switch to dark theme' : 'Switch to light theme';
}

// Apply theme immediately (before DOM ready)
initTheme();

// Initialize
async function init() {
    if (authToken) {
        try {
            const health = await fetch('/api/health').then(r => r.json());
            if (health.auth_enabled) {
                const user = await fetchWithAuth('/api/auth/me').then(r => r.json());
                currentUser = user;
                showApp();
            } else {
                showApp();
            }
        } catch (e) {
            showLogin();
        }
    } else {
        // Check if auth is enabled
        const health = await fetch('/api/health').then(r => r.json());
        if (!health.auth_enabled) {
            authToken = 'anonymous';
            showApp();
        } else {
            showLogin();
        }
    }
}

async function showLogin() {
    document.getElementById('login-page').style.display = 'flex';
    document.getElementById('app').classList.remove('active');

    // Use server-injected auth mode if available (instant, no fetch needed)
    if (window.__AUTH_MODE__) {
        updateLoginPageForAuthMode({ auth_mode: window.__AUTH_MODE__ });
        return;
    }

    // Fallback: fetch auth status from API
    try {
        const status = await fetch('/api/auth/status').then(r => r.json());
        updateLoginPageForAuthMode(status);
    } catch (e) {
        console.error('Failed to fetch auth status:', e);
        // Default to showing token form
        document.getElementById('token-login-form').classList.add('active');
        document.getElementById('password-login-form').classList.remove('active');
    }
}

// Update login page UI based on auth mode (token vs local)
function updateLoginPageForAuthMode(status) {
    const authModeEl = document.getElementById('auth-mode-indicator');
    const tokenForm = document.getElementById('token-login-form');
    const passwordForm = document.getElementById('password-login-form');

    const authMode = status.auth_mode || status.mode || 'token';

    if (authMode === 'token') {
        // Token authentication mode - show token form only
        authModeEl.className = 'auth-mode-indicator token-mode';
        authModeEl.innerHTML = '🔐 Kubernetes Token 인증 모드';
        tokenForm.classList.add('active');
        passwordForm.classList.remove('active');

        // Focus on token input
        setTimeout(() => {
            document.getElementById('login-token').focus();
        }, 100);
    } else if (authMode === 'local') {
        // Local authentication mode - show password form only
        authModeEl.className = 'auth-mode-indicator local-mode';
        authModeEl.innerHTML = '👤 로컬 계정 인증 모드';
        tokenForm.classList.remove('active');
        passwordForm.classList.add('active');

        // Focus on username input
        setTimeout(() => {
            document.getElementById('login-username').focus();
        }, 100);
    } else {
        // Default or mixed mode - show token form
        authModeEl.style.display = 'none';
        tokenForm.classList.add('active');
        passwordForm.classList.remove('active');
    }
}

// Handle Enter key in token textarea
function handleTokenKeydown(event) {
    if (event.key === 'Enter' && !event.shiftKey) {
        event.preventDefault();
        loginWithToken();
    }
}

// Handle Enter key in password form
function handlePasswordKeydown(event) {
    if (event.key === 'Enter') {
        event.preventDefault();
        login();
    }
}

// Toggle token help dropdown
function toggleTokenHelp() {
    const box = document.getElementById('token-help-box');
    if (box) {
        box.classList.toggle('expanded');
    }
}

// Login with kubeconfig credentials (local mode only)
async function loginWithKubeconfig() {
    const errorEl = document.getElementById('login-error');
    errorEl.textContent = '';

    try {
        const resp = await fetch('/api/auth/kubeconfig', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });

        const data = await resp.json();
        if (resp.ok) {
            authToken = data.token;
            localStorage.setItem('k13d_token', authToken);
            currentUser = { username: data.username, role: data.role };
            showApp();
        } else {
            errorEl.textContent = data.error || 'Kubeconfig login failed';
        }
    } catch (e) {
        errorEl.textContent = 'Login failed: ' + e.message;
    }
}

function showApp() {
    document.getElementById('login-page').style.display = 'none';
    document.getElementById('app').classList.add('active');
    if (currentUser) {
        document.getElementById('user-badge').textContent = currentUser.username;
    } else if (authToken === 'anonymous') {
        document.getElementById('user-badge').textContent = 'anonymous';
        // Hide logout button when auth is disabled
        document.getElementById('logout-btn').style.display = 'none';
    }
    // Restore sidebar state
    if (sidebarCollapsed) {
        document.getElementById('sidebar').classList.add('collapsed');
        document.getElementById('hamburger-btn').classList.add('active');
        var toggleIcon = document.getElementById('sidebar-toggle-icon');
        if (toggleIcon) toggleIcon.textContent = '»';
    }
    // Restore debug mode
    if (debugMode) {
        document.getElementById('debug-panel').classList.add('active');
        document.getElementById('debug-toggle').style.background = 'var(--accent-purple)';
    }
    if (typeof initWorkspaceFeatures === 'function') {
        initWorkspaceFeatures().catch((e) => {
            console.error('Failed to initialize workspace features:', e);
        });
    }
    loadClusterContexts();
    loadNamespaces();
    switchResource('pods');
    initMobileNavSections();
    setupResizeHandle();
    setupHealthCheck();
    // Initialize auto-refresh
    updateAutoRefreshUI();
    updateLastRefreshTime();
    if (autoRefreshEnabled) {
        startAutoRefresh();
    }
    // Initialize AI status (model name and connection status)
    updateAIStatus();
    // Load user permissions for feature gating
    loadUserPermissions();
}

// Login tab switching
function switchLoginTab(tab) {
    document.querySelectorAll('.login-tab').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('.login-form').forEach(f => f.classList.remove('active'));

    if (tab === 'token') {
        document.querySelector('.login-tab:first-child').classList.add('active');
        document.getElementById('token-login-form').classList.add('active');
    } else {
        document.querySelector('.login-tab:last-child').classList.add('active');
        document.getElementById('password-login-form').classList.add('active');
    }
}

// Token-based login (K8s RBAC)
async function loginWithToken() {
    const token = document.getElementById('login-token').value.trim();
    if (!token) {
        document.getElementById('login-error').textContent = 'Please enter a token';
        return;
    }

    try {
        const resp = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token })
        });

        const data = await resp.json();
        if (resp.ok) {
            authToken = data.token;
            localStorage.setItem('k13d_token', authToken);
            currentUser = { username: data.username, role: data.role };
            showApp();
        } else {
            document.getElementById('login-error').textContent = data.error || 'Invalid token';
        }
    } catch (e) {
        document.getElementById('login-error').textContent = 'Login failed: ' + e.message;
    }
}

// Username/password login
async function login() {
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;

    try {
        const resp = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        if (resp.ok) {
            const data = await resp.json();
            authToken = data.token;
            localStorage.setItem('k13d_token', authToken);
            currentUser = { username: data.username, role: data.role };
            showApp();
        } else {
            document.getElementById('login-error').textContent = 'Invalid credentials';
        }
    } catch (e) {
        document.getElementById('login-error').textContent = 'Login failed';
    }
}

async function logout() {
    try {
        // Send logout request with credentials (cookies) and auth header
        await fetch('/api/auth/logout', {
            method: 'POST',
            credentials: 'include',
            headers: authToken ? { 'Authorization': `Bearer ${authToken}` } : {}
        });
    } catch (e) {
        console.error('Logout request failed:', e);
    }
    // Clear local storage and state regardless of server response
    localStorage.removeItem('k13d_token');
    localStorage.removeItem('k13d_auto_refresh');
    localStorage.removeItem('k13d_refresh_interval');
    authToken = null;
    currentUser = null;
    // Stop auto-refresh timer
    if (autoRefreshTimer) {
        clearInterval(autoRefreshTimer);
        autoRefreshTimer = null;
    }
    location.reload();
}

function setCurrentClusterContext(name) {
    currentClusterContext = name || 'default';
    localStorage.setItem('k13d_current_context', currentClusterContext);
}

function getCacheScope() {
    return currentClusterContext || document.getElementById('cluster-name')?.textContent?.trim() || 'default';
}

function buildScopedCacheKey(section, key) {
    return `ctx:${getCacheScope()}:${section}:${key}`;
}

function isDashboardLoadActive(loadId) {
    return loadId === activeDashboardLoadId;
}

function formatCacheAge(ageMs) {
    if (!Number.isFinite(ageMs) || ageMs < 1000) return 'just now';
    if (ageMs < 60 * 1000) return `${Math.max(1, Math.round(ageMs / 1000))}s`;
    if (ageMs < 60 * 60 * 1000) return `${Math.round(ageMs / (60 * 1000))}m`;
    return `${Math.round(ageMs / (60 * 60 * 1000))}h`;
}

function clearDataFreshnessState() {
    const indicator = document.getElementById('data-freshness-indicator');
    if (!indicator) return;
    if (dataFreshnessResetTimer) {
        clearTimeout(dataFreshnessResetTimer);
        dataFreshnessResetTimer = null;
    }
    indicator.hidden = true;
    indicator.textContent = '';
    indicator.className = 'data-freshness-indicator';
}

function setDataFreshnessState(state, label, autoHideMs = 0) {
    const indicator = document.getElementById('data-freshness-indicator');
    if (!indicator) return;

    if (dataFreshnessResetTimer) {
        clearTimeout(dataFreshnessResetTimer);
        dataFreshnessResetTimer = null;
    }

    if (!state || !label) {
        clearDataFreshnessState();
        return;
    }

    indicator.hidden = false;
    indicator.textContent = label;
    indicator.className = `data-freshness-indicator ${state}`;

    if (autoHideMs > 0) {
        dataFreshnessResetTimer = setTimeout(() => {
            clearDataFreshnessState();
        }, autoHideMs);
    }
}

function buildResourceRequest(resource) {
    const isClusterScoped = clusterScopedResources.includes(resource);
    const namespace = isClusterScoped ? '' : currentNamespace;
    return {
        url: namespace ? `/api/k8s/${resource}?namespace=${namespace}` : `/api/k8s/${resource}`,
        cacheKey: buildScopedCacheKey('resource', `${resource}:${isClusterScoped ? 'cluster' : (namespace || 'all')}`),
    };
}

function renderTableLoadingState(resource) {
    renderTableHeaders(resource);
    const columns = tableHeaders[resource] || ['NAME'];
    const body = document.getElementById('table-body');
    if (!body) return;

    body.innerHTML = `<tr>
            <td colspan="${columns.length}" style="text-align:center;padding:36px;color:var(--text-secondary);">
                <div class="loading-dots" style="justify-content:center;margin-bottom:10px;">
                    <span></span><span></span><span></span>
                </div>
                Loading ${escapeHtml(resource)}...
            </td>
        </tr>`;
    updatePaginationUI(0, 0);
}

async function runWithConcurrency(items, limit, worker) {
    let nextIndex = 0;
    const runners = Array.from({ length: Math.min(limit, items.length) }, async () => {
        while (nextIndex < items.length) {
            const item = items[nextIndex++];
            await worker(item);
        }
    });

    await Promise.allSettled(runners);
}

function renderNamespaceOptions(items) {
    const select = document.getElementById('namespace-select');
    if (!select) return;

    const selectedNamespace = currentNamespace || select.value || '';
    let selectionExists = selectedNamespace === '';

    select.innerHTML = '<option value="">All Namespaces</option>';
    (items || []).forEach((ns) => {
        const option = document.createElement('option');
        option.value = ns.name;
        option.textContent = ns.name;
        if (ns.name === selectedNamespace) {
            selectionExists = true;
        }
        select.appendChild(option);
    });

    if (selectionExists) {
        select.value = selectedNamespace;
    } else {
        currentNamespace = '';
        select.value = '';
    }
}

async function loadNamespaces(options = {}) {
    const cacheKey = buildScopedCacheKey('namespaces', 'all');
    const policy = {
        ...namespacesCachePolicy,
        cacheKey,
        forceNetwork: !!options.forceNetwork,
    };

    const preview = K13D.SWR?.peekJSON(cacheKey, policy);
    if (preview?.data?.items) {
        renderNamespaceOptions(preview.data.items);
    }

    try {
        const result = await K13D.SWR.fetchJSON('/api/k8s/namespaces', {}, policy);
        renderNamespaceOptions(result.data?.items || []);

        if (result.revalidatePromise) {
            result.revalidatePromise.then((revalidated) => {
                renderNamespaceOptions(revalidated.data?.items || []);
            }).catch((e) => {
                console.error('Failed to refresh namespaces:', e);
            });
        }
    } catch (e) {
        console.error('Failed to load namespaces:', e);
    }
}

function applyResourcePayload(resource, data, meta, loadId) {
    if (!isDashboardLoadActive(loadId)) {
        return;
    }

    if (!data || data.error) {
        console.error(`API returned error for ${resource}:`, data?.error);
        clearResourceData(resource);
        if (resource === currentResource) {
            setDataFreshnessState('offline', meta.cached ? 'Offline cache' : 'Failed to refresh', 4500);
        }
        return;
    }

    const countEl = document.getElementById(`${resource}-count`);
    if (countEl) {
        countEl.textContent = data.items ? data.items.length : 0;
    }

    if (resource !== currentResource) {
        return;
    }

    renderTable(resource, data.items || []);

    if (meta.error && meta.cached) {
        setDataFreshnessState('offline', 'Offline cache', 5000);
        return;
    }

    if (meta.cached && meta.revalidating) {
        const ageText = meta.ageMs ? formatCacheAge(meta.ageMs) : 'just now';
        setDataFreshnessState('refreshing', `Cached ${ageText} ago · refreshing`);
        return;
    }

    if (meta.cached && meta.stale) {
        const ageText = meta.ageMs ? formatCacheAge(meta.ageMs) : 'just now';
        setDataFreshnessState('cached', `Cached ${ageText} ago`);
        return;
    }

    if (!meta.cached) {
        setDataFreshnessState('live', 'Live', 2200);
        updateLastRefreshTime();
        return;
    }

    clearDataFreshnessState();
}

async function loadResourceSnapshot(resource, loadId, options = {}) {
    const request = buildResourceRequest(resource);
    const renderTarget = resource === currentResource;
    const policy = {
        ...resourceCachePolicy,
        cacheKey: request.cacheKey,
        forceNetwork: !!options.forceNetwork,
    };

    const preview = K13D.SWR?.peekJSON(request.cacheKey, policy);
    if (renderTarget && !preview) {
        renderTableLoadingState(resource);
        setDataFreshnessState('refreshing', 'Loading live...');
    }

    try {
        const result = await K13D.SWR.fetchJSON(request.url, {}, policy);
        applyResourcePayload(resource, result.data, {
            cached: !!result.cached,
            stale: !!result.stale,
            ageMs: result.ageMs,
            error: result.error,
            revalidating: !!result.revalidatePromise,
        }, loadId);

        if (result.revalidatePromise) {
            result.revalidatePromise.then((revalidated) => {
                applyResourcePayload(resource, revalidated.data, {
                    cached: !!revalidated.cached,
                    stale: !!revalidated.stale,
                    ageMs: revalidated.ageMs,
                    error: revalidated.error,
                    revalidating: false,
                }, loadId);
            }).catch((e) => {
                console.error(`Failed to refresh ${resource}:`, e);
                if (renderTarget && isDashboardLoadActive(loadId) && preview) {
                    setDataFreshnessState('offline', 'Offline cache', 5000);
                }
            });
        }
    } catch (e) {
        if (!isDashboardLoadActive(loadId)) {
            return;
        }
        console.error(`Failed to load ${resource}:`, e);
        clearResourceData(resource);
        if (renderTarget) {
            setDataFreshnessState('offline', 'Failed to refresh', 4500);
        }
    }
}

async function loadData(options = {}) {
    const loadId = ++activeDashboardLoadId;
    const prioritizedResource = allResources.includes(currentResource) ? currentResource : null;
    const remainingResources = prioritizedResource
        ? allResources.filter((resource) => resource !== prioritizedResource)
        : [...allResources];

    if (prioritizedResource) {
        await loadResourceSnapshot(prioritizedResource, loadId, options);
    } else {
        clearDataFreshnessState();
    }

    void runWithConcurrency(remainingResources, backgroundRefreshConcurrency, (resource) =>
        loadResourceSnapshot(resource, loadId, options)
    );

    void loadCRDs(options);
}

function clearResourceData(resource) {
    const countEl = document.getElementById(`${resource}-count`);
    if (countEl) countEl.textContent = '-';
    if (resource === currentResource) {
        renderTable(resource, []);
    }
}

// Load Custom Resource Definitions
async function loadCRDs(options = {}) {
    const cacheKey = buildScopedCacheKey('crds', 'all');
    const policy = {
        ...namespacesCachePolicy,
        cacheKey,
        forceNetwork: !!options.forceNetwork,
    };

    const renderCRDNavigation = (data) => {
        if (!data) return;

        if (data.error) {
            console.error('CRD API error:', data.error);
            document.getElementById('crd-count').textContent = '-';
            const errorMsg = data.error.includes('forbidden') || data.error.includes('Forbidden')
                ? 'No permission'
                : 'Error loading';
            document.getElementById('crd-nav-items').innerHTML = `<div style="font-size: 11px; color: var(--accent-yellow); padding: 4px 8px;" title="${escapeHtml(data.error)}">${errorMsg}</div>`;
            return;
        }

        if (data.items && data.items.length > 0) {
            loadedCRDs = data.items;
            document.getElementById('crd-count').textContent = data.items.length;

            const grouped = {};
            data.items.forEach(crd => {
                const group = crd.group || 'core';
                if (!grouped[group]) grouped[group] = [];
                grouped[group].push(crd);
            });

            const container = document.getElementById('crd-nav-items');
            if (!container) return;
            const sortedGroups = Object.keys(grouped).sort();
            let html = '';
            let count = 0;

            for (const group of sortedGroups) {
                for (const crd of grouped[group]) {
                    if (count >= 15) break;
                    const shortGroup = group.split('.')[0] || 'core';
                    html += `<div class="nav-item" data-crd="${crd.name}" onclick="switchToCRD('${crd.name}')" title="${crd.name}">
                                <span style="font-size: 11px;">${crd.kind}</span>
                                <span class="count" style="font-size: 9px; opacity: 0.7;">${shortGroup}</span>
                            </div>`;
                    count++;
                }
                if (count >= 15) break;
            }

            if (data.items.length > 15) {
                html += `<div class="nav-item" onclick="showAllCRDs()" style="font-style: italic; opacity: 0.8;">
                            <span>View all ${data.items.length} CRDs...</span>
                        </div>`;
            }

            container.innerHTML = html;
        } else {
            document.getElementById('crd-count').textContent = '0';
            document.getElementById('crd-nav-items').innerHTML = '<div style="font-size: 11px; color: var(--text-secondary); padding: 4px 8px;">No CRDs found</div>';
        }
    };

    const preview = K13D.SWR?.peekJSON(cacheKey, policy);
    if (preview?.data) {
        renderCRDNavigation(preview.data);
    }

    try {
        const result = await K13D.SWR.fetchJSON('/api/crd/', {}, policy);
        renderCRDNavigation(result.data);

        if (result.revalidatePromise) {
            result.revalidatePromise.then((revalidated) => {
                renderCRDNavigation(revalidated.data);
            }).catch((e) => {
                console.error('Failed to refresh CRDs:', e);
            });
        }
    } catch (e) {
        console.error('Failed to load CRDs:', e);
        const countEl = document.getElementById('crd-count');
        if (countEl) countEl.textContent = 'err';
        const navEl = document.getElementById('crd-nav-items');
        if (navEl) navEl.innerHTML = `<div style="font-size: 11px; color: var(--accent-red); padding: 4px 8px;" title="${escapeHtml(e.message)}">Failed to load</div>`;
    }
}

// Switch to viewing a Custom Resource's instances
async function switchToCRD(crdName) {
    closeMobileSidebar();
    currentCRD = loadedCRDs.find(c => c.name === crdName);
    if (!currentCRD) return;

    currentResource = `crd:${crdName}`;

    // Update active nav item
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
    });
    document.querySelector(`[data-crd="${crdName}"]`)?.classList.add('active');

    // Update panel title
    document.getElementById('panel-title').textContent = `${currentCRD.kind} (${currentCRD.group})`;
    document.getElementById('resource-summary').innerHTML = '';

    // Clear filters
    columnFilters = {};
    sortColumn = null;
    sortDirection = 'asc';
    updateActiveColumnFiltersDisplay();

    // Load instances
    await loadCRDInstances(currentCRD);
}

// Load instances of a Custom Resource
async function loadCRDInstances(crdInfo) {
    try {
        // For namespaced resources, use current namespace (empty = all namespaces)
        const ns = crdInfo.namespaced ? currentNamespace : '';
        const url = ns ? `/api/crd/${crdInfo.name}/instances?namespace=${encodeURIComponent(ns)}` : `/api/crd/${crdInfo.name}/instances`;

        console.log(`Loading CR instances: ${url} (namespaced: ${crdInfo.namespaced}, ns: "${ns}")`);

        const resp = await fetchWithAuth(url);

        if (!resp.ok) {
            const errorText = await resp.text();
            throw new Error(`HTTP ${resp.status}: ${errorText}`);
        }

        const data = await resp.json();
        console.log(`CR instances response:`, data);

        // Check for API error in response
        if (data.error) {
            throw new Error(data.error);
        }

        // Build dynamic headers from printerColumns
        const printerCols = data.printerColumns || crdInfo.printerColumns || [];
        const extraColNames = printerCols
            .filter(c => {
                const key = c.name.toLowerCase();
                return key !== 'age' && key !== 'name' && key !== 'namespace' && (c.priority || 0) === 0;
            })
            .map(c => c.name.toUpperCase());

        let headers;
        if (crdInfo.namespaced) {
            headers = ['NAME', 'NAMESPACE', ...extraColNames, 'STATUS', 'AGE'];
        } else {
            headers = ['NAME', ...extraColNames, 'STATUS', 'AGE'];
        }

        // Store printer column info for renderTableBody
        crdInfo._extraColumns = extraColNames;

        // Store headers dynamically
        tableHeaders[`crd:${crdInfo.name}`] = headers;

        // Render table
        allItems = data.items || [];
        filteredItems = [...allItems];
        currentPage = 1;

        // Update summary for CRD instances
        const summaryEl = document.getElementById('resource-summary');
        if (summaryEl) {
            summaryEl.innerHTML = `<span class="summary-item"><span class="summary-count">${allItems.length}</span> instances</span>`;
        }

        // Render headers with filter row
        const headerRow = `<tr>${headers.map(h => {
            const sortClass = sortColumn === h ? (sortDirection === 'asc' ? 'sort-asc' : 'sort-desc') : '';
            return `<th class="${sortClass}" onclick="onColumnSort('${h}', this)">${h}<span class="sort-icon"></span></th>`;
        }).join('')}</tr>`;

        const filterRow = `<tr class="column-filter-row ${columnFiltersVisible ? 'active' : ''}" id="column-filter-row">
                    ${headers.map(h => {
            const filterValue = columnFilters[h] || '';
            return `<th><input type="text" class="column-filter-input" placeholder="Filter ${h.toLowerCase()}..."
                            value="${filterValue}"
                            data-column="${h}"
                            onkeyup="onColumnFilterChange(event, '${h}')"
                            onclick="event.stopPropagation()"></th>`;
        }).join('')}
                </tr>`;

        document.getElementById('table-header').innerHTML = headerRow + filterRow;

        if (!data.items || data.items.length === 0) {
            const nsInfo = crdInfo.namespaced ? (ns ? ` in namespace "${ns}"` : ' (all namespaces)') : '';
            document.getElementById('table-body').innerHTML =
                `<tr><td colspan="${headers.length}" style="text-align:center;padding:40px;">
                            <div style="color:var(--text-secondary);">No ${crdInfo.kind} instances found${nsInfo}</div>
                            <div style="font-size:11px;color:var(--text-secondary);margin-top:8px;">
                                CRD: ${crdInfo.group}/${crdInfo.version}
                            </div>
                        </td></tr>`;
            updatePaginationUI(0, 0);
            return;
        }

        applyFilterAndSort();
    } catch (e) {
        console.error('Failed to load CR instances:', e);
        const headers = crdInfo.namespaced ? ['NAME', 'NAMESPACE', 'STATUS', 'AGE'] : ['NAME', 'STATUS', 'AGE'];
        document.getElementById('table-body').innerHTML =
            `<tr><td colspan="${headers.length}" style="text-align:center;padding:40px;">
                        <div style="color:var(--accent-red);">Failed to load ${crdInfo.kind} instances</div>
                        <div style="font-size:11px;color:var(--text-secondary);margin-top:8px;">${escapeHtml(e.message)}</div>
                    </td></tr>`;
    }
}

// Show all CRDs in a modal
function showAllCRDs() {
    let html = `
                <div class="modal-overlay" onclick="closeModal(event)">
                    <div class="modal detail-modal" style="max-width: 800px;" onclick="event.stopPropagation()">
                        <div class="modal-header">
                            <h3>All Custom Resource Definitions (${loadedCRDs.length})</h3>
                            <button class="modal-close" onclick="closeAllModals()">&times;</button>
                        </div>
                        <div class="modal-body" style="max-height: 70vh; overflow-y: auto;">
                            <input type="text" id="crd-search" placeholder="Search CRDs..." style="width: 100%; padding: 8px; margin-bottom: 12px; background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary);" oninput="filterCRDList(this.value)">
                            <div id="crd-list-container">
                                ${renderCRDList(loadedCRDs)}
                            </div>
                        </div>
                    </div>
                </div>
            `;

    const modalContainer = document.createElement('div');
    modalContainer.id = 'crd-modal';
    modalContainer.innerHTML = html;
    document.body.appendChild(modalContainer);
}

// Render CRD list for modal
function renderCRDList(crds) {
    if (!crds || crds.length === 0) {
        return '<p style="color: var(--text-secondary);">No CRDs found</p>';
    }

    // Group by group
    const grouped = {};
    crds.forEach(crd => {
        const group = crd.group || 'core';
        if (!grouped[group]) grouped[group] = [];
        grouped[group].push(crd);
    });

    let html = '';
    const sortedGroups = Object.keys(grouped).sort();

    for (const group of sortedGroups) {
        html += `<div style="margin-bottom: 16px;">
                    <div style="font-size: 12px; color: var(--text-secondary); margin-bottom: 8px; border-bottom: 1px solid var(--border-color); padding-bottom: 4px;">${group}</div>`;

        for (const crd of grouped[group]) {
            const shortNames = crd.shortNames?.length ? ` (${crd.shortNames.join(', ')})` : '';
            const scope = crd.namespaced ? 'Namespaced' : 'Cluster';
            html += `<div class="nav-item" style="margin: 4px 0; padding: 8px; cursor: pointer;" onclick="closeAllModals(); switchToCRD('${crd.name}')">
                        <div style="display: flex; justify-content: space-between; width: 100%;">
                            <span><strong>${crd.kind}</strong>${shortNames}</span>
                            <span style="font-size: 11px; color: var(--text-secondary);">${scope} • ${crd.version}</span>
                        </div>
                    </div>`;
        }

        html += '</div>';
    }

    return html;
}

// Filter CRD list in modal
function filterCRDList(query) {
    const filtered = loadedCRDs.filter(crd => {
        const q = query.toLowerCase();
        return crd.name.toLowerCase().includes(q) ||
            crd.kind.toLowerCase().includes(q) ||
            crd.group.toLowerCase().includes(q) ||
            (crd.shortNames || []).some(s => s.toLowerCase().includes(q));
    });
    document.getElementById('crd-list-container').innerHTML = renderCRDList(filtered);
}

function switchResource(resource) {
    closeMobileSidebar();
    currentResource = resource;

    // Clear column filters when switching resources
    columnFilters = {};

    // Reset sort when switching resources
    sortColumn = null;
    sortDirection = 'asc';

    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.toggle('active', item.dataset.resource === resource);
    });
    document.getElementById('panel-title').textContent = resource.charAt(0).toUpperCase() + resource.slice(1);

    // Hide topology view, custom views and overview panel, show main panel
    hideTopologyView();
    hideAllCustomViews();
    hideOverviewPanel();

    // Update active column filters display
    updateActiveColumnFiltersDisplay();

    loadData();
}

function onNamespaceChange() {
    currentNamespace = document.getElementById('namespace-select').value;
    trackNamespaceUsage(currentNamespace);
    loadData();
}

function refreshData() {
    void manualRefresh();
}

// Auto-refresh functions
function startAutoRefresh() {
    if (autoRefreshTimer) {
        clearInterval(autoRefreshTimer);
    }
    if (autoRefreshEnabled && autoRefreshInterval > 0) {
        autoRefreshTimer = setInterval(() => {
            loadData({ forceNetwork: true });
        }, autoRefreshInterval * 1000);
        updateAutoRefreshUI();
    }
}

function stopAutoRefresh() {
    if (autoRefreshTimer) {
        clearInterval(autoRefreshTimer);
        autoRefreshTimer = null;
    }
    updateAutoRefreshUI();
}

function toggleAutoRefresh() {
    autoRefreshEnabled = !autoRefreshEnabled;
    localStorage.setItem('k13d_auto_refresh', autoRefreshEnabled);
    if (autoRefreshEnabled) {
        startAutoRefresh();
    } else {
        stopAutoRefresh();
    }
}

function setAutoRefreshInterval(seconds) {
    autoRefreshInterval = Math.max(5, Math.min(300, seconds)); // 5s to 5min
    localStorage.setItem('k13d_refresh_interval', autoRefreshInterval);
    if (autoRefreshEnabled) {
        startAutoRefresh();
    }
}

function updateAutoRefreshUI() {
    const toggle = document.getElementById('auto-refresh-toggle');
    const intervalSelect = document.getElementById('refresh-interval');
    if (toggle) {
        toggle.classList.toggle('active', autoRefreshEnabled);
        toggle.title = autoRefreshEnabled
            ? `Auto-refresh: ON (every ${autoRefreshInterval}s)`
            : 'Auto-refresh: OFF';
    }
    if (intervalSelect) {
        intervalSelect.value = autoRefreshInterval;
    }
}

async function manualRefresh() {
    const btn = document.querySelector('.refresh-btn');
    if (btn) {
        btn.classList.add('spinning');
    }
    try {
        await loadData({ forceNetwork: true });
        await loadNamespaces({ forceNetwork: true });
    } finally {
        if (btn) {
            setTimeout(() => btn.classList.remove('spinning'), 500);
        }
    }
}

function updateLastRefreshTime() {
    const el = document.getElementById('last-refresh-time');
    if (el) {
        el.textContent = formatTimeShort(new Date());
    }
}

function updateResourceSummary(resource, items) {
    const summaryEl = document.getElementById('resource-summary');
    if (!summaryEl) return;

    if (!items || items.length === 0) {
        summaryEl.innerHTML = '<span class="summary-item"><span class="summary-count">0</span> total</span>';
        return;
    }

    const total = items.length;
    let html = `<span class="summary-item"><span class="summary-count">${total}</span> total</span>`;

    // Resource-specific status breakdown
    if (resource === 'pods') {
        const statusCounts = {};
        items.forEach(item => {
            const status = (item.status || 'Unknown').toLowerCase();
            statusCounts[status] = (statusCounts[status] || 0) + 1;
        });
        const statusOrder = ['running', 'pending', 'succeeded', 'failed', 'unknown'];
        statusOrder.forEach(status => {
            if (statusCounts[status]) {
                html += `<span class="summary-item"><span class="summary-count status-${status}">${statusCounts[status]}</span> ${status}</span>`;
            }
        });
        // Handle other statuses
        Object.keys(statusCounts).forEach(status => {
            if (!statusOrder.includes(status) && statusCounts[status]) {
                html += `<span class="summary-item"><span class="summary-count">${statusCounts[status]}</span> ${status}</span>`;
            }
        });
    } else if (resource === 'deployments' || resource === 'statefulsets' || resource === 'replicasets') {
        let ready = 0, notReady = 0;
        items.forEach(item => {
            const readyStr = String(item.ready || '0/0');
            const parts = readyStr.includes('/') ? readyStr.split('/') : [readyStr, readyStr];
            if (parts.length === 2 && parts[0] === parts[1] && parts[0] !== '0') {
                ready++;
            } else {
                notReady++;
            }
        });
        if (ready > 0) html += `<span class="summary-item"><span class="summary-count status-running">${ready}</span> ready</span>`;
        if (notReady > 0) html += `<span class="summary-item"><span class="summary-count status-pending">${notReady}</span> not ready</span>`;
    } else if (resource === 'nodes') {
        let ready = 0, notReady = 0;
        items.forEach(item => {
            if (item.status === 'Ready') ready++;
            else notReady++;
        });
        if (ready > 0) html += `<span class="summary-item"><span class="summary-count status-running">${ready}</span> ready</span>`;
        if (notReady > 0) html += `<span class="summary-item"><span class="summary-count status-failed">${notReady}</span> not ready</span>`;
    } else if (resource === 'jobs') {
        let complete = 0, running = 0, failed = 0;
        items.forEach(item => {
            const status = (item.status || '').toLowerCase();
            if (status.includes('complete') || status === 'succeeded') complete++;
            else if (status.includes('fail')) failed++;
            else running++;
        });
        if (complete > 0) html += `<span class="summary-item"><span class="summary-count status-succeeded">${complete}</span> complete</span>`;
        if (running > 0) html += `<span class="summary-item"><span class="summary-count status-pending">${running}</span> running</span>`;
        if (failed > 0) html += `<span class="summary-item"><span class="summary-count status-failed">${failed}</span> failed</span>`;
    } else if (resource === 'events') {
        const typeCounts = {};
        items.forEach(item => {
            const type = item.type || 'Unknown';
            typeCounts[type] = (typeCounts[type] || 0) + 1;
        });
        if (typeCounts['Normal']) html += `<span class="summary-item"><span class="summary-count status-running">${typeCounts['Normal']}</span> normal</span>`;
        if (typeCounts['Warning']) html += `<span class="summary-item"><span class="summary-count status-pending">${typeCounts['Warning']}</span> warning</span>`;
    } else if (resource === 'services') {
        const typeCounts = {};
        items.forEach(item => {
            const type = item.type || 'ClusterIP';
            typeCounts[type] = (typeCounts[type] || 0) + 1;
        });
        Object.keys(typeCounts).forEach(type => {
            html += `<span class="summary-item"><span class="summary-count">${typeCounts[type]}</span> ${type}</span>`;
        });
    }

    summaryEl.innerHTML = html;
}

function renderTable(resource, items) {
    // Standardize items and update state
    allItems = items || [];
    
    // Render headers for standard resources
    // CRD headers are handled in renderCRDInstances, but we support both here
    renderTableHeaders(resource);
    
    // Apply initial filtering and sorting, then render body via renderCurrentPage
    applyFilterAndSort();

    updateResourceSummary(resource, allItems);
}


// Show Custom Resource detail using the shared detail-modal
async function showCRDetail(crdName, namespace, name) {
    const crdInfo = loadedCRDs.find(c => c.name === crdName);
    if (!crdInfo) return;

    try {
        // Fetch full CR as JSON for overview
        const ns = namespace ? `&namespace=${namespace}` : '';
        const resp = await fetchWithAuth(`/api/crd/${crdName}/instances/${name}?${ns}`);
        const crData = await resp.json();

        // Store as selectedResource for YAML/Events tabs
        selectedResource = {
            name: name,
            namespace: namespace,
            _isCR: true,
            _crdName: crdName,
            _crdInfo: crdInfo,
            _crData: crData,
        };

        document.getElementById('detail-title').textContent = `${crdInfo.kind}: ${name}`;

        // Overview tab
        document.getElementById('detail-overview').innerHTML = generateCROverview(crdInfo, crData);

        // YAML tab - load on demand
        document.getElementById('detail-yaml').innerHTML = '<div class="yaml-viewer" style="color: var(--text-secondary);">Click the YAML tab to load...</div>';
        document.getElementById('detail-yaml').dataset.loaded = 'false';

        // Events tab - load on demand
        document.getElementById('detail-events').innerHTML = '<p style="color: var(--text-secondary);">Click the Events tab to load...</p>';
        document.getElementById('detail-events').dataset.loaded = 'false';

        // Hide Related Pods tab
        document.getElementById('detail-pods-tab').style.display = 'none';

        document.getElementById('detail-modal').classList.add('active');
        switchDetailTab('overview');
    } catch (e) {
        console.error('Failed to load CR detail:', e);
    }
}

// Generate rich overview for Custom Resources
function generateCROverview(crdInfo, crData) {
    const metadata = crData.metadata || {};
    const spec = crData.spec || {};
    const status = crData.status || {};
    const labels = metadata.labels || {};
    const annotations = metadata.annotations || {};

    // Determine status from common patterns
    let statusText = '-';
    let statusColor = 'var(--text-secondary)';
    const conditions = status.conditions || [];

    if (status.phase) {
        statusText = status.phase;
    } else if (status.state) {
        statusText = status.state;
    } else if (typeof status.ready === 'boolean') {
        statusText = status.ready ? 'Ready' : 'NotReady';
    } else if (conditions.length > 0) {
        const readyCond = conditions.find(c => c.type === 'Ready' || c.type === 'Available' || c.type === 'Synced');
        if (readyCond) {
            statusText = readyCond.status === 'True' ? readyCond.type : `Not${readyCond.type}`;
        }
    }

    // Also check printer columns for status
    if (statusText === '-' && crdInfo.printerColumns) {
        for (const col of crdInfo.printerColumns) {
            const key = col.name.toLowerCase();
            if (key === 'status' || key === 'phase' || key === 'state' || key === 'ready') {
                const val = resolveJSONPathClient(crData, col.jsonPath || col.JSONPath);
                if (val) { statusText = String(val); break; }
            }
        }
    }

    const readyStates = ['ready', 'running', 'active', 'healthy', 'synced', 'true', 'available', 'bound', 'succeeded', 'complete'];
    const failedStates = ['failed', 'error', 'notready', 'unavailable', 'false', 'degraded', 'crashloopbackoff'];
    const statusLower = statusText.toLowerCase();
    if (readyStates.some(s => statusLower.includes(s))) {
        statusColor = 'var(--accent-green)';
    } else if (failedStates.some(s => statusLower.includes(s))) {
        statusColor = 'var(--accent-red)';
    } else if (statusText !== '-') {
        statusColor = 'var(--accent-yellow)';
    }

    // Build labels HTML
    const labelHtml = Object.keys(labels).length > 0
        ? Object.entries(labels).map(([k, v]) =>
            `<span style="display:inline-block;padding:2px 8px;margin:2px;border-radius:4px;background:var(--accent-blue)15;color:var(--accent-blue);font-size:11px;border:1px solid var(--accent-blue)30;font-family:monospace;">${escapeHtml(k)}=${escapeHtml(v)}</span>`
        ).join('')
        : '<span style="color:var(--text-secondary);font-size:12px;">None</span>';

    // Build spec fields (top-level only, skip large nested objects)
    const specEntries = Object.entries(spec).filter(([k, v]) => {
        if (v === null || v === undefined) return false;
        if (typeof v === 'object' && !Array.isArray(v) && Object.keys(v).length > 5) return false;
        return true;
    }).slice(0, 12);

    const specHtml = specEntries.length > 0
        ? specEntries.map(([k, v]) => {
            let display;
            if (typeof v === 'object') {
                display = Array.isArray(v) ? `[${v.length} items]` : JSON.stringify(v);
                if (display.length > 80) display = display.substring(0, 77) + '...';
            } else {
                display = String(v);
            }
            return `<div class="overview-stat">
                        <span class="stat-label">${escapeHtml(k)}</span>
                        <span class="stat-value" style="font-family:monospace;font-size:12px;word-break:break-all;">${escapeHtml(display)}</span>
                    </div>`;
        }).join('')
        : '<div style="color:var(--text-secondary);font-size:12px;padding:8px;">No spec fields</div>';

    // Build conditions table
    let conditionsHtml = '';
    if (conditions.length > 0) {
        conditionsHtml = `
                    <div class="overview-card" style="grid-column: 1 / -1;">
                        <div class="overview-card-title">Conditions</div>
                        <div style="overflow-x:auto;">
                            <table style="width:100%;border-collapse:collapse;font-size:12px;">
                                <thead>
                                    <tr style="border-bottom:1px solid var(--border-color);">
                                        <th style="text-align:left;padding:6px 8px;color:var(--text-secondary);">TYPE</th>
                                        <th style="text-align:left;padding:6px 8px;color:var(--text-secondary);">STATUS</th>
                                        <th style="text-align:left;padding:6px 8px;color:var(--text-secondary);">REASON</th>
                                        <th style="text-align:left;padding:6px 8px;color:var(--text-secondary);">MESSAGE</th>
                                        <th style="text-align:left;padding:6px 8px;color:var(--text-secondary);">LAST TRANSITION</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${conditions.map(c => {
            const condColor = c.status === 'True' ? 'var(--accent-green)' : c.status === 'False' ? 'var(--accent-red)' : 'var(--accent-yellow)';
            const age = c.lastTransitionTime ? formatTimeShort(c.lastTransitionTime) : '-';
            return `<tr style="border-bottom:1px solid var(--border-color)20;">
                                            <td style="padding:6px 8px;font-weight:500;">${escapeHtml(c.type || '-')}</td>
                                            <td style="padding:6px 8px;color:${condColor};font-weight:600;">${escapeHtml(c.status || '-')}</td>
                                            <td style="padding:6px 8px;color:var(--text-secondary);">${escapeHtml(c.reason || '-')}</td>
                                            <td style="padding:6px 8px;color:var(--text-secondary);max-width:300px;overflow:hidden;text-overflow:ellipsis;" title="${escapeHtml(c.message || '')}">${escapeHtml(c.message || '-')}</td>
                                            <td style="padding:6px 8px;color:var(--text-secondary);">${age}</td>
                                        </tr>`;
        }).join('')}
                                </tbody>
                            </table>
                        </div>
                    </div>`;
    }

    // Build status fields (excluding conditions)
    const statusEntries = Object.entries(status).filter(([k]) => k !== 'conditions').slice(0, 8);
    const statusFieldsHtml = statusEntries.length > 0
        ? statusEntries.map(([k, v]) => {
            let display;
            if (typeof v === 'object') {
                display = JSON.stringify(v);
                if (display.length > 80) display = display.substring(0, 77) + '...';
            } else {
                display = String(v);
            }
            return `<div class="overview-stat">
                        <span class="stat-label">${escapeHtml(k)}</span>
                        <span class="stat-value" style="font-family:monospace;font-size:12px;">${escapeHtml(display)}</span>
                    </div>`;
        }).join('')
        : '';

    // Build printer columns card
    let printerColsHtml = '';
    const printerCols = crdInfo.printerColumns || [];
    const displayCols = printerCols.filter(c => {
        const key = c.name.toLowerCase();
        return key !== 'age' && key !== 'name' && key !== 'namespace';
    });
    if (displayCols.length > 0) {
        const colValues = displayCols.map(c => {
            const val = resolveJSONPathClient(crData, c.jsonPath || c.JSONPath) || '-';
            return `<div class="overview-stat">
                        <span class="stat-label">${escapeHtml(c.name)}</span>
                        <span class="stat-value" style="font-family:monospace;font-size:12px;">${escapeHtml(String(val))}</span>
                    </div>`;
        }).join('');
        printerColsHtml = `
                    <div class="overview-card">
                        <div class="overview-card-title">Key Fields</div>
                        <div class="overview-card-content">${colValues}</div>
                    </div>`;
    }

    // Annotations (show first 5, truncated)
    const annotationEntries = Object.entries(annotations).slice(0, 5);
    const annotationsHtml = annotationEntries.length > 0
        ? annotationEntries.map(([k, v]) => {
            const shortVal = v.length > 60 ? v.substring(0, 57) + '...' : v;
            return `<div class="overview-stat">
                        <span class="stat-label" style="font-size:11px;" title="${escapeHtml(k)}">${escapeHtml(k.split('/').pop())}</span>
                        <span class="stat-value" style="font-size:11px;font-family:monospace;" title="${escapeHtml(v)}">${escapeHtml(shortVal)}</span>
                    </div>`;
        }).join('')
        : '';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(statusText)}
                    </div>
                    <span style="color:var(--text-secondary);font-size:12px;margin-left:12px;">${escapeHtml(crdInfo.group)}/${escapeHtml(crdInfo.version)}</span>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Name</span>
                                <span class="stat-value" style="font-family:monospace;">${escapeHtml(metadata.name || '-')}</span>
                            </div>
                            ${metadata.namespace ? `<div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(metadata.namespace)}</span>
                            </div>` : ''}
                            <div class="overview-stat">
                                <span class="stat-label">Created</span>
                                <span class="stat-value">${metadata.creationTimestamp ? formatTimeShort(metadata.creationTimestamp) : '-'}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Generation</span>
                                <span class="stat-value">${metadata.generation || '-'}</span>
                            </div>
                        </div>
                    </div>
                    ${printerColsHtml}
                    <div class="overview-card">
                        <div class="overview-card-title">Spec</div>
                        <div class="overview-card-content">${specHtml}</div>
                    </div>
                    ${statusFieldsHtml ? `<div class="overview-card">
                        <div class="overview-card-title">Status</div>
                        <div class="overview-card-content">${statusFieldsHtml}</div>
                    </div>` : ''}
                </div>
                ${conditionsHtml}
                <div class="overview-card" style="margin-top:12px;">
                    <div class="overview-card-title">Labels</div>
                    <div style="padding:8px;">${labelHtml}</div>
                </div>
                ${annotationsHtml ? `<div class="overview-card" style="margin-top:12px;">
                    <div class="overview-card-title">Annotations</div>
                    <div class="overview-card-content">${annotationsHtml}</div>
                </div>` : ''}
            `;
}

// Client-side JSONPath resolver (mirrors Go's ResolveJSONPath)
function resolveJSONPathClient(obj, path) {
    if (!path || !obj) return null;
    path = path.replace(/^\./, '');
    return _resolvePathRec(obj, path);
}

function _resolvePathRec(current, path) {
    if (!path || current === null || current === undefined) return current;
    if (typeof current !== 'object' || Array.isArray(current)) return null;

    const dotIdx = path.indexOf('.');
    const bracketIdx = path.indexOf('[');

    // Simple field (no dot, no bracket)
    if (dotIdx < 0 && bracketIdx < 0) return current[path];

    // Array bracket comes before dot (or no dot)
    if (bracketIdx >= 0 && (dotIdx < 0 || bracketIdx < dotIdx)) {
        const fieldName = path.substring(0, bracketIdx);
        const rest = path.substring(bracketIdx);
        const arr = current[fieldName];
        if (!Array.isArray(arr)) return null;

        const bracketEnd = rest.indexOf(']');
        if (bracketEnd < 0) return null;
        const bracketContent = rest.substring(1, bracketEnd);
        let remaining = rest.substring(bracketEnd + 1);
        if (remaining.startsWith('.')) remaining = remaining.substring(1);

        // Array filter: ?(@.key=="value")
        if (bracketContent.startsWith('?(@.')) {
            const expr = bracketContent.substring(4, bracketContent.length - 1); // strip ?(@. and )
            const eqParts = expr.split('==');
            if (eqParts.length === 2) {
                const key = eqParts[0];
                const value = eqParts[1].replace(/['"]/g, '');
                const found = arr.find(item => item && String(item[key]) === value);
                return remaining ? _resolvePathRec(found, remaining) : found;
            }
            return null;
        }

        // Numeric index
        const idx = parseInt(bracketContent);
        if (!isNaN(idx) && idx >= 0 && idx < arr.length) {
            return remaining ? _resolvePathRec(arr[idx], remaining) : arr[idx];
        }
        return null;
    }

    // Dot-separated path
    const fieldName = path.substring(0, dotIdx);
    const rest = path.substring(dotIdx + 1);
    return _resolvePathRec(current[fieldName], rest);
}

// Agentic Mode State - DEFAULT ON for tool execution
let pendingApproval = null;

// Resource name mappings for AI command parsing
const resourceAliases = {
    // Korean aliases
    '파드': 'pods', '팟': 'pods', '포드': 'pods',
    '디플로이먼트': 'deployments', '배포': 'deployments',
    '서비스': 'services', '서비스들': 'services',
    '노드': 'nodes', '노드들': 'nodes',
    '네임스페이스': 'namespaces', '네임스페이스들': 'namespaces',
    '컨피그맵': 'configmaps', '설정': 'configmaps',
    '시크릿': 'secrets', '비밀': 'secrets',
    '인그레스': 'ingresses',
    '이벤트': 'events', '이벤트들': 'events',
    '스테이트풀셋': 'statefulsets',
    '데몬셋': 'daemonsets',
    '레플리카셋': 'replicasets',
    '잡': 'jobs', '작업': 'jobs',
    '크론잡': 'cronjobs', '스케줄잡': 'cronjobs',
    '볼륨': 'persistentvolumeclaims', 'pvc': 'persistentvolumeclaims',
    '롤': 'roles', '역할': 'roles',
    '서비스계정': 'serviceaccounts',
    // English aliases
    'pod': 'pods', 'deployment': 'deployments', 'deploy': 'deployments',
    'service': 'services', 'svc': 'services',
    'node': 'nodes', 'namespace': 'namespaces', 'ns': 'namespaces',
    'configmap': 'configmaps', 'cm': 'configmaps',
    'secret': 'secrets', 'ingress': 'ingresses', 'ing': 'ingresses',
    'event': 'events', 'ev': 'events',
    'statefulset': 'statefulsets', 'sts': 'statefulsets',
    'daemonset': 'daemonsets', 'ds': 'daemonsets',
    'replicaset': 'replicasets', 'rs': 'replicasets',
    'job': 'jobs', 'cronjob': 'cronjobs', 'cj': 'cronjobs',
    'pv': 'persistentvolumes', 'persistentvolume': 'persistentvolumes',
    'role': 'roles', 'rolebinding': 'rolebindings', 'rb': 'rolebindings',
    'clusterrole': 'clusterroles', 'cr': 'clusterroles',
    'clusterrolebinding': 'clusterrolebindings', 'crb': 'clusterrolebindings',
    'serviceaccount': 'serviceaccounts', 'sa': 'serviceaccounts',
    'networkpolicy': 'networkpolicies', 'netpol': 'networkpolicies'
};

// Parse user message and AI response for dashboard commands
async function handleAIDashboardCommands(aiResponse, userMessage) {
    const msg = userMessage.toLowerCase();
    const resp = aiResponse.toLowerCase();

    // Detect show/list resource commands from user message
    const showPatterns = [
        /(?:show|display|list|get|보여|조회|확인|봐|봐줘|보기|리스트).*?(pods?|deployments?|services?|nodes?|namespaces?|configmaps?|secrets?|ingress(?:es)?|events?|statefulsets?|daemonsets?|replicasets?|jobs?|cronjobs?|persistentvolume(?:claim)?s?|roles?|rolebindings?|clusterroles?|clusterrolebindings?|serviceaccounts?|networkpolic(?:y|ies)|파드|팟|포드|디플로이먼트|배포|서비스|노드|네임스페이스|컨피그맵|설정|시크릿|비밀|인그레스|이벤트|스테이트풀셋|데몬셋|레플리카셋|잡|작업|크론잡|스케줄잡|볼륨|pvc|pv|롤|역할|서비스계정|svc|ns|cm|ing|ev|sts|ds|rs|cj|rb|cr|crb|sa|netpol)/i,
        /(?:pods?|deployments?|services?|nodes?|namespaces?|configmaps?|secrets?|ingress(?:es)?|events?|statefulsets?|daemonsets?|replicasets?|jobs?|cronjobs?|persistentvolume(?:claim)?s?|roles?|rolebindings?|clusterroles?|clusterrolebindings?|serviceaccounts?|networkpolic(?:y|ies)|파드|팟|포드|디플로이먼트|배포|서비스|노드|네임스페이스|컨피그맵|설정|시크릿|비밀|인그레스|이벤트|스테이트풀셋|데몬셋|레플리카셋|잡|작업|크론잡|스케줄잡|볼륨|pvc|pv|롤|역할|서비스계정|svc|ns|cm|ing|ev|sts|ds|rs|cj|rb|cr|crb|sa|netpol).*?(?:show|display|list|보여|조회|확인|봐|봐줘|보기|리스트)/i
    ];

    let detectedResource = null;
    let detectedNamespace = null;

    // Check user message for resource commands
    for (const pattern of showPatterns) {
        const match = msg.match(pattern);
        if (match) {
            const resourceWord = match[1] || match[0];
            detectedResource = resourceAliases[resourceWord.toLowerCase()] || resourceWord.toLowerCase();
            // Ensure it's a valid resource
            if (allResources.includes(detectedResource)) {
                break;
            }
            detectedResource = null;
        }
    }

    // Check for namespace specification
    const nsPatterns = [
        /(?:namespace|ns|네임스페이스)[:\s=]+([a-z0-9-]+)/i,
        /(?:in|from|에서|의)\s+([a-z0-9-]+)\s+(?:namespace|ns|네임스페이스)/i,
        /-n\s+([a-z0-9-]+)/i
    ];

    for (const pattern of nsPatterns) {
        const match = msg.match(pattern);
        if (match) {
            detectedNamespace = match[1];
            break;
        }
    }

    // Also check AI response for explicit dashboard commands
    // AI can include special markers like [[SHOW:pods]] or [[NAMESPACE:default]]
    const aiShowMatch = aiResponse.match(/\[\[SHOW:([a-z]+)\]\]/i);
    const aiNsMatch = aiResponse.match(/\[\[NAMESPACE:([a-z0-9-]*)\]\]/i);

    if (aiShowMatch) {
        detectedResource = aiShowMatch[1].toLowerCase();
    }
    if (aiNsMatch) {
        detectedNamespace = aiNsMatch[1] || ''; // empty string means all namespaces
    }

    // Execute dashboard navigation if resource detected
    if (detectedResource && allResources.includes(detectedResource)) {
        // Show notification
        showDashboardActionNotification(`Switching to ${detectedResource}...`);

        // Switch namespace first if specified
        if (detectedNamespace !== null) {
            const nsSelect = document.getElementById('namespace-select');
            if (nsSelect) {
                // Check if namespace exists in dropdown
                const nsExists = Array.from(nsSelect.options).some(opt => opt.value === detectedNamespace);
                if (nsExists || detectedNamespace === '') {
                    nsSelect.value = detectedNamespace;
                    currentNamespace = detectedNamespace;
                }
            }
        }

        // Switch to the resource view
        switchResource(detectedResource);

        // Scroll dashboard into view on mobile
        const dashboardPanel = document.querySelector('.dashboard-panel');
        if (dashboardPanel && window.innerWidth < 768) {
            dashboardPanel.scrollIntoView({ behavior: 'smooth' });
        }
    }

    // Check for filter commands
    const filterPatterns = [
        /(?:filter|find|search|필터|검색|찾아)[:\s]+["']?([^"'\n]+)["']?/i,
        /["']([^"']+)["'].*?(?:filter|find|search|필터|검색|찾아)/i
    ];

    for (const pattern of filterPatterns) {
        const match = msg.match(pattern);
        if (match && match[1]) {
            const filterText = match[1].trim();
            if (filterText && filterText.length > 1) {
                const filterInput = document.getElementById('filter-input');
                if (filterInput) {
                    filterInput.value = filterText;
                    filterTable(filterText.toLowerCase());
                    showDashboardActionNotification(`Filtering by "${filterText}"...`);
                }
            }
            break;
        }
    }
}

// Show a brief notification for dashboard actions
function showDashboardActionNotification(message) {
    const notification = document.createElement('div');
    notification.className = 'dashboard-action-notification';
    notification.textContent = message;
    notification.style.cssText = `
                position: fixed;
                top: 60px;
                left: 50%;
                transform: translateX(-50%);
                background: var(--accent-blue);
                color: white;
                padding: 8px 16px;
                border-radius: 4px;
                z-index: 10000;
                font-size: 13px;
                animation: fadeInOut 2s ease-in-out;
            `;
    document.body.appendChild(notification);

    setTimeout(() => {
        notification.remove();
    }, 2000);
}

// AI Chat
async function sendMessage() {
    const input = document.getElementById('ai-input');
    const message = input.value.trim();
    
    // If it's already generation, skip validation to allow Stop button to work
    if (isLoading) return;
    
    if (!message) {
        const originalPlaceholder = input.placeholder;
        input.placeholder = t('msg_enter_question') || '질문을 입력해 주세요.';
        input.classList.add('error');
        input.focus();
        
        // Remove error state when user types
        const onInput = () => {
            input.classList.remove('error');
            input.placeholder = originalPlaceholder;
            input.removeEventListener('input', onInput);
        };
        input.addEventListener('input', onInput);
        return;
    }
    
    if (isLoading) return;

    // Check guardrails (K8s safety analysis)
    const guardrailCheck = checkGuardrails(message);

    if (!guardrailCheck.allowed) {
        showToast(guardrailCheck.reason, 'error');
        return;
    }

    // Show safety confirmation dialog for risky operations
    if (guardrailCheck.requireConfirmation) {
        const analysis = {
            riskLevel: guardrailCheck.riskLevel || 'warning',
            explanation: guardrailCheck.reason,
            warnings: [guardrailCheck.reason],
            recommendations: guardrailCheck.riskLevel === 'critical' ?
                ['Consider using --dry-run=client first', 'Verify the correct cluster context'] :
                ['Review the operation before proceeding']
        };

        return new Promise((resolve) => {
            showSafetyConfirmation(analysis,
                () => {
                    // User confirmed - proceed
                    proceedWithMessage(message);
                    resolve();
                },
                () => {
                    // User cancelled
                    showToast('Operation cancelled', 'info');
                    resolve();
                }
            );
        });
    }

    await proceedWithMessage(message);
}

function stopGeneration() {
    if (aiAbortController) {
        aiAbortController.abort();
        aiAbortController = null;
        console.log('[AI] Generation stopped by user');
        showToast('Generation stopped', 'info');
    }
}

async function proceedWithMessage(message) {
    isLoading = true;
    const sendBtn = document.getElementById('send-btn');
    const aiInput = document.getElementById('ai-input');
    if (sendBtn) sendBtn.disabled = true;
    aiInput.value = '';
    aiInput.disabled = true;

    saveQueryToHistory(message);
    aiHistoryIndex = -1;
    aiCurrentDraft = '';

    // User message is saved by backend in handleAgenticChat
    addMessage(message, true);

    try {
        // Always use agentic mode
        await sendMessageAgentic(message);
    } finally {
        isLoading = false;
        if (sendBtn) sendBtn.disabled = false;
        aiInput.disabled = false;
        aiInput.focus();
    }
}

// Format resource links in AI responses to make them clickable
function formatResourceLinks(text) {
    // Common Kubernetes resource patterns
    // Match patterns like: pod/nginx-xxx, deployment/my-app, service/my-svc
    // or just: nginx-pod, my-deployment (when context is clear)

    // Pattern 1: explicit resource/name format (e.g., pod/nginx-xxx, deployment/my-app)
    const explicitPattern = /\b(pod|deployment|service|statefulset|daemonset|configmap|secret|ingress|node|namespace|replicaset|job|cronjob)s?\/([a-z0-9][-a-z0-9]*[a-z0-9])\b/gi;

    text = text.replace(explicitPattern, (match, kind, name) => {
        const resourceMap = {
            'pod': 'pods', 'pods': 'pods',
            'deployment': 'deployments', 'deployments': 'deployments',
            'service': 'services', 'services': 'services',
            'statefulset': 'statefulsets', 'statefulsets': 'statefulsets',
            'daemonset': 'daemonsets', 'daemonsets': 'daemonsets',
            'configmap': 'configmaps', 'configmaps': 'configmaps',
            'secret': 'secrets', 'secrets': 'secrets',
            'ingress': 'ingresses', 'ingresses': 'ingresses',
            'node': 'nodes', 'nodes': 'nodes',
            'namespace': 'namespaces', 'namespaces': 'namespaces',
            'replicaset': 'replicasets', 'replicasets': 'replicasets',
            'job': 'jobs', 'jobs': 'jobs',
            'cronjob': 'cronjobs', 'cronjobs': 'cronjobs'
        };
        const resourceType = resourceMap[kind.toLowerCase()] || 'pods';
        return `<a href="#" class="resource-link" onclick="navigateToResource('${resourceType}', '${name}'); return false;">${match}</a>`;
    });

    // Pattern 2: backtick-quoted names that look like k8s resources
    // e.g., `nginx-deployment`, `my-service`, `coredns-xxxxx`
    const backtickPattern = /`([a-z][a-z0-9]*(?:-[a-z0-9]+)+)`/gi;
    text = text.replace(backtickPattern, (match, name) => {
        // Only convert if it looks like a k8s resource name (has hyphens)
        if (name.includes('-')) {
            return `<a href="#" class="resource-link" onclick="searchAndNavigateToResource('${name}'); return false;">\`${name}\`</a>`;
        }
        return match;
    });

    return text;
}

// Navigate directly to a known resource type
function navigateToResource(resourceType, name) {
    switchResource(resourceType);
    setTimeout(() => {
        document.getElementById('filter-input').value = name;
        currentFilter = name.toLowerCase();
        applyFilterAndSort();
    }, 500);
}

// Search for resource and navigate (when type is unknown)
async function searchAndNavigateToResource(name) {
    try {
        const response = await fetch(`/api/search?q=${encodeURIComponent(name)}&namespace=${currentNamespace || ''}`, {
            headers: { 'Authorization': `Bearer ${authToken}` }
        });
        if (response.ok) {
            const data = await response.json();
            if (data.results && data.results.length > 0) {
                navigateToSearchResult(data.results[0]);
                return;
            }
        }
    } catch (e) {
        console.error('Search error:', e);
    }
    // Fallback: just filter current view
    document.getElementById('filter-input').value = name;
    currentFilter = name.toLowerCase();
    applyFilterAndSort();
}

// Agentic chat with tool calling and Decision Required flow
async function sendMessageAgentic(message) {
    const container = document.getElementById('ai-messages');
    const div = document.createElement('div');
    div.className = 'message assistant streaming';
    div.id = 'streaming-message';
    div.innerHTML = `<div class="message-content"><span class="cursor">▊</span></div>`;
    container.appendChild(div);
    aiForceScrollToBottom();

    const contentEl = div.querySelector('.message-content');
    let fullContent = '';

    aiAbortController = new AbortController();
    const signal = aiAbortController.signal;

    try {
        const response = await fetch('/api/chat/agentic', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify({ message, language: currentLanguage, session_id: currentSessionId }),
            signal: signal
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `HTTP ${response.status}`);
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();

        let currentEventType = null;

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            const chunk = decoder.decode(value, { stream: true });
            const lines = chunk.split('\n');

            for (const line of lines) {
                // Handle event type lines
                if (line.startsWith('event: ')) {
                    currentEventType = line.slice(7).trim();
                    continue;
                }

                if (line.startsWith('data: ')) {
                    const data = line.slice(6);

                    if (data === '[DONE]') {
                        break;
                    }

                    // Handle session events - save session_id for conversation continuity
                    if (currentEventType === 'session') {
                        try {
                            const sessionInfo = JSON.parse(data);
                            if (sessionInfo.session_id) {
                                currentSessionId = sessionInfo.session_id;
                                sessionStorage.setItem('k13d_session_id', currentSessionId);
                            }
                        } catch (e) {
                            console.error('Failed to parse session:', e);
                        }
                        currentEventType = null;
                        continue;
                    }

                    // Handle tool_execution events - insert before the AI response text
                    if (currentEventType === 'tool_execution') {
                        try {
                            const execInfo = JSON.parse(data);
                            showToolExecution(execInfo, div, contentEl);
                        } catch (e) {
                            console.error('Failed to parse tool_execution:', e);
                        }
                        currentEventType = null;
                        continue;
                    }

                    // Check if this is an approval request
                    if (currentEventType === 'approval') {
                        try {
                            const parsed = JSON.parse(data);
                            if (parsed.type === 'approval_required') {
                                showApprovalModal(parsed);
                            } else if (parsed.type === 'approval_blocked') {
                                showApprovalBlocked(parsed);
                            }
                        } catch (e) {
                            console.error('Failed to parse approval:', e);
                        }
                        currentEventType = null;
                        continue;
                    }

                    // Try parsing as JSON for other event types
                    try {
                        const parsed = JSON.parse(data);
                        if (parsed.type === 'approval_required') {
                            showApprovalModal(parsed);
                            continue;
                        }
                        if (parsed.type === 'approval_blocked') {
                            showApprovalBlocked(parsed);
                            continue;
                        }
                        if (parsed.type === 'tool_execution') {
                            showToolExecution(parsed, div, contentEl);
                            continue;
                        }
                    } catch (e) {
                        // Not JSON, treat as regular text
                    }

                    // Regular text streaming
                    const text = data.replace(/\\n/g, '\n');
                    fullContent += text;

                    let formatted = fullContent;
                    formatted = formatted.replace(/```(\w*)\n?([\s\S]*?)```/g, '<pre><code>$2</code></pre>');
                    formatted = formatted.replace(/\n/g, '<br>');
                    contentEl.innerHTML = formatted + '<span class="cursor">▊</span>';
                    aiScrollToBottom();

                    currentEventType = null;
                }
            }
        }

        // Finalize
        div.classList.remove('streaming');
        div.id = '';
        let formatted = fullContent;
        formatted = formatted.replace(/```(\w*)\n?([\s\S]*?)```/g, '<pre><code>$2</code></pre>');
        formatted = formatResourceLinks(formatted);
        formatted = formatted.replace(/\n/g, '<br>');
        contentEl.innerHTML = formatted;

        // AI response is saved by backend in handleAgenticChat
        // Refresh chat history list to reflect updated title/message count
        if (fullContent.trim()) {
            loadChatHistory().catch(() => {});
        }

        // Parse AI response for dashboard commands and execute them
        await handleAIDashboardCommands(fullContent, message);

        // Refresh resource list after potential changes
        await loadData();

    } catch (e) {
        if (e.name === 'AbortError') {
            console.log('[AI] Stream aborted');
            if (fullContent.trim()) {
                // Keep what we already got
                div.classList.remove('streaming');
                div.id = '';
                contentEl.innerHTML = formatResourceLinks(fullContent.replace(/\n/g, '<br>'));
            } else {
                div.remove();
            }
            return;
        }

        div.classList.remove('streaming');
        div.id = '';

        // Provide user-friendly error messages
        let errorMsg = e.message;
        if (e.message.includes('AI client not configured') || e.message.includes('503')) {
            errorMsg = `<strong>AI Assistant Not Configured</strong><br><br>
                        The AI assistant requires an LLM provider to be configured. Please go to
                        <strong>Settings → AI/LLM Settings</strong> to configure your preferred provider
                        (OpenAI, Anthropic, Ollama, etc.).<br><br>
                        <em>Note: You need an API key from your chosen provider.</em>`;
        } else if (e.message.includes('does not support tool calling')) {
            errorMsg = `<strong>Tool Calling Not Supported</strong><br><br>
                        The current AI provider does not support tool calling (agentic mode).
                        Please configure a provider that supports tool calling, such as:<br>
                        • OpenAI (GPT-4, GPT-3.5-turbo)<br>
                        • Anthropic Claude<br>
                        • Ollama with compatible models`;
        }

        contentEl.innerHTML = `<span style="color: var(--accent-red)">${errorMsg}</span>`;
    } finally {
        aiAbortController = null;
    }
}

// Show tool execution info with expandable result
// messageDiv: the AI message div, contentEl: the text content element inside it
function showToolExecution(execInfo, messageDiv, contentEl) {
    const execDiv = document.createElement('div');
    execDiv.className = 'tool-execution';

    const isError = execInfo.is_error;
    const statusIcon = isError ? '❌' : '✅';
    const statusColor = isError ? 'var(--accent-red)' : 'var(--accent-green)';

    const uniqueId = 'tool-result-' + Date.now();
    const resultLength = execInfo.result ? execInfo.result.length : 0;

    const toolLabelParts = [];
    if (execInfo.tool) {
        toolLabelParts.push(execInfo.tool);
    }
    if (execInfo.server) {
        toolLabelParts.push(`(${execInfo.server})`);
    }
    const toolLabel = toolLabelParts.join(' ');

    execDiv.innerHTML = `
                <div class="tool-header" style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px;">
                    <span style="color: ${statusColor};">${statusIcon}</span>
                    <span class="tool-name">${toolLabel}</span>
                </div>
                <div class="tool-command" style="background: var(--bg-primary); padding: 8px; border-radius: 4px; font-family: monospace; font-size: 12px; margin-bottom: 8px; word-break: break-all;">
                    $ ${escapeHtml(execInfo.command || 'N/A')}
                </div>
                ${execInfo.result ? `
                    <div class="tool-result-container">
                        <div class="tool-result-full" id="${uniqueId}-full" style="display: none; background: var(--bg-primary); padding: 8px; border-radius: 4px; font-family: monospace; font-size: 11px; max-height: 400px; overflow: auto; white-space: pre-wrap; word-break: break-all; color: ${isError ? 'var(--accent-red)' : 'var(--text-secondary)'};">
${escapeHtml(execInfo.result)}</div>
                        <button onclick="toggleToolResult('${uniqueId}')" id="${uniqueId}-btn" style="margin-top: 6px; padding: 4px 8px; font-size: 11px; background: var(--bg-tertiary); border: none; border-radius: 4px; color: var(--text-primary); cursor: pointer;">
                            ▼ Show Result (${resultLength} chars)
                        </button>
                    </div>
                ` : ''}
            `;

    // Insert tool execution before the content element (AI response text)
    messageDiv.insertBefore(execDiv, contentEl);
    aiScrollToBottom();

    // Log to debug panel
    addDebugLog('tool', 'Tool Executed', {
        tool: execInfo.tool,
        tool_type: execInfo.tool_type,
        server: execInfo.server,
        command: execInfo.command,
        result_length: resultLength,
        is_error: isError
    });
}

// Toggle tool result expansion
function toggleToolResult(uniqueId) {
    const full = document.getElementById(uniqueId + '-full');
    const btn = document.getElementById(uniqueId + '-btn');

    if (full.style.display === 'none') {
        full.style.display = 'block';
        btn.textContent = '▲ Hide Result';
    } else {
        full.style.display = 'none';
        btn.textContent = btn.textContent.replace('▲ Hide Result', '▼ Show Result');
    }
}

// Show Decision Required approval modal
function showApprovalModal(approval) {
    pendingApproval = approval;

    const isDangerous = approval.category === 'dangerous';
    const icon = isDangerous ? '⚠️' : '🔧';
    const title = isDangerous ? 'Dangerous Operation' : 'Decision Required';
    const warnings = Array.isArray(approval.warnings) ? approval.warnings : [];
    const warningsHtml = warnings.length > 0
        ? `<div class="approval-warnings" style="margin:12px 0 0 0;padding:10px 12px;border-radius:8px;background:rgba(255,184,0,0.12);border:1px solid rgba(255,184,0,0.28);font-size:12px;color:var(--text-secondary);">
                ${warnings.map(w => `<div style="margin-top:4px;">• ${escapeHtml(w)}</div>`).join('')}
           </div>`
        : '';

    const modal = document.createElement('div');
    modal.className = 'approval-modal';
    modal.id = 'approval-modal';
    modal.innerHTML = `
                <div class="approval-box ${isDangerous ? 'dangerous' : ''}">
                    <div class="approval-header">
                        <span class="approval-icon">${icon}</span>
                        <span class="approval-title">${title}</span>
                    </div>
                    <div class="approval-category ${approval.category}">${approval.category}</div>
                    <p>The AI wants to execute the following command:</p>
                    <div class="approval-command">${escapeHtml(approval.command)}</div>
                    <p style="font-size: 12px; color: var(--text-secondary);">
                        Tool: <strong>${approval.tool_name}</strong>
                    </p>
                    ${warningsHtml}
                    <div class="approval-buttons">
                        <button class="btn-reject" onclick="respondToApproval(false)">
                            ✕ Reject
                        </button>
                        <button class="btn-approve" onclick="respondToApproval(true)">
                            ✓ Approve
                        </button>
                    </div>
                </div>
            `;

    document.body.appendChild(modal);

    // Add keyboard handlers
    document.addEventListener('keydown', handleApprovalKeypress);
}

function showApprovalBlocked(approval) {
    const reason = approval.reason || 'Blocked by tool approval policy';
    const warnings = Array.isArray(approval.warnings) && approval.warnings.length > 0
        ? '<br>' + approval.warnings.map(w => escapeHtml(w)).join('<br>')
        : '';
    showToast(`Blocked: ${reason}`, 'error');

    const container = document.getElementById('ai-messages');
    const div = document.createElement('div');
    div.className = 'message assistant';
    div.innerHTML = `
        <div class="message-content" style="color: var(--accent-red);">
            <strong>Command blocked by policy</strong><br>
            <code>${escapeHtml(approval.command || '')}</code><br>
            ${escapeHtml(reason)}${warnings}
        </div>
    `;
    container.appendChild(div);
    aiScrollToBottom();
}

function handleApprovalKeypress(e) {
    if (!pendingApproval) return;

    if (e.key === 'Enter' || e.key === 'y' || e.key === 'Y') {
        respondToApproval(true);
    } else if (e.key === 'Escape' || e.key === 'n' || e.key === 'N') {
        respondToApproval(false);
    }
}

async function respondToApproval(approved) {
    if (!pendingApproval) return;

    const approvalId = pendingApproval.id;
    pendingApproval = null;

    // Remove modal
    const modal = document.getElementById('approval-modal');
    if (modal) {
        modal.remove();
    }

    // Remove keyboard handler
    document.removeEventListener('keydown', handleApprovalKeypress);

    // Send response to server
    try {
        await fetch('/api/tool/approve', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify({ id: approvalId, approved })
        });

        // Add temporary status message to chat (auto-removes after 5 seconds)
        const container = document.getElementById('ai-messages');
        const statusDiv = document.createElement('div');
        statusDiv.className = 'tool-execution';
        statusDiv.style.transition = 'opacity 0.3s ease-out';
        statusDiv.innerHTML = approved
            ? `<span class="tool-name">✓ Approved:</span> Command execution proceeding...`
            : `<span class="tool-name" style="color: var(--accent-red)">✕ Rejected:</span> Command was cancelled by user.`;
        container.appendChild(statusDiv);
        aiScrollToBottom();

        // Auto-remove the status message after 5 seconds
        setTimeout(() => {
            statusDiv.style.opacity = '0';
            setTimeout(() => statusDiv.remove(), 300);
        }, 5000);

    } catch (e) {
        console.error('Failed to send approval response:', e);
    }
}

function addMessage(content, isUser = false) {
    const container = document.getElementById('ai-messages');
    const div = document.createElement('div');
    div.className = `message ${isUser ? 'user' : 'assistant'}`;

    let formatted = content;
    if (!isUser) {
        formatted = content.replace(/```(\w*)\n?([\s\S]*?)```/g, '<pre><code>$2</code></pre>');
        formatted = formatted.replace(/\n/g, '<br>');
    }

    div.innerHTML = `<div class="message-content">${formatted}</div>`;
    container.appendChild(div);
    if (isUser) {
        aiForceScrollToBottom();
    } else {
        aiScrollToBottom();
    }
}

function addLoadingMessage() {
    const container = document.getElementById('ai-messages');
    const div = document.createElement('div');
    div.className = 'message assistant';
    div.id = 'loading-message';
    div.innerHTML = `<div class="message-content"><div class="loading-dots"><span></span><span></span><span></span></div></div>`;
    container.appendChild(div);
    aiScrollToBottom();
}

function removeLoadingMessage() {
    const loading = document.getElementById('loading-message');
    if (loading) loading.remove();
}

// AI messages auto-scroll state
let aiAutoScroll = true;
const AI_SCROLL_STEP = 60; // pixels per arrow key press

function aiScrollToBottom() {
    if (!aiAutoScroll) return;
    const container = document.getElementById('ai-messages');
    if (container) container.scrollTop = container.scrollHeight;
}

function aiForceScrollToBottom() {
    const container = document.getElementById('ai-messages');
    if (container) {
        container.scrollTop = container.scrollHeight;
        aiAutoScroll = true;
    }
}

// Detect manual scroll to toggle auto-scroll
(function initAiScrollListener() {
    const container = document.getElementById('ai-messages');
    if (!container) return;
    container.addEventListener('scroll', () => {
        const atBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 30;
        aiAutoScroll = atBottom;
    });
    // Arrow key scrolling on the messages container
    container.setAttribute('tabindex', '-1');
    container.addEventListener('keydown', (e) => {
        if (e.key === 'ArrowUp') {
            e.preventDefault();
            container.scrollTop -= AI_SCROLL_STEP;
        } else if (e.key === 'ArrowDown') {
            e.preventDefault();
            container.scrollTop += AI_SCROLL_STEP;
        } else if (e.key === 'PageUp') {
            e.preventDefault();
            container.scrollTop -= container.clientHeight;
        } else if (e.key === 'PageDown') {
            e.preventDefault();
            container.scrollTop += container.clientHeight;
        } else if (e.key === 'Home') {
            e.preventDefault();
            container.scrollTop = 0;
            aiAutoScroll = false;
        } else if (e.key === 'End') {
            e.preventDefault();
            container.scrollTop = container.scrollHeight;
            aiAutoScroll = true;
        }
    });
})();

// AI input query history
let aiQueryHistory = [];
let aiHistoryIndex = -1;
let aiCurrentDraft = '';

function loadAIQueryHistory() {
    try {
        const parsed = JSON.parse(localStorage.getItem('k13d_query_history') || '[]');
        aiQueryHistory = Array.isArray(parsed) ? parsed : [];
    } catch (error) {
        aiQueryHistory = [];
    }
}

function aiInputHasSelection(input) {
    return input.selectionStart !== input.selectionEnd;
}

function aiInputCursorOnFirstLine(input) {
    if (!input || input.value.indexOf('\n') === -1) {
        return true;
    }
    const cursor = Math.max(0, input.selectionStart || 0);
    return !input.value.slice(0, cursor).includes('\n');
}

function aiInputCursorOnLastLine(input) {
    if (!input || input.value.indexOf('\n') === -1) {
        return true;
    }
    const cursor = Math.max(0, input.selectionEnd || 0);
    return !input.value.slice(cursor).includes('\n');
}

function setAIInputValueFromHistory(input, value) {
    input.value = value;
    const end = input.value.length;
    if (typeof input.setSelectionRange === 'function') {
        input.setSelectionRange(end, end);
    }
}

function saveQueryToHistory(query) {
    loadAIQueryHistory();
    if (!query.trim()) return;
    // Avoid duplicates at the end
    if (aiQueryHistory.length > 0 && aiQueryHistory[aiQueryHistory.length - 1] === query) return;
    aiQueryHistory.push(query);
    // Keep last 50 entries
    if (aiQueryHistory.length > 50) aiQueryHistory = aiQueryHistory.slice(-50);
    localStorage.setItem('k13d_query_history', JSON.stringify(aiQueryHistory));
}

function clearAiInput() {
    const input = document.getElementById('ai-input');
    input.value = '';
    aiHistoryIndex = -1;
    aiCurrentDraft = '';
    input.focus();
}

function toggleAiExpand() {
    const aiPanel = document.getElementById('ai-panel');
    const btn = document.getElementById('ai-expand-btn');
    const input = document.getElementById('ai-input');
    const inputContainer = document.getElementById('ai-input-container');
    const contextChips = document.getElementById('context-chips');
    const aiActions = inputContainer ? inputContainer.querySelector('.ai-actions') : null;
    const aiHint = inputContainer ? inputContainer.querySelector('.ai-hint') : null;

    aiPanel.classList.toggle('expanded');

    if (aiPanel.classList.contains('expanded')) {
        btn.innerHTML = '&#x2716;'; // X to close
        btn.title = 'Exit fullscreen';

        // Ensure panel children stretch to full width
        aiPanel.style.alignItems = 'stretch';

        // Use setAttribute on style to support !important
        if (inputContainer) {
            inputContainer.setAttribute('style',
                'width: 100% !important;' +
                'max-width: none !important;' +
                'min-width: 0 !important;' +
                'box-sizing: border-box !important;' +
                'padding: 24px 40px !important;' +
                'display: flex !important;' +
                'flex-direction: column !important;'
            );
        }
        if (input) {
            input.setAttribute('style',
                'width: 100% !important;' +
                'font-size: 16px !important;' +
                'box-sizing: border-box !important;' +
                'height: auto !important;' +
                'resize: none !important;'
            );
        }
        if (contextChips) {
            contextChips.setAttribute('style',
                'width: 100% !important;' +
                'box-sizing: border-box !important;'
            );
        }
    } else {
        btn.innerHTML = '&#x26F6;'; // expand icon
        btn.title = 'Expand AI panel';
        input.rows = 2;

        // Clear all inline styles
        aiPanel.style.alignItems = '';
        if (inputContainer) inputContainer.removeAttribute('style');
        if (input) input.removeAttribute('style');
        if (contextChips) contextChips.removeAttribute('style');
    }
    input.focus();
}

document.getElementById('ai-input').addEventListener('keydown', (e) => {
    if (e.isComposing) {
        return;
    }
    if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
        loadAIQueryHistory();
    }
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
    } else if (e.key === 'Escape') {
        const aiPanel = document.getElementById('ai-panel');
        if (aiPanel.classList.contains('expanded')) {
            toggleAiExpand();
        }
    } else if (e.key === 'ArrowUp' && !e.shiftKey && !e.altKey && !e.ctrlKey && !e.metaKey) {
        const input = e.target;
        if (!aiInputHasSelection(input) && aiQueryHistory.length > 0 && aiInputCursorOnFirstLine(input)) {
            e.preventDefault();
            if (aiHistoryIndex === -1) {
                aiCurrentDraft = input.value;
                aiHistoryIndex = aiQueryHistory.length - 1;
            } else if (aiHistoryIndex > 0) {
                aiHistoryIndex--;
            }
            setAIInputValueFromHistory(input, aiQueryHistory[aiHistoryIndex]);
        }
    } else if (e.key === 'ArrowDown' && !e.shiftKey && !e.altKey && !e.ctrlKey && !e.metaKey) {
        const input = e.target;
        if (!aiInputHasSelection(input) && aiHistoryIndex !== -1 && aiInputCursorOnLastLine(input)) {
            e.preventDefault();
            if (aiHistoryIndex < aiQueryHistory.length - 1) {
                aiHistoryIndex++;
                setAIInputValueFromHistory(input, aiQueryHistory[aiHistoryIndex]);
            } else {
                aiHistoryIndex = -1;
                setAIInputValueFromHistory(input, aiCurrentDraft);
            }
        }
    }
});

loadAIQueryHistory();

// Resizable panel
function setupResizeHandle() {
    const handle = document.getElementById('resize-handle');
    const aiPanel = document.getElementById('ai-panel');
    let isResizing = false;

    handle.addEventListener('mousedown', (e) => {
        isResizing = true;
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    });

    document.addEventListener('mousemove', (e) => {
        if (!isResizing) return;
        const containerWidth = document.querySelector('.content-wrapper').offsetWidth;
        const newWidth = containerWidth - e.clientX + document.querySelector('.sidebar').offsetWidth;
        if (newWidth >= 280 && newWidth <= containerWidth * 0.75) {
            aiPanel.style.width = newWidth + 'px';
        }
    });

    document.addEventListener('mouseup', () => {
        isResizing = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
    });
}

// Health check
function setupHealthCheck() {
    setInterval(async () => {
        try {
            const resp = await fetch('/api/health');
            const data = await resp.json();
            const dot = document.getElementById('health-dot');
            const status = document.getElementById('health-status');

            if (data.status === 'ok' && data.k8s_ready) {
                dot.className = 'health-dot ok';
                status.textContent = 'Connected';
            } else {
                dot.className = 'health-dot warning';
                status.textContent = 'Degraded';
            }
        } catch (e) {
            document.getElementById('health-dot').className = 'health-dot error';
            document.getElementById('health-status').textContent = 'Disconnected';
        }
    }, 10000);
}

function handleGlobalSearchInput(event) {
    const query = event.target.value.trim();

    // Clear previous timeout
    if (searchTimeout) {
        clearTimeout(searchTimeout);
    }

    if (query.length < 2) {
        hideSearchResults();
        return;
    }

    // Debounce search
    searchTimeout = setTimeout(() => {
        performGlobalSearch(query);
    }, 300);
}

function handleGlobalSearchKeydown(event) {
    const resultsDiv = document.getElementById('search-results');
    const items = resultsDiv.querySelectorAll('.search-result-item');

    switch (event.key) {
        case 'ArrowDown':
            event.preventDefault();
            searchSelectedIndex = Math.min(searchSelectedIndex + 1, items.length - 1);
            updateSearchSelection(items);
            break;
        case 'ArrowUp':
            event.preventDefault();
            searchSelectedIndex = Math.max(searchSelectedIndex - 1, 0);
            updateSearchSelection(items);
            break;
        case 'Enter':
            event.preventDefault();
            if (searchSelectedIndex >= 0 && searchResults[searchSelectedIndex]) {
                navigateToSearchResult(searchResults[searchSelectedIndex]);
            }
            break;
        case 'Escape':
            hideSearchResults();
            event.target.blur();
            break;
    }
}

function updateSearchSelection(items) {
    items.forEach((item, idx) => {
        if (idx === searchSelectedIndex) {
            item.style.background = 'var(--bg-tertiary)';
            item.scrollIntoView({ block: 'nearest' });
        } else {
            item.style.background = '';
        }
    });
}

async function performGlobalSearch(query) {
    const resultsDiv = document.getElementById('search-results');
    resultsDiv.innerHTML = '<div class="search-loading">Searching...</div>';
    resultsDiv.style.display = 'block';

    try {
        const response = await fetch(`/api/search?q=${encodeURIComponent(query)}&namespace=${currentNamespace || ''}`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) throw new Error('Search failed');

        const data = await response.json();
        searchResults = data.results || [];
        searchSelectedIndex = -1;

        if (searchResults.length === 0) {
            resultsDiv.innerHTML = '<div class="search-no-results">No results found</div>';
        } else {
            resultsDiv.innerHTML = searchResults.map((result, idx) => `
                        <div class="search-result-item" onclick="navigateToSearchResult(searchResults[${idx}])">
                            <span class="search-result-kind ${result.kind.toLowerCase()}">${result.kind}</span>
                            <div class="search-result-info">
                                <div class="search-result-name">${escapeHtml(result.name)}</div>
                                ${result.namespace ? `<div class="search-result-namespace">${escapeHtml(result.namespace)}</div>` : ''}
                            </div>
                            ${result.status ? `<span class="search-result-status ${result.status.toLowerCase().replace(/\s/g, '')}">${result.status}</span>` : ''}
                        </div>
                    `).join('');
        }
    } catch (e) {
        resultsDiv.innerHTML = '<div class="search-no-results">Search error</div>';
        console.error('Search error:', e);
    }
}

function navigateToSearchResult(result) {
    hideSearchResults();
    document.getElementById('global-search').value = '';

    // Map kind to resource type
    const kindToResource = {
        'Pod': 'pods',
        'Deployment': 'deployments',
        'Service': 'services',
        'StatefulSet': 'statefulsets',
        'DaemonSet': 'daemonsets',
        'ConfigMap': 'configmaps',
        'Secret': 'secrets',
        'Ingress': 'ingresses',
        'Node': 'nodes',
        'Namespace': 'namespaces',
        'ReplicaSet': 'replicasets',
        'Job': 'jobs',
        'CronJob': 'cronjobs'
    };

    const resourceType = kindToResource[result.kind] || result.kind.toLowerCase() + 's';

    // Switch namespace if needed
    if (result.namespace && result.namespace !== currentNamespace) {
        currentNamespace = result.namespace;
        document.getElementById('namespace-select').value = result.namespace;
    }

    // Switch to the resource type
    switchResource(resourceType);

    // Set filter to highlight the specific resource
    setTimeout(() => {
        document.getElementById('filter-input').value = result.name;
        currentFilter = result.name.toLowerCase();
        applyFilterAndSort();
    }, 500);
}

function showSearchResults() {
    const query = document.getElementById('global-search').value.trim();
    if (query.length >= 2 && searchResults.length > 0) {
        document.getElementById('search-results').style.display = 'block';
    }
}

function hideSearchResults() {
    document.getElementById('search-results').style.display = 'none';
    searchSelectedIndex = -1;
}

// Hide search results when clicking outside
document.addEventListener('click', (e) => {
    if (!e.target.closest('.search-container')) {
        hideSearchResults();
    }
});

// Filter functionality
let currentFilter = '';
let cachedData = [];

function handleFilter(event) {
    currentFilter = event.target.value.trim().toLowerCase();
    // Use the new filtering system that works with sorting/pagination
    currentPage = 1;
    applyFilterAndSort();
}

// Legacy filterTable for compatibility (now uses new system)
function filterTable(query) {
    document.getElementById('filter-input').value = query;
    currentPage = 1;
    applyFilterAndSort();
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    // Check if command bar is open
    const commandBarOpen = document.getElementById('command-bar-overlay').classList.contains('active');
    const yamlEditorOpen = document.getElementById('yaml-editor-modal').classList.contains('active');

    // Handle command bar input separately
    if (commandBarOpen) {
        handleCommandBarKeydown(e);
        return;
    }

    // Handle YAML editor shortcuts
    if (yamlEditorOpen) {
        handleYamlEditorKeydown(e);
        return;
    }

    // Ignore if in input/textarea (except for specific shortcuts)
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
        if (e.key === 'Escape') {
            e.target.blur();
        }
        return;
    }

    // Check for modifiers
    const isMeta = e.metaKey || e.ctrlKey;
    const isAlt = e.altKey;

    // Alt+number for namespace switching
    if (isAlt && e.key >= '0' && e.key <= '9') {
        e.preventDefault();
        switchToRecentNamespace(parseInt(e.key));
        return;
    }

    switch (e.key.toLowerCase()) {
        case 'k':
            if (isMeta) {
                e.preventDefault();
                document.getElementById('global-search').focus();
            }
            break;
        case 'f':
            if (isMeta) {
                e.preventDefault();
                toggleColumnFilters();
            }
            break;
        case '/':
            e.preventDefault();
            document.getElementById('filter-input').focus();
            break;
        case ':':
            e.preventDefault();
            openCommandBar();
            break;
        case 'r':
            e.preventDefault();
            refreshData();
            break;
        case 'a':
            e.preventDefault();
            toggleAIPanel();
            break;
        case 'b':
            e.preventDefault();
            toggleSidebar();
            break;
        case 'd':
            e.preventDefault();
            toggleDebugMode();
            break;
        case 'e':
            e.preventDefault();
            openYamlEditor();
            break;
        case 'n':
            e.preventDefault();
            showNamespaceIndicator();
            break;
        case '1':
            e.preventDefault();
            switchResource('pods');
            break;
        case '2':
            e.preventDefault();
            switchResource('deployments');
            break;
        case '3':
            e.preventDefault();
            switchResource('services');
            break;
        case '4':
            e.preventDefault();
            switchResource('nodes');
            break;
        case 's':
            e.preventDefault();
            showSettings();
            break;
        case '?':
            e.preventDefault();
            showShortcuts();
            break;
        case 'escape':
            closeAllModals();
            hideNamespaceIndicator();
            break;
    }
});

function toggleAIPanel() {
    const panel = document.getElementById('ai-panel');
    const handle = document.getElementById('resize-handle');
    const btn = document.getElementById('ai-toggle-btn');
    const isMobile = window.innerWidth <= 768;

    if (isMobile) {
        // Mobile: use class toggle (CSS transform-based)
        const isOpen = panel.classList.contains('mobile-open');
        panel.classList.toggle('mobile-open', !isOpen);
        if (btn) btn.classList.toggle('active', !isOpen);
        localStorage.setItem('k13d_ai_panel', !isOpen ? 'open' : 'closed');
    } else {
        // Desktop: use display toggle
        const isHidden = panel.style.display === 'none';
        panel.style.display = isHidden ? 'flex' : 'none';
        handle.style.display = isHidden ? 'block' : 'none';
        if (btn) btn.classList.toggle('active', isHidden);
        localStorage.setItem('k13d_ai_panel', isHidden ? 'open' : 'closed');
    }
}

// Restore AI panel state on load
(function initAIPanelState() {
    const saved = localStorage.getItem('k13d_ai_panel');
    if (saved === 'closed') {
        const panel = document.getElementById('ai-panel');
        const handle = document.getElementById('resize-handle');
        const btn = document.getElementById('ai-toggle-btn');
        if (panel) panel.style.display = 'none';
        if (handle) handle.style.display = 'none';
        if (btn) btn.classList.remove('active');
    } else {
        const btn = document.getElementById('ai-toggle-btn');
        if (btn) btn.classList.add('active');
    }
})();

// Add message to DOM (without saving)
function addMessageToDOM(content, isUser, scroll = true) {
    const container = document.getElementById('ai-messages');
    const div = document.createElement('div');
    div.className = `message ${isUser ? 'user' : 'assistant'}`;

    let formattedContent = content;
    if (!isUser) {
        formattedContent = formatResourceLinks(marked.parse(content));
    }

    div.innerHTML = `<div class="message-content">${formattedContent}</div>`;
    container.appendChild(div);

    if (scroll) {
        if (isUser) {
            aiForceScrollToBottom();
        } else {
            aiScrollToBottom();
        }
    }
}

// Shortcuts modal
function showShortcuts() {
    document.getElementById('shortcuts-modal').classList.add('active');
}

function closeShortcuts() {
    document.getElementById('shortcuts-modal').classList.remove('active');
}

// Resource detail modal
let selectedResource = null;

// Generate resource-specific overview HTML
function generateResourceOverview(resource, item) {
    switch (resource) {
        case 'pods':
            return generatePodOverview(item);
        case 'deployments':
            return generateDeploymentOverview(item);
        case 'services':
            return generateServiceOverview(item);
        case 'statefulsets':
            return generateStatefulSetOverview(item);
        case 'daemonsets':
            return generateDaemonSetOverview(item);
        case 'nodes':
            return generateNodeOverview(item);
        case 'configmaps':
            return generateConfigMapOverview(item);
        case 'secrets':
            return generateSecretOverview(item);
        case 'ingresses':
            return generateIngressOverview(item);
        case 'jobs':
            return generateJobOverview(item);
        case 'cronjobs':
            return generateCronJobOverview(item);
        case 'pvcs':
            return generatePVCOverview(item);
        case 'pvs':
            return generatePVOverview(item);
        default:
            return generateDefaultOverview(item);
    }
}

// Default overview (key-value pairs)
function generateDefaultOverview(item) {
    const html = Object.entries(item).map(([key, value]) =>
        `<div class="property-label">${key}</div><div class="property-value">${escapeHtml(String(value || '-'))}</div>`
    ).join('');
    return `<div class="property-grid">${html}</div>`;
}

// Pod Overview
function generatePodOverview(item) {
    const statusColor = item.status === 'Running' ? 'var(--accent-green)' :
        item.status === 'Pending' ? 'var(--accent-yellow)' :
            item.status === 'Failed' || item.status === 'Error' ? 'var(--accent-red)' : 'var(--text-secondary)';
    const restarts = parseInt(item.restarts) || 0;
    const restartColor = restarts > 5 ? 'var(--accent-red)' : restarts > 0 ? 'var(--accent-yellow)' : 'var(--accent-green)';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(item.status)}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">📦 Container Status</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Ready</span>
                                <span class="stat-value" style="color: var(--accent-green);">${escapeHtml(item.ready || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Restarts</span>
                                <span class="stat-value" style="color: ${restartColor};">${restarts}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🖥️ Node & Network</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Node</span>
                                <span class="stat-value">${escapeHtml(item.node || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Pod IP</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.ip || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="overview-actions">
                    <button class="btn btn-secondary" onclick="openLogViewerDirect('${escapeHtml(item.name)}', '${escapeHtml(item.namespace || '')}')">📋 View Logs</button>
                </div>
            `;
}

// Deployment Overview
function generateDeploymentOverview(item) {
    const ready = item.ready || '0/0';
    const [readyCount, totalCount] = ready.split('/').map(n => parseInt(n) || 0);
    const healthPercent = totalCount > 0 ? Math.round((readyCount / totalCount) * 100) : 0;
    const healthColor = healthPercent === 100 ? 'var(--accent-green)' : healthPercent >= 50 ? 'var(--accent-yellow)' : 'var(--accent-red)';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${healthColor}20; color: ${healthColor}; border: 1px solid ${healthColor}40;">
                        <span class="status-dot" style="background: ${healthColor};"></span>
                        ${healthPercent === 100 ? 'Healthy' : healthPercent > 0 ? 'Degraded' : 'Unavailable'}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">📊 Replicas</div>
                        <div class="overview-card-content">
                            <div class="overview-progress">
                                <div class="progress-bar" style="width: ${healthPercent}%; background: ${healthColor};"></div>
                            </div>
                            <div class="overview-stat" style="margin-top: 8px;">
                                <span class="stat-label">Ready</span>
                                <span class="stat-value" style="color: ${healthColor};">${ready}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Up-to-date</span>
                                <span class="stat-value">${escapeHtml(item.upToDate || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Available</span>
                                <span class="stat-value">${escapeHtml(item.available || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🐳 Container Image</div>
                        <div class="overview-card-content">
                            <div class="image-tag" title="${escapeHtml(item.image || '-')}">${escapeHtml(item.image || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// Service Overview
function generateServiceOverview(item) {
    const typeColors = {
        'ClusterIP': 'var(--accent-blue)',
        'NodePort': 'var(--accent-purple)',
        'LoadBalancer': 'var(--accent-green)',
        'ExternalName': 'var(--accent-yellow)'
    };
    const typeColor = typeColors[item.type] || 'var(--text-secondary)';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${typeColor}20; color: ${typeColor}; border: 1px solid ${typeColor}40;">
                        ${escapeHtml(item.type || 'Unknown')}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">🌐 Network</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Cluster IP</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.clusterIP || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">External IP</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.externalIP || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🔌 Ports</div>
                        <div class="overview-card-content">
                            <div class="ports-list">${escapeHtml(item.ports || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// StatefulSet Overview
function generateStatefulSetOverview(item) {
    const ready = item.ready || '0/0';
    const [readyCount, totalCount] = ready.split('/').map(n => parseInt(n) || 0);
    const healthPercent = totalCount > 0 ? Math.round((readyCount / totalCount) * 100) : 0;
    const healthColor = healthPercent === 100 ? 'var(--accent-green)' : healthPercent >= 50 ? 'var(--accent-yellow)' : 'var(--accent-red)';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${healthColor}20; color: ${healthColor}; border: 1px solid ${healthColor}40;">
                        <span class="status-dot" style="background: ${healthColor};"></span>
                        ${healthPercent === 100 ? 'Healthy' : healthPercent > 0 ? 'Degraded' : 'Unavailable'}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">📊 Replicas</div>
                        <div class="overview-card-content">
                            <div class="overview-progress">
                                <div class="progress-bar" style="width: ${healthPercent}%; background: ${healthColor};"></div>
                            </div>
                            <div class="overview-stat" style="margin-top: 8px;">
                                <span class="stat-label">Ready</span>
                                <span class="stat-value" style="color: ${healthColor};">${ready}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🐳 Container Image</div>
                        <div class="overview-card-content">
                            <div class="image-tag" title="${escapeHtml(item.image || '-')}">${escapeHtml(item.image || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// DaemonSet Overview
function generateDaemonSetOverview(item) {
    const ready = parseInt(item.ready) || 0;
    const desired = parseInt(item.desired) || 0;
    const healthPercent = desired > 0 ? Math.round((ready / desired) * 100) : 0;
    const healthColor = healthPercent === 100 ? 'var(--accent-green)' : healthPercent >= 50 ? 'var(--accent-yellow)' : 'var(--accent-red)';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${healthColor}20; color: ${healthColor}; border: 1px solid ${healthColor}40;">
                        <span class="status-dot" style="background: ${healthColor};"></span>
                        ${healthPercent === 100 ? 'Healthy' : healthPercent > 0 ? 'Degraded' : 'Unavailable'}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">📊 Node Coverage</div>
                        <div class="overview-card-content">
                            <div class="overview-progress">
                                <div class="progress-bar" style="width: ${healthPercent}%; background: ${healthColor};"></div>
                            </div>
                            <div class="overview-stat" style="margin-top: 8px;">
                                <span class="stat-label">Desired</span>
                                <span class="stat-value">${desired}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Current</span>
                                <span class="stat-value">${escapeHtml(item.current || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Ready</span>
                                <span class="stat-value" style="color: ${healthColor};">${ready}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Available</span>
                                <span class="stat-value">${escapeHtml(item.available || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🐳 Container Image</div>
                        <div class="overview-card-content">
                            <div class="image-tag" title="${escapeHtml(item.image || '-')}">${escapeHtml(item.image || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// Node Overview
function generateNodeOverview(item) {
    const statusColor = item.status === 'Ready' ? 'var(--accent-green)' : 'var(--accent-red)';
    const roles = item.roles || '-';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(item.status)}
                    </div>
                    <div class="overview-roles">
                        ${roles.split(',').map(r => `<span class="role-badge">${escapeHtml(r.trim())}</span>`).join('')}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">💻 System Info</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Version</span>
                                <span class="stat-value">${escapeHtml(item.version || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">OS</span>
                                <span class="stat-value">${escapeHtml(item.os || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Arch</span>
                                <span class="stat-value">${escapeHtml(item.arch || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📦 Capacity</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">CPU</span>
                                <span class="stat-value">${escapeHtml(item.cpu || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Memory</span>
                                <span class="stat-value">${escapeHtml(item.memory || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Pods</span>
                                <span class="stat-value">${escapeHtml(item.pods || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🌐 Network</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Internal IP</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.internalIP || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// ConfigMap Overview
function generateConfigMapOverview(item) {
    return `
                <div class="overview-cards">
                    <div class="overview-card" style="grid-column: span 2;">
                        <div class="overview-card-title">📝 Data</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Keys</span>
                                <span class="stat-value">${escapeHtml(item.data || '0')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// Secret Overview
function generateSecretOverview(item) {
    const typeColors = {
        'Opaque': 'var(--accent-blue)',
        'kubernetes.io/service-account-token': 'var(--accent-purple)',
        'kubernetes.io/dockerconfigjson': 'var(--accent-green)',
        'kubernetes.io/tls': 'var(--accent-yellow)'
    };
    const typeColor = typeColors[item.type] || 'var(--text-secondary)';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${typeColor}20; color: ${typeColor}; border: 1px solid ${typeColor}40;">
                        🔒 ${escapeHtml(item.type || 'Unknown')}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card" style="grid-column: span 2;">
                        <div class="overview-card-title">🔐 Data</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Keys</span>
                                <span class="stat-value">${escapeHtml(item.data || '0')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// Ingress Overview
function generateIngressOverview(item) {
    return `
                <div class="overview-cards">
                    <div class="overview-card" style="grid-column: span 2;">
                        <div class="overview-card-title">🌐 Routing</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Class</span>
                                <span class="stat-value">${escapeHtml(item.class || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Hosts</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.hosts || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Address</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.address || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// Job Overview
function generateJobOverview(item) {
    const statusColor = item.status === 'Complete' ? 'var(--accent-green)' :
        item.status === 'Running' ? 'var(--accent-blue)' :
            item.status === 'Failed' ? 'var(--accent-red)' : 'var(--text-secondary)';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(item.status || 'Unknown')}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">📊 Completion</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Completions</span>
                                <span class="stat-value">${escapeHtml(item.completions || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Duration</span>
                                <span class="stat-value">${escapeHtml(item.duration || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🐳 Container Image</div>
                        <div class="overview-card-content">
                            <div class="image-tag" title="${escapeHtml(item.image || '-')}">${escapeHtml(item.image || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// CronJob Overview
function generateCronJobOverview(item) {
    const suspendColor = item.suspend === 'True' ? 'var(--accent-yellow)' : 'var(--accent-green)';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${suspendColor}20; color: ${suspendColor}; border: 1px solid ${suspendColor}40;">
                        ${item.suspend === 'True' ? '⏸️ Suspended' : '▶️ Active'}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">⏰ Schedule</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Schedule</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.schedule || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Last Schedule</span>
                                <span class="stat-value">${escapeHtml(item.lastSchedule || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Active Jobs</span>
                                <span class="stat-value">${escapeHtml(item.active || '0')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🐳 Container Image</div>
                        <div class="overview-card-content">
                            <div class="image-tag" title="${escapeHtml(item.image || '-')}">${escapeHtml(item.image || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// PVC Overview
function generatePVCOverview(item) {
    const statusColor = item.status === 'Bound' ? 'var(--accent-green)' :
        item.status === 'Pending' ? 'var(--accent-yellow)' : 'var(--accent-red)';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(item.status)}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">💾 Storage</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Capacity</span>
                                <span class="stat-value">${escapeHtml(item.capacity || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Access Modes</span>
                                <span class="stat-value">${escapeHtml(item.accessModes || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Storage Class</span>
                                <span class="stat-value">${escapeHtml(item.storageClass || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🔗 Volume</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Volume</span>
                                <span class="stat-value" style="font-family: monospace; font-size: 11px;">${escapeHtml(item.volume || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

// PV Overview
function generatePVOverview(item) {
    const statusColor = item.status === 'Available' ? 'var(--accent-green)' :
        item.status === 'Bound' ? 'var(--accent-blue)' :
            item.status === 'Released' ? 'var(--accent-yellow)' : 'var(--accent-red)';

    return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(item.status)}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">💾 Storage</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Capacity</span>
                                <span class="stat-value">${escapeHtml(item.capacity || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Access Modes</span>
                                <span class="stat-value">${escapeHtml(item.accessModes || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Reclaim Policy</span>
                                <span class="stat-value">${escapeHtml(item.reclaimPolicy || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🔗 Claim</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Claim</span>
                                <span class="stat-value" style="font-family: monospace; font-size: 11px;">${escapeHtml(item.claim || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Storage Class</span>
                                <span class="stat-value">${escapeHtml(item.storageClass || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
}

function showResourceDetail(item) {
    selectedResource = item;
    document.getElementById('detail-title').textContent = `${currentResource.slice(0, -1)}: ${item.name}`;

    // Overview tab - use resource-specific generator
    const overviewHtml = generateResourceOverview(currentResource, item);
    document.getElementById('detail-overview').innerHTML = overviewHtml;

    // YAML tab - will be loaded on demand
    document.getElementById('detail-yaml').innerHTML = `<div class="yaml-viewer" style="color: var(--text-secondary);">Click the YAML tab to load...</div>`;
    document.getElementById('detail-yaml').dataset.loaded = 'false';

    // Events tab - will be loaded on demand
    document.getElementById('detail-events').innerHTML = '<p style="color: var(--text-secondary);">Click the Events tab to load...</p>';
    document.getElementById('detail-events').dataset.loaded = 'false';

    // Related Pods tab - only for Services, Deployments, StatefulSets, DaemonSets, ReplicaSets
    const podsTab = document.getElementById('detail-pods-tab');
    const podsContent = document.getElementById('detail-pods');
    const workloadResources = ['services', 'deployments', 'statefulsets', 'daemonsets', 'replicasets'];
    if (workloadResources.includes(currentResource)) {
        podsTab.style.display = 'inline-block';
        podsContent.innerHTML = '<p style="color: var(--text-secondary);">Click the Related Pods tab to load...</p>';
        podsContent.dataset.loaded = 'false';
    } else {
        podsTab.style.display = 'none';
    }

    // Referenced By tab - only for Secrets and ConfigMaps
    const refsTab = document.getElementById('detail-refs-tab');
    const refsContent = document.getElementById('detail-refs');
    if (['secrets', 'configmaps'].includes(currentResource)) {
        refsTab.style.display = 'inline-block';
        refsContent.innerHTML = '<p style="color: var(--text-secondary);">Click the Referenced By tab to load...</p>';
        refsContent.dataset.loaded = 'false';
    } else {
        refsTab.style.display = 'none';
    }

    document.getElementById('detail-modal').classList.add('active');
    switchDetailTab('overview');
}

async function switchDetailTab(tab) {
    document.querySelectorAll('.detail-tab').forEach(t => t.classList.remove('active'));
    document.querySelector(`.detail-tab[onclick*="${tab}"]`).classList.add('active');

    document.getElementById('detail-overview').style.display = tab === 'overview' ? 'block' : 'none';
    document.getElementById('detail-yaml').style.display = tab === 'yaml' ? 'block' : 'none';
    document.getElementById('detail-events').style.display = tab === 'events' ? 'block' : 'none';
    document.getElementById('detail-pods').style.display = tab === 'pods' ? 'block' : 'none';
    document.getElementById('detail-refs').style.display = tab === 'refs' ? 'block' : 'none';

    // Load YAML on demand
    if (tab === 'yaml' && selectedResource) {
        const yamlEl = document.getElementById('detail-yaml');
        if (yamlEl.dataset.loaded !== 'true') {
            yamlEl.innerHTML = `<div class="yaml-viewer" style="color: var(--text-secondary);">Loading YAML...</div>`;
            try {
                let url;
                if (selectedResource._isCR) {
                    // Custom Resource: use CRD API
                    const crdName = selectedResource._crdName;
                    const ns = selectedResource.namespace ? `&namespace=${encodeURIComponent(selectedResource.namespace)}` : '';
                    url = `/api/crd/${crdName}/instances/${encodeURIComponent(selectedResource.name)}?format=yaml${ns}`;
                } else {
                    // Built-in resource: use k8s API
                    const ns = selectedResource.namespace || '';
                    url = `/api/k8s/${currentResource}?name=${encodeURIComponent(selectedResource.name)}&namespace=${encodeURIComponent(ns)}&format=yaml`;
                }
                const response = await fetchWithAuth(url);
                if (!response.ok) {
                    throw new Error(await response.text());
                }
                const yaml = await response.text();
                yamlEl.innerHTML = `<pre class="yaml-viewer">${escapeHtml(yaml)}</pre>`;
                yamlEl.dataset.loaded = 'true';
            } catch (error) {
                yamlEl.innerHTML = `<div class="yaml-viewer" style="color: var(--accent-red);">Error loading YAML: ${escapeHtml(error.message)}</div>`;
            }
        }
    }

    // Load Events on demand
    if (tab === 'events' && selectedResource) {
        const eventsEl = document.getElementById('detail-events');
        if (eventsEl.dataset.loaded !== 'true') {
            eventsEl.innerHTML = '<p style="color: var(--text-secondary);">Loading events...</p>';
            try {
                const ns = selectedResource.namespace || '';
                const url = `/api/k8s/events?namespace=${encodeURIComponent(ns)}`;
                const response = await fetchWithAuth(url);
                if (!response.ok) {
                    throw new Error(await response.text());
                }
                const data = await response.json();
                const relatedEvents = getRelatedEvents(data.items || [], currentResource, selectedResource);

                if (relatedEvents.length === 0) {
                    eventsEl.innerHTML = '<p style="color: var(--text-secondary);">No events found for this resource.</p>';
                } else {
                    const eventsHtml = relatedEvents.map(e => `
                                <div class="event-item" style="padding: 8px; margin-bottom: 8px; border-left: 3px solid ${e.type === 'Warning' ? 'var(--accent-yellow)' : 'var(--accent-green)'}; background: var(--bg-secondary);">
                                    <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
                                        <span style="font-weight: 500; color: ${e.type === 'Warning' ? 'var(--accent-yellow)' : 'var(--accent-green)'}">${escapeHtml(e.reason || 'Unknown')}</span>
                                        <span style="color: var(--text-secondary); font-size: 12px;">${escapeHtml(e.lastSeen || '')}</span>
                                    </div>
                                    <div style="color: var(--text-primary); font-size: 13px;">${escapeHtml(e.message || '')}</div>
                                    ${e.count > 1 ? `<div style="color: var(--text-secondary); font-size: 11px; margin-top: 4px;">Count: ${e.count}</div>` : ''}
                                </div>
                            `).join('');
                    eventsEl.innerHTML = eventsHtml;
                }
                eventsEl.dataset.loaded = 'true';
            } catch (error) {
                eventsEl.innerHTML = `<p style="color: var(--accent-red);">Error loading events: ${escapeHtml(error.message)}</p>`;
            }
        }
    }

    // Load Related Pods on demand (for Services, Deployments, etc.)
    if (tab === 'pods' && selectedResource) {
        const podsEl = document.getElementById('detail-pods');
        if (podsEl.dataset.loaded !== 'true') {
            podsEl.innerHTML = '<p style="color: var(--text-secondary);">Loading related pods...</p>';
            try {
                const ns = selectedResource.namespace || '';
                let labelSelector = resolveRelatedPodsSelector(currentResource, selectedResource, '');
                if (!labelSelector) {
                    const yamlUrl = `/api/k8s/${currentResource}?name=${encodeURIComponent(selectedResource.name)}&namespace=${encodeURIComponent(ns)}&format=yaml`;
                    const yamlResp = await fetchWithAuth(yamlUrl);
                    if (!yamlResp.ok) {
                        throw new Error('Failed to fetch resource details');
                    }
                    const yamlText = await yamlResp.text();
                    labelSelector = resolveRelatedPodsSelector(currentResource, selectedResource, yamlText);
                }

                if (!labelSelector) {
                    podsEl.innerHTML = '<p style="color: var(--text-secondary);">No selector found for this resource.</p>';
                    podsEl.dataset.loaded = 'true';
                    return;
                }

                // Fetch pods with the label selector
                const podsUrl = `/api/k8s/pods?namespace=${encodeURIComponent(ns)}&labelSelector=${encodeURIComponent(labelSelector)}`;
                const podsResp = await fetchWithAuth(podsUrl);
                const podsData = await podsResp.json();

                if (!podsData.items || podsData.items.length === 0) {
                    podsEl.innerHTML = `
                                <p style="color: var(--text-secondary);">No pods found matching selector:</p>
                                <code style="display: block; padding: 8px; background: var(--bg-secondary); border-radius: 4px; font-size: 12px; margin-top: 8px;">${escapeHtml(labelSelector)}</code>
                            `;
                    podsEl.dataset.loaded = 'true';
                    return;
                }

                // Render pods table
                let podsHtml = `
                            <div style="margin-bottom: 12px;">
                                <span style="color: var(--text-secondary); font-size: 12px;">Selector: </span>
                                <code style="padding: 2px 6px; background: var(--bg-secondary); border-radius: 3px; font-size: 11px;">${escapeHtml(labelSelector)}</code>
                                <span style="color: var(--text-secondary); font-size: 12px; margin-left: 12px;">${podsData.items.length} pod(s)</span>
                            </div>
                            <table class="data-table" style="font-size: 12px;">
                                <thead>
                                    <tr>
                                        <th>NAME</th>
                                        <th>STATUS</th>
                                        <th>READY</th>
                                        <th>RESTARTS</th>
                                        <th>NODE</th>
                                        <th>AGE</th>
                                        <th>LOGS</th>
                                    </tr>
                                </thead>
                                <tbody>
                        `;

                for (const pod of podsData.items) {
                    const statusColor = pod.status === 'Running' ? 'var(--accent-green)' :
                        pod.status === 'Pending' ? 'var(--accent-yellow)' :
                            pod.status === 'Failed' ? 'var(--accent-red)' : 'var(--text-secondary)';

                    podsHtml += `
                                <tr style="cursor: pointer;" onclick="viewPodFromDetail('${escapeHtml(pod.name)}', '${escapeHtml(pod.namespace || '')}')">
                                    <td style="color: var(--accent-blue);">${escapeHtml(pod.name)}</td>
                                    <td><span style="color: ${statusColor};">${escapeHtml(pod.status)}</span></td>
                                    <td>${escapeHtml(pod.ready || '-')}</td>
                                    <td>${escapeHtml(pod.restarts || '0')}</td>
                                    <td style="color: var(--text-secondary);">${escapeHtml(pod.node || '-')}</td>
                                    <td style="color: var(--text-secondary);">${escapeHtml(pod.age || '-')}</td>
                                    <td class="resource-actions" onclick="event.stopPropagation();">
                                        <button class="resource-action-btn" onclick="openLogViewerDirect('${escapeHtml(pod.name)}', '${escapeHtml(pod.namespace || '')}')" title="View Logs">📋</button>
                                    </td>
                                </tr>
                            `;
                }

                podsHtml += '</tbody></table>';
                podsEl.innerHTML = podsHtml;
                podsEl.dataset.loaded = 'true';
            } catch (error) {
                podsEl.innerHTML = `<p style="color: var(--accent-red);">Error loading related pods: ${escapeHtml(error.message)}</p>`;
            }
        }
    }

    // Load Referenced By on demand
    if (tab === 'refs' && selectedResource) {
        const refsEl = document.getElementById('detail-refs');
        if (refsEl.dataset.loaded !== 'true') {
            refsEl.innerHTML = '<p style="color: var(--text-secondary);">Loading references...</p>';
            try {
                const kind = currentResource === 'secrets' ? 'Secret' : 'ConfigMap';
                const ns = selectedResource.namespace || '';
                const params = new URLSearchParams({ kind, name: selectedResource.name, namespace: ns });
                const resp = await fetchWithAuth(`/api/resource/references?${params}`);
                const data = await resp.json();

                if (!data.references || data.references.length === 0) {
                    refsEl.innerHTML = '<p style="color: var(--text-secondary);">No resources reference this ' + kind + '.</p>';
                    refsEl.dataset.loaded = 'true';
                    return;
                }

                let html = `<div style="margin-bottom:8px;font-size:12px;color:var(--text-secondary);">${data.references.length} resource(s) reference this ${kind}</div>`;
                html += `<table class="data-table" style="font-size:12px;">
                            <thead><tr><th>Kind</th><th>Name</th><th>Namespace</th><th>Reference Type</th></tr></thead>
                            <tbody>`;
                for (const ref of data.references) {
                    html += `<tr>
                                <td><span style="padding:2px 6px;border-radius:3px;background:var(--bg-tertiary);font-size:11px;">${escapeHtml(ref.kind)}</span></td>
                                <td style="color:var(--accent-blue);">${escapeHtml(ref.name)}</td>
                                <td style="color:var(--text-secondary);">${escapeHtml(ref.namespace)}</td>
                                <td>${escapeHtml(ref.ref_type)}</td>
                            </tr>`;
                }
                html += '</tbody></table>';
                refsEl.innerHTML = html;
                refsEl.dataset.loaded = 'true';
            } catch (error) {
                refsEl.innerHTML = `<p style="color: var(--accent-red);">Error loading references: ${escapeHtml(error.message)}</p>`;
            }
        }
    }
}

// Helper function to view pod details from the related pods tab
function viewPodFromDetail(podName, namespace) {
    closeDetail();
    // Switch to pods view and find the pod
    switchResource('pods');
    setTimeout(() => {
        // Try to find and highlight the pod in the table
        const rows = document.querySelectorAll('#table-body tr');
        for (const row of rows) {
            const nameCell = row.querySelector('td:first-child');
            if (nameCell && nameCell.textContent.trim() === podName) {
                row.click();
                row.scrollIntoView({ behavior: 'smooth', block: 'center' });
                break;
            }
        }
    }, 500);
}

// Helper function to open log viewer directly (without row context)
function openLogViewerDirect(podName, namespace) {
    openLogViewer(podName, namespace, ['default']);
}

function closeDetail() {
    document.getElementById('detail-modal').classList.remove('active');
    selectedResource = null;
}

function resourceKindForEvents(resource) {
    const kindMap = {
        pods: 'Pod',
        deployments: 'Deployment',
        daemonsets: 'DaemonSet',
        statefulsets: 'StatefulSet',
        replicasets: 'ReplicaSet',
        services: 'Service',
        ingresses: 'Ingress',
        cronjobs: 'CronJob',
        jobs: 'Job',
        configmaps: 'ConfigMap',
        secrets: 'Secret',
        namespaces: 'Namespace',
        nodes: 'Node',
    };
    return kindMap[resource] || '';
}

function leadingSpaceCount(line) {
    const match = line.match(/^\s*/);
    return match ? match[0].length : 0;
}

function extractSelectorFromYAMLBlock(yamlText, blockName) {
    const lines = yamlText.split('\n');

    for (let i = 0; i < lines.length; i++) {
        if (lines[i].trim() !== blockName) {
            continue;
        }

        const blockIndent = leadingSpaceCount(lines[i]);
        const selectors = [];

        for (let j = i + 1; j < lines.length; j++) {
            const rawLine = lines[j];
            const trimmed = rawLine.trim();

            if (!trimmed) {
                continue;
            }

            const indent = leadingSpaceCount(rawLine);
            if (indent <= blockIndent) {
                break;
            }

            const match = trimmed.match(/^([\w.-]+):\s*(.+)$/);
            if (!match || match[2].endsWith(':')) {
                continue;
            }

            selectors.push(`${match[1]}=${match[2].trim().replace(/^['"]|['"]$/g, '')}`);
        }

        if (selectors.length > 0) {
            return selectors.join(',');
        }
    }

    return '';
}

function resolveRelatedPodsSelector(resource, item, yamlText) {
    const directSelector = item && typeof item.selector === 'string' ? item.selector.trim() : '';
    if (directSelector && directSelector !== '*') {
        return directSelector;
    }

    if (!yamlText) {
        return '';
    }

    if (resource === 'services') {
        return extractSelectorFromYAMLBlock(yamlText, 'selector:');
    }

    return extractSelectorFromYAMLBlock(yamlText, 'matchLabels:');
}

function getRelatedEvents(events, resource, item) {
    const resourceName = item?.name || '';
    const resourceNamespace = item?.namespace || '';
    const resourceKind = resourceKindForEvents(resource);

    const directMatches = (events || []).filter((event) => {
        const involved = event.involvedObject || {};
        if (involved.name !== resourceName) {
            return false;
        }
        if (resourceNamespace && involved.namespace && involved.namespace !== resourceNamespace) {
            return false;
        }
        if (!resourceKind || !involved.kind) {
            return true;
        }
        return involved.kind.toLowerCase() === resourceKind.toLowerCase();
    });

    if (directMatches.length > 0) {
        return directMatches;
    }

    return (events || []).filter((event) => {
        const involved = event.involvedObject || {};
        return involved.name === resourceName ||
            (event.message && event.message.includes(resourceName));
    });
}

function analyzeWithAI() {
    if (selectedResource) {
        const msg = `Analyze this ${currentResource.slice(0, -1)}: ${selectedResource.name} in namespace ${selectedResource.namespace || 'N/A'}. Current status: ${selectedResource.status || 'unknown'}`;
        document.getElementById('ai-input').value = msg;
        closeDetail();
        document.getElementById('ai-input').focus();
    }
}

// Override renderTable to include click handlers and cache data
const originalRenderTable = renderTable;
renderTable = function (resource, items) {
    cachedData = items || [];
    originalRenderTable(resource, items);
    addRowClickHandlers();
};

// ==========================================
// Sidebar Toggle
// ==========================================
function toggleSidebar() {
    const sidebar = document.getElementById('sidebar');
    const hamburger = document.getElementById('hamburger-btn');
    const overlay = document.getElementById('sidebar-overlay');
    const toggleIcon = document.getElementById('sidebar-toggle-icon');
    const isMobile = window.innerWidth <= 768;

    if (isMobile) {
        const isOpen = sidebar.classList.contains('mobile-open');
        // Remove desktop collapsed class to prevent width:0/overflow:hidden conflict
        if (!isOpen && sidebar.classList.contains('collapsed')) {
            sidebar.classList.remove('collapsed');
        }
        sidebar.classList.toggle('mobile-open', !isOpen);
        hamburger.classList.toggle('active', !isOpen);
        hamburger.setAttribute('aria-expanded', String(!isOpen));
        if (overlay) overlay.classList.toggle('active', !isOpen);
        // Auto-scroll to active nav item when opening
        if (!isOpen) {
            requestAnimationFrame(function () {
                const activeItem = sidebar.querySelector('.nav-item.active');
                if (activeItem) {
                    activeItem.scrollIntoView({ block: 'center', behavior: 'smooth' });
                    // Expand the section containing the active item
                    const section = activeItem.closest('.nav-section');
                    if (section && section.classList.contains('collapsed')) {
                        section.classList.remove('collapsed');
                    }
                }
            });
        }
    } else {
        sidebarCollapsed = !sidebarCollapsed;
        sidebar.classList.toggle('collapsed', sidebarCollapsed);
        hamburger.classList.toggle('active', sidebarCollapsed);
        hamburger.setAttribute('aria-expanded', String(!sidebarCollapsed));
        if (toggleIcon) toggleIcon.textContent = sidebarCollapsed ? '»' : '«';
        localStorage.setItem('k13d_sidebar_collapsed', sidebarCollapsed);
    }
}

// Close mobile sidebar when a nav item is clicked
function closeMobileSidebar() {
    if (window.innerWidth <= 768) {
        const sidebar = document.getElementById('sidebar');
        if (sidebar.classList.contains('mobile-open')) {
            toggleSidebar();
        }
    }
}

// Toggle nav section collapse (mobile only)
function toggleNavSection(titleEl) {
    if (window.innerWidth > 768) return;
    const section = titleEl.closest('.nav-section');
    if (section) section.classList.toggle('collapsed');
}

// Auto-collapse inactive nav sections on mobile load
function initMobileNavSections() {
    if (window.innerWidth > 768) return;
    // Skip if no active item yet (will be called again after switchResource)
    if (!document.querySelector('#sidebar .nav-item.active')) return;
    document.querySelectorAll('#sidebar .nav-section').forEach(function (section) {
        // Skip the overview section (no nav-title)
        if (!section.querySelector('.nav-title')) return;
        // Collapse sections that don't contain the active item
        if (!section.querySelector('.nav-item.active')) {
            section.classList.add('collapsed');
        }
    });
}
// Run on load and resize
window.addEventListener('load', initMobileNavSections);
window.addEventListener('resize', function () {
    if (window.innerWidth > 768) {
        // Remove collapsed state when switching to desktop
        document.querySelectorAll('#sidebar .nav-section.collapsed').forEach(function (s) {
            s.classList.remove('collapsed');
        });
    }
});

// ==========================================
// Debug Mode (MCP Tool Calling)
// ==========================================
let debugLogs = [];

function toggleDebugMode() {
    debugMode = !debugMode;
    const panel = document.getElementById('debug-panel');
    const toggle = document.getElementById('debug-toggle');

    panel.classList.toggle('active', debugMode);
    toggle.style.background = debugMode ? 'var(--accent-purple)' : 'transparent';
    localStorage.setItem('k13d_debug_mode', debugMode);
}

function addDebugLog(type, title, content) {
    if (!debugMode) return;

    const timestamp = formatTimeShort(new Date());
    debugLogs.push({ type, title, content, timestamp });

    const container = document.getElementById('debug-content');
    const entry = document.createElement('div');
    entry.className = `debug-entry ${type}`;
    entry.innerHTML = `
                <div class="debug-entry-header">
                    <span>${title}</span>
                    <span>${timestamp}</span>
                </div>
                <div class="debug-entry-body">${typeof content === 'object' ? JSON.stringify(content, null, 2) : content}</div>
            `;
    container.appendChild(entry);
    container.scrollTop = container.scrollHeight;
}

function clearDebugLogs() {
    debugLogs = [];
    document.getElementById('debug-content').innerHTML = `
                <div style="color: var(--text-secondary); text-align: center; padding: 20px;">
                    Debug logs cleared. Tool calls will appear here.
                </div>
            `;
}

// ==========================================
// AI Context Management
// ==========================================
function addToAIContext(item) {
    // Check if already exists
    const exists = aiContextItems.find(i => i.name === item.name && i.namespace === item.namespace);
    if (exists) return;

    aiContextItems.push({
        type: currentResource,
        name: item.name,
        namespace: item.namespace || '',
        data: item
    });

    renderContextChips();
}

function removeFromAIContext(index) {
    aiContextItems.splice(index, 1);
    renderContextChips();
}

function clearAIContext() {
    aiContextItems = [];
    renderContextChips();
}

function renderContextChips() {
    const container = document.getElementById('context-chips');
    if (aiContextItems.length === 0) {
        container.innerHTML = '<span style="font-size: 11px; color: var(--text-secondary);">Context: Click resources to add</span>';
        return;
    }

    container.innerHTML = aiContextItems.map((item, index) => `
                <span class="context-chip">
                    ${item.type.slice(0, -1)}: ${item.name}
                    <span class="remove" onclick="event.stopPropagation(); removeFromAIContext(${index})">×</span>
                </span>
            `).join('') + `<span class="context-chip" style="background: var(--bg-tertiary); cursor: pointer;" onclick="clearAIContext()">Clear all</span>`;
}

function getContextForAI() {
    if (aiContextItems.length === 0) return '';

    return '\n\nContext from selected resources:\n' + aiContextItems.map(item => {
        return `[${item.type}] ${item.name}${item.namespace ? ` (ns: ${item.namespace})` : ''}: ${JSON.stringify(item.data)}`;
    }).join('\n');
}

// Update addRowClickHandlers - explicit button for AI context only
function addRowClickHandlers() {
    document.querySelectorAll('#table-body tr[data-index]').forEach((row) => {
        const dataIndex = parseInt(row.dataset.index || '', 10);
        const item = Number.isNaN(dataIndex) ? null : cachedData[dataIndex];
        if (!item) {
            return;
        }

        // Left click - show detail modal (but not if clicking on action buttons)
        row.onclick = (e) => {
            // Ignore clicks on action buttons or their container
            if (e.target.closest('.resource-actions') || e.target.closest('.resource-action-btn')) {
                return;
            }
            showResourceDetail(item);
        };

        // Add explicit + button for adding to context
        const firstCell = row.querySelector('td:first-child');
        if (firstCell && !firstCell.querySelector('.add-context-btn')) {
            const btn = document.createElement('button');
            btn.className = 'add-context-btn';
            btn.textContent = '+';
            btn.title = 'Add to AI context';
            btn.onclick = (e) => {
                e.stopPropagation();
                addToAIContext(item);
                // Visual feedback
                btn.textContent = '✓';
                btn.style.background = 'var(--accent-green)';
                setTimeout(() => {
                    btn.textContent = '+';
                    btn.style.background = '';
                }, 1000);
            };
            firstCell.prepend(btn);
            firstCell.prepend(document.createTextNode(' '));
        }
    });
}

// Override sendMessage to include context (uses agentic mode)
const originalSendMessage = sendMessage;
sendMessage = async function () {
    const input = document.getElementById('ai-input');
    let message = input.value.trim();
        if (isLoading) return;
    if (!message) {
        const originalPlaceholder = input.placeholder;
        input.placeholder = t('msg_enter_question') || '질문을 입력해 주세요.';
        input.classList.add('error');
        input.focus();
        
        // Remove error state when user types
        const onInput = () => {
            input.classList.remove('error');
            input.placeholder = originalPlaceholder;
            input.removeEventListener('input', onInput);
        };
        input.addEventListener('input', onInput);
        return;
    }

    // Add context if available
    const contextStr = getContextForAI();
    if (contextStr) {
        message += contextStr;
    }

    // Log request in debug mode
    addDebugLog('request', 'AI Request', { message, context: aiContextItems });

    // Save query to history for arrow key navigation
    saveQueryToHistory(message.split('\n\nContext from selected resources:')[0]);
    aiHistoryIndex = -1;
    aiCurrentDraft = '';

    isLoading = true;
    document.getElementById('send-btn').disabled = true;
    input.value = '';
    input.disabled = true;

    addMessage(message.split('\n\nContext from selected resources:')[0], true);

    // Use agentic mode
    await sendMessageAgentic(message);

    isLoading = false;
    document.getElementById('send-btn').disabled = false;
    input.disabled = false;
    input.focus();
};

// ==========================================
// Terminal Functions (xterm.js + WebSocket)
// ==========================================
let currentTerminal = null;
let currentTerminalWs = null;
let terminalFitAddon = null;
let terminalReconnectAttempts = 0;
let terminalReconnectTimer = null;
let terminalHeartbeatInterval = null;
let terminalPodName = null;
let terminalNamespace = null;
let terminalContainer = null;
let terminalShouldReconnect = true;

function openTerminal(podName, namespace, container = '') {
    // Store connection params for reconnection
    terminalPodName = podName;
    terminalNamespace = namespace;
    terminalContainer = container;
    terminalShouldReconnect = true;
    const modal = document.getElementById('terminal-modal');
    document.getElementById('terminal-pod-name').textContent = podName;
    document.getElementById('terminal-container-name').textContent = container || 'default';

    modal.classList.add('active');

    // Initialize xterm.js
    const terminalEl = document.getElementById('terminal-container');
    terminalEl.innerHTML = '';

    currentTerminal = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: "'SF Mono', 'Monaco', 'Menlo', monospace",
        theme: {
            background: '#1a1b26',
            foreground: '#c0caf5',
            cursor: '#c0caf5',
            selection: '#33467c',
            black: '#15161e',
            red: '#f7768e',
            green: '#9ece6a',
            yellow: '#e0af68',
            blue: '#7aa2f7',
            magenta: '#bb9af7',
            cyan: '#7dcfff',
            white: '#a9b1d6'
        }
    });

    terminalFitAddon = new FitAddon.FitAddon();
    currentTerminal.loadAddon(terminalFitAddon);

    if (typeof WebLinksAddon !== 'undefined') {
        const webLinksAddon = new WebLinksAddon.WebLinksAddon();
        currentTerminal.loadAddon(webLinksAddon);
    }

    currentTerminal.open(terminalEl);
    terminalFitAddon.fit();

    // Connect WebSocket with reconnection support
    connectTerminalWebSocket();
}

function connectTerminalWebSocket() {
    // Clean up existing connection
    if (terminalHeartbeatInterval) {
        clearInterval(terminalHeartbeatInterval);
        terminalHeartbeatInterval = null;
    }
    if (currentTerminalWs) {
        currentTerminalWs.onclose = null; // Prevent reconnection loop
        currentTerminalWs.close();
        currentTerminalWs = null;
    }

    // Build WebSocket URL with auth token (WebSocket cannot set headers)
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsParams = new URLSearchParams();
    if (terminalContainer) wsParams.set('container', terminalContainer);
    if (authToken && authToken !== 'anonymous') wsParams.set('token', authToken);
    const wsQuery = wsParams.toString() ? '?' + wsParams.toString() : '';
    const wsUrl = `${wsProtocol}//${window.location.host}/api/terminal/${terminalNamespace}/${terminalPodName}${wsQuery}`;

    currentTerminalWs = new WebSocket(wsUrl);

    currentTerminalWs.onopen = () => {
        // Reset reconnect attempts on successful connection
        terminalReconnectAttempts = 0;

        if (currentTerminal) {
            currentTerminal.writeln('\x1b[32m● Connected to pod: ' + terminalPodName + '\x1b[0m');
            currentTerminal.writeln('');

            const dims = terminalFitAddon.proposeDimensions();
            if (dims) {
                currentTerminalWs.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }));
            }
        }

        // Start heartbeat/keepalive (ping every 30 seconds)
        terminalHeartbeatInterval = setInterval(() => {
            if (currentTerminalWs && currentTerminalWs.readyState === WebSocket.OPEN) {
                currentTerminalWs.send(JSON.stringify({ type: 'ping' }));
            }
        }, 30000);
    };

    currentTerminalWs.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            if (msg.type === 'output') {
                if (currentTerminal) currentTerminal.write(msg.data);
            } else if (msg.type === 'error') {
                if (currentTerminal) currentTerminal.writeln('\x1b[31mError: ' + msg.data + '\x1b[0m');
            } else if (msg.type === 'pong') {
                // Heartbeat response received
            }
        } catch (e) {
            if (currentTerminal) currentTerminal.write(event.data);
        }
    };

    currentTerminalWs.onclose = (event) => {
        // Clear heartbeat
        if (terminalHeartbeatInterval) {
            clearInterval(terminalHeartbeatInterval);
            terminalHeartbeatInterval = null;
        }

        if (!currentTerminal) return;

        // Show disconnection message
        currentTerminal.writeln('\x1b[33m\r\n● Connection closed.\x1b[0m');

        // Attempt reconnection with exponential backoff
        if (terminalShouldReconnect) {
            const delay = Math.min(1000 * Math.pow(2, terminalReconnectAttempts), 30000); // Max 30s
            terminalReconnectAttempts++;

            currentTerminal.writeln('\x1b[90mReconnecting in ' + (delay / 1000) + 's... (attempt ' + terminalReconnectAttempts + ')\x1b[0m');

            terminalReconnectTimer = setTimeout(() => {
                if (terminalShouldReconnect && currentTerminal) {
                    currentTerminal.writeln('\x1b[36m● Reconnecting...\x1b[0m');
                    connectTerminalWebSocket();
                }
            }, delay);
        }
    };

    currentTerminalWs.onerror = (err) => {
        if (!currentTerminal) return;

        // Only show detailed error on first connection attempt
        if (terminalReconnectAttempts === 0) {
            const isRemoteAccess = window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1';
            currentTerminal.writeln('\x1b[31m');
            currentTerminal.writeln('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
            currentTerminal.writeln('  ✗ WebSocket Connection Failed');
            currentTerminal.writeln('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
            currentTerminal.writeln('\x1b[0m');
            if (isRemoteAccess) {
                currentTerminal.writeln('\x1b[33mYou are accessing from a remote IP address.\x1b[0m');
                currentTerminal.writeln('\x1b[33mWebSocket connections require additional configuration.\x1b[0m');
                currentTerminal.writeln('');
                currentTerminal.writeln('\x1b[36mSolution 1: Enable development mode\x1b[0m');
                currentTerminal.writeln('  export K13D_DEV=true');
                currentTerminal.writeln('');
                currentTerminal.writeln('\x1b[36mSolution 2: Allow your origin explicitly\x1b[0m');
                currentTerminal.writeln('  export K13D_WS_ALLOWED_ORIGINS="' + window.location.origin + '"');
                currentTerminal.writeln('');
                currentTerminal.writeln('\x1b[90mThen restart k13d server.\x1b[0m');
            } else {
                currentTerminal.writeln('\x1b[33mFailed to connect to terminal WebSocket.\x1b[0m');
                currentTerminal.writeln('\x1b[33mPlease check if the pod is running and accessible.\x1b[0m');
            }
        }
    };

    currentTerminal.onData((data) => {
        if (currentTerminalWs && currentTerminalWs.readyState === WebSocket.OPEN) {
            currentTerminalWs.send(JSON.stringify({ type: 'input', data: data }));
        }
    });

    window.addEventListener('resize', handleTerminalResize);
    currentTerminal.onResize(({ cols, rows }) => {
        if (currentTerminalWs && currentTerminalWs.readyState === WebSocket.OPEN) {
            currentTerminalWs.send(JSON.stringify({ type: 'resize', cols, rows }));
        }
    });

    setTimeout(() => terminalFitAddon.fit(), 100);
}

function handleTerminalResize() { if (terminalFitAddon) terminalFitAddon.fit(); }

function closeTerminal() {
    // Disable reconnection
    terminalShouldReconnect = false;

    // Clear timers
    if (terminalReconnectTimer) {
        clearTimeout(terminalReconnectTimer);
        terminalReconnectTimer = null;
    }
    if (terminalHeartbeatInterval) {
        clearInterval(terminalHeartbeatInterval);
        terminalHeartbeatInterval = null;
    }

    // Close WebSocket
    if (currentTerminalWs) {
        currentTerminalWs.onclose = null; // Prevent reconnection
        currentTerminalWs.close();
        currentTerminalWs = null;
    }

    // Dispose terminal
    if (currentTerminal) {
        currentTerminal.dispose();
        currentTerminal = null;
    }

    // Remove event listeners
    window.removeEventListener('resize', handleTerminalResize);

    // Hide modal
    document.getElementById('terminal-modal').classList.remove('active');

    // Reset state
    terminalReconnectAttempts = 0;
    terminalPodName = null;
    terminalNamespace = null;
    terminalContainer = null;
}

// ==========================================
// Log Viewer Functions
// ==========================================
let currentLogPod = null, currentLogNamespace = null, currentLogContainer = null;
let currentLogPods = []; // For multi-pod logging
let logEventSource = null, logFollowMode = true, allLogs = [], ansiUp = null;
let isMultiPodMode = false;
let podColorMap = {};
let hiddenPods = new Set();

const POD_COLORS = [
    { name: 'blue', class: 'log-pod-0' },
    { name: 'green', class: 'log-pod-1' },
    { name: 'yellow', class: 'log-pod-2' },
    { name: 'purple', class: 'log-pod-3' },
    { name: 'cyan', class: 'log-pod-4' },
    { name: 'red', class: 'log-pod-5' },
    { name: 'teal', class: 'log-pod-6' },
    { name: 'orange', class: 'log-pod-7' }
];

// Helper function to open log viewer from row button click
function openLogViewerFromRow(btn, podName, namespace) {
    const row = btn.closest('tr');
    let containers = ['default'];
    if (row && row.dataset.containers) {
        try {
            containers = JSON.parse(row.dataset.containers);
        } catch (e) {
            console.warn('Failed to parse containers:', e);
        }
    }
    openLogViewer(podName, namespace, containers);
}

// Open multi-pod log viewer for a workload (deployment, replicaset, etc.)
async function openMultiPodLogViewer(workloadName, namespace, labelSelector) {
    isMultiPodMode = true;
    currentLogNamespace = namespace;
    currentLogPods = [];
    podColorMap = {};
    hiddenPods.clear();

    document.getElementById('log-pod-name').textContent = workloadName;
    document.getElementById('log-container-name').textContent = '(multiple pods)';

    // Hide container select for multi-pod mode
    document.getElementById('log-container-select').style.display = 'none';

    document.getElementById('log-viewer-modal').classList.add('active');
    if (typeof AnsiUp !== 'undefined') { ansiUp = new AnsiUp(); ansiUp.use_classes = true; }
    // Ensure Follow button shows correct state
    document.getElementById('log-follow-btn').classList.toggle('active', logFollowMode);

    // Fetch pods for this workload
    try {
        const resp = await fetchWithAuth(`/api/k8s/pods?namespace=${namespace}&labelSelector=${encodeURIComponent(labelSelector)}`);
        const data = await resp.json();
        currentLogPods = (data.items || []).map(p => p.name);

        // Assign colors to pods
        currentLogPods.forEach((pod, idx) => {
            podColorMap[pod] = POD_COLORS[idx % POD_COLORS.length];
        });

        // Show pod legend
        renderPodLegend();
        await loadMultiPodLogs();
    } catch (e) {
        document.getElementById('log-content').innerHTML = `<p style="color: var(--accent-red);">Error: ${e.message}</p>`;
    }
}

function renderPodLegend() {
    const legend = document.getElementById('log-pod-legend');
    if (currentLogPods.length <= 1) {
        legend.style.display = 'none';
        return;
    }

    legend.style.display = 'flex';
    legend.innerHTML = currentLogPods.map((pod, idx) => {
        const color = podColorMap[pod];
        const shortName = pod.length > 30 ? pod.substring(0, 27) + '...' : pod;
        const hidden = hiddenPods.has(pod) ? 'hidden' : '';
        return `<div class="log-pod-legend-item ${hidden}" onclick="togglePodVisibility('${pod}')" title="${pod}">
                    <span class="log-pod-legend-dot legend-${color.class.replace('log-', '')}"></span>
                    <span>${shortName}</span>
                </div>`;
    }).join('');
}

function togglePodVisibility(podName) {
    if (hiddenPods.has(podName)) {
        hiddenPods.delete(podName);
    } else {
        hiddenPods.add(podName);
    }
    renderPodLegend();
    // Re-render logs with filter
    filterLogs();
}

async function openLogViewer(podName, namespace, containers = []) {
    isMultiPodMode = false;
    currentLogPod = podName;
    currentLogNamespace = namespace;
    currentLogPods = [podName];
    podColorMap = { [podName]: POD_COLORS[0] };

    document.getElementById('log-pod-name').textContent = podName;
    document.getElementById('log-pod-legend').style.display = 'none';
    document.getElementById('log-container-select').style.display = '';

    // Filter out 'default' placeholder - use actual container names only
    const validContainers = containers.filter(c => c && c !== 'default');

    const containerSelect = document.getElementById('log-container-select');
    if (validContainers.length > 0) {
        containerSelect.innerHTML = validContainers.map((c, i) => `<option value="${c}" ${i === 0 ? 'selected' : ''}>${c}</option>`).join('');
        currentLogContainer = validContainers[0];
        document.getElementById('log-container-name').textContent = currentLogContainer;
    } else {
        // No containers specified - let the backend use the default container
        containerSelect.innerHTML = '<option value="">default</option>';
        currentLogContainer = '';
        document.getElementById('log-container-name').textContent = 'default';
    }

    document.getElementById('log-viewer-modal').classList.add('active');
    if (typeof AnsiUp !== 'undefined') { ansiUp = new AnsiUp(); ansiUp.use_classes = true; }
    // Ensure Follow button shows correct state
    document.getElementById('log-follow-btn').classList.toggle('active', logFollowMode);
    await loadLogs();
}

function switchLogContainer() {
    currentLogContainer = document.getElementById('log-container-select').value;
    document.getElementById('log-container-name').textContent = currentLogContainer;
    loadLogs();
}

async function loadMultiPodLogs() {
    const tailLines = document.getElementById('log-lines-select').value;
    const logContent = document.getElementById('log-content');
    logContent.innerHTML = '<p style="color: var(--text-secondary);">Loading logs from multiple pods...</p>';
    allLogs = [];

    try {
        // Fetch logs from all pods in parallel
        const logPromises = currentLogPods.map(async (pod) => {
            try {
                const url = `/api/pods/${currentLogNamespace}/${pod}/logs?tailLines=${Math.floor(tailLines / currentLogPods.length)}`;
                const resp = await fetchWithAuth(url);
                const text = await resp.text();
                return text.split('\n').filter(l => l.trim()).map(line => ({ pod, line }));
            } catch (e) {
                return [{ pod, line: `[Error fetching logs: ${e.message}]` }];
            }
        });

        const results = await Promise.all(logPromises);
        const allPodLogs = results.flat();

        // Sort by timestamp if present, otherwise keep order
        // For now, interleave logs from different pods
        logContent.innerHTML = '';
        allPodLogs.forEach(({ pod, line }) => {
            appendLogLine(line, pod);
        });
        // Auto-scroll to bottom after loading logs
        logContent.scrollTop = logContent.scrollHeight;

    } catch (e) {
        logContent.innerHTML = `<p style="color: var(--accent-red);">Error loading logs: ${e.message}</p>`;
    }
}

async function loadLogs() {
    if (isMultiPodMode) {
        return loadMultiPodLogs();
    }

    const tailLines = document.getElementById('log-lines-select').value;
    const logContent = document.getElementById('log-content');
    logContent.innerHTML = '<p style="color: var(--text-secondary);">Loading logs...</p>';
    allLogs = [];
    if (logEventSource) { logEventSource.close(); logEventSource = null; }

    try {
        // Always use follow=false to get plain text response
        // SSE streaming is not properly supported in this fetch pattern
        let url = `/api/pods/${currentLogNamespace}/${currentLogPod}/logs?tailLines=${tailLines}&follow=false`;
        if (currentLogContainer) {
            url += `&container=${currentLogContainer}`;
        }
        const resp = await fetchWithAuth(url);
        if (!resp.ok) {
            const errorText = await resp.text();
            throw new Error(errorText || `HTTP ${resp.status}`);
        }
        const text = await resp.text();
        logContent.innerHTML = '';
        if (text.trim()) {
            text.split('\n').forEach(line => { if (line.trim()) appendLogLine(line, currentLogPod); });
        } else {
            logContent.innerHTML = '<p style="color: var(--text-secondary);">No logs available for this pod.</p>';
        }
        // Auto-scroll to bottom after loading logs
        logContent.scrollTop = logContent.scrollHeight;
    } catch (e) {
        logContent.innerHTML = `<p style="color: var(--accent-red);">Error loading logs: ${e.message}</p>`;
    }
}

function appendLogLine(line, podName = null) {
    const logContent = document.getElementById('log-content');
    const pod = podName || currentLogPod;
    allLogs.push({ line, pod });

    const div = document.createElement('div');
    div.className = 'log-line';
    div.dataset.pod = pod;

    // Add pod color class for multi-pod mode
    const podColor = podColorMap[pod];
    if (podColor && currentLogPods.length > 1) {
        div.classList.add(podColor.class);
    }

    // Detect log level
    const lineLower = line.toLowerCase();
    if (lineLower.includes('error') || lineLower.includes('fatal') || lineLower.includes('panic')) {
        div.classList.add('error');
    } else if (lineLower.includes('warn') || lineLower.includes('warning')) {
        div.classList.add('warn');
    }

    // Add pod tag for multi-pod mode
    if (currentLogPods.length > 1) {
        const podTag = document.createElement('span');
        podTag.className = 'log-pod-tag';
        // Show short pod name (last part after last dash or first 15 chars)
        const shortPod = pod.split('-').slice(-2).join('-').substring(0, 15);
        podTag.textContent = shortPod;
        podTag.title = pod;
        div.appendChild(podTag);
    }

    // Create content wrapper
    const content = document.createElement('span');
    content.className = 'log-line-content';
    content.innerHTML = ansiUp ? ansiUp.ansi_to_html(line) : escapeHtml(line);
    div.appendChild(content);

    logContent.appendChild(div);
    if (logFollowMode) logContent.scrollTop = logContent.scrollHeight;
}

function reloadLogs() { loadLogs(); }
function toggleLogFollow() {
    logFollowMode = !logFollowMode;
    document.getElementById('log-follow-btn').classList.toggle('active', logFollowMode);
    loadLogs();
}

function filterLogs() {
    const searchTerm = document.getElementById('log-search-input').value.toLowerCase();
    let matchCount = 0;
    document.querySelectorAll('#log-content .log-line').forEach(lineEl => {
        const pod = lineEl.dataset.pod;
        const isPodHidden = hiddenPods.has(pod);
        const matchesSearch = searchTerm === '' || lineEl.textContent.toLowerCase().includes(searchTerm);
        const visible = !isPodHidden && matchesSearch;
        lineEl.style.display = visible ? 'flex' : 'none';
        if (visible && searchTerm) matchCount++;
    });
    document.getElementById('log-match-count').textContent = searchTerm ? `${matchCount} matches` : '';
}

function downloadLogs() {
    const logsText = allLogs.map(l => {
        if (typeof l === 'object') {
            return currentLogPods.length > 1 ? `[${l.pod}] ${l.line}` : l.line;
        }
        return l;
    }).join('\n');
    const blob = new Blob([logsText], { type: 'text/plain' });
    const a = document.createElement('a'); a.href = URL.createObjectURL(blob);
    const dateStr = new Date().toISOString().slice(0, 19).replace(/[T:]/g, '-');
    const podName = currentLogPod || currentLogPods[0] || 'unknown';
    const filename = currentLogPods.length > 1
        ? `${podName}-multi-${dateStr}.log`
        : `${podName}-${dateStr}.log`;
    a.download = filename; a.click();
}

function closeLogViewer() {
    document.getElementById('log-viewer-modal').classList.remove('active');
    if (logEventSource) { logEventSource.close(); logEventSource = null; }
    allLogs = []; logFollowMode = true; // Reset to default (follow mode on)
}

// ==========================================
// Metrics Functions
// ==========================================
let cpuChart = null, memoryChart = null, llmUsageChart = null;
let metricsHistory = { cpu: [], memory: [], timestamps: [], pods: [], nodes: [] };
let llmUsageHistory = { requests: [], tokens: [], timestamps: [] };
let metricsInterval = null;
let metricsHistoryLoaded = false;
let metricsTimeRangeMinutes = 30; // Default time range in minutes

function setMetricsTimeRangeMinutes(minutes) {
    metricsTimeRangeMinutes = minutes;
    // Update active button state
    document.querySelectorAll('.time-range-btn').forEach(btn => {
        btn.classList.remove('active');
        if (parseInt(btn.dataset.minutes) === minutes) {
            btn.classList.add('active');
        }
    });
    // Reload historical data with new time range
    loadHistoricalMetrics();
    loadLLMUsageStats();
}

function setMetricsTimeRange(hours) {
    metricsTimeRangeMinutes = hours * 60;
    // Update active button state
    document.querySelectorAll('.time-range-btn').forEach(btn => {
        btn.classList.remove('active');
        if (parseInt(btn.dataset.hours) === hours) {
            btn.classList.add('active');
        }
    });
    // Reload historical data with new time range
    loadHistoricalMetrics();
    loadLLMUsageStats();
}

async function showMetrics() {
    document.getElementById('metrics-modal').classList.add('active');
    const metricsNsSelect = document.getElementById('metrics-namespace-select');
    metricsNsSelect.innerHTML = document.getElementById('namespace-select').innerHTML;

    // Load Prometheus status
    try {
        const resp = await fetchWithAuth('/api/prometheus/settings');
        const data = await resp.json();
        if (!data.error) {
            updatePrometheusStatus(data.expose_metrics, data.external_url);
        }
    } catch (e) {
        console.error('Failed to load Prometheus status:', e);
    }

    // Load historical metrics first, then real-time
    await loadHistoricalMetrics();
    await loadMetrics();
    await loadLLMUsageStats();

    // Set up auto-refresh interval if checkbox is checked
    const autoRefresh = document.getElementById('metrics-auto-refresh');
    if (autoRefresh && autoRefresh.checked) {
        metricsInterval = setInterval(loadMetrics, 30000);
    }
}

async function loadHistoricalMetrics() {
    try {
        const resp = await fetchWithAuth(`/api/metrics/history/cluster?minutes=${metricsTimeRangeMinutes}&limit=100`);
        const data = await resp.json();

        if (!data.error && data.items && data.items.length > 0) {
            // Sort by timestamp ascending
            const sorted = data.items.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));

            metricsHistory.timestamps = sorted.map(m => formatTimeShort(m.timestamp));
            metricsHistory.cpu = sorted.map(m => m.used_cpu_millis || 0);
            metricsHistory.memory = sorted.map(m => m.used_memory_mb || 0);
            metricsHistory.pods = sorted.map(m => m.running_pods || 0);
            metricsHistory.nodes = sorted.map(m => m.ready_nodes || m.total_nodes || 0);

            // Check if metrics-server data is available (all zeros means unavailable)
            const hasCPUData = metricsHistory.cpu.some(v => v > 0);
            const hasMemData = metricsHistory.memory.some(v => v > 0);

            metricsHistoryLoaded = true;
            updateMetricsCharts(hasCPUData, hasMemData);

            // Update summary from latest metrics
            const latest = sorted[sorted.length - 1];
            if (latest) {
                if (hasCPUData) {
                    document.getElementById('metrics-total-cpu').textContent = `${latest.used_cpu_millis || 0}m`;
                } else {
                    document.getElementById('metrics-total-cpu').textContent = 'N/A';
                    document.getElementById('metrics-total-cpu').title = 'Install metrics-server for CPU data';
                }
                if (hasMemData) {
                    document.getElementById('metrics-total-memory').textContent = formatBytes((latest.used_memory_mb || 0) * 1024 * 1024);
                } else {
                    document.getElementById('metrics-total-memory').textContent = 'N/A';
                    document.getElementById('metrics-total-memory').title = 'Install metrics-server for memory data';
                }
                document.getElementById('metrics-total-pods').textContent = latest.running_pods || 0;
            }
        } else {
            // No data collected yet
            const cpuEl = document.getElementById('metrics-total-cpu');
            const memEl = document.getElementById('metrics-total-memory');
            const podEl = document.getElementById('metrics-total-pods');
            if (cpuEl) cpuEl.textContent = 'Collecting...';
            if (memEl) memEl.textContent = 'Collecting...';
            if (podEl) podEl.textContent = '0';
        }
    } catch (e) {
        console.error('Failed to load historical metrics:', e);
    }
}

async function loadMetrics() {
    const namespace = document.getElementById('metrics-namespace-select').value;
    try {
        // Load pod metrics
        const url = namespace ? `/api/metrics/pods?namespace=${namespace}` : '/api/metrics/pods';
        const resp = await fetchWithAuth(url);
        const data = await resp.json();

        if (data.error) {
            document.getElementById('metrics-cpu-value').textContent = 'N/A';
            document.getElementById('metrics-mem-value').textContent = 'N/A';
            document.getElementById('metrics-pods-value').textContent = 'N/A';
            // Also update legacy elements for backward compatibility
            const legacyCpu = document.getElementById('metrics-total-cpu');
            const legacyMem = document.getElementById('metrics-total-memory');
            const legacyPods = document.getElementById('metrics-total-pods');
            if (legacyCpu) legacyCpu.textContent = 'N/A';
            if (legacyMem) legacyMem.textContent = 'N/A';
            if (legacyPods) legacyPods.textContent = 'N/A';
            return;
        }

        const totalCpu = data.items?.reduce((sum, p) => sum + (p.cpu || 0), 0) || 0;
        const totalMem = data.items?.reduce((sum, p) => sum + (p.memory || 0), 0) || 0;
        const podCount = data.items?.length || 0;

        // Update new dashboard stat cards
        document.getElementById('metrics-cpu-value').textContent = `${totalCpu.toFixed(0)}m`;
        document.getElementById('metrics-mem-value').textContent = formatBytes(totalMem * 1024 * 1024);
        document.getElementById('metrics-pods-value').textContent = podCount;

        // Also update legacy elements for backward compatibility
        const legacyCpu = document.getElementById('metrics-total-cpu');
        const legacyMem = document.getElementById('metrics-total-memory');
        const legacyPods = document.getElementById('metrics-total-pods');
        if (legacyCpu) legacyCpu.textContent = `${totalCpu.toFixed(0)}m`;
        if (legacyMem) legacyMem.textContent = formatBytes(totalMem * 1024 * 1024);
        if (legacyPods) legacyPods.textContent = podCount;

        // Append real-time data point to history
        metricsHistory.timestamps.push(formatTimeShort(new Date()));
        metricsHistory.cpu.push(totalCpu);
        metricsHistory.memory.push(totalMem);
        metricsHistory.pods.push(podCount);
        // Keep last known node count for real-time updates
        metricsHistory.nodes.push(metricsHistory.nodes.length > 0 ? metricsHistory.nodes[metricsHistory.nodes.length - 1] : 0);
        const maxHistory = 100;
        while (metricsHistory.timestamps.length > maxHistory) {
            metricsHistory.timestamps.shift();
            metricsHistory.cpu.shift();
            metricsHistory.memory.shift();
            metricsHistory.pods.shift();
            metricsHistory.nodes.shift();
        }
        updateMetricsCharts();
        updateTopConsumers(data.items || []);

        // Load node health info
        await loadNodeHealth();
    } catch (e) { console.error('Failed to load metrics:', e); }
}

async function loadNodeHealth() {
    try {
        const resp = await fetchWithAuth('/api/metrics/nodes');
        const data = await resp.json();

        // Also get node list for status info
        const nodesResp = await fetchWithAuth('/api/nodes');
        const nodesData = await nodesResp.json();

        const nodeHealthGrid = document.getElementById('node-health-grid');
        if (!nodeHealthGrid) return;

        // Build node info map
        const nodeInfo = {};
        if (nodesData.items) {
            nodesData.items.forEach(node => {
                const readyCondition = node.status?.conditions?.find(c => c.type === 'Ready');
                nodeInfo[node.metadata.name] = {
                    ready: readyCondition?.status === 'True',
                    capacity: {
                        cpu: parseCpuToMillicores(node.status?.capacity?.cpu || '0'),
                        memory: parseMemoryToMB(node.status?.capacity?.memory || '0')
                    }
                };
            });
        }

        // Update nodes stat card
        const totalNodes = nodesData.items?.length || 0;
        const readyNodes = Object.values(nodeInfo).filter(n => n.ready).length;
        document.getElementById('metrics-nodes-value').textContent = `${readyNodes}/${totalNodes}`;

        // Update last nodes value in history for real-time sync
        if (metricsHistory.nodes.length > 0) {
            metricsHistory.nodes[metricsHistory.nodes.length - 1] = readyNodes;
        }

        // Build node health cards
        if (data.items && data.items.length > 0) {
            const cards = data.items.map(node => {
                const info = nodeInfo[node.name] || { ready: true, capacity: { cpu: 4000, memory: 8192 } };
                const cpuPercent = Math.min(100, (node.cpu / info.capacity.cpu) * 100);
                const memPercent = Math.min(100, (node.memory / info.capacity.memory) * 100);
                const status = !info.ready ? 'critical' : (cpuPercent > 80 || memPercent > 80) ? 'warning' : 'healthy';

                return `
                            <div class="node-health-card">
                                <div class="node-name">
                                    <span class="node-status ${status}"></span>
                                    <span>${escapeHtml(node.name)}</span>
                                </div>
                                <div class="usage-bar-container">
                                    <div class="usage-bar-label">
                                        <span>CPU</span>
                                        <span>${node.cpu}m / ${info.capacity.cpu}m (${cpuPercent.toFixed(0)}%)</span>
                                    </div>
                                    <div class="usage-bar">
                                        <div class="fill cpu ${cpuPercent > 80 ? 'high' : ''}" style="width: ${cpuPercent}%"></div>
                                    </div>
                                </div>
                                <div class="usage-bar-container">
                                    <div class="usage-bar-label">
                                        <span>Memory</span>
                                        <span>${formatBytes(node.memory * 1024 * 1024)} / ${formatBytes(info.capacity.memory * 1024 * 1024)} (${memPercent.toFixed(0)}%)</span>
                                    </div>
                                    <div class="usage-bar">
                                        <div class="fill memory ${memPercent > 80 ? 'high' : ''}" style="width: ${memPercent}%"></div>
                                    </div>
                                </div>
                            </div>
                        `;
            }).join('');
            nodeHealthGrid.innerHTML = cards;
        } else {
            nodeHealthGrid.innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 20px;">No node metrics available</p>';
        }
    } catch (e) {
        console.error('Failed to load node health:', e);
        const nodeHealthGrid = document.getElementById('node-health-grid');
        if (nodeHealthGrid) {
            nodeHealthGrid.innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 20px;">Failed to load node health</p>';
        }
    }
}

function parseCpuToMillicores(cpuStr) {
    if (!cpuStr) return 0;
    if (cpuStr.endsWith('m')) {
        return parseInt(cpuStr) || 0;
    }
    // Assume it's in cores, convert to millicores
    return (parseFloat(cpuStr) || 0) * 1000;
}

function parseMemoryToMB(memStr) {
    if (!memStr) return 0;
    const units = { 'Ki': 1 / 1024, 'Mi': 1, 'Gi': 1024, 'Ti': 1024 * 1024, 'K': 1 / 1000, 'M': 1, 'G': 1000, 'T': 1000000 };
    for (const [unit, multiplier] of Object.entries(units)) {
        if (memStr.endsWith(unit)) {
            return (parseFloat(memStr) || 0) * multiplier;
        }
    }
    // Assume bytes
    return (parseInt(memStr) || 0) / (1024 * 1024);
}

function updateMetricsCharts(hasCPUData, hasMemData) {
    // Default to checking history data if not passed
    if (hasCPUData === undefined) hasCPUData = metricsHistory.cpu.some(v => v > 0);
    if (hasMemData === undefined) hasMemData = metricsHistory.memory.some(v => v > 0);

    const opts = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: { display: false },
            tooltip: {
                mode: 'index',
                intersect: false,
            }
        },
        scales: {
            x: {
                ticks: { color: '#a9b1d6', maxTicksLimit: 10 },
                grid: { color: '#414868' }
            },
            y: {
                ticks: { color: '#a9b1d6' },
                grid: { color: '#414868' },
                beginAtZero: true
            }
        },
        interaction: {
            mode: 'nearest',
            axis: 'x',
            intersect: false
        }
    };

    // Update chart titles based on data availability
    const cpuTitle = document.querySelector('#cpu-chart')?.closest('.metric-card')?.querySelector('h4');
    const memTitle = document.querySelector('#memory-chart')?.closest('.metric-card')?.querySelector('h4');

    const cpuCtx = document.getElementById('cpu-chart')?.getContext('2d');
    if (cpuCtx) {
        // Choose data: CPU if available, otherwise Pod Count
        const chartData = hasCPUData ? metricsHistory.cpu : metricsHistory.pods;
        const chartLabel = hasCPUData ? 'CPU (millicores)' : 'Running Pods';
        const chartColor = hasCPUData ? '#7dcfff' : '#9ece6a';
        const chartBg = hasCPUData ? 'rgba(125,207,255,0.1)' : 'rgba(158,206,106,0.1)';

        if (cpuTitle) {
            cpuTitle.textContent = hasCPUData ? 'CPU Usage Over Time' : 'Running Pods Over Time';
            if (!hasCPUData) cpuTitle.title = 'Install metrics-server for CPU data';
        }

        if (cpuChart) {
            cpuChart.data.labels = metricsHistory.timestamps;
            cpuChart.data.datasets[0].data = chartData;
            cpuChart.data.datasets[0].label = chartLabel;
            cpuChart.data.datasets[0].borderColor = chartColor;
            cpuChart.data.datasets[0].backgroundColor = chartBg;
            cpuChart.update();
        } else {
            cpuChart = new Chart(cpuCtx, {
                type: 'line',
                data: {
                    labels: metricsHistory.timestamps,
                    datasets: [{
                        label: chartLabel,
                        data: chartData,
                        borderColor: chartColor,
                        backgroundColor: chartBg,
                        fill: true,
                        tension: 0.4,
                        pointRadius: 0,
                        pointHoverRadius: 4
                    }]
                },
                options: opts
            });
        }
    }

    const memCtx = document.getElementById('memory-chart')?.getContext('2d');
    if (memCtx) {
        // Choose data: Memory if available, otherwise Node Count
        const chartData = hasMemData ? metricsHistory.memory : metricsHistory.nodes;
        const chartLabel = hasMemData ? 'Memory (MB)' : 'Ready Nodes';
        const chartColor = hasMemData ? '#bb9af7' : '#e0af68';
        const chartBg = hasMemData ? 'rgba(187,154,247,0.1)' : 'rgba(224,175,104,0.1)';

        if (memTitle) {
            memTitle.textContent = hasMemData ? 'Memory Usage Over Time' : 'Ready Nodes Over Time';
            if (!hasMemData) memTitle.title = 'Install metrics-server for Memory data';
        }

        if (memoryChart) {
            memoryChart.data.labels = metricsHistory.timestamps;
            memoryChart.data.datasets[0].data = chartData;
            memoryChart.data.datasets[0].label = chartLabel;
            memoryChart.data.datasets[0].borderColor = chartColor;
            memoryChart.data.datasets[0].backgroundColor = chartBg;
            memoryChart.update();
        } else {
            memoryChart = new Chart(memCtx, {
                type: 'line',
                data: {
                    labels: metricsHistory.timestamps,
                    datasets: [{
                        label: chartLabel,
                        data: chartData,
                        borderColor: chartColor,
                        backgroundColor: chartBg,
                        fill: true,
                        tension: 0.4,
                        pointRadius: 0,
                        pointHoverRadius: 4
                    }]
                },
                options: opts
            });
        }
    }
}

function updateTopConsumers(pods) {
    const topCpu = [...pods].sort((a, b) => (b.cpu || 0) - (a.cpu || 0)).slice(0, 5);
    document.getElementById('top-cpu-list').innerHTML = topCpu.map(p => `<div style="display:flex;justify-content:space-between;padding:8px;border-bottom:1px solid var(--border-color);"><span style="font-size:12px;">${escapeHtml(p.name)}</span><span style="font-size:12px;color:var(--accent-cyan);">${p.cpu || 0}m</span></div>`).join('');
    const topMem = [...pods].sort((a, b) => (b.memory || 0) - (a.memory || 0)).slice(0, 5);
    document.getElementById('top-memory-list').innerHTML = topMem.map(p => `<div style="display:flex;justify-content:space-between;padding:8px;border-bottom:1px solid var(--border-color);"><span style="font-size:12px;">${escapeHtml(p.name)}</span><span style="font-size:12px;color:var(--accent-purple);">${formatBytes((p.memory || 0) * 1024 * 1024)}</span></div>`).join('');
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024, sizes = ['B', 'Ki', 'Mi', 'Gi'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatNumber(num) {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
}

// ==========================================
// LLM Usage Functions
// ==========================================
async function loadLLMUsageStats() {
    try {
        const resp = await fetchWithAuth(`/api/llm/usage/stats?minutes=${metricsTimeRangeMinutes}`);
        const data = await resp.json();

        if (data.error) {
            document.getElementById('llm-total-requests').textContent = 'N/A';
            document.getElementById('llm-total-tokens').textContent = 'N/A';
            document.getElementById('llm-prompt-tokens').textContent = 'N/A';
            document.getElementById('llm-completion-tokens').textContent = 'N/A';
            return;
        }

        // Update summary stats
        document.getElementById('llm-total-requests').textContent = formatNumber(data.total_requests || 0);
        document.getElementById('llm-total-tokens').textContent = formatNumber(data.total_tokens || 0);
        document.getElementById('llm-prompt-tokens').textContent = formatNumber(data.prompt_tokens || 0);
        document.getElementById('llm-completion-tokens').textContent = formatNumber(data.completion_tokens || 0);

        // Update time series chart
        if (data.hourly && data.hourly.length > 0) {
            llmUsageHistory.timestamps = data.hourly.map(h => {
                const d = new Date(h.hour);
                return formatTimeShort(d);
            });
            llmUsageHistory.requests = data.hourly.map(h => h.requests || 0);
            llmUsageHistory.tokens = data.hourly.map(h => h.total_tokens || 0);
            updateLLMUsageChart();
        }

        // Update model breakdown list
        if (data.by_model && data.by_model.length > 0) {
            document.getElementById('llm-model-list').innerHTML = data.by_model.map(m =>
                `<div style="display:flex;justify-content:space-between;padding:8px;border-bottom:1px solid var(--border-color);">
                            <span style="font-size:12px;">${escapeHtml(m.model || 'unknown')}</span>
                            <span style="font-size:12px;color:var(--accent-yellow);">${formatNumber(m.total_tokens || 0)} tokens</span>
                        </div>`
            ).join('');
        } else {
            document.getElementById('llm-model-list').innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 20px;">No LLM usage data</p>';
        }
    } catch (e) {
        console.error('Failed to load LLM usage stats:', e);
        document.getElementById('llm-model-list').innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 20px;">Failed to load data</p>';
    }
}

function updateLLMUsageChart() {
    const opts = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: { display: true, position: 'top', labels: { color: '#a9b1d6', font: { size: 10 } } },
            tooltip: { mode: 'index', intersect: false }
        },
        scales: {
            x: {
                ticks: { color: '#a9b1d6', maxTicksLimit: 8 },
                grid: { color: '#414868' }
            },
            y: {
                type: 'linear',
                position: 'left',
                ticks: { color: '#e0af68' },
                grid: { color: '#414868' },
                beginAtZero: true,
                title: { display: true, text: 'Requests', color: '#e0af68' }
            },
            y1: {
                type: 'linear',
                position: 'right',
                ticks: { color: '#9ece6a' },
                grid: { drawOnChartArea: false },
                beginAtZero: true,
                title: { display: true, text: 'Tokens', color: '#9ece6a' }
            }
        },
        interaction: { mode: 'nearest', axis: 'x', intersect: false }
    };

    const ctx = document.getElementById('llm-usage-chart')?.getContext('2d');
    if (ctx) {
        if (llmUsageChart) {
            llmUsageChart.data.labels = llmUsageHistory.timestamps;
            llmUsageChart.data.datasets[0].data = llmUsageHistory.requests;
            llmUsageChart.data.datasets[1].data = llmUsageHistory.tokens;
            llmUsageChart.update();
        } else {
            llmUsageChart = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: llmUsageHistory.timestamps,
                    datasets: [
                        {
                            label: 'Requests',
                            data: llmUsageHistory.requests,
                            borderColor: '#e0af68',
                            backgroundColor: 'rgba(224,175,104,0.1)',
                            fill: false,
                            tension: 0.4,
                            pointRadius: 2,
                            pointHoverRadius: 4,
                            yAxisID: 'y'
                        },
                        {
                            label: 'Tokens',
                            data: llmUsageHistory.tokens,
                            borderColor: '#9ece6a',
                            backgroundColor: 'rgba(158,206,106,0.1)',
                            fill: true,
                            tension: 0.4,
                            pointRadius: 0,
                            pointHoverRadius: 4,
                            yAxisID: 'y1'
                        }
                    ]
                },
                options: opts
            });
        }
    }
}

function closeMetrics() {
    document.getElementById('metrics-modal').classList.remove('active');
    if (metricsInterval) { clearInterval(metricsInterval); metricsInterval = null; }
    metricsHistoryLoaded = false;
    // Reset history to avoid stale data on reopen
    metricsHistory = { cpu: [], memory: [], timestamps: [], pods: [], nodes: [] };
    // Destroy all charts so they're recreated fresh on next open
    if (cpuChart) { cpuChart.destroy(); cpuChart = null; }
    if (memoryChart) { memoryChart.destroy(); memoryChart = null; }
    if (llmUsageChart) { llmUsageChart.destroy(); llmUsageChart = null; }
}

async function collectMetricsNow() {
    try {
        const resp = await fetchWithAuth('/api/metrics/collect', { method: 'POST' });
        const data = await resp.json();
        if (data.success) {
            // Reload data after collection completes
            await loadHistoricalMetrics();
            await loadMetrics();
        }
    } catch (e) {
        console.error('Failed to trigger metrics collection:', e);
    }
}

// ==========================================
// Port Forward Functions
// ==========================================
let currentPfPod = null, currentPfNamespace = null, activePortForwards = [];

function openPortForward(podName, namespace) {
    currentPfPod = podName; currentPfNamespace = namespace;
    document.getElementById('pf-target').value = `${namespace}/${podName}`;
    document.getElementById('portforward-modal').classList.add('active');
    loadActivePortForwards();
}

async function startPortForward() {
    const localPort = document.getElementById('pf-local-port').value;
    const remotePort = document.getElementById('pf-remote-port').value;
    if (!localPort || !remotePort) { alert('Please enter both ports'); return; }
    try {
        const resp = await fetchWithAuth('/api/portforward/start', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ namespace: currentPfNamespace, pod: currentPfPod, localPort: parseInt(localPort), remotePort: parseInt(remotePort) })
        });
        const data = await resp.json();
        if (data.error) alert('Error: ' + data.error);
        else { showToast(`Port forward started: localhost:${localPort}`); loadActivePortForwards(); }
    } catch (e) { alert('Failed: ' + e.message); }
}

async function loadActivePortForwards() {
    try {
        const resp = await fetchWithAuth('/api/portforward/list');
        activePortForwards = (await resp.json()).items || [];
        renderPortForwardList();
    } catch (e) { console.error(e); }
}

function renderPortForwardList() {
    const list = document.getElementById('portforward-list');
    if (activePortForwards.length === 0) { list.innerHTML = '<p style="color:var(--text-secondary);text-align:center;padding:20px;">No active port forwards</p>'; return; }
    list.innerHTML = activePortForwards.map(pf => `<div class="portforward-item"><div class="info"><div class="ports">localhost:${parseInt(pf.localPort) || 0} → :${parseInt(pf.remotePort) || 0}</div><div class="target">${escapeHtml(pf.namespace)}/${escapeHtml(pf.pod)}</div></div><div class="status"><span class="status-dot ${pf.active ? 'active' : 'stopped'}"></span><button onclick="stopPortForward('${escapeHtml(pf.id)}')">Stop</button></div></div>`).join('');
}

async function stopPortForward(id) { try { await fetchWithAuth(`/api/portforward/${id}`, { method: 'DELETE' }); loadActivePortForwards(); } catch (e) { console.error(e); } }
function closePortForward() { document.getElementById('portforward-modal').classList.remove('active'); }

// ==========================================
// AI-Dashboard Interactive Actions
// ==========================================
function executeAIAction(action) {
    switch (action.type) {
        case 'navigate': navigateToResource(action.target, action.params); break;
        case 'highlight': highlightResource(action.target, action.params); break;
        case 'open_terminal': openTerminal(action.params.pod, action.params.namespace, action.params.container); break;
        case 'show_logs': fetchPodContainers(action.params.pod, action.params.namespace).then(c => openLogViewer(action.params.pod, action.params.namespace, c)); break;
        case 'show_metrics': showMetrics(); break;
        case 'port_forward': openPortForward(action.params.pod, action.params.namespace); break;
    }
    showToast(`AI Action: ${action.type}`);
}

function navigateToResource(resource, params) {
    switchResource(resource);
    if (params.namespace) { document.getElementById('namespace-select').value = params.namespace; currentNamespace = params.namespace; }
    if (params.filter) { document.getElementById('filter-input').value = params.filter; }
    loadData();
}

function highlightResource(resourceType, params) {
    setTimeout(() => {
        document.querySelectorAll('#table-body tr').forEach(row => {
            const nameCell = row.querySelector('td:first-child');
            if (nameCell && nameCell.textContent.includes(params.name)) {
                row.classList.add('ai-highlight');
                row.scrollIntoView({ behavior: 'smooth', block: 'center' });
            }
        });
    }, 500);
}

async function fetchPodContainers(podName, namespace) {
    try { const resp = await fetchWithAuth(`/api/k8s/pods/${namespace}/${podName}`); return (await resp.json()).containers || ['default']; }
    catch (e) { return ['default']; }
}

function showToast(message, type) {
    const toast = document.createElement('div');
    toast.className = 'ai-action-toast'; toast.textContent = message;
    if (type === 'error') {
        toast.style.background = 'var(--accent-red)';
        toast.style.color = '#fff';
    }
    document.body.appendChild(toast);
    setTimeout(() => toast.remove(), type === 'error' ? 5000 : 3000);
}

// ==========================================
// Enhanced renderTable with action buttons
// ==========================================
// Add ACTIONS column to workload and networking types
['pods', 'deployments', 'daemonsets', 'statefulsets', 'replicasets', 'services', 'ingresses'].forEach(resource => {
    if (!tableHeaders[resource].includes('ACTIONS')) {
        tableHeaders[resource].push('ACTIONS');
    }
});

// Generate HTML for a single table row
function generateRowHTML(resource, item, index) {
    const headers = tableHeaders[resource];
    switch (resource) {
            case 'pods':
                const podContainers = item.containers || ['default'];
                const podContainersJson = JSON.stringify(podContainers).replace(/'/g, "\\'");
                return `<tr data-index="${index}" data-containers='${podContainersJson}'><td>${item.name}</td><td>${item.namespace}</td><td>${item.ready}</td><td class="status-${item.status.toLowerCase()}">${item.status}</td><td>${item.restarts}</td><td>${item.age}</td><td>${item.ip || '-'}</td><td class="resource-actions"><button class="resource-action-btn terminal" onclick="event.stopPropagation(); openTerminal('${item.name}', '${item.namespace}')">Terminal</button><button class="resource-action-btn logs" onclick="event.stopPropagation(); openLogViewerFromRow(this, '${item.name}', '${item.namespace}')">Logs</button><button class="resource-action-btn portforward" onclick="event.stopPropagation(); openPortForward('${item.name}', '${item.namespace}')">Forward</button><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Pod', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
            case 'deployments':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.ready}</td><td>${item.upToDate || item.up_to_date || '-'}</td><td>${item.available || '-'}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Deployment', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
            case 'daemonsets':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.desired || '-'}</td><td>${item.current || '-'}</td><td>${item.ready || '-'}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('DaemonSet', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
            case 'statefulsets':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.ready || '-'}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('StatefulSet', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
            case 'replicasets':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.desired || '-'}</td><td>${item.current || '-'}</td><td>${item.ready || '-'}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('ReplicaSet', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
            case 'jobs':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.completions || '-'}</td><td>${item.duration || '-'}</td><td>${item.age}</td></tr>`;
            case 'cronjobs':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.schedule || '-'}</td><td>${item.suspend ? 'Yes' : 'No'}</td><td>${item.active || 0}</td><td>${item.lastSchedule || '-'}</td></tr>`;
            case 'services':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.type}</td><td>${item.clusterIP}</td><td>${item.ports}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Service', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
            case 'ingresses':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.class || item.ingressClass || '-'}</td><td>${item.hosts || '-'}</td><td>${item.address || '-'}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Ingress', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
            case 'networkpolicies':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.podSelector || '-'}</td><td>${item.age}</td></tr>`;
            case 'configmaps':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.data || item.dataCount || 0}</td><td>${item.age}</td></tr>`;
            case 'secrets':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.type || '-'}</td><td>${item.data || item.dataCount || 0}</td><td>${item.age}</td></tr>`;
            case 'serviceaccounts':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.secrets || 0}</td><td>${item.age}</td></tr>`;
            case 'persistentvolumes':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.capacity || '-'}</td><td>${item.accessModes || '-'}</td><td>${item.reclaimPolicy || '-'}</td><td class="status-${(item.status || '').toLowerCase()}">${item.status || '-'}</td><td>${item.claim || '-'}</td></tr>`;
            case 'persistentvolumeclaims':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td class="status-${(item.status || '').toLowerCase()}">${item.status || '-'}</td><td>${item.volume || '-'}</td><td>${item.capacity || '-'}</td><td>${item.accessModes || '-'}</td></tr>`;
            case 'nodes':
                return `<tr data-index="${index}"><td>${item.name}</td><td class="status-${(item.status || '').toLowerCase()}">${item.status}</td><td>${item.roles || '-'}</td><td>${item.version || '-'}</td><td>${item.age}</td></tr>`;
            case 'namespaces':
                return `<tr data-index="${index}"><td>${item.name}</td><td class="status-active">${item.status}</td><td>${item.age}</td></tr>`;
            case 'events':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.type}</td><td>${item.reason}</td><td>${item.message?.substring(0, 50) || '-'}...</td><td>${item.count}</td><td>${item.lastSeen}</td></tr>`;
            case 'roles':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.age}</td></tr>`;
            case 'rolebindings':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.role || '-'}</td><td>${item.age}</td></tr>`;
            case 'clusterroles':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.age}</td></tr>`;
            case 'clusterrolebindings':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.role || '-'}</td><td>${item.age}</td></tr>`;
            case 'hpa':
                return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.reference || '-'}</td><td>${item.minReplicas || '-'}</td><td>${item.maxReplicas || '-'}</td><td>${item.replicas || '-'}</td><td>${item.age}</td></tr>`;
            default:
                // Handle CRDs and unknown types with fallback
                // Generic fallback for CRDs and unknown resource types
                const values = (headers || ['NAME']).map(h => {
                    const key = h.toLowerCase().replace(/[- ]/g, '');
                    return item[key] || item[h] || item.name || '-';
                });
                return `<tr data-index="${index}">${values.map(v => `<td>${escapeHtml(String(v))}</td>`).join('')}</tr>`;
    }
}

// Render table body using generateRowHTML
function renderTableBody(resource, items) {
    const headers = tableHeaders[resource];
    if (!items || items.length === 0) {
        document.getElementById('table-body').innerHTML =
            `<tr><td colspan="${headers ? headers.length : 1}" style="text-align:center;padding:40px;">No ${resource} found</td></tr>`;
        return;
    }
    document.getElementById('table-body').innerHTML = items.map((item, index) => generateRowHTML(resource, item, index)).join('');
    addRowClickHandlers();
}

// Add Metrics nav item
setTimeout(() => {
    // Find Monitoring section by its title text
    const navSections = document.querySelectorAll('.nav-section');
    let monitoringSection = null;
    for (const section of navSections) {
        const title = section.querySelector('.nav-title');
        if (title && title.textContent.trim() === 'Monitoring') {
            monitoringSection = section;
            break;
        }
    }
    if (monitoringSection && !document.querySelector('[onclick="showMetrics()"]')) {
        const metricsItem = document.createElement('div');
        metricsItem.className = 'nav-item';
        metricsItem.onclick = showMetrics;
        metricsItem.innerHTML = '<span>Metrics</span>';
        const firstChild = monitoringSection.querySelector('.nav-item');
        if (firstChild) monitoringSection.insertBefore(metricsItem, firstChild);
    }
}, 100);

// ==========================================
// Cluster Overview
// ==========================================

function loadOverviewData() {
    loadClusterOverview();
    loadRecentEvents();
}

async function loadClusterOverview() {
    const cacheKey = buildScopedCacheKey('overview', currentNamespace || 'all');
    const policy = {
        ...overviewCachePolicy,
        cacheKey,
    };

    const renderOverview = (data) => {
        if (!data) return;

        const ctxEl = document.getElementById('overview-context');
        if (ctxEl && data.context) {
            ctxEl.textContent = data.context;
        }

        const nodesEl = document.getElementById('ov-nodes-ready');
        if (nodesEl) nodesEl.textContent = `${data.nodes?.ready || 0}/${data.nodes?.total || 0}`;

        const podsEl = document.getElementById('ov-pods-running');
        if (podsEl) podsEl.textContent = `${data.pods?.running || 0}/${data.pods?.total || 0}`;

        const deployEl = document.getElementById('ov-deploy-healthy');
        if (deployEl) deployEl.textContent = `${data.deployments?.healthy || 0}/${data.deployments?.total || 0}`;

        const nsEl = document.getElementById('ov-namespaces');
        if (nsEl) nsEl.textContent = `${data.namespaces || 0}`;
    };

    const preview = K13D.SWR?.peekJSON(cacheKey, policy);
    if (preview?.data) {
        renderOverview(preview.data);
    }

    try {
        const result = await K13D.SWR.fetchJSON('/api/overview', {}, policy);
        renderOverview(result.data);
        if (result.revalidatePromise) {
            result.revalidatePromise.then((revalidated) => {
                renderOverview(revalidated.data);
            }).catch((e) => {
                console.error('Failed to refresh cluster overview:', e);
            });
        }
    } catch (e) {
        console.error('Failed to load cluster overview:', e);
    }
}

async function loadRecentEvents() {
    const cacheKey = buildScopedCacheKey('overview-events', currentNamespace || 'all');
    const policy = {
        ...resourceCachePolicy,
        cacheKey,
    };

    const renderEvents = (data) => {
        const eventsEl = document.getElementById('overview-events');
        if (!eventsEl) return;

        const events = (data?.items || []).slice(0, 10);
        if (events.length === 0) {
            eventsEl.innerHTML = '<p class="text-muted">No recent events</p>';
            return;
        }

        eventsEl.innerHTML = events.map(evt => {
            const typeLower = (evt.type || 'normal').toLowerCase();
            const typeClass = typeLower === 'warning' ? 'warning' : 'normal';
            const msg = K13D?.Utils?.escapeHtml ? K13D.Utils.escapeHtml(evt.message || '') : (evt.message || '');
            return `<div class="overview-event-item ${typeClass}">
                        <span class="event-type ${typeClass}">${evt.type || 'Normal'}</span>
                        <span class="event-message">${msg.substring(0, 120)}${msg.length > 120 ? '...' : ''}</span>
                        <span class="event-time">${evt.lastSeen || evt.age || ''}</span>
                    </div>`;
        }).join('');
    };

    const preview = K13D.SWR?.peekJSON(cacheKey, policy);
    if (preview?.data) {
        renderEvents(preview.data);
    }

    try {
        const result = await K13D.SWR.fetchJSON('/api/k8s/events?namespace=', {}, policy);
        renderEvents(result.data);
        if (result.revalidatePromise) {
            result.revalidatePromise.then((revalidated) => {
                renderEvents(revalidated.data);
            }).catch((e) => {
                console.error('Failed to refresh recent events:', e);
            });
        }
    } catch (e) {
        console.error('Failed to load recent events:', e);
    }
}

// ============================
// Helm View
// ============================
function showHelmView() {
    showCustomView('helm-container', 'helm');
    loadHelmData();
}

async function loadHelmData() {
    const body = document.getElementById('helm-body');
    const ns = document.getElementById('helm-ns-select')?.value || '';
    body.innerHTML = '<div class="loading-placeholder">Loading Helm releases...</div>';
    try {
        let params = ns ? `?namespace=${encodeURIComponent(ns)}` : '?all=true';
        const resp = await fetchWithAuth(`/api/helm/releases${params}`);
        const data = await resp.json();
        const items = data.items || [];
        if (items.length === 0) {
            body.innerHTML = '<div class="loading-placeholder">No Helm releases found.</div>';
            return;
        }
        body.innerHTML = `
                    <table class="helm-table">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Namespace</th>
                                <th>Revision</th>
                                <th>Status</th>
                                <th>Chart</th>
                                <th>App Version</th>
                                <th>Updated</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${items.map(r => `
                                <tr>
                                    <td style="font-weight:600;color:var(--accent-cyan);cursor:pointer;" onclick="showHelmReleaseDetail('${escapeHtml(r.name)}','${escapeHtml(r.namespace || '')}')">${escapeHtml(r.name)}</td>
                                    <td>${escapeHtml(r.namespace || '-')}</td>
                                    <td>${r.revision || '-'}</td>
                                    <td><span class="helm-status ${(r.status || '').toLowerCase()}">${escapeHtml(r.status || '-')}</span></td>
                                    <td style="font-family:monospace;">${escapeHtml(r.chart || '-')}</td>
                                    <td>${escapeHtml(r.appVersion || '-')}</td>
                                    <td style="color:var(--text-secondary);">${r.updated ? formatDateTime(r.updated) : '-'}</td>
                                    <td class="helm-actions">
                                        <button onclick="showHelmReleaseDetail('${escapeHtml(r.name)}','${escapeHtml(r.namespace || '')}')">Details</button>
                                        <button onclick="helmRollback('${escapeHtml(r.name)}','${escapeHtml(r.namespace || '')}')">Rollback</button>
                                        <button style="color:var(--accent-red);" onclick="helmUninstall('${escapeHtml(r.name)}','${escapeHtml(r.namespace || '')}')">Uninstall</button>
                                    </td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                    <div id="helm-detail-area"></div>`;
    } catch (e) {
        body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load Helm releases: ${escapeHtml(e.message)}</div>`;
    }
}

async function showHelmReleaseDetail(name, namespace) {
    const area = document.getElementById('helm-detail-area');
    if (!area) return;
    area.innerHTML = '<div class="loading-placeholder">Loading release details...</div>';
    try {
        const nsParam = namespace ? `?namespace=${encodeURIComponent(namespace)}` : '';
        const [valuesResp, historyResp] = await Promise.all([
            fetchWithAuth(`/api/helm/release/${encodeURIComponent(name)}/values${nsParam}&all=true`),
            fetchWithAuth(`/api/helm/release/${encodeURIComponent(name)}/history${nsParam}`)
        ]);
        const values = await valuesResp.json();
        const history = await historyResp.json();
        const historyItems = history.items || [];

        area.innerHTML = `
                    <div class="helm-detail-panel">
                        <h3>Release: ${escapeHtml(name)} (${escapeHtml(namespace || 'default')})</h3>
                        <div style="display:flex;gap:16px;margin-bottom:16px;">
                            <button onclick="showHelmDetailTab('values','${escapeHtml(name)}','${escapeHtml(namespace)}')" style="padding:6px 14px;border-radius:6px;border:1px solid var(--border-color);background:var(--accent-blue);color:#fff;cursor:pointer;">Values</button>
                            <button onclick="showHelmDetailTab('history','${escapeHtml(name)}','${escapeHtml(namespace)}')" style="padding:6px 14px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-tertiary);color:var(--text-primary);cursor:pointer;">History</button>
                            <button onclick="showHelmDetailTab('manifest','${escapeHtml(name)}','${escapeHtml(namespace)}')" style="padding:6px 14px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-tertiary);color:var(--text-primary);cursor:pointer;">Manifest</button>
                        </div>
                        <div id="helm-detail-content">
                            <pre>${escapeHtml(JSON.stringify(values, null, 2))}</pre>
                        </div>
                        ${historyItems.length > 0 ? `
                        <div style="margin-top:16px;">
                            <h4 style="margin-bottom:8px;">Revision History</h4>
                            <table class="helm-table" style="font-size:12px;">
                                <thead><tr><th>Rev</th><th>Status</th><th>Chart</th><th>Description</th><th>Updated</th></tr></thead>
                                <tbody>
                                    ${historyItems.map(h => `
                                        <tr>
                                            <td>${h.revision || '-'}</td>
                                            <td><span class="helm-status ${(h.status || '').toLowerCase()}">${escapeHtml(h.status || '')}</span></td>
                                            <td style="font-family:monospace;">${escapeHtml(h.chart || '-')}</td>
                                            <td>${escapeHtml(h.description || '-')}</td>
                                            <td style="color:var(--text-secondary);">${h.updated ? formatDateTime(h.updated) : '-'}</td>
                                        </tr>
                                    `).join('')}
                                </tbody>
                            </table>
                        </div>` : ''}
                    </div>`;
    } catch (e) {
        area.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load details: ${escapeHtml(e.message)}</div>`;
    }
}

async function showHelmDetailTab(tab, name, namespace) {
    const content = document.getElementById('helm-detail-content');
    if (!content) return;
    content.innerHTML = '<div class="loading-placeholder">Loading...</div>';
    const nsParam = namespace ? `?namespace=${encodeURIComponent(namespace)}` : '';
    try {
        if (tab === 'values') {
            const resp = await fetchWithAuth(`/api/helm/release/${encodeURIComponent(name)}/values${nsParam}&all=true`);
            const data = await resp.json();
            content.innerHTML = `<pre>${escapeHtml(JSON.stringify(data, null, 2))}</pre>`;
        } else if (tab === 'manifest') {
            const resp = await fetchWithAuth(`/api/helm/release/${encodeURIComponent(name)}/manifest${nsParam}`);
            const text = await resp.text();
            content.innerHTML = `<pre>${escapeHtml(text)}</pre>`;
        } else if (tab === 'history') {
            const resp = await fetchWithAuth(`/api/helm/release/${encodeURIComponent(name)}/history${nsParam}`);
            const data = await resp.json();
            content.innerHTML = `<pre>${escapeHtml(JSON.stringify(data, null, 2))}</pre>`;
        }
    } catch (e) {
        content.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed: ${escapeHtml(e.message)}</div>`;
    }
}

async function helmRollback(name, namespace) {
    const revision = prompt(`Rollback "${name}" to which revision? (Enter revision number)`);
    if (!revision) return;
    try {
        await fetchWithAuth('/api/helm/rollback', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, namespace, revision: parseInt(revision) })
        });
        alert(`Release "${name}" rolled back to revision ${revision}`);
        loadHelmData();
    } catch (e) {
        alert('Rollback failed: ' + e.message);
    }
}

async function helmUninstall(name, namespace) {
    if (!confirm(`Uninstall Helm release "${name}" from "${namespace || 'default'}"?`)) return;
    try {
        await fetchWithAuth('/api/helm/uninstall', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, namespace })
        });
        alert(`Release "${name}" uninstalled`);
        loadHelmData();
    } catch (e) {
        alert('Uninstall failed: ' + e.message);
    }
}

// ============================
// RBAC Visualization View
// ============================
function showRBACVizView() {
    showCustomView('rbac-viz-container', 'rbac-viz');
    loadRBACVizData();
}

async function loadRBACVizData() {
    const body = document.getElementById('rbac-viz-body');
    const ns = document.getElementById('rbac-viz-ns-select')?.value || '';
    const filter = document.getElementById('rbac-viz-filter')?.value || '';
    body.innerHTML = '<div class="loading-placeholder">Loading RBAC data...</div>';
    try {
        let url = '/api/rbac/visualization';
        const params = [];
        if (ns) params.push(`namespace=${encodeURIComponent(ns)}`);
        if (filter) params.push(`subject_kind=${encodeURIComponent(filter)}`);
        if (params.length) url += '?' + params.join('&');
        const resp = await fetchWithAuth(url);
        const data = await resp.json();
        if (!data.subjects || data.subjects.length === 0) {
            body.innerHTML = '<div class="loading-placeholder">No RBAC bindings found</div>';
            return;
        }
        body.innerHTML = data.subjects.map(s => {
            const iconClass = s.kind === 'ServiceAccount' ? 'sa' : s.kind === 'User' ? 'user' : 'group';
            const initial = s.kind === 'ServiceAccount' ? 'SA' : s.kind === 'User' ? 'U' : 'G';
            return `<div class="rbac-card" style="cursor:pointer;" onclick="showRBACSubjectDetail('${escapeHtml(s.name)}','${escapeHtml(s.kind)}','${escapeHtml(s.namespace || '')}')">
                        <div class="rbac-card-header">
                            <div class="rbac-subject-icon ${iconClass}">${initial}</div>
                            <div>
                                <div style="font-weight:600;">${escapeHtml(s.name)}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">${escapeHtml(s.kind)}${s.namespace ? ' · ' + escapeHtml(s.namespace) : ''}</div>
                            </div>
                        </div>
                        <div style="display:flex;flex-wrap:wrap;gap:4px;">
                            ${(s.roles || []).map(r => `<span class="rbac-role-badge ${r.cluster_scope ? 'cluster' : ''}">${r.cluster_scope ? '⊕ ' : ''}${escapeHtml(r.role_name)}</span>`).join('')}
                        </div>
                    </div>`;
        }).join('');
    } catch (e) {
        body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load RBAC: ${escapeHtml(e.message)}</div>`;
    }
}

async function showRBACSubjectDetail(name, kind, namespace) {
    const modal = document.getElementById('rbac-detail-modal');
    const title = document.getElementById('rbac-detail-title');
    const content = document.getElementById('rbac-detail-content');
    title.textContent = `${kind}: ${name}`;
    content.innerHTML = '<div class="loading-placeholder">Loading subject details...</div>';
    modal.classList.add('active');

    try {
        const params = new URLSearchParams({ name, kind });
        if (namespace) params.set('namespace', namespace);
        const resp = await fetchWithAuth(`/api/rbac/subject/detail?${params}`);
        const data = await resp.json();

        if (!data.bindings || data.bindings.length === 0) {
            content.innerHTML = '<div class="loading-placeholder">No bindings found for this subject</div>';
            return;
        }

        content.innerHTML = data.bindings.map(b => `
                    <div style="border:1px solid var(--border-color);border-radius:8px;padding:12px;margin-bottom:12px;">
                        <div style="display:flex;align-items:center;gap:8px;margin-bottom:8px;">
                            <span style="font-weight:600;color:var(--accent-blue);">${escapeHtml(b.role_name)}</span>
                            <span style="font-size:11px;padding:2px 6px;border-radius:4px;background:var(--bg-tertiary);color:var(--text-secondary);">${escapeHtml(b.role_kind)}</span>
                            <span style="font-size:11px;color:var(--text-secondary);">via ${escapeHtml(b.binding_kind)}: ${escapeHtml(b.binding_name)}</span>
                            ${b.namespace ? `<span style="font-size:11px;color:var(--text-secondary);">(${escapeHtml(b.namespace)})</span>` : ''}
                        </div>
                        ${(b.rules && b.rules.length > 0) ? `
                        <table style="width:100%;font-size:12px;border-collapse:collapse;">
                            <thead>
                                <tr style="border-bottom:1px solid var(--border-color);">
                                    <th style="text-align:left;padding:4px 8px;color:var(--text-secondary);">Verbs</th>
                                    <th style="text-align:left;padding:4px 8px;color:var(--text-secondary);">Resources</th>
                                    <th style="text-align:left;padding:4px 8px;color:var(--text-secondary);">API Groups</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${b.rules.map(r => `<tr style="border-bottom:1px solid var(--border-color);">
                                    <td style="padding:4px 8px;">${(r.verbs || []).map(v => `<span style="padding:1px 4px;border-radius:3px;background:var(--bg-tertiary);margin-right:2px;">${escapeHtml(v)}</span>`).join(' ')}</td>
                                    <td style="padding:4px 8px;">${(r.resources || []).join(', ')}</td>
                                    <td style="padding:4px 8px;">${(r.api_groups || []).map(g => g || 'core').join(', ')}</td>
                                </tr>`).join('')}
                            </tbody>
                        </table>` : '<div style="font-size:12px;color:var(--text-secondary);">No rules defined</div>'}
                    </div>
                `).join('');
    } catch (e) {
        content.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load details: ${escapeHtml(e.message)}</div>`;
    }
}

function viewNetPolInTopology(namespace, name) {
    // Switch to topology view with Network Policies enabled and focus on the specific policy
    const netpolCheckbox = document.getElementById('topology-show-netpol');
    if (netpolCheckbox) netpolCheckbox.checked = true;
    const nsSelect = document.getElementById('topology-ns-select');
    if (nsSelect && namespace) nsSelect.value = namespace;
    if (name) {
        topologyFocusNodeId = `NetworkPolicy/${namespace}/${name}`;
    }
    showTopology();
}

// ============================
// Network Policy Visualization
// ============================
function showNetPolVizView() {
    showCustomView('netpol-viz-container', 'netpol-viz');
    loadNetPolVizData();
}

async function loadNetPolVizData() {
    const body = document.getElementById('netpol-viz-body');
    const ns = document.getElementById('netpol-viz-ns-select')?.value || '';
    body.innerHTML = '<div class="loading-placeholder">Loading network policies...</div>';
    try {
        const params = ns ? `?namespace=${encodeURIComponent(ns)}` : '';
        const resp = await fetchWithAuth(`/api/netpol/visualization${params}`);
        const data = await resp.json();
        if (!data.policies || data.policies.length === 0) {
            body.innerHTML = '<div class="loading-placeholder">No network policies found</div>';
            return;
        }
        body.innerHTML = data.policies.map(p => `
                    <div class="netpol-card">
                        <div class="netpol-card-header">
                            <div>
                                <div style="font-weight:600;">${escapeHtml(p.name)}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">${escapeHtml(p.namespace)}</div>
                            </div>
                            <div style="display:flex;align-items:center;gap:8px;">
                                <div class="netpol-selector">Selector: ${escapeHtml(p.pod_selector || '*')}</div>
                                <button onclick="viewNetPolInTopology('${escapeHtml(p.namespace)}','${escapeHtml(p.name)}')" style="padding:4px 10px;font-size:11px;border-radius:4px;border:1px solid var(--accent-blue);background:transparent;color:var(--accent-blue);cursor:pointer;white-space:nowrap;">View in Topology</button>
                            </div>
                        </div>
                        <div class="netpol-direction">
                            <div class="netpol-direction-col">
                                <div class="netpol-direction-label">↓ Ingress (${(p.ingress_rules || []).length} rules)</div>
                                ${(p.ingress_rules || []).map(r => `<div class="netpol-rule">${escapeHtml(r)}</div>`).join('') || '<div style="font-size:12px;color:var(--text-secondary);">No ingress rules</div>'}
                            </div>
                            <div class="netpol-direction-col">
                                <div class="netpol-direction-label">↑ Egress (${(p.egress_rules || []).length} rules)</div>
                                ${(p.egress_rules || []).map(r => `<div class="netpol-rule">${escapeHtml(r)}</div>`).join('') || '<div style="font-size:12px;color:var(--text-secondary);">No egress rules</div>'}
                            </div>
                        </div>
                    </div>
                `).join('');
    } catch (e) {
        body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed: ${escapeHtml(e.message)}</div>`;
    }
}

// ============================
// Event Timeline View
// ============================
function showTimelineView() {
    showCustomView('timeline-container', 'timeline');
    loadTimelineData();
}

async function loadTimelineData() {
    const body = document.getElementById('timeline-body');
    const ns = document.getElementById('timeline-ns-select')?.value || '';
    const hours = document.getElementById('timeline-hours')?.value || '24';
    const warningsOnly = document.getElementById('timeline-warnings-only')?.checked || false;
    body.innerHTML = '<div class="loading-placeholder">Loading events...</div>';
    try {
        const params = new URLSearchParams();
        if (ns) params.set('namespace', ns);
        params.set('hours', hours);
        if (warningsOnly) params.set('warnings_only', 'true');
        const resp = await fetchWithAuth(`/api/events/timeline?${params}`);
        const data = await resp.json();
        const totalEvents = (data.totalNormal || 0) + (data.totalWarning || 0);
        let html = `<div class="timeline-stats">
                    <div class="timeline-stat"><div class="timeline-stat-value">${totalEvents}</div><div class="timeline-stat-label">Total Events</div></div>
                    <div class="timeline-stat"><div class="timeline-stat-value" style="color:var(--accent-green);">${data.totalNormal || 0}</div><div class="timeline-stat-label">Normal</div></div>
                    <div class="timeline-stat"><div class="timeline-stat-value" style="color:var(--accent-red);">${data.totalWarning || 0}</div><div class="timeline-stat-label">Warning</div></div>
                </div>`;
        if (data.windows && data.windows.length > 0) {
            html += '<div class="timeline-container"><div class="timeline-line"></div>';
            data.windows.forEach(w => {
                const dotClass = w.warningCount > 0 ? 'warning' : '';
                const windowTime = formatTime(w.timestamp);
                const windowCount = (w.normalCount || 0) + (w.warningCount || 0);
                html += `<div class="timeline-group">
                            <div class="timeline-dot ${dotClass}"></div>
                            <div class="timeline-time">${escapeHtml(windowTime)} (${windowCount} events)</div>
                            ${(w.events || []).slice(0, 10).map(e => `
                                <div class="timeline-event">
                                    <span class="evt-type ${escapeHtml(e.type)}">${escapeHtml(e.type)}</span>
                                    <span class="evt-reason">${escapeHtml(e.reason || '')}</span>
                                    <span class="evt-msg">${escapeHtml(e.message || '').substring(0, 120)}</span>
                                </div>
                            `).join('')}
                            ${(w.events || []).length > 10 ? `<div style="font-size:11px;color:var(--text-secondary);padding:4px 12px;">...and ${(w.events || []).length - 10} more</div>` : ''}
                        </div>`;
            });
            html += '</div>';
        } else {
            html += '<div class="loading-placeholder">No events in the selected time range</div>';
        }
        body.innerHTML = html;
    } catch (e) {
        body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed: ${escapeHtml(e.message)}</div>`;
    }
}

// ============================
// GitOps View
// ============================
function showGitOpsView() {
    showCustomView('gitops-container', 'gitops');
    loadGitOpsData();
}

async function loadGitOpsData() {
    const body = document.getElementById('gitops-body');
    const ns = document.getElementById('gitops-ns-select')?.value || '';
    body.innerHTML = '<div class="loading-placeholder">Loading GitOps status...</div>';
    try {
        const params = ns ? `?namespace=${encodeURIComponent(ns)}` : '';
        const resp = await fetchWithAuth(`/api/gitops/status${params}`);
        const data = await resp.json();
        const argoApps = data.argocd || [];
        const fluxApps = data.flux || [];
        if (argoApps.length === 0 && fluxApps.length === 0) {
            body.innerHTML = `<div class="gitops-empty">
                        <div style="font-size:48px;margin-bottom:16px;">🔄</div>
                        <h3>No GitOps Resources Found</h3>
                        <p style="margin-top:8px;">${escapeHtml(data.message || 'Install ArgoCD or Flux to enable GitOps features')}</p>
                    </div>`;
            return;
        }
        let html = '';
        if (argoApps.length > 0) {
            html += `<h3 style="margin-bottom:12px;display:flex;align-items:center;gap:8px;"><span style="font-size:20px;">🐙</span> ArgoCD Applications (${argoApps.length})</h3>`;
            html += argoApps.map(a => {
                const syncStatus = (a.syncStatus || 'unknown').toLowerCase();
                const dotClass = syncStatus === 'synced' ? 'synced' : syncStatus === 'outofsync' ? 'outofsync' : 'unknown';
                return `<div class="gitops-card">
                            <div class="gitops-status-dot ${dotClass}"></div>
                            <div class="gitops-info">
                                <div class="gitops-name">${escapeHtml(a.name)}</div>
                                <div class="gitops-meta">${escapeHtml(a.namespace)} · Health: ${escapeHtml(a.status || 'Unknown')}</div>
                                <div class="gitops-repo">${escapeHtml(a.source || '')}</div>
                            </div>
                            <span class="gitops-badge ${dotClass}">${escapeHtml(a.syncStatus || 'Unknown')}</span>
                        </div>`;
            }).join('');
        }
        if (fluxApps.length > 0) {
            html += `<h3 style="margin:20px 0 12px;display:flex;align-items:center;gap:8px;"><span style="font-size:20px;">🌊</span> Flux Kustomizations (${fluxApps.length})</h3>`;
            html += fluxApps.map(f => {
                const isReady = f.status === 'Ready';
                const readyClass = isReady ? 'synced' : 'degraded';
                return `<div class="gitops-card">
                            <div class="gitops-status-dot ${readyClass}"></div>
                            <div class="gitops-info">
                                <div class="gitops-name">${escapeHtml(f.name)}</div>
                                <div class="gitops-meta">${escapeHtml(f.namespace)} · Source: ${escapeHtml(f.source || '')}</div>
                            </div>
                            <span class="gitops-badge ${readyClass}">${isReady ? 'Ready' : 'Not Ready'}</span>
                        </div>`;
            }).join('');
        }
        body.innerHTML = html;
    } catch (e) {
        body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed: ${escapeHtml(e.message)}</div>`;
    }
}

// ============================
// Resource Diff Modal
// ============================
async function showResourceDiff(kind, name, namespace) {
    document.getElementById('diff-resource-label').textContent = `${kind}/${name} (${namespace || 'default'})`;
    document.getElementById('diff-left').textContent = 'Loading...';
    document.getElementById('diff-right').textContent = 'Loading...';
    document.getElementById('diff-modal').classList.add('active');
    try {
        const resp = await fetchWithAuth('/api/diff', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ resource: kind, name, namespace })
        });
        const data = await resp.json();
        document.getElementById('diff-left').textContent = data.lastApplied || '(no last-applied annotation found)';
        document.getElementById('diff-right').textContent = data.currentYaml || '(failed to get current)';
    } catch (e) {
        document.getElementById('diff-left').textContent = 'Error: ' + e.message;
        document.getElementById('diff-right').textContent = '';
    }
}

function closeDiffModal() {
    document.getElementById('diff-modal').classList.remove('active');
}

// ============================
// AI Auto-Troubleshoot
// ============================
async function runAutoTroubleshoot() {
    const ns = currentNamespace || '';
    const aiInput = document.getElementById('ai-input');
    const prompt = ns
        ? `Analyze namespace "${ns}" for issues. Check for CrashLoopBackOff pods, OOMKilled, pending pods, failed deployments, and recent warning events. Provide a diagnosis and remediation steps.`
        : `Analyze the entire cluster for issues. Check all namespaces for CrashLoopBackOff pods, OOMKilled, pending pods, failed deployments, and recent warning events. Provide a diagnosis and remediation steps.`;
    if (aiInput) {
        aiInput.value = prompt;
        sendMessage();
    }
}

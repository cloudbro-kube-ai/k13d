/**
 * k13d Documentation - Search Functionality
 */

(function() {
    'use strict';

    // Search index - will be populated with page data
    const searchIndex = [
        {
            title: 'Overview',
            path: 'index.html',
            content: 'k13d Kubernetes management tool TUI Web UI AI assistant kubectl intelligence',
            section: 'Getting Started'
        },
        {
            title: 'Installation',
            path: 'pages/installation.html',
            content: 'install binary docker kubernetes helm go build make',
            section: 'Getting Started'
        },
        {
            title: 'Quick Start',
            path: 'pages/quick-start.html',
            content: 'quick start tutorial getting started first steps',
            section: 'Getting Started'
        },
        {
            title: 'Configuration',
            path: 'pages/configuration.html',
            content: 'config yaml settings llm provider api key',
            section: 'Getting Started'
        },
        {
            title: 'Architecture',
            path: 'pages/architecture.html',
            content: 'architecture design system structure components',
            section: 'Concepts'
        },
        {
            title: 'AI Assistant',
            path: 'pages/ai-assistant.html',
            content: 'AI assistant chat agent tool calling kubectl bash',
            section: 'Concepts'
        },
        {
            title: 'MCP Integration',
            path: 'pages/mcp-integration.html',
            content: 'MCP model context protocol tools servers integration',
            section: 'Concepts'
        },
        {
            title: 'Security & RBAC',
            path: 'pages/security.html',
            content: 'security RBAC authentication authorization JWT tokens',
            section: 'Concepts'
        },
        {
            title: 'TUI Dashboard',
            path: 'pages/tui-guide.html',
            content: 'TUI terminal dashboard k9s vim keybindings navigation',
            section: 'User Guide'
        },
        {
            title: 'Web Dashboard',
            path: 'pages/web-guide.html',
            content: 'web dashboard browser UI streaming chat',
            section: 'User Guide'
        },
        {
            title: 'Keyboard Shortcuts',
            path: 'pages/keyboard-shortcuts.html',
            content: 'keyboard shortcuts keybindings hotkeys vim',
            section: 'User Guide'
        },
        {
            title: 'LLM Providers',
            path: 'pages/llm-providers.html',
            content: 'LLM providers OpenAI Ollama Azure Anthropic Solar',
            section: 'AI & LLM'
        },
        {
            title: 'Embedded LLM',
            path: 'pages/embedded-llm.html',
            content: 'embedded LLM llama.cpp SLLM offline air-gapped',
            section: 'AI & LLM'
        },
        {
            title: 'Tool Calling',
            path: 'pages/tool-calling.html',
            content: 'tool calling function calling kubectl bash execution',
            section: 'AI & LLM'
        },
        {
            title: 'Benchmarks',
            path: 'pages/benchmarks.html',
            content: 'benchmarks performance evaluation k8s-ai-bench',
            section: 'AI & LLM'
        },
        {
            title: 'Docker',
            path: 'pages/docker.html',
            content: 'docker container compose image deployment',
            section: 'Deployment'
        },
        {
            title: 'Kubernetes',
            path: 'pages/kubernetes.html',
            content: 'kubernetes deployment manifest yaml service',
            section: 'Deployment'
        },
        {
            title: 'Helm Chart',
            path: 'pages/helm.html',
            content: 'helm chart values installation upgrade',
            section: 'Deployment'
        },
        {
            title: 'Air-Gapped',
            path: 'pages/air-gapped.html',
            content: 'air-gapped offline disconnected installation',
            section: 'Deployment'
        },
        {
            title: 'API Reference',
            path: 'pages/api-reference.html',
            content: 'API reference endpoints REST HTTP',
            section: 'Reference'
        },
        {
            title: 'CLI Reference',
            path: 'pages/cli-reference.html',
            content: 'CLI command line flags options arguments',
            section: 'Reference'
        },
        {
            title: 'Environment Variables',
            path: 'pages/environment-variables.html',
            content: 'environment variables env config',
            section: 'Reference'
        },
        {
            title: 'Changelog',
            path: 'pages/changelog.html',
            content: 'changelog releases versions updates',
            section: 'Reference'
        }
    ];

    const searchInput = document.getElementById('search-input');
    const searchModal = document.getElementById('search-modal');
    const searchResults = document.getElementById('search-results');

    if (!searchInput || !searchModal || !searchResults) return;

    // Search function
    function search(query) {
        if (!query || query.length < 2) {
            return [];
        }

        const terms = query.toLowerCase().split(/\s+/);
        const results = [];

        searchIndex.forEach(item => {
            const searchText = `${item.title} ${item.content} ${item.section}`.toLowerCase();
            let score = 0;

            terms.forEach(term => {
                if (item.title.toLowerCase().includes(term)) {
                    score += 10;
                }
                if (searchText.includes(term)) {
                    score += 1;
                }
            });

            if (score > 0) {
                results.push({ ...item, score });
            }
        });

        return results.sort((a, b) => b.score - a.score).slice(0, 10);
    }

    // Render results
    function renderResults(results, query) {
        if (results.length === 0) {
            searchResults.innerHTML = `
                <div class="search-empty">
                    <p>No results found for "<strong>${escapeHtml(query)}</strong>"</p>
                    <p class="search-hint">Try different keywords or check the spelling.</p>
                </div>
            `;
            return;
        }

        const html = results.map(result => `
            <a href="${result.path}" class="search-result-item">
                <span class="search-result-section">${result.section}</span>
                <span class="search-result-title">${highlightMatch(result.title, query)}</span>
            </a>
        `).join('');

        searchResults.innerHTML = html;
    }

    // Highlight matching text
    function highlightMatch(text, query) {
        const terms = query.toLowerCase().split(/\s+/);
        let result = escapeHtml(text);

        terms.forEach(term => {
            const regex = new RegExp(`(${escapeRegExp(term)})`, 'gi');
            result = result.replace(regex, '<mark>$1</mark>');
        });

        return result;
    }

    // Escape HTML
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Escape RegExp special characters
    function escapeRegExp(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    }

    // Event handlers
    let debounceTimer;

    searchInput.addEventListener('input', (e) => {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(() => {
            const query = e.target.value.trim();

            if (query.length >= 2) {
                const results = search(query);
                renderResults(results, query);
                searchModal.classList.add('active');
            } else {
                searchModal.classList.remove('active');
            }
        }, 150);
    });

    searchInput.addEventListener('focus', () => {
        const query = searchInput.value.trim();
        if (query.length >= 2) {
            searchModal.classList.add('active');
        }
    });

    // Close on click outside
    document.addEventListener('click', (e) => {
        if (!searchInput.contains(e.target) && !searchModal.contains(e.target)) {
            searchModal.classList.remove('active');
        }
    });

    // Keyboard navigation
    searchInput.addEventListener('keydown', (e) => {
        const items = searchResults.querySelectorAll('.search-result-item');
        const activeItem = searchResults.querySelector('.search-result-item.active');

        if (e.key === 'ArrowDown') {
            e.preventDefault();
            if (activeItem) {
                activeItem.classList.remove('active');
                const next = activeItem.nextElementSibling || items[0];
                next?.classList.add('active');
            } else if (items.length > 0) {
                items[0].classList.add('active');
            }
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            if (activeItem) {
                activeItem.classList.remove('active');
                const prev = activeItem.previousElementSibling || items[items.length - 1];
                prev?.classList.add('active');
            } else if (items.length > 0) {
                items[items.length - 1].classList.add('active');
            }
        } else if (e.key === 'Enter') {
            e.preventDefault();
            if (activeItem) {
                window.location.href = activeItem.getAttribute('href');
            } else if (items.length > 0) {
                window.location.href = items[0].getAttribute('href');
            }
        }
    });

    // Add search result styles
    const style = document.createElement('style');
    style.textContent = `
        .search-result-item {
            display: flex;
            flex-direction: column;
            gap: 4px;
            padding: var(--spacing-md);
            border-bottom: 1px solid var(--border-light);
            transition: background var(--transition-fast);
        }
        .search-result-item:hover,
        .search-result-item.active {
            background: var(--bg-hover);
        }
        .search-result-section {
            font-size: 11px;
            text-transform: uppercase;
            color: var(--text-muted);
        }
        .search-result-title {
            font-size: 14px;
            font-weight: 500;
            color: var(--text-primary);
        }
        .search-result-title mark {
            background: rgba(122, 162, 247, 0.3);
            color: var(--accent-blue);
            border-radius: 2px;
            padding: 0 2px;
        }
        .search-empty {
            padding: var(--spacing-xl);
            text-align: center;
            color: var(--text-muted);
        }
        .search-hint {
            font-size: 13px;
            margin-top: var(--spacing-sm);
        }
    `;
    document.head.appendChild(style);
})();

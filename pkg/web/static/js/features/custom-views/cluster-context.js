async function loadClusterContexts() {
    const cacheKey = buildScopedCacheKey('contexts', 'all');
    const policy = {
        ...namespacesCachePolicy,
        cacheKey,
    };

    const renderContexts = (data) => {
        const list = document.getElementById('cluster-dropdown-list');
        const nameEl = document.getElementById('cluster-name');
        if (!list || !data) return;

        const previousContext = currentClusterContext;
        if (data.currentContext) {
            nameEl.textContent = data.currentContext;
            setCurrentClusterContext(data.currentContext);
        }

        list.innerHTML = (data.contexts || []).map((ctx, i) => `
                    <div class="cluster-dropdown-item ${ctx.name === data.currentContext ? 'active' : ''}" data-ctx-index="${i}">
                        <span class="ctx-icon"></span>
                        <div style="flex:1;">
                            <div style="font-weight:${ctx.name === data.currentContext ? '600' : '400'}">${escapeHtml(ctx.name)}</div>
                            <div style="font-size:11px;color:var(--text-secondary);">${escapeHtml(ctx.cluster || '')}</div>
                        </div>
                        ${ctx.name === data.currentContext ? '<span style="color:var(--accent-green);">●</span>' : ''}
                    </div>
                `).join('');

        list.querySelectorAll('[data-ctx-index]').forEach((el) => {
            el.addEventListener('click', () => {
                const idx = parseInt(el.dataset.ctxIndex, 10);
                const ctxName = (data.contexts || [])[idx]?.name;
                if (ctxName) switchClusterContext(ctxName);
            });
        });

        if (
            previousContext &&
            data.currentContext &&
            previousContext !== data.currentContext &&
            document.getElementById('app')?.classList.contains('active') &&
            allResources.includes(currentResource)
        ) {
            loadNamespaces({ forceNetwork: true });
            loadData({ forceNetwork: true });
        }
    };

    const preview = K13D.SWR?.peekJSON(cacheKey, policy);
    if (preview?.data) {
        renderContexts(preview.data);
    }

    try {
        const result = await K13D.SWR.fetchJSON('/api/contexts', {}, policy);
        renderContexts(result.data);
        if (result.revalidatePromise) {
            result.revalidatePromise.then((revalidated) => {
                renderContexts(revalidated.data);
            }).catch((e) => {
                console.warn('Failed to refresh contexts:', e);
            });
        }
    } catch (e) {
        console.warn('Failed to load contexts:', e);
    }
}

function toggleClusterDropdown() {
    const dd = document.getElementById('cluster-dropdown');
    if (!dd) return;
    dd.classList.toggle('active');
    if (dd.classList.contains('active')) loadClusterContexts();
}

async function switchClusterContext(name) {
    try {
        const resp = await fetchWithAuth('/api/contexts/switch', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ context: name })
        });
        if (!resp.ok) {
            const errText = await resp.text();
            throw new Error(errText || `HTTP ${resp.status}`);
        }

        const result = await resp.json();
        setCurrentClusterContext(name);
        document.getElementById('cluster-name').textContent = name;
        document.getElementById('cluster-dropdown').classList.remove('active');

        for (const resource of allResources) {
            clearResourceData(resource);
        }

        if (!result.reachable) {
            showToast(`Context "${name}" is not reachable. Check cluster connectivity.`, 'error');
            currentNamespace = '';
            document.getElementById('namespace-select').value = '';
            return;
        }

        showToast(`Switched to context: ${name}`);
        currentNamespace = '';
        document.getElementById('namespace-select').value = '';
        await loadNamespaces({ forceNetwork: true });
        syncCustomViewNamespaces();
        loadData({ forceNetwork: true });
    } catch (e) {
        alert('Failed to switch context: ' + e.message);
    }
}

document.addEventListener('click', (e) => {
    const switcher = document.getElementById('cluster-switcher');
    if (switcher && !switcher.contains(e.target)) {
        document.getElementById('cluster-dropdown')?.classList.remove('active');
    }
});

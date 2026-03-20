/**
 * Lightweight stale-while-revalidate cache for the Web UI.
 *
 * Inspired by query-client patterns used in Headlamp and stale-first mobile UX
 * used by Open WebUI, but implemented as a tiny framework-free module.
 */

(function(global) {
    'use strict';

    const CACHE_PREFIX = 'k13d_swr_v1:';
    const DEFAULT_TTL_MS = 15 * 1000;
    const DEFAULT_MAX_STALE_MS = 5 * 60 * 1000;
    const memoryCache = new Map();
    const inflight = new Map();

    function getStorage(mode) {
        try {
            return mode === 'local' ? global.localStorage : global.sessionStorage;
        } catch (e) {
            return null;
        }
    }

    function buildStorageKey(cacheKey) {
        return `${CACHE_PREFIX}${cacheKey}`;
    }

    function readEntry(cacheKey, persist) {
        const memoryEntry = memoryCache.get(cacheKey);
        if (memoryEntry) {
            return memoryEntry;
        }

        const storage = getStorage(persist);
        if (!storage) {
            return null;
        }

        try {
            const raw = storage.getItem(buildStorageKey(cacheKey));
            if (!raw) {
                return null;
            }

            const parsed = JSON.parse(raw);
            if (!parsed || typeof parsed.savedAt !== 'number') {
                return null;
            }

            memoryCache.set(cacheKey, parsed);
            return parsed;
        } catch (e) {
            return null;
        }
    }

    function writeEntry(cacheKey, payload, persist) {
        const entry = {
            payload,
            savedAt: Date.now(),
        };

        memoryCache.set(cacheKey, entry);

        const storage = getStorage(persist);
        if (storage) {
            try {
                storage.setItem(buildStorageKey(cacheKey), JSON.stringify(entry));
            } catch (e) {
                // Ignore storage quota or serialization errors and keep memory cache.
            }
        }

        return entry;
    }

    function clearPrefix(prefix) {
        const normalizedPrefix = prefix || '';
        memoryCache.forEach((_, key) => {
            if (key.startsWith(normalizedPrefix)) {
                memoryCache.delete(key);
            }
        });

        ['session', 'local'].forEach((mode) => {
            const storage = getStorage(mode);
            if (!storage) {
                return;
            }

            try {
                const keys = [];
                for (let i = 0; i < storage.length; i++) {
                    const key = storage.key(i);
                    if (!key || !key.startsWith(CACHE_PREFIX)) {
                        continue;
                    }

                    const cacheKey = key.slice(CACHE_PREFIX.length);
                    if (cacheKey.startsWith(normalizedPrefix)) {
                        keys.push(key);
                    }
                }

                keys.forEach((key) => storage.removeItem(key));
            } catch (e) {
                // Ignore storage access issues.
            }
        });
    }

    function peekJSON(cacheKey, policy = {}) {
        const persist = policy.persist || 'session';
        const ttlMs = Number.isFinite(policy.ttlMs) ? policy.ttlMs : DEFAULT_TTL_MS;
        const maxStaleMs = Number.isFinite(policy.maxStaleMs) ? policy.maxStaleMs : DEFAULT_MAX_STALE_MS;
        const entry = readEntry(cacheKey, persist);

        if (!entry) {
            return null;
        }

        const ageMs = Date.now() - entry.savedAt;
        if (ageMs > maxStaleMs) {
            return null;
        }

        return {
            data: entry.payload,
            cached: true,
            stale: ageMs > ttlMs,
            ageMs,
            savedAt: entry.savedAt,
        };
    }

    async function fetchNetworkJSON(url, options, policy, fallbackEntry, background) {
        const cacheKey = policy.cacheKey || url;
        const persist = policy.persist || 'session';

        if (inflight.has(cacheKey)) {
            return inflight.get(cacheKey);
        }

        const requestOptions = {
            ...options,
            silentErrors: background || options.silentErrors,
        };

        const promise = (async () => {
            const response = await fetchWithAuth(url, requestOptions);
            if (!response.ok) {
                const error = new Error(`HTTP ${response.status}`);
                error.status = response.status;
                throw error;
            }

            const data = await response.json();

            if (!data || !data.error) {
                const saved = writeEntry(cacheKey, data, persist);
                return {
                    data,
                    response,
                    cached: false,
                    stale: false,
                    savedAt: saved.savedAt,
                };
            }

            return {
                data,
                response,
                cached: false,
                stale: false,
                savedAt: Date.now(),
            };
        })().catch((error) => {
            if (!fallbackEntry) {
                throw error;
            }

            return {
                data: fallbackEntry.payload,
                error,
                cached: true,
                stale: true,
                fallback: true,
                ageMs: Date.now() - fallbackEntry.savedAt,
                savedAt: fallbackEntry.savedAt,
            };
        }).finally(() => {
            inflight.delete(cacheKey);
        });

        inflight.set(cacheKey, promise);
        return promise;
    }

    async function fetchJSON(url, options = {}, policy = {}) {
        const method = (options.method || 'GET').toUpperCase();
        if (method !== 'GET') {
            const response = await fetchWithAuth(url, options);
            return {
                data: await response.json(),
                response,
                cached: false,
                stale: false,
                savedAt: Date.now(),
            };
        }

        const cacheKey = policy.cacheKey || url;
        const persist = policy.persist || 'session';
        const ttlMs = Number.isFinite(policy.ttlMs) ? policy.ttlMs : DEFAULT_TTL_MS;
        const maxStaleMs = Number.isFinite(policy.maxStaleMs) ? policy.maxStaleMs : DEFAULT_MAX_STALE_MS;
        const entry = readEntry(cacheKey, persist);
        const ageMs = entry ? Date.now() - entry.savedAt : Number.POSITIVE_INFINITY;
        const hasUsableCache = !!entry && ageMs <= maxStaleMs;
        const isFresh = !!entry && ageMs <= ttlMs;
        const useCachedWhileRefreshing = policy.useCachedWhileRefreshing !== false;
        const forceNetwork = !!policy.forceNetwork;
        const revalidateIfFresh = !!policy.revalidateIfFresh;
        const shouldRevalidate = !!entry && (forceNetwork || !isFresh || revalidateIfFresh);

        if (hasUsableCache && useCachedWhileRefreshing) {
            return {
                data: entry.payload,
                cached: true,
                stale: !isFresh,
                ageMs,
                savedAt: entry.savedAt,
                revalidatePromise: shouldRevalidate
                    ? fetchNetworkJSON(url, options, { ...policy, cacheKey, persist }, entry, true)
                    : null,
            };
        }

        return fetchNetworkJSON(
            url,
            options,
            { ...policy, cacheKey, persist },
            forceNetwork && hasUsableCache ? entry : null,
            false
        );
    }

    global.K13D = global.K13D || {};
    global.K13D.SWR = {
        fetchJSON,
        peekJSON,
        clearPrefix,
    };
})(window);

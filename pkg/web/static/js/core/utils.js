/**
 * k13d Core Utilities
 * Common helper functions used across the application
 */

(function(global) {
    'use strict';

    const Utils = {
        /**
         * Escape HTML special characters to prevent XSS
         * @param {string} text - Text to escape
         * @returns {string} Escaped text
         */
        escapeHtml: function(text) {
            if (text === null || text === undefined) return '';
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        },

        /**
         * Format bytes to human readable string
         * @param {number} bytes - Number of bytes
         * @param {number} decimals - Decimal places
         * @returns {string} Formatted string (e.g., "1.5 GB")
         */
        formatBytes: function(bytes, decimals = 2) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const dm = decimals < 0 ? 0 : decimals;
            const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
        },

        /**
         * Format duration in seconds to human readable string
         * @param {number} seconds - Duration in seconds
         * @returns {string} Formatted string (e.g., "2d 5h 30m")
         */
        formatDuration: function(seconds) {
            if (seconds < 60) return `${seconds}s`;
            if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
            if (seconds < 86400) {
                const h = Math.floor(seconds / 3600);
                const m = Math.floor((seconds % 3600) / 60);
                return m > 0 ? `${h}h ${m}m` : `${h}h`;
            }
            const d = Math.floor(seconds / 86400);
            const h = Math.floor((seconds % 86400) / 3600);
            return h > 0 ? `${d}d ${h}h` : `${d}d`;
        },

        /**
         * Parse Kubernetes age string to seconds
         * @param {string} age - Age string (e.g., "5d", "2h30m", "45s")
         * @returns {number} Age in seconds
         */
        parseAge: function(age) {
            if (!age) return 0;
            let seconds = 0;
            const dayMatch = age.match(/(\d+)d/);
            const hourMatch = age.match(/(\d+)h/);
            const minMatch = age.match(/(\d+)m/);
            const secMatch = age.match(/(\d+)s/);

            if (dayMatch) seconds += parseInt(dayMatch[1]) * 86400;
            if (hourMatch) seconds += parseInt(hourMatch[1]) * 3600;
            if (minMatch) seconds += parseInt(minMatch[1]) * 60;
            if (secMatch) seconds += parseInt(secMatch[1]);

            return seconds;
        },

        /**
         * Parse Kubernetes CPU value to millicores
         * @param {string} cpu - CPU string (e.g., "100m", "2", "0.5")
         * @returns {number} CPU in millicores
         */
        parseCPU: function(cpu) {
            if (!cpu) return 0;
            if (cpu.endsWith('m')) {
                return parseInt(cpu);
            }
            return parseFloat(cpu) * 1000;
        },

        /**
         * Parse Kubernetes memory value to bytes
         * @param {string} memory - Memory string (e.g., "128Mi", "1Gi", "1000000")
         * @returns {number} Memory in bytes
         */
        parseMemory: function(memory) {
            if (!memory) return 0;
            const units = {
                'Ki': 1024,
                'Mi': 1024 * 1024,
                'Gi': 1024 * 1024 * 1024,
                'Ti': 1024 * 1024 * 1024 * 1024,
                'K': 1000,
                'M': 1000000,
                'G': 1000000000,
                'T': 1000000000000
            };

            for (const [suffix, multiplier] of Object.entries(units)) {
                if (memory.endsWith(suffix)) {
                    return parseFloat(memory) * multiplier;
                }
            }
            return parseFloat(memory);
        },

        /**
         * Truncate string to specified length
         * @param {string} str - String to truncate
         * @param {number} maxLen - Maximum length
         * @returns {string} Truncated string with ellipsis
         */
        truncate: function(str, maxLen) {
            if (!str || str.length <= maxLen) return str || '';
            return str.substring(0, maxLen - 3) + '...';
        },

        /**
         * Debounce function execution
         * @param {Function} func - Function to debounce
         * @param {number} wait - Wait time in milliseconds
         * @returns {Function} Debounced function
         */
        debounce: function(func, wait) {
            let timeout;
            return function executedFunction(...args) {
                const later = () => {
                    clearTimeout(timeout);
                    func(...args);
                };
                clearTimeout(timeout);
                timeout = setTimeout(later, wait);
            };
        },

        /**
         * Throttle function execution
         * @param {Function} func - Function to throttle
         * @param {number} limit - Minimum time between calls in milliseconds
         * @returns {Function} Throttled function
         */
        throttle: function(func, limit) {
            let inThrottle;
            return function(...args) {
                if (!inThrottle) {
                    func.apply(this, args);
                    inThrottle = true;
                    setTimeout(() => inThrottle = false, limit);
                }
            };
        },

        /**
         * Deep clone an object
         * @param {Object} obj - Object to clone
         * @returns {Object} Cloned object
         */
        deepClone: function(obj) {
            if (obj === null || typeof obj !== 'object') return obj;
            if (obj instanceof Date) return new Date(obj.getTime());
            if (obj instanceof Array) return obj.map(item => Utils.deepClone(item));
            if (obj instanceof Object) {
                const copy = {};
                Object.keys(obj).forEach(key => {
                    copy[key] = Utils.deepClone(obj[key]);
                });
                return copy;
            }
            return obj;
        },

        /**
         * Generate a random ID
         * @param {number} length - Length of ID
         * @returns {string} Random ID
         */
        generateId: function(length = 8) {
            const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
            let result = '';
            for (let i = 0; i < length; i++) {
                result += chars.charAt(Math.floor(Math.random() * chars.length));
            }
            return result;
        },

        /**
         * Check if running in dark mode
         * @returns {boolean} True if dark mode
         */
        isDarkMode: function() {
            return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
        },

        /**
         * Copy text to clipboard
         * @param {string} text - Text to copy
         * @returns {Promise<boolean>} Success status
         */
        copyToClipboard: async function(text) {
            try {
                await navigator.clipboard.writeText(text);
                return true;
            } catch (err) {
                // Fallback for older browsers
                const textarea = document.createElement('textarea');
                textarea.value = text;
                textarea.style.position = 'fixed';
                textarea.style.opacity = '0';
                document.body.appendChild(textarea);
                textarea.select();
                try {
                    document.execCommand('copy');
                    return true;
                } catch (e) {
                    return false;
                } finally {
                    document.body.removeChild(textarea);
                }
            }
        },

        /**
         * Get relative time string (e.g., "5 minutes ago")
         * @param {Date|string|number} date - Date to format
         * @returns {string} Relative time string
         */
        relativeTime: function(date) {
            const now = new Date();
            const then = new Date(date);
            const diffMs = now - then;
            const diffSec = Math.floor(diffMs / 1000);
            const diffMin = Math.floor(diffSec / 60);
            const diffHour = Math.floor(diffMin / 60);
            const diffDay = Math.floor(diffHour / 24);

            if (diffSec < 60) return 'just now';
            if (diffMin < 60) return `${diffMin}m ago`;
            if (diffHour < 24) return `${diffHour}h ago`;
            if (diffDay < 30) return `${diffDay}d ago`;
            return then.toLocaleDateString();
        }
    };

    // Export to global namespace
    global.K13D = global.K13D || {};
    global.K13D.Utils = Utils;

})(window);

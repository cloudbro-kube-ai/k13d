/**
 * k13d Documentation - Main JavaScript
 */

(function() {
    'use strict';

    // Theme Toggle
    const themeToggle = document.getElementById('theme-toggle');
    const savedTheme = localStorage.getItem('theme') || 'dark';

    document.documentElement.setAttribute('data-theme', savedTheme);

    if (themeToggle) {
        themeToggle.addEventListener('click', () => {
            const currentTheme = document.documentElement.getAttribute('data-theme');
            const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
        });
    }

    // Tab Navigation
    const tabButtons = document.querySelectorAll('.tab-btn');
    const tabContents = document.querySelectorAll('.tab-content');

    tabButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            const tabId = btn.getAttribute('data-tab');

            // Update buttons
            tabButtons.forEach(b => b.classList.remove('active'));
            btn.classList.add('active');

            // Update content
            tabContents.forEach(content => {
                content.classList.remove('active');
                if (content.id === tabId) {
                    content.classList.add('active');
                }
            });
        });
    });

    // Sidebar Active State
    const currentPath = window.location.pathname;
    const sidebarLinks = document.querySelectorAll('.sidebar-links a');

    sidebarLinks.forEach(link => {
        const href = link.getAttribute('href');
        if (currentPath.endsWith(href) ||
            (currentPath.endsWith('/') && href === 'index.html')) {
            link.classList.add('active');
        } else {
            link.classList.remove('active');
        }
    });

    // Smooth Scroll for TOC
    const tocLinks = document.querySelectorAll('.toc a');

    tocLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const targetId = link.getAttribute('href').slice(1);
            const target = document.getElementById(targetId);
            if (target) {
                target.scrollIntoView({ behavior: 'smooth', block: 'start' });
                history.pushState(null, null, `#${targetId}`);
            }
        });
    });

    // Active TOC on Scroll
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                const id = entry.target.getAttribute('id');
                tocLinks.forEach(link => {
                    link.classList.remove('active');
                    if (link.getAttribute('href') === `#${id}`) {
                        link.classList.add('active');
                    }
                });
            }
        });
    }, { rootMargin: '-20% 0% -80% 0%' });

    document.querySelectorAll('h2[id], h3[id]').forEach(heading => {
        observer.observe(heading);
    });

    // Copy Code Button
    document.querySelectorAll('pre code').forEach(block => {
        const wrapper = document.createElement('div');
        wrapper.className = 'code-block';
        block.parentNode.insertBefore(wrapper, block.parentNode.firstChild);
        wrapper.appendChild(block.parentNode);

        // Actually wrap pre element
        const pre = block.parentNode;
        if (pre.parentNode !== wrapper) {
            pre.parentNode.insertBefore(wrapper, pre);
            wrapper.appendChild(pre);
        }

        const copyBtn = document.createElement('button');
        copyBtn.className = 'copy-btn';
        copyBtn.textContent = 'Copy';
        wrapper.appendChild(copyBtn);

        copyBtn.addEventListener('click', async () => {
            try {
                await navigator.clipboard.writeText(block.textContent);
                copyBtn.textContent = 'Copied!';
                copyBtn.classList.add('copied');
                setTimeout(() => {
                    copyBtn.textContent = 'Copy';
                    copyBtn.classList.remove('copied');
                }, 2000);
            } catch (err) {
                console.error('Failed to copy:', err);
            }
        });
    });

    // Mobile Sidebar Toggle
    const sidebar = document.getElementById('sidebar');
    const sidebarToggle = document.createElement('button');
    sidebarToggle.className = 'sidebar-toggle';
    sidebarToggle.innerHTML = `
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="3" y1="12" x2="21" y2="12"></line>
            <line x1="3" y1="6" x2="21" y2="6"></line>
            <line x1="3" y1="18" x2="21" y2="18"></line>
        </svg>
    `;
    document.body.appendChild(sidebarToggle);

    sidebarToggle.addEventListener('click', () => {
        sidebar.classList.toggle('open');
    });

    // Close sidebar on link click (mobile)
    sidebarLinks.forEach(link => {
        link.addEventListener('click', () => {
            if (window.innerWidth <= 768) {
                sidebar.classList.remove('open');
            }
        });
    });

    // Keyboard Shortcuts
    document.addEventListener('keydown', (e) => {
        // Cmd/Ctrl + K for search
        if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
            e.preventDefault();
            document.getElementById('search-input')?.focus();
        }

        // Escape to close search
        if (e.key === 'Escape') {
            document.getElementById('search-modal')?.classList.remove('active');
            document.getElementById('search-input')?.blur();
        }
    });

    // Add IDs to headings for TOC
    document.querySelectorAll('.content h2, .content h3').forEach(heading => {
        if (!heading.id) {
            heading.id = heading.textContent
                .toLowerCase()
                .replace(/[^a-z0-9]+/g, '-')
                .replace(/(^-|-$)/g, '');
        }
    });

    console.log('k13d Documentation initialized');
})();

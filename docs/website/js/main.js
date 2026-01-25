// k13d Documentation Website JavaScript

document.addEventListener('DOMContentLoaded', () => {
    // Tab functionality for keybindings section
    const tabButtons = document.querySelectorAll('.tab-btn');
    const tabContents = document.querySelectorAll('.tab-content');

    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            const tabId = button.dataset.tab;

            // Remove active class from all buttons and contents
            tabButtons.forEach(btn => btn.classList.remove('active'));
            tabContents.forEach(content => content.classList.remove('active'));

            // Add active class to clicked button and corresponding content
            button.classList.add('active');
            document.getElementById(tabId).classList.add('active');
        });
    });

    // Smooth scroll for anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                const navHeight = document.querySelector('.navbar').offsetHeight;
                const targetPosition = target.offsetTop - navHeight - 20;
                window.scrollTo({
                    top: targetPosition,
                    behavior: 'smooth'
                });
            }
        });
    });

    // Navbar background on scroll
    const navbar = document.querySelector('.navbar');
    window.addEventListener('scroll', () => {
        if (window.scrollY > 50) {
            navbar.style.background = 'rgba(15, 23, 42, 0.98)';
            navbar.style.boxShadow = '0 4px 20px rgba(0, 0, 0, 0.3)';
        } else {
            navbar.style.background = 'rgba(15, 23, 42, 0.95)';
            navbar.style.boxShadow = 'none';
        }
    });

    // Keyboard shortcut demo in terminal
    const terminalContent = document.querySelector('.terminal-content');
    if (terminalContent) {
        // Add subtle animation to terminal
        let cursorVisible = true;
        const cursor = document.createElement('span');
        cursor.textContent = '█';
        cursor.style.animation = 'blink 1s step-end infinite';

        // Add cursor blink animation
        const style = document.createElement('style');
        style.textContent = `
            @keyframes blink {
                0%, 100% { opacity: 1; }
                50% { opacity: 0; }
            }
        `;
        document.head.appendChild(style);
    }

    // Copy code blocks
    const codeBlocks = document.querySelectorAll('.code-block');
    codeBlocks.forEach(block => {
        const copyBtn = document.createElement('button');
        copyBtn.textContent = 'Copy';
        copyBtn.className = 'copy-btn';
        copyBtn.style.cssText = `
            position: absolute;
            top: 8px;
            right: 8px;
            padding: 6px 12px;
            background: rgba(99, 102, 241, 0.2);
            border: 1px solid rgba(99, 102, 241, 0.3);
            border-radius: 4px;
            color: #6366f1;
            font-size: 12px;
            cursor: pointer;
            opacity: 0;
            transition: opacity 0.2s;
        `;

        block.style.position = 'relative';
        block.appendChild(copyBtn);

        block.addEventListener('mouseenter', () => {
            copyBtn.style.opacity = '1';
        });

        block.addEventListener('mouseleave', () => {
            copyBtn.style.opacity = '0';
        });

        copyBtn.addEventListener('click', () => {
            const code = block.querySelector('code').textContent;
            navigator.clipboard.writeText(code).then(() => {
                copyBtn.textContent = 'Copied!';
                copyBtn.style.background = 'rgba(34, 197, 94, 0.2)';
                copyBtn.style.borderColor = 'rgba(34, 197, 94, 0.3)';
                copyBtn.style.color = '#22c55e';
                setTimeout(() => {
                    copyBtn.textContent = 'Copy';
                    copyBtn.style.background = 'rgba(99, 102, 241, 0.2)';
                    copyBtn.style.borderColor = 'rgba(99, 102, 241, 0.3)';
                    copyBtn.style.color = '#6366f1';
                }, 2000);
            });
        });
    });

    // Intersection Observer for fade-in animations
    const observerOptions = {
        root: null,
        rootMargin: '0px',
        threshold: 0.1
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '1';
                entry.target.style.transform = 'translateY(0)';
            }
        });
    }, observerOptions);

    // Apply fade-in to feature cards and other elements
    const animatedElements = document.querySelectorAll('.feature-card, .ai-feature, .step');
    animatedElements.forEach(el => {
        el.style.opacity = '0';
        el.style.transform = 'translateY(20px)';
        el.style.transition = 'opacity 0.5s ease, transform 0.5s ease';
        observer.observe(el);
    });

    // Mobile menu toggle (if needed in future)
    const createMobileMenu = () => {
        const navbar = document.querySelector('.navbar');
        const navLinks = document.querySelector('.nav-links');

        if (window.innerWidth <= 768) {
            // Add hamburger button if not exists
            if (!document.querySelector('.hamburger')) {
                const hamburger = document.createElement('button');
                hamburger.className = 'hamburger';
                hamburger.innerHTML = '☰';
                hamburger.style.cssText = `
                    display: none;
                    background: none;
                    border: none;
                    color: white;
                    font-size: 24px;
                    cursor: pointer;
                `;
                navbar.appendChild(hamburger);
            }
        }
    };

    // Search functionality (placeholder for future)
    const initSearch = () => {
        // Could add search functionality here
    };

    // Initialize
    createMobileMenu();
    window.addEventListener('resize', createMobileMenu);
});

// Utility: Detect keyboard navigation
document.addEventListener('keydown', (e) => {
    // Quick navigation shortcuts for the docs site
    if (e.key === '/' && !e.target.matches('input, textarea')) {
        e.preventDefault();
        // Could open a search modal here
    }
});

console.log('k13d Documentation loaded. Press ? for help.');

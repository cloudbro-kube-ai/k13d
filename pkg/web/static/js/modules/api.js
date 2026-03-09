/**
 * Unified API Fetch Wrapper with Error Handling
 */

async function fetchWithAuth(url, options = {}) {
    const headers = { ...options.headers };

    // Attach auth token if available (reads global `authToken` from app.js)
    if (window.authToken && window.authToken !== 'anonymous') {
        headers['Authorization'] = `Bearer ${window.authToken}`;
    }

    try {
        const response = await fetch(url, { ...options, headers });

        // Handle common API HTTP errors globally
        if (!response.ok) {
            if (response.status === 401) {
                // Only auto-logout if we're not already on the login page
                const loginPage = document.getElementById('login-page');
                const isOnLoginPage = loginPage && loginPage.style.display !== 'none';
                if (!isOnLoginPage && typeof logout === 'function') {
                    console.warn('[API] Unauthorized access or token expired. Logging out.');
                    logout();
                }
            } else if (response.status === 403) {
                console.warn('[API] Forbidden access to:', url);
                showErrorToast(`Forbidden: You don't have permission to access this resource.`);
            } else if (response.status >= 500) {
                console.error('[API] Server error on:', url, response.status);
            }
        }

        return response;
    } catch (e) {
        console.error('[API] Fetch exception:', e);
        showErrorToast(`Network Error: ${e.message}`);
        throw e;
    }
}

// Global UI Toast for API Errors
function showErrorToast(message) {
    const errorToast = document.getElementById('login-error'); // fallback to login error if outside app
    if (errorToast && document.getElementById('login-page').style.display !== 'none') {
        errorToast.textContent = message;
        errorToast.style.display = 'block';
        setTimeout(() => { errorToast.style.display = 'none'; }, 5000);
    } else {
        // Attempt to create a global app toast if not exists
        let toast = document.getElementById('app-error-toast');
        if (!toast) {
            toast = document.createElement('div');
            toast.id = 'app-error-toast';
            toast.style.position = 'fixed';
            toast.style.bottom = '20px';
            toast.style.right = '20px';
            toast.style.backgroundColor = '#f44336';
            toast.style.color = '#fff';
            toast.style.padding = '12px 24px';
            toast.style.borderRadius = '4px';
            toast.style.zIndex = '9999';
            toast.style.boxShadow = '0 2px 10px rgba(0,0,0,0.2)';
            toast.style.transition = 'opacity 0.3s ease-in-out';
            toast.style.opacity = '0';
            document.body.appendChild(toast);
        }

        toast.textContent = message;
        toast.style.display = 'block';

        // fade in
        setTimeout(() => { toast.style.opacity = '1'; }, 10);

        // fade out
        setTimeout(() => {
            toast.style.opacity = '0';
            setTimeout(() => { toast.style.display = 'none'; }, 300);
        }, 3000);
    }
}

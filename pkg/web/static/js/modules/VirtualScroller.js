class TableVirtualScroller {
    constructor(containerSelector, tbodyId, rowHeight) {
        this.container = document.querySelector(containerSelector);
        this.tbody = document.getElementById(tbodyId);
        this.rowHeight = rowHeight || 43; // Default row height (px)
        this.items = [];
        this.renderRowFn = null;
        this.visibleNodes = 50;

        // Add scroll listener with rAF throttling for performance
        this.ticking = false;
        this.container.addEventListener('scroll', () => {
            if (!this.ticking) {
                window.requestAnimationFrame(() => {
                    this.onScroll();
                    this.ticking = false;
                });
                this.ticking = true;
            }
        });
    }

    setItems(items, renderRowFn) {
        this.items = items;
        this.renderRowFn = renderRowFn;
        // Reset scroll position on new data if needed, or just re-render at current scroll
        this.onScroll();
    }

    onScroll() {
        if (!this.items || !this.items.length) {
            return; // Empty state is handled by app.js directly
        }

        const scrollTop = this.container.scrollTop;
        const totalHeight = this.items.length * this.rowHeight;

        let startIndex = Math.floor(scrollTop / this.rowHeight);
        startIndex = Math.max(0, startIndex - 10); // buffer 10 rows above

        let endIndex = startIndex + this.visibleNodes;
        endIndex = Math.min(this.items.length, endIndex);

        const topPadding = startIndex * this.rowHeight;
        const bottomPadding = Math.max(0, totalHeight - (endIndex * this.rowHeight));

        let rowsHtml = '';
        for (let i = startIndex; i < endIndex; i++) {
            rowsHtml += this.renderRowFn(this.items[i], i);
        }

        const parts = [];
        if (topPadding > 0) {
            parts.push(
                `<tr data-spacer="true" aria-hidden="true" style="height: ${topPadding}px; border: none !important;"><td colspan="100" style="border: none !important; padding: 0;"></td></tr>`
            );
        }
        parts.push(rowsHtml);
        if (bottomPadding > 0) {
            parts.push(
                `<tr data-spacer="true" aria-hidden="true" style="height: ${bottomPadding}px; border: none !important;"><td colspan="100" style="border: none !important; padding: 0;"></td></tr>`
            );
        }

        this.tbody.innerHTML = parts.join('');

        if (typeof window.addRowClickHandlers === 'function') {
            window.addRowClickHandlers();
        }
    }
}

// Instantiate globally for the main table
window.virtualScroller = null;
document.addEventListener('DOMContentLoaded', () => {
    window.virtualScroller = new TableVirtualScroller('.table-container', 'table-body', 44);
});

/**
 * Lucide Icons Helper for k13d
 * Provides icon rendering functions for consistent icon usage
 */

// Icon name mapping for common icons
const ICONS = {
  // Status icons
  'success': 'check-circle',
  'error': 'x-circle',
  'warning': 'alert-triangle',
  'info': 'info',
  
  // Actions
  'refresh': 'refresh-cw',
  'delete': 'trash-2',
  'edit': 'pencil',
  'search': 'search',
  'clear': 'x',
  'expand': 'maximize-2',
  'collapse': 'minimize-2',
  'close': 'x',
  
  // Navigation
  'arrow-up': 'chevron-up',
  'arrow-down': 'chevron-down',
  'arrow-left': 'chevron-left',
  'arrow-right': 'chevron-right',
  'menu': 'menu',
  'sidebar': 'panel-left',
  
  // Resource types
  'pod': 'box',
  'deployment': 'layers',
  'service': 'network',
  'configmap': 'file-text',
  'secret': 'lock',
  'namespace': 'folder',
  'node': 'server',
  'pvc': 'hard-drive',
  'ingress': 'globe',
  'job': 'clock',
  'cronjob': 'calendar',
  'daemonset': 'copy',
  'statefulset': 'database',
  'replicaset': 'repeat',
  'hpa': 'activity',
  'crd': 'file-code',
  
  // Features
  'ai': 'bot',
  'terminal': 'terminal',
  'logs': 'file-text',
  'metrics': 'bar-chart-2',
  'topology': 'git-branch',
  'settings': 'settings',
  'help': 'help-circle',
  'shortcuts': 'keyboard',
  'history': 'clock',
  'chat': 'message-square',
  'bookmark': 'bookmark',
  'pin': 'pin',
  'unpin': 'pin-off',
  'copy': 'copy',
  'download': 'download',
  'upload': 'upload',
  'link': 'external-link',
  'lock': 'lock',
  'unlock': 'unlock',
  
  // Objects
  'package': 'package',
  'clipboard': 'clipboard',
  'lightbulb': 'lightbulb',
  'wrench': 'wrench',
  'tool': 'settings',
  'robot': 'bot',
  'kubernetes': 'container',
  'docker': 'box',
  'github': 'github',
  
  // Sort
  'sort-asc': 'arrow-up',
  'sort-desc': 'arrow-down',
  'sort-none': 'arrow-up-down',
  
  // Misc
  'loading': 'loader',
  'spinner': 'loader',
  'check': 'check',
  'plus': 'plus',
  'minus': 'minus',
  'more': 'more-vertical',
  'filter': 'filter',
  'download-cloud': 'download-cloud',
  'upload-cloud': 'upload-cloud',
};

/**
 * Render a Lucide icon as HTML string
 * @param {string} name - Icon name (e.g., 'search', 'check-circle')
 * @param {Object} options - Icon options
 * @param {number} options.size - Icon size in pixels (default: 16)
 * @param {string} options.className - Additional CSS class
 * @param {string} options.color - Icon color
 * @returns {string} HTML string with the icon
 */
function lucideIcon(name, options = {}) {
  const {
    size = 16,
    className = '',
    color = 'currentColor',
    strokeWidth = 2
  } = options;
  
  const iconName = ICONS[name] || name;
  
  return `<i data-lucide="${iconName}" 
    style="width:${size}px;height:${size}px;color:${color};stroke-width:${strokeWidth}px" 
    class="lucide-icon ${className}"></i>`;
}

/**
 * Render a Lucide icon with label
 * @param {string} name - Icon name
 * @param {string} label - Text label
 * @param {Object} options - Icon options
 * @returns {string} HTML string with icon and label
 */
function lucideIconLabel(name, label, options = {}) {
  const { labelClass = '', ...iconOptions } = options;
  return `<span class="icon-label ${labelClass}">${lucideIcon(name, iconOptions)} ${label}</span>`;
}

/**
 * Replace emoji with Lucide icon in a string
 * @param {string} text - Text containing emoji
 * @returns {string} Text with emoji replaced by Lucide icons
 */
function replaceEmojiIcons(text) {
  const emojiMap = {
    '📦': 'package',
    '📋': 'clipboard',
    '📊': 'bar-chart-2',
    '🔧': 'wrench',
    '🔍': 'search',
    '⚠️': 'alert-triangle',
    '❌': 'x-circle',
    '✅': 'check-circle',
    '💡': 'lightbulb',
    '🗑️': 'trash-2',
    '🔄': 'refresh-cw',
    '🤖': 'bot',
    '⎈': 'container',
    '✖': 'x',
  };
  
  let result = text;
  for (const [emoji, iconName] of Object.entries(emojiMap)) {
    result = result.split(emoji).join(lucideIcon(iconName));
  }
  return result;
}

/**
 * Initialize all Lucide icons on the page
 * Call this after DOM is ready
 */
function initLucideIcons() {
  if (typeof lucide !== 'undefined') {
    lucide.createIcons();
  }
}

// Export for use in other modules
window.ICONS = ICONS;
window.lucideIcon = lucideIcon;
window.lucideIconLabel = lucideIconLabel;
window.replaceEmojiIcons = replaceEmojiIcons;
window.initLucideIcons = initLucideIcons;

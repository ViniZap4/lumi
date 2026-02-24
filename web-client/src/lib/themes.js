// Theme definitions matching tui-client/theme/builtin.go exactly.

export const themeOrder = [
  'tokyo-night',
  'tokyo-day',
  'catppuccin-mocha',
  'catppuccin-latte',
  'dracula',
  'lgbt-dark',
  'lgbt-light',
  'trans-dark',
  'trans-light',
  'transfem',
  'transmasc',
  'obsidian',
];

export const themes = {
  'tokyo-night': {
    name: 'tokyo-night',
    isDark: true,
    primary: '#7aa2f7',
    secondary: '#bb9af7',
    accent: '#7dcfff',
    muted: '#565f89',
    background: '#1a1b26',
    selectedBg: '#292e42',
    overlayBg: '#16161e',
    text: '#c0caf5',
    textDim: '#565f89',
    border: '#3b4261',
    separator: '#292e42',
    error: '#f7768e',
    warning: '#e0af68',
    info: '#7dcfff',
    logoColors: ['#7aa2f7', '#7dcfff', '#bb9af7', '#7aa2f7', '#7dcfff', '#bb9af7'],
  },
  'tokyo-day': {
    name: 'tokyo-day',
    isDark: false,
    primary: '#2e7de9',
    secondary: '#9854f1',
    accent: '#007197',
    muted: '#8990b3',
    background: '#d5d6db',
    selectedBg: '#c4c8da',
    overlayBg: '#e9e9ec',
    text: '#3760bf',
    textDim: '#8990b3',
    border: '#a8aecb',
    separator: '#c4c8da',
    error: '#f52a65',
    warning: '#8c6c3e',
    info: '#007197',
    logoColors: ['#2e7de9', '#007197', '#9854f1', '#2e7de9', '#007197', '#9854f1'],
  },
  'catppuccin-mocha': {
    name: 'catppuccin-mocha',
    isDark: true,
    primary: '#cba6f7',
    secondary: '#89b4fa',
    accent: '#f5c2e7',
    muted: '#6c7086',
    background: '#1e1e2e',
    selectedBg: '#313244',
    overlayBg: '#181825',
    text: '#cdd6f4',
    textDim: '#6c7086',
    border: '#45475a',
    separator: '#313244',
    error: '#f38ba8',
    warning: '#f9e2af',
    info: '#89dceb',
    logoColors: ['#cba6f7', '#f5c2e7', '#89b4fa', '#cba6f7', '#f5c2e7', '#89b4fa'],
  },
  'catppuccin-latte': {
    name: 'catppuccin-latte',
    isDark: false,
    primary: '#8839ef',
    secondary: '#1e66f5',
    accent: '#ea76cb',
    muted: '#9ca0b0',
    background: '#ccd0da',
    selectedBg: '#bcc0cc',
    overlayBg: '#e6e9ef',
    text: '#4c4f69',
    textDim: '#9ca0b0',
    border: '#acb0be',
    separator: '#bcc0cc',
    error: '#d20f39',
    warning: '#df8e1d',
    info: '#04a5e5',
    logoColors: ['#8839ef', '#ea76cb', '#1e66f5', '#8839ef', '#ea76cb', '#1e66f5'],
  },
  'dracula': {
    name: 'dracula',
    isDark: true,
    primary: '#bd93f9',
    secondary: '#8be9fd',
    accent: '#ff79c6',
    muted: '#6272a4',
    background: '#282a36',
    selectedBg: '#44475a',
    overlayBg: '#21222c',
    text: '#f8f8f2',
    textDim: '#6272a4',
    border: '#44475a',
    separator: '#44475a',
    error: '#ff5555',
    warning: '#f1fa8c',
    info: '#8be9fd',
    logoColors: ['#bd93f9', '#ff79c6', '#8be9fd', '#bd93f9', '#ff79c6', '#8be9fd'],
  },
  'lgbt-dark': {
    name: 'lgbt-dark',
    isDark: true,
    primary: '#ff6b6b',
    secondary: '#ffd93d',
    accent: '#6bcb77',
    muted: '#7f8c8d',
    background: '#1a1a2e',
    selectedBg: '#2d2d44',
    overlayBg: '#141425',
    text: '#ecf0f1',
    textDim: '#7f8c8d',
    border: '#3d3d5c',
    separator: '#2d2d44',
    error: '#ff6b6b',
    warning: '#ffd93d',
    info: '#4ecdc4',
    logoColors: ['#ff6b6b', '#ff8e53', '#ffd93d', '#6bcb77', '#4ecdc4', '#9b59b6'],
  },
  'lgbt-light': {
    name: 'lgbt-light',
    isDark: false,
    primary: '#e63946',
    secondary: '#f4a261',
    accent: '#2a9d8f',
    muted: '#8d99ae',
    background: '#d6d6d6',
    selectedBg: '#c8c8c8',
    overlayBg: '#edf2f4',
    text: '#264653',
    textDim: '#8d99ae',
    border: '#a8b2c1',
    separator: '#c8c8c8',
    error: '#e63946',
    warning: '#e76f51',
    info: '#2a9d8f',
    logoColors: ['#e63946', '#e76f51', '#f4a261', '#2a9d8f', '#264653', '#6a4c93'],
  },
  'trans-dark': {
    name: 'trans-dark',
    isDark: true,
    primary: '#5bcefa',
    secondary: '#f5a9b8',
    accent: '#ffffff',
    muted: '#7f8c8d',
    background: '#1a1a2e',
    selectedBg: '#2d2d44',
    overlayBg: '#141425',
    text: '#ecf0f1',
    textDim: '#7f8c8d',
    border: '#3d3d5c',
    separator: '#2d2d44',
    error: '#ff6b6b',
    warning: '#ffd93d',
    info: '#5bcefa',
    logoColors: ['#5bcefa', '#f5a9b8', '#ffffff', '#f5a9b8', '#5bcefa', '#ffffff'],
  },
  'trans-light': {
    name: 'trans-light',
    isDark: false,
    primary: '#5bcefa',
    secondary: '#f5a9b8',
    accent: '#333333',
    muted: '#8d99ae',
    background: '#d6d6d6',
    selectedBg: '#c8c8c8',
    overlayBg: '#edf2f4',
    text: '#2b2d42',
    textDim: '#8d99ae',
    border: '#a8b2c1',
    separator: '#c8c8c8',
    error: '#e63946',
    warning: '#e76f51',
    info: '#5bcefa',
    logoColors: ['#5bcefa', '#f5a9b8', '#333333', '#f5a9b8', '#5bcefa', '#333333'],
  },
  'transfem': {
    name: 'transfem',
    isDark: true,
    primary: '#f4a7b9',
    secondary: '#c9b1e8',
    accent: '#87ceeb',
    muted: '#7a6e7e',
    background: '#1a1520',
    selectedBg: '#2e2238',
    overlayBg: '#130f1a',
    text: '#f0e6f6',
    textDim: '#7a6e7e',
    border: '#3d2e4a',
    separator: '#2e2238',
    error: '#ff6b8a',
    warning: '#ffd6a5',
    info: '#87ceeb',
    logoColors: ['#87ceeb', '#f4a7b9', '#f0e6f6', '#f4a7b9', '#87ceeb', '#c9b1e8'],
  },
  'transmasc': {
    name: 'transmasc',
    isDark: true,
    primary: '#5ba3d9',
    secondary: '#f4a7b9',
    accent: '#d8eaf8',
    muted: '#6e7a8a',
    background: '#0f1820',
    selectedBg: '#1c2e3f',
    overlayBg: '#0a1018',
    text: '#ddeeff',
    textDim: '#6e7a8a',
    border: '#2a4060',
    separator: '#1c2e3f',
    error: '#ff7096',
    warning: '#ffd6a5',
    info: '#5ba3d9',
    logoColors: ['#5ba3d9', '#d8eaf8', '#f4a7b9', '#5ba3d9', '#d8eaf8', '#f4a7b9'],
  },
  'obsidian': {
    name: 'obsidian',
    isDark: true,
    primary: '#a882ff',
    secondary: '#7c6f9e',
    accent: '#c792ea',
    muted: '#5c5c5c',
    background: '#1e1e1e',
    selectedBg: '#2d2d2d',
    overlayBg: '#161616',
    text: '#dcddde',
    textDim: '#5c5c5c',
    border: '#3c3c3c',
    separator: '#2d2d2d',
    error: '#e05561',
    warning: '#d19a66',
    info: '#56b6c2',
    logoColors: ['#a882ff', '#c792ea', '#7c6f9e', '#a882ff', '#c792ea', '#7c6f9e'],
  },
};

/** Filtered theme lists by isDark */
export const darkThemeOrder = themeOrder.filter(n => themes[n].isDark);
export const lightThemeOrder = themeOrder.filter(n => !themes[n].isDark);

const STORAGE_KEY = 'lumi-theme';
const SETTINGS_KEY = 'lumi-theme-settings';

/**
 * Apply a theme by name — sets CSS custom properties on :root.
 */
export function applyTheme(name) {
  const t = themes[name];
  if (!t) return;

  const root = document.documentElement;
  root.style.setProperty('--color-primary', t.primary);
  root.style.setProperty('--color-secondary', t.secondary);
  root.style.setProperty('--color-accent', t.accent);
  root.style.setProperty('--color-muted', t.muted);
  root.style.setProperty('--color-background', t.background);
  root.style.setProperty('--color-selected-bg', t.selectedBg);
  root.style.setProperty('--color-overlay-bg', t.overlayBg);
  root.style.setProperty('--color-text', t.text);
  root.style.setProperty('--color-text-dim', t.textDim);
  root.style.setProperty('--color-border', t.border);
  root.style.setProperty('--color-separator', t.separator);
  root.style.setProperty('--color-error', t.error);
  root.style.setProperty('--color-warning', t.warning);
  root.style.setProperty('--color-info', t.info);

  for (let i = 0; i < 6; i++) {
    root.style.setProperty(`--color-logo-${i}`, t.logoColors[i]);
  }
}

/**
 * Resolve which theme name to use based on mode.
 */
export function resolveTheme(mode, darkName, lightName) {
  if (mode === 'dark') return darkName;
  if (mode === 'light') return lightName;
  // auto — use OS preference
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  return prefersDark ? darkName : lightName;
}

/**
 * Load theme settings from localStorage, migrating from old single-key format.
 */
export function loadThemeSettings() {
  const saved = localStorage.getItem(SETTINGS_KEY);
  if (saved) {
    try {
      const s = JSON.parse(saved);
      if (s.mode && s.darkName && s.lightName) return s;
    } catch {}
  }
  // Migrate from old single-key format
  const oldTheme = localStorage.getItem(STORAGE_KEY);
  if (oldTheme && themes[oldTheme]) {
    const isDark = themes[oldTheme].isDark;
    return {
      mode: isDark ? 'dark' : 'light',
      darkName: isDark ? oldTheme : darkThemeOrder[0],
      lightName: isDark ? lightThemeOrder[0] : oldTheme,
    };
  }
  return { mode: 'dark', darkName: 'tokyo-night', lightName: 'tokyo-day' };
}

/**
 * Persist theme settings to localStorage.
 */
export function saveThemeSettings(mode, darkName, lightName) {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify({ mode, darkName, lightName }));
}

/**
 * Watch for OS color-scheme changes. Returns a cleanup function.
 */
export function watchSystemTheme(callback) {
  const mq = window.matchMedia('(prefers-color-scheme: dark)');
  const handler = (e) => callback(e.matches);
  mq.addEventListener('change', handler);
  return () => mq.removeEventListener('change', handler);
}

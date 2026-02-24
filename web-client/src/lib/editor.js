import { EditorView, ViewPlugin, lineNumbers, highlightActiveLine, keymap } from '@codemirror/view';
import { EditorState, Compartment } from '@codemirror/state';
import { history, defaultKeymap, historyKeymap } from '@codemirror/commands';
import { markdown } from '@codemirror/lang-markdown';
import { languages } from '@codemirror/language-data';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { tags } from '@lezer/highlight';
import { vim, Vim, getCM } from '@replit/codemirror-vim';

const themeCompartment = new Compartment();

function getCSSVar(name) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

function hexToRgba(hex, alpha) {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

function buildTheme() {
  const bg = getCSSVar('--color-background');
  const text = getCSSVar('--color-text');
  const selectedBg = getCSSVar('--color-selected-bg');
  const border = getCSSVar('--color-border');
  const muted = getCSSVar('--color-muted');
  const primary = getCSSVar('--color-primary');
  const secondary = getCSSVar('--color-secondary');
  const accent = getCSSVar('--color-accent');
  const warning = getCSSVar('--color-warning');
  const info = getCSSVar('--color-info');
  const overlayBg = getCSSVar('--color-overlay-bg');

  const editorTheme = EditorView.theme({
    '&': {
      backgroundColor: bg,
      color: text,
      height: '100%',
    },
    '.cm-content': {
      caretColor: primary,
      fontFamily: 'inherit',
      fontSize: '0.95rem',
      lineHeight: '1.7',
      padding: '1.25rem',
    },
    '.cm-cursor, .cm-dropCursor': {
      borderLeftColor: primary,
    },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
      backgroundColor: hexToRgba(primary, 0.35) + ' !important',
    },
    '.cm-activeLine': {
      backgroundColor: selectedBg,
    },
    '.cm-gutters': {
      backgroundColor: overlayBg,
      color: muted,
      border: 'none',
      borderRight: `1px solid ${border}`,
    },
    '.cm-activeLineGutter': {
      backgroundColor: selectedBg,
      color: primary,
    },
    '.cm-lineNumbers .cm-gutterElement': {
      padding: '0 0.5rem 0 0.75rem',
      minWidth: '2.5rem',
    },
    // Vim cursor styling
    '.cm-fat-cursor': {
      backgroundColor: primary + ' !important',
      color: bg + ' !important',
    },
    '&:not(.cm-focused) .cm-fat-cursor': {
      backgroundColor: 'transparent !important',
      outline: `1px solid ${primary}`,
      color: text + ' !important',
    },
    // Vim command line
    '.cm-vim-panel': {
      backgroundColor: overlayBg,
      color: text,
      borderTop: `1px solid ${border}`,
      padding: '0.25rem 0.5rem',
      fontFamily: 'inherit',
    },
    '.cm-vim-panel input': {
      backgroundColor: 'transparent',
      color: text,
      fontFamily: 'inherit',
    },
  }, { dark: true });

  const highlightStyles = HighlightStyle.define([
    { tag: tags.heading1, color: primary, fontWeight: '700' },
    { tag: tags.heading2, color: secondary, fontWeight: '700' },
    { tag: tags.heading3, color: accent, fontWeight: '700' },
    { tag: [tags.heading4, tags.heading5, tags.heading6], color: muted, fontWeight: '700' },
    { tag: tags.emphasis, fontStyle: 'italic', color: text },
    { tag: tags.strong, fontWeight: '700', color: text },
    { tag: [tags.monospace, tags.processingInstruction], color: accent },
    { tag: tags.link, color: info, textDecoration: 'underline' },
    { tag: tags.url, color: info },
    { tag: tags.quote, color: muted, fontStyle: 'italic' },
    { tag: tags.string, color: warning },
    { tag: [tags.meta, tags.comment], color: muted },
    { tag: tags.keyword, color: secondary },
    { tag: tags.number, color: warning },
    { tag: tags.punctuation, color: muted },
    { tag: tags.contentSeparator, color: border },
  ]);

  return [editorTheme, syntaxHighlighting(highlightStyles)];
}

/**
 * Create a CodeMirror 6 editor.
 * @param {HTMLElement} container
 * @param {string} content
 * @param {object} opts - { onChange, onSave, onQuit, vimEnabled, jjEscape, relativeLineNumbers }
 */
export function createEditor(container, content, { onChange, onSave, onQuit, vimEnabled = true, jjEscape = false, relativeLineNumbers = false }) {
  const extensions = [];

  if (vimEnabled) {
    Vim.defineEx('write', 'w', () => { if (onSave) onSave(); });
    Vim.defineEx('quit', 'q', () => { if (onQuit) onQuit(); });
    Vim.defineEx('wquit', 'wq', () => {
      if (onSave) onSave();
      if (onQuit) onQuit();
    });

    if (jjEscape) {
      Vim.map('jj', '<Esc>', 'insert');
    } else {
      try { Vim.unmap('jj', 'insert'); } catch {}
    }

    extensions.push(vim());
  }

  if (relativeLineNumbers) {
    // Track cursor line to force gutter re-render on cursor movement
    let prevCursorLine = 0;
    extensions.push(
      lineNumbers({
        formatNumber: (lineNo, state) => {
          const curLine = state.doc.lineAt(state.selection.main.head).number;
          if (lineNo === curLine) return String(lineNo);
          return String(Math.abs(lineNo - curLine));
        },
      }),
      ViewPlugin.define(() => ({
        update(update) {
          const curLine = update.state.doc.lineAt(update.state.selection.main.head).number;
          if (curLine !== prevCursorLine) {
            prevCursorLine = curLine;
            // Force gutter re-render by requesting a measure pass
            update.view.requestMeasure();
          }
        },
      })),
    );
  } else {
    extensions.push(lineNumbers());
  }

  extensions.push(
    highlightActiveLine(),
    history(),
    EditorView.lineWrapping,
    markdown({ codeLanguages: languages }),
    keymap.of([
      { key: 'Mod-s', run: () => { if (onSave) onSave(); return true; } },
      ...defaultKeymap,
      ...historyKeymap,
    ]),
    themeCompartment.of(buildTheme()),
    EditorView.updateListener.of((update) => {
      if (update.docChanged && onChange) {
        onChange(update.state.doc.toString());
      }
    }),
    // Esc handler: exit edit mode when appropriate
    EditorView.domEventHandlers({
      keydown(event, view) {
        if (event.key === 'Escape') {
          // Don't intercept when vim command panel is active
          if (event.target.closest('.cm-vim-panel')) return false;

          if (!vimEnabled) {
            if (onQuit) onQuit();
            return true;
          }
          // Vim mode: only quit if in normal mode with no pending operator
          const cm = getCM(view);
          const vimState = cm?.state?.vim;
          if (vimState && !vimState.insertMode && vimState.mode !== 'visual' && !vimState.operator) {
            if (onQuit) onQuit();
            return true;
          }
        }
        return false;
      }
    }),
  );

  const view = new EditorView({
    state: EditorState.create({
      doc: content,
      extensions,
    }),
    parent: container,
  });

  return view;
}

/**
 * Destroy the editor and clean up.
 */
export function destroyEditor(view) {
  if (view) view.destroy();
}

/**
 * Hot-swap the theme compartment with fresh CSS var colors.
 */
export function updateTheme(view) {
  if (!view) return;
  view.dispatch({
    effects: themeCompartment.reconfigure(buildTheme()),
  });
}

/**
 * Replace the entire document content.
 */
export function setContent(view, content) {
  if (!view) return;
  view.dispatch({
    changes: { from: 0, to: view.state.doc.length, insert: content },
  });
}

/**
 * Get the current vim mode string. Returns null if vim is not active.
 */
export function getVimMode(view) {
  if (!view) return null;
  const cm = getCM(view);
  if (!cm) return null;
  const state = cm.state;
  if (!state) return null;
  const vimState = state.vim;
  if (!vimState) return null;
  if (vimState.visualLine) return 'V-LINE';
  if (vimState.visualBlock) return 'V-BLOCK';
  if (vimState.mode === 'visual') return 'VISUAL';
  if (vimState.mode === 'insert' || vimState.insertMode) return 'INSERT';
  if (vimState.mode === 'replace') return 'REPLACE';
  return 'NORMAL';
}

/**
 * Get the cursor position as { line, col } (1-indexed).
 */
export function getCursorPosition(view) {
  if (!view) return { line: 1, col: 1 };
  const pos = view.state.selection.main.head;
  const line = view.state.doc.lineAt(pos);
  return { line: line.number, col: pos - line.from + 1 };
}

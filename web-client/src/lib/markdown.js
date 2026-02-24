/**
 * Improved markdown renderer — line-by-line processing matching the TUI's
 * codeBlockLines + mdLineStyle approach. Returns HTML with semantic CSS classes
 * that map to theme-aware styles defined in app.css.
 */

/**
 * Escape HTML entities in a string.
 */
function escapeHtml(str) {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/**
 * Process inline markdown formatting within a line.
 * Handles: bold-italic, bold, italic, strikethrough, code spans, wikilinks, standard links.
 */
function processInline(line) {
  // Code spans first (so inner formatting is not processed)
  let result = '';
  let i = 0;
  while (i < line.length) {
    if (line[i] === '`') {
      const end = line.indexOf('`', i + 1);
      if (end !== -1) {
        result += `<span class="md-code">${escapeHtml(line.slice(i + 1, end))}</span>`;
        i = end + 1;
        continue;
      }
    }
    result += line[i];
    i++;
  }

  line = result;

  // Wikilinks: [[target|display]] and [[target]]
  line = line.replace(/\[\[([^\]|]+)\|([^\]]+)\]\]/g, '<a class="md-link md-wikilink" data-note="$1">$2</a>');
  line = line.replace(/\[\[([^\]]+)\]\]/g, '<a class="md-link md-wikilink" data-note="$1">$1</a>');

  // Standard links: [text](url) — external opens in new tab, relative treated as note link
  line = line.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, text, url) => {
    if (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('mailto:')) {
      return `<a class="md-link" href="${url}" target="_blank" rel="noopener">${text}</a>`;
    }
    return `<a class="md-link md-wikilink" data-note="${url}">${text}</a>`;
  });

  // Bold-italic: ***text*** or ___text___
  line = line.replace(/\*\*\*(.*?)\*\*\*/g, '<span class="md-bold-italic">$1</span>');
  line = line.replace(/___(.*?)___/g, '<span class="md-bold-italic">$1</span>');

  // Bold: **text** or __text__
  line = line.replace(/\*\*(.*?)\*\*/g, '<span class="md-bold">$1</span>');
  line = line.replace(/__(.*?)__/g, '<span class="md-bold">$1</span>');

  // Italic: *text* or _text_
  line = line.replace(/\*([^*]+)\*/g, '<span class="md-italic">$1</span>');
  line = line.replace(/(?<!\w)_([^_]+)_(?!\w)/g, '<span class="md-italic">$1</span>');

  // Strikethrough: ~~text~~
  line = line.replace(/~~(.*?)~~/g, '<span class="md-strike">$1</span>');

  return line;
}

/**
 * Render a markdown string to themed HTML.
 * @param {string} md - raw markdown text
 * @returns {string} HTML string
 */
export function renderMarkdown(md) {
  if (!md) return '';

  const lines = md.split('\n');
  const html = [];
  let inCodeBlock = false;
  let codeLang = '';
  let codeLines = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trimStart();

    // Fenced code block toggle
    if (trimmed.startsWith('```')) {
      if (!inCodeBlock) {
        inCodeBlock = true;
        codeLang = trimmed.slice(3).trim();
        codeLines = [];
      } else {
        // Close code block
        const langAttr = codeLang ? ` data-lang="${escapeHtml(codeLang)}"` : '';
        const langLabel = codeLang
          ? `<span class="md-code-lang">${escapeHtml(codeLang)}</span>`
          : '';
        html.push(
          `<div class="md-code-block"${langAttr}>${langLabel}<pre>${escapeHtml(codeLines.join('\n'))}</pre></div>`
        );
        inCodeBlock = false;
        codeLang = '';
        codeLines = [];
      }
      continue;
    }

    if (inCodeBlock) {
      codeLines.push(line);
      continue;
    }

    // Horizontal rule
    if (/^(\s*)(---+|___+|\*\*\*+)\s*$/.test(line)) {
      html.push('<div class="md-hr"></div>');
      continue;
    }

    // Headings
    const headingMatch = trimmed.match(/^(#{1,6})\s+(.*)$/);
    if (headingMatch) {
      const level = headingMatch[1].length;
      const text = processInline(escapeHtml(headingMatch[2]));
      html.push(`<div class="md-h${level}">${headingMatch[1]} ${text}</div>`);
      continue;
    }

    // Blockquote
    if (trimmed.startsWith('> ')) {
      const text = processInline(escapeHtml(trimmed.slice(2)));
      html.push(`<div class="md-blockquote">&gt; ${text}</div>`);
      continue;
    }

    // Unordered list
    const ulMatch = trimmed.match(/^([-*+])\s+(.*)$/);
    if (ulMatch) {
      const text = processInline(escapeHtml(ulMatch[2]));
      html.push(`<div class="md-list-item"><span class="md-list-marker">${ulMatch[1]}</span> ${text}</div>`);
      continue;
    }

    // Ordered list
    const olMatch = trimmed.match(/^(\d+\.)\s+(.*)$/);
    if (olMatch) {
      const text = processInline(escapeHtml(olMatch[2]));
      html.push(`<div class="md-list-item"><span class="md-list-marker">${olMatch[1]}</span> ${text}</div>`);
      continue;
    }

    // Empty line
    if (trimmed === '') {
      html.push('<div class="md-blank">&nbsp;</div>');
      continue;
    }

    // Normal paragraph line
    html.push(`<div class="md-line">${processInline(escapeHtml(line))}</div>`);
  }

  // Handle unclosed code block
  if (inCodeBlock && codeLines.length > 0) {
    const langAttr = codeLang ? ` data-lang="${escapeHtml(codeLang)}"` : '';
    html.push(
      `<div class="md-code-block"${langAttr}><pre>${escapeHtml(codeLines.join('\n'))}</pre></div>`
    );
  }

  return html.join('\n');
}

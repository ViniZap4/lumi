<script>
  import { onMount } from 'svelte';
  import { getNotes, getFolders, getNote, updateNote } from './lib/api.js';
  import { themes, themeOrder, applyTheme, loadSavedTheme, cycleTheme } from './lib/themes.js';
  import { renderMarkdown } from './lib/markdown.js';

  let viewMode = 'home'; // 'home', 'tree', 'note', 'config'
  let editMode = false;
  let allNotes = [];
  let allFolders = [];
  let displayItems = [];
  let cursor = 0;
  let selectedNote = null;
  let content = '';
  let title = '';
  let error = null;
  let saving = false;
  let currentDir = '/';

  // Search modal
  let showSearch = false;
  let searchQuery = '';
  let searchType = 'filename';
  let searchResults = [];
  let searchCursor = 0;

  // Theme state
  let currentThemeName = 'tokyo-night';
  $: currentTheme = themes[currentThemeName];

  // Config state
  let configCursor = 1; // skip first header
  let previousView = 'home';

  // Logo lines for per-line coloring
  const logoLines = [
    '  ██╗     ██╗   ██╗███╗   ███╗██╗',
    '  ██║     ██║   ██║████╗ ████║██║',
    '  ██║     ██║   ██║██╔████╔██║██║',
    '  ██║     ██║   ██║██║╚██╔╝██║██║',
    '  ███████╗╚██████╔╝██║ ╚═╝ ██║██║',
    '  ╚══════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝',
  ];

  // Sample markdown for config preview (matches TUI's previewSamples)
  const sampleMarkdown = [
    '# Heading 1',
    '## Heading 2',
    '### Heading 3',
    '',
    'Normal text with **bold** and *italic*.',
    'A `code span` and a [link](url).',
    '',
    '- List item one',
    '- Another with [[wikilink]]',
    '',
    '> Blockquote text here',
    '',
    '```',
    'code block line',
    '```',
    '',
    '---',
  ].join('\n');

  // Config items definition
  function buildConfigItems() {
    return [
      { label: 'Theme', kind: 'header' },
      { label: 'Theme', kind: 'cycle', key: 'theme', value: currentThemeName, options: themeOrder },
      { label: 'Display', kind: 'header' },
      { label: 'Search', kind: 'cycle', key: 'search_type', value: searchType, options: ['filename', 'content'] },
    ];
  }

  $: configItems = buildConfigItems();

  onMount(async () => {
    currentThemeName = loadSavedTheme();
    applyTheme(currentThemeName);
    await loadAll();
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  });

  async function loadAll() {
    try {
      error = null;
      allNotes = await getNotes('');
      allFolders = await getFolders();
      buildItems();
    } catch (err) {
      error = 'Failed to load: ' + err.message;
    }
  }

  function buildItems() {
    displayItems = [];
    allFolders.forEach(f => {
      displayItems.push({ type: 'folder', name: f.name, path: f.path, data: f });
    });
    allNotes.forEach(n => {
      displayItems.push({ type: 'note', name: n.title || n.id, id: n.id, data: n });
    });
    cursor = Math.min(cursor, Math.max(0, displayItems.length - 1));
  }

  $: currentItem = displayItems[cursor];
  $: previewNote = currentItem?.type === 'note' ? currentItem.data : null;

  function performSearch() {
    if (searchQuery === '') {
      searchResults = allNotes.map(n => ({
        type: 'note', name: n.title || n.id, id: n.id, data: n
      }));
    } else {
      const query = searchQuery.toLowerCase();
      searchResults = allNotes.filter(n => {
        if (searchType === 'filename') {
          return (n.title || n.id).toLowerCase().includes(query);
        } else {
          return (n.content || '').toLowerCase().includes(query);
        }
      }).map(n => ({
        type: 'note', name: n.title || n.id, id: n.id, data: n
      }));
    }
    searchCursor = 0;
  }

  async function openNote(item) {
    try {
      error = null;
      const note = await getNote(item.id);
      selectedNote = note;
      title = note.title || '';
      content = note.content || '';
      viewMode = 'note';
      editMode = false;
    } catch (err) {
      error = `Failed to open: ${err.message}`;
    }
  }

  async function save() {
    if (!selectedNote) return;
    try {
      saving = true;
      error = null;
      await updateNote(selectedNote.id, { title, content });
    } catch (err) {
      error = `Save failed: ${err.message}`;
    } finally {
      saving = false;
    }
  }

  // Config navigation
  function enterConfig() {
    previousView = viewMode;
    configCursor = 1; // skip first header
    viewMode = 'config';
  }

  function configMoveDown() {
    for (let i = configCursor + 1; i < configItems.length; i++) {
      if (configItems[i].kind !== 'header') { configCursor = i; return; }
    }
  }

  function configMoveUp() {
    for (let i = configCursor - 1; i >= 0; i--) {
      if (configItems[i].kind !== 'header') { configCursor = i; return; }
    }
  }

  function configCycleOption(direction) {
    const item = configItems[configCursor];
    if (!item || item.kind !== 'cycle') return;
    const opts = item.options;
    let idx = opts.indexOf(item.value);
    if (idx === -1) idx = 0;
    idx += direction;
    if (idx < 0) idx = opts.length - 1;
    if (idx >= opts.length) idx = 0;
    const newValue = opts[idx];

    // Apply
    if (item.key === 'theme') {
      currentThemeName = newValue;
      applyTheme(newValue);
    } else if (item.key === 'search_type') {
      searchType = newValue;
    }
    // Trigger rebuild
    configItems = buildConfigItems();
  }

  function formatDate(dateStr) {
    if (!dateStr) return '';
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
    } catch {
      return '';
    }
  }

  async function handleKey(e) {
    // Search modal
    if (showSearch) {
      if (e.key === 'Escape') {
        e.preventDefault();
        showSearch = false;
        searchQuery = '';
        searchCursor = 0;
        return;
      } else if ((e.key === 'j' || e.key === 'ArrowDown') && e.ctrlKey) {
        e.preventDefault();
        if (searchCursor < searchResults.length - 1) searchCursor++;
        return;
      } else if ((e.key === 'k' || e.key === 'ArrowUp') && e.ctrlKey) {
        e.preventDefault();
        if (searchCursor > 0) searchCursor--;
        return;
      } else if (e.key === 'Enter') {
        e.preventDefault();
        if (searchResults[searchCursor]) {
          await openNote(searchResults[searchCursor]);
          showSearch = false;
        }
        return;
      }
    }

    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

    // Home view
    if (viewMode === 'home' && !showSearch) {
      if (e.key === '/') { e.preventDefault(); showSearch = true; performSearch(); }
      else if (e.key === 't' || e.key === 'Enter') { e.preventDefault(); viewMode = 'tree'; }
      else if (e.key === 'c') { e.preventDefault(); enterConfig(); }
      return;
    }

    // Tree view
    if (viewMode === 'tree') {
      if (e.key === '/') { e.preventDefault(); showSearch = true; performSearch(); }
      else if (e.key === 'j' || e.key === 'ArrowDown') {
        e.preventDefault();
        if (cursor < displayItems.length - 1) cursor++;
      } else if (e.key === 'k' || e.key === 'ArrowUp') {
        e.preventDefault();
        if (cursor > 0) cursor--;
      } else if (e.key === 'Enter' || e.key === 'l') {
        e.preventDefault();
        if (displayItems[cursor]?.type === 'note') openNote(displayItems[cursor]);
      } else if (e.key === 'Escape' || e.key === 'h') {
        e.preventDefault();
        viewMode = 'home';
      } else if (e.key === 'c') {
        e.preventDefault();
        enterConfig();
      }
      return;
    }

    // Note view
    if (viewMode === 'note') {
      if (e.key === 'Escape') {
        e.preventDefault();
        if (editMode) { editMode = false; } else { viewMode = 'tree'; selectedNote = null; }
      } else if (e.key === 'h' && !editMode) {
        e.preventDefault();
        viewMode = 'home';
        selectedNote = null;
      } else if (e.key === 'e' && !editMode) {
        e.preventDefault();
        editMode = true;
      } else if (e.key === 'j' && !editMode) {
        e.preventDefault();
        const el = document.querySelector('.note-view-content');
        if (el) el.scrollBy(0, 40);
      } else if (e.key === 'k' && !editMode) {
        e.preventDefault();
        const el = document.querySelector('.note-view-content');
        if (el) el.scrollBy(0, -40);
      } else if (e.key === '/' && !editMode) {
        e.preventDefault();
        showSearch = true;
        performSearch();
      } else if (e.key === 'c' && !editMode) {
        e.preventDefault();
        enterConfig();
      }
      return;
    }

    // Config view
    if (viewMode === 'config') {
      if (e.key === 'Escape') { e.preventDefault(); viewMode = previousView; }
      else if (e.key === 'j' || e.key === 'ArrowDown') { e.preventDefault(); configMoveDown(); }
      else if (e.key === 'k' || e.key === 'ArrowUp') { e.preventDefault(); configMoveUp(); }
      else if (e.key === 'l' || e.key === 'ArrowRight') { e.preventDefault(); configCycleOption(1); }
      else if (e.key === 'h' || e.key === 'ArrowLeft') { e.preventDefault(); configCycleOption(-1); }
      return;
    }
  }

  $: if (showSearch) performSearch();
</script>

{#if error}
  <div class="error-toast">{error}</div>
{/if}

<!-- ─── Home View ─────────────────────────────────────────────── -->
{#if viewMode === 'home'}
  <div class="home-view">
    <div class="logo">
      {#each logoLines as line, i}
        <div class="logo-line" style="color: var(--color-logo-{i})">{line}</div>
      {/each}
    </div>
    <div class="subtitle">Local-first Markdown notes</div>
    <div class="home-keys">
      <div><span class="key-hint">/</span> Search notes</div>
      <div><span class="key-hint">t</span> Browse tree</div>
      <div><span class="key-hint">c</span> Settings</div>
    </div>
  </div>
{/if}

<!-- ─── Tree View ─────────────────────────────────────────────── -->
{#if viewMode === 'tree'}
  <div class="tree-view">
    <!-- Parent Column -->
    <div class="tree-col parent-col">
      <div class="col-header">Parent</div>
      <div class="col-body">
        <div class="parent-item">~</div>
      </div>
    </div>

    <!-- Center Column -->
    <div class="tree-col center-col">
      <div class="col-header col-header-primary">{currentDir}</div>
      <div class="col-body">
        {#each displayItems as item, i}
          <div
            class="tree-item"
            class:active={i === cursor}
            on:click={() => { cursor = i; if (item.type === 'note') openNote(item); }}
          >
            <span class="tree-icon">{item.type === 'folder' ? '/' : ' '}</span>
            <span class="tree-name">{item.name}</span>
          </div>
        {/each}
      </div>
      <div class="col-help">
        <span class="key-hint">hjkl</span> move
        <span class="key-hint">enter</span> open
        <span class="key-hint">/</span> search
        <span class="key-hint">c</span> config
        <span class="key-hint">esc</span> back
      </div>
    </div>

    <!-- Preview Column -->
    <div class="tree-col preview-col">
      <div class="col-header">Preview</div>
      <div class="col-body preview-body">
        {#if previewNote}
          <div class="preview-title">{previewNote.title || previewNote.id}</div>
          <div class="sep"></div>
          <div class="preview-md">
            {@html renderMarkdown(previewNote.content?.split('\n').slice(0, 20).join('\n') || '')}
          </div>
        {:else}
          <div class="empty-msg">Select a note to preview</div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- ─── Note View ─────────────────────────────────────────────── -->
{#if viewMode === 'note' && selectedNote}
  {#if editMode}
    <div class="note-split">
      <div class="note-editor-pane">
        <div class="note-bar">
          <input
            bind:value={title}
            placeholder="Title"
            class="title-input"
          />
          <div class="note-actions">
            <button on:click={save} disabled={saving}>
              {saving ? 'Saving...' : 'Save'}
            </button>
            <button on:click={() => editMode = false}>View (esc)</button>
          </div>
        </div>
        <div class="editor-body">
          <textarea
            bind:value={content}
            placeholder="Write your note..."
          ></textarea>
        </div>
        <div class="status-help">
          <span class="key-hint">ctrl+s</span> save
          <span class="key-hint">esc</span> view
        </div>
      </div>

      <div class="note-preview-pane">
        <div class="note-bar">
          <span class="bar-label">Live Preview</span>
        </div>
        <div class="preview-md-body">
          {@html renderMarkdown(content || '')}
        </div>
      </div>
    </div>
  {:else}
    <div class="note-full">
      <!-- Header bar -->
      <div class="note-header-bar">
        <span class="note-title-text">{title || 'Untitled'}</span>
        {#if selectedNote.tags?.length}
          {#each selectedNote.tags as tag}
            <span class="note-tag">#{tag}</span>
          {/each}
        {/if}
        <span class="note-date">{formatDate(selectedNote.updated_at || selectedNote.created_at)}</span>
      </div>
      <div class="sep"></div>

      <!-- Content -->
      <div class="note-view-content">
        {@html renderMarkdown(content || '')}
      </div>

      <!-- Footer -->
      <div class="sep"></div>
      <div class="note-status-bar">
        <span>View</span>
      </div>
      <div class="status-help">
        <span class="key-hint">jk</span> scroll
        <span class="key-hint">e</span> edit
        <span class="key-hint">c</span> config
        <span class="key-hint">h</span> home
        <span class="key-hint">esc</span> back
      </div>
    </div>
  {/if}
{/if}

<!-- ─── Config View ───────────────────────────────────────────── -->
{#if viewMode === 'config'}
  <div class="config-view">
    <!-- Left: Settings -->
    <div class="config-left">
      <div class="config-title">Lumi Settings</div>

      <div class="config-items">
        {#each configItems as item, i}
          {#if item.kind === 'header'}
            <div class="config-header" class:first={i === 0}>{item.label}</div>
          {:else if item.kind === 'cycle'}
            <div class="config-row" class:active={i === configCursor}>
              <span class="config-label">{item.label}</span>
              <span class="config-value">&lt; {item.value} &gt;</span>
            </div>
          {/if}
        {/each}
      </div>

      <!-- Color swatches -->
      <div class="config-swatches">
        <span class="swatch" style="color: var(--color-primary)">██</span>
        <span class="swatch" style="color: var(--color-secondary)">██</span>
        <span class="swatch" style="color: var(--color-accent)">██</span>
        <span class="swatch" style="color: var(--color-muted)">██</span>
        <span class="swatch" style="color: var(--color-text)">██</span>
        <span class="swatch" style="color: var(--color-error)">██</span>
        <span class="swatch" style="color: var(--color-warning)">██</span>
        <span class="swatch" style="color: var(--color-info)">██</span>
      </div>

      <div class="status-help">
        <span class="key-hint">j/k</span> move
        <span class="key-hint">h/l</span> change
        <span class="key-hint">esc</span> back
      </div>
    </div>

    <!-- Separator -->
    <div class="config-sep">
      {#each Array(60) as _}
        <div class="sep-char">│</div>
      {/each}
    </div>

    <!-- Right: Preview -->
    <div class="config-right">
      <!-- Preview header -->
      <div class="preview-header-row">
        <span class="preview-h-title">Sample Note</span>
        <span class="preview-h-tag">#demo</span>
        <span class="preview-h-tag">#theme</span>
        <span class="preview-h-date">Jan 1, 2026</span>
      </div>
      <div class="sep"></div>

      <!-- Sample markdown -->
      <div class="config-preview-md">
        {@html renderMarkdown(sampleMarkdown)}
      </div>

      <!-- Footer -->
      <div class="config-preview-footer">
        <div class="sep"></div>
        <div class="note-status-bar">
          <span>Ln 1  Col 1</span>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- ─── Search Modal ──────────────────────────────────────────── -->
{#if showSearch}
  <div class="modal-overlay" on:click={() => showSearch = false}>
    <div class="search-modal" on:click|stopPropagation>
      <div class="modal-title">Find Notes</div>

      <div class="search-input-box">
        <input
          type="text"
          bind:value={searchQuery}
          on:input={() => performSearch()}
          placeholder={`[${searchType === 'filename' ? 'Filename' : 'Content'}] Type to search...`}
          autofocus
        />
      </div>

      <div class="search-content">
        <div class="search-results">
          <div class="results-header">Results ({searchResults.length})</div>
          {#each searchResults.slice(0, 10) as result, i}
            <div
              class="result-item"
              class:active={i === searchCursor}
              on:click={() => { searchCursor = i; openNote(result); showSearch = false; }}
            >
              <span class="tree-name">{result.name}</span>
            </div>
          {/each}
          {#if searchResults.length === 0}
            <div class="empty-msg">No results</div>
          {/if}
        </div>

        <div class="search-preview">
          {#if searchResults[searchCursor]?.data}
            <div class="preview-title">{searchResults[searchCursor].name}</div>
            <div class="sep"></div>
            <div class="preview-md">
              {@html renderMarkdown(searchResults[searchCursor].data.content?.split('\n').slice(0, 10).join('\n') || '')}
            </div>
          {:else}
            <div class="empty-msg">No preview available</div>
          {/if}
        </div>
      </div>

      <div class="status-help">
        <span class="key-hint">ctrl+j/k</span> navigate
        <span class="key-hint">enter</span> open
        <span class="key-hint">esc</span> close
      </div>
    </div>
  </div>
{/if}

<style>
  /* ── Shared ──────────────────────────────────────────────────── */
  .sep {
    height: 1px;
    background: var(--color-separator);
    width: 100%;
  }

  .key-hint {
    color: var(--color-secondary);
    font-weight: 700;
  }

  .status-help {
    padding: 0.75rem 1.5rem;
    border-top: 1px solid var(--color-separator);
    font-size: 0.85rem;
    color: var(--color-muted);
    text-align: center;
    display: flex;
    gap: 1rem;
    justify-content: center;
  }

  .empty-msg {
    padding: 2rem;
    text-align: center;
    color: var(--color-muted);
    font-style: italic;
  }

  .error-toast {
    position: fixed;
    top: 1rem;
    right: 1rem;
    background: var(--color-error);
    color: var(--color-background);
    padding: 0.75rem 1.25rem;
    border-radius: 4px;
    z-index: 2000;
    font-weight: 600;
    font-size: 0.9rem;
  }

  /* ── Home View ───────────────────────────────────────────────── */
  .home-view {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
    width: 100vw;
    background: var(--color-background);
    color: var(--color-text);
  }

  .logo {
    font-size: 0.75rem;
    line-height: 1.2;
    font-weight: bold;
    white-space: pre;
    margin-bottom: 2rem;
  }

  .logo-line {
    /* color set via inline style from logoColors */
  }

  .subtitle {
    color: var(--color-muted);
    margin-bottom: 3rem;
  }

  .home-keys {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    color: var(--color-text);
    font-size: 0.95rem;
    text-align: center;
  }

  /* ── Tree View ───────────────────────────────────────────────── */
  .tree-view {
    display: grid;
    grid-template-columns: 1fr 1.3fr 1.7fr;
    height: 100vh;
    width: 100vw;
    background: var(--color-background);
    color: var(--color-text);
  }

  .tree-col {
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--color-separator);
    overflow: hidden;
  }

  .preview-col {
    border-right: none;
  }

  .col-header {
    padding: 1rem 1.25rem;
    border-bottom: 1px solid var(--color-separator);
    font-weight: 600;
    font-size: 0.85rem;
    color: var(--color-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .col-header-primary {
    color: var(--color-primary);
    text-transform: none;
    font-size: 0.95rem;
  }

  .col-body {
    flex: 1;
    overflow-y: auto;
    padding: 0.5rem;
  }

  .col-help {
    padding: 0.75rem 1.25rem;
    border-top: 1px solid var(--color-separator);
    font-size: 0.8rem;
    color: var(--color-muted);
    text-align: center;
    display: flex;
    gap: 0.75rem;
    justify-content: center;
    flex-wrap: wrap;
  }

  .parent-item {
    padding: 0.5rem 0.75rem;
    color: var(--color-muted);
  }

  .tree-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.75rem;
    cursor: pointer;
  }

  .tree-item:hover {
    background: var(--color-selected-bg);
  }

  .tree-item.active {
    background: var(--color-selected-bg);
    color: var(--color-accent);
  }

  .tree-icon {
    color: var(--color-muted);
    width: 1ch;
    flex-shrink: 0;
  }

  .tree-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .preview-body {
    padding: 1rem 1.25rem;
  }

  .preview-title {
    font-size: 1.1rem;
    font-weight: 600;
    color: var(--color-primary);
    margin-bottom: 0.75rem;
  }

  .preview-md {
    color: var(--color-text);
    line-height: 1.6;
    font-size: 0.9rem;
  }

  /* ── Note View — Full ────────────────────────────────────────── */
  .note-full {
    display: flex;
    flex-direction: column;
    height: 100vh;
    width: 100vw;
    background: var(--color-background);
    color: var(--color-text);
  }

  .note-header-bar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1rem 1.5rem;
    flex-wrap: wrap;
  }

  .note-title-text {
    font-weight: 700;
    font-size: 1.15rem;
    color: var(--color-primary);
  }

  .note-tag {
    color: var(--color-accent);
    font-size: 0.85rem;
  }

  .note-date {
    color: var(--color-muted);
    font-size: 0.85rem;
    margin-left: auto;
  }

  .note-view-content {
    flex: 1;
    padding: 1.5rem;
    overflow-y: auto;
    line-height: 1.7;
    scroll-behavior: smooth;
  }

  .note-status-bar {
    padding: 0.5rem 1.5rem;
    background: var(--color-selected-bg);
    color: var(--color-primary);
    font-size: 0.85rem;
  }

  /* ── Note View — Split (Edit) ────────────────────────────────── */
  .note-split {
    display: grid;
    grid-template-columns: 1fr 1fr;
    height: 100vh;
    width: 100vw;
    background: var(--color-background);
    color: var(--color-text);
  }

  .note-editor-pane {
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--color-separator);
  }

  .note-preview-pane {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .note-bar {
    display: flex;
    gap: 0.75rem;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid var(--color-separator);
    align-items: center;
  }

  .bar-label {
    color: var(--color-muted);
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .title-input {
    flex: 1;
    padding: 0.5rem 0.75rem;
    background: var(--color-overlay-bg);
    border: 1px solid var(--color-border);
    border-radius: 3px;
    color: var(--color-text);
    font-size: 1.05rem;
    font-weight: 600;
    font-family: inherit;
  }

  .title-input:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .note-actions {
    display: flex;
    gap: 0.5rem;
  }

  button {
    padding: 0.5rem 1rem;
    background: var(--color-selected-bg);
    color: var(--color-text);
    border: 1px solid var(--color-border);
    border-radius: 3px;
    font-weight: 600;
    cursor: pointer;
    font-size: 0.85rem;
    font-family: inherit;
  }

  button:hover:not(:disabled) {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .editor-body {
    flex: 1;
    padding: 1.25rem;
  }

  .editor-body textarea {
    width: 100%;
    height: 100%;
    background: transparent;
    border: none;
    color: var(--color-text);
    font-family: inherit;
    font-size: 0.95rem;
    line-height: 1.7;
    resize: none;
  }

  .editor-body textarea:focus {
    outline: none;
  }

  .preview-md-body {
    flex: 1;
    padding: 1.25rem;
    overflow-y: auto;
    line-height: 1.7;
    font-size: 0.95rem;
  }

  /* ── Config View ─────────────────────────────────────────────── */
  .config-view {
    display: flex;
    height: 100vh;
    width: 100vw;
    background: var(--color-background);
    color: var(--color-text);
  }

  .config-left {
    width: 38%;
    display: flex;
    flex-direction: column;
    padding: 1.5rem;
  }

  .config-title {
    text-align: center;
    font-weight: 700;
    color: var(--color-primary);
    font-size: 1.1rem;
    margin-bottom: 1.5rem;
  }

  .config-items {
    flex: 1;
  }

  .config-header {
    font-weight: 700;
    color: var(--color-primary);
    padding: 0.25rem 0.5rem;
    margin-top: 1rem;
  }
  .config-header.first {
    margin-top: 0;
  }

  .config-row {
    display: flex;
    justify-content: space-between;
    padding: 0.4rem 1.5rem;
  }

  .config-row .config-label {
    color: var(--color-text);
  }

  .config-row .config-value {
    color: var(--color-muted);
  }

  .config-row.active {
    background: var(--color-selected-bg);
  }

  .config-row.active .config-label {
    color: var(--color-accent);
  }

  .config-row.active .config-value {
    color: var(--color-secondary);
    font-weight: 700;
  }

  .config-swatches {
    text-align: center;
    padding: 1rem 0;
    display: flex;
    gap: 0.5rem;
    justify-content: center;
    font-size: 0.9rem;
  }

  .config-sep {
    display: flex;
    flex-direction: column;
    color: var(--color-separator);
    font-size: 0.85rem;
    line-height: 1;
    padding: 0;
    user-select: none;
  }

  .sep-char {
    text-align: center;
    height: calc(100vh / 60);
  }

  .config-right {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 1.5rem;
    overflow: hidden;
  }

  .preview-header-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding-bottom: 0.5rem;
  }

  .preview-h-title {
    font-weight: 700;
    color: var(--color-primary);
    font-size: 1.05rem;
  }

  .preview-h-tag {
    color: var(--color-accent);
    font-size: 0.85rem;
  }

  .preview-h-date {
    color: var(--color-muted);
    font-size: 0.85rem;
    margin-left: auto;
  }

  .config-preview-md {
    flex: 1;
    padding: 0.75rem 0;
    overflow-y: auto;
    line-height: 1.6;
    font-size: 0.9rem;
  }

  .config-preview-footer {
    margin-top: auto;
  }

  /* ── Search Modal ────────────────────────────────────────────── */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .search-modal {
    width: 90%;
    max-width: 850px;
    max-height: 80vh;
    background: var(--color-overlay-bg);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .modal-title {
    font-size: 1.1rem;
    font-weight: 700;
    color: var(--color-primary);
  }

  .search-input-box input {
    width: 100%;
    padding: 0.6rem 0.75rem;
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: 3px;
    color: var(--color-text);
    font-size: 0.95rem;
    font-family: inherit;
  }

  .search-input-box input:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .search-content {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
    flex: 1;
    min-height: 300px;
  }

  .search-results {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    overflow-y: auto;
  }

  .results-header {
    font-size: 0.8rem;
    color: var(--color-muted);
    margin-bottom: 0.5rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .result-item {
    display: flex;
    align-items: center;
    padding: 0.4rem 0.75rem;
    cursor: pointer;
  }

  .result-item:hover {
    background: var(--color-selected-bg);
  }

  .result-item.active {
    background: var(--color-selected-bg);
    color: var(--color-accent);
  }

  .search-preview {
    padding: 1rem;
    background: var(--color-background);
    border: 1px solid var(--color-separator);
    border-radius: 3px;
    overflow-y: auto;
  }
</style>

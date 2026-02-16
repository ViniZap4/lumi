<script>
  import { onMount } from 'svelte';
  import { getNotes, getNote, updateNote } from './lib/api.js';

  let notes = [];
  let items = [];
  let cursor = 0;
  let search = '';
  let previewContent = '';
  let selectedNote = null;
  let content = '';
  let title = '';
  let error = null;
  let saving = false;
  let viewMode = 'tree';

  onMount(async () => {
    await loadNotes();
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  });

  async function loadNotes() {
    try {
      error = null;
      notes = await getNotes('');
      items = notes.map(n => ({
        type: 'note',
        name: n.title || n.id,
        id: n.id,
        note: n
      }));
      updatePreview();
    } catch (err) {
      error = 'Failed to load: ' + err.message;
    }
  }

  function updatePreview() {
    if (cursor >= 0 && cursor < filteredItems.length) {
      const item = filteredItems[cursor];
      previewContent = item.note?.content || '';
    }
  }

  async function openNote(item) {
    try {
      error = null;
      const note = await getNote(item.id);
      selectedNote = note;
      title = note.title || '';
      content = note.content || '';
      viewMode = 'full';
    } catch (err) {
      error = `Failed to open: ${err.message}`;
      console.error('Open error:', err, 'Item:', item);
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

  function handleKey(e) {
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
    
    if (viewMode === 'full') {
      if (e.key === 'Escape') {
        e.preventDefault();
        viewMode = 'tree';
        selectedNote = null;
      }
      return;
    }

    switch (e.key) {
      case 'j':
        e.preventDefault();
        if (cursor < filteredItems.length - 1) {
          cursor++;
          updatePreview();
        }
        break;
      case 'k':
        e.preventDefault();
        if (cursor > 0) {
          cursor--;
          updatePreview();
        }
        break;
      case 'Enter':
        e.preventDefault();
        if (filteredItems[cursor]) openNote(filteredItems[cursor]);
        break;
      case '/':
        e.preventDefault();
        document.getElementById('search')?.focus();
        break;
      case 'Escape':
        e.preventDefault();
        search = '';
        break;
    }
  }

  $: filteredItems = search
    ? items.filter(i => i.name.toLowerCase().includes(search.toLowerCase()))
    : items;
  
  $: if (cursor >= filteredItems.length) cursor = Math.max(0, filteredItems.length - 1);
  $: if (cursor >= 0) updatePreview();
</script>

{#if error}
  <div class="error">{error}</div>
{/if}

{#if viewMode === 'tree'}
  <div class="container">
    <div class="sidebar">
      <div class="header">
        <h2>📝 Notes</h2>
        <input 
          id="search"
          type="text" 
          bind:value={search} 
          placeholder="Search (/)..."
        />
      </div>
      
      <div class="items">
        {#each filteredItems as item, i}
          <div 
            class="item"
            class:active={i === cursor}
            on:click={() => { cursor = i; openNote(item); }}
          >
            <span class="icon">📄</span>
            <span class="name">{item.name}</span>
          </div>
        {/each}
        {#if filteredItems.length === 0}
          <div class="empty">No notes found</div>
        {/if}
      </div>
      
      <div class="help">
        j/k • enter • /search • esc
      </div>
    </div>

    <div class="preview">
      <div class="preview-header">Preview</div>
      <div class="preview-content">{@html renderMarkdown(previewContent)}</div>
    </div>
  </div>
{:else}
  <div class="editor-view">
    <div class="toolbar">
      <input 
        bind:value={title} 
        placeholder="Title"
        class="title-input"
      />
      <div class="actions">
        <button on:click={save} disabled={saving}>
          {saving ? 'Saving...' : 'Save'}
        </button>
        <button on:click={() => viewMode = 'tree'}>
          Back (esc)
        </button>
      </div>
    </div>
    <textarea 
      bind:value={content}
      class="editor"
      placeholder="Write your note..."
    ></textarea>
  </div>
{/if}

<style>
  * { box-sizing: border-box; }

  .error {
    position: fixed;
    top: 1rem;
    right: 1rem;
    background: #dc2626;
    color: white;
    padding: 1rem 1.5rem;
    border-radius: 8px;
    box-shadow: 0 4px 6px rgba(0,0,0,0.3);
    z-index: 1000;
  }

  .container {
    display: grid;
    grid-template-columns: 400px 1fr;
    height: 100vh;
    background: #0a0a0a;
    color: #e5e5e5;
  }

  .sidebar {
    display: flex;
    flex-direction: column;
    border-right: 1px solid #262626;
    background: #0f0f0f;
  }

  .header {
    padding: 1.5rem;
    border-bottom: 1px solid #262626;
  }

  .header h2 {
    margin: 0 0 1rem 0;
    font-size: 1.25rem;
    color: #fbbf24;
  }

  input[type="text"] {
    width: 100%;
    padding: 0.75rem;
    background: #1a1a1a;
    border: 1px solid #404040;
    border-radius: 6px;
    color: #e5e5e5;
    font-size: 0.95rem;
  }

  input[type="text"]:focus {
    outline: none;
    border-color: #3b82f6;
  }

  .items {
    flex: 1;
    overflow-y: auto;
    padding: 0.5rem;
  }

  .item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    margin-bottom: 0.25rem;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .item:hover {
    background: #1a1a1a;
  }

  .item.active {
    background: #3b82f6;
    color: white;
  }

  .icon {
    font-size: 1.25rem;
  }

  .name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty {
    padding: 2rem;
    text-align: center;
    color: #737373;
  }

  .help {
    padding: 1rem 1.5rem;
    border-top: 1px solid #262626;
    font-size: 0.85rem;
    color: #737373;
    text-align: center;
  }

  .preview {
    display: flex;
    flex-direction: column;
    background: #0a0a0a;
  }

  .preview-header {
    padding: 1.5rem;
    border-bottom: 1px solid #262626;
    font-weight: 600;
    color: #a3a3a3;
  }

  .preview-content {
    flex: 1;
    padding: 2rem;
    overflow-y: auto;
    font-size: 0.95rem;
    line-height: 1.7;
    color: #d4d4d4;
  }

  .preview-content :global(h1) { color: #fbbf24; font-size: 2em; margin: 1em 0 0.5em; }
  .preview-content :global(h2) { color: #60a5fa; font-size: 1.5em; margin: 1em 0 0.5em; }
  .preview-content :global(h3) { color: #34d399; font-size: 1.25em; margin: 1em 0 0.5em; }
  .preview-content :global(code) { background: #1a1a1a; padding: 0.2em 0.4em; border-radius: 3px; color: #f87171; }
  .preview-content :global(pre) { background: #1a1a1a; padding: 1em; border-radius: 6px; overflow-x: auto; }
  .preview-content :global(pre code) { background: none; color: #e5e5e5; }
  .preview-content :global(a) { color: #60a5fa; text-decoration: none; }
  .preview-content :global(a:hover) { text-decoration: underline; }
  .preview-content :global(blockquote) { border-left: 4px solid #404040; padding-left: 1em; color: #a3a3a3; }
  .preview-content :global(ul), .preview-content :global(ol) { padding-left: 2em; }
  .preview-content :global(li) { margin: 0.5em 0; }

  .editor-view {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: #0a0a0a;
  }

  .toolbar {
    display: flex;
    gap: 1rem;
    padding: 1.5rem;
    border-bottom: 1px solid #262626;
    background: #0f0f0f;
  }

  .title-input {
    flex: 1;
    padding: 0.75rem 1rem;
    background: #1a1a1a;
    border: 1px solid #404040;
    border-radius: 6px;
    color: #e5e5e5;
    font-size: 1.25rem;
    font-weight: 600;
  }

  .title-input:focus {
    outline: none;
    border-color: #3b82f6;
  }

  .actions {
    display: flex;
    gap: 0.75rem;
  }

  button {
    padding: 0.75rem 1.5rem;
    background: #3b82f6;
    color: white;
    border: none;
    border-radius: 6px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  button:hover {
    background: #2563eb;
  }

  button:disabled {
    background: #404040;
    cursor: not-allowed;
  }

  .editor {
    flex: 1;
    padding: 2rem;
    background: #0a0a0a;
    color: #e5e5e5;
    border: none;
    font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
    font-size: 1rem;
    line-height: 1.7;
    resize: none;
  }

  .editor:focus {
    outline: none;
  }

  ::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }

  ::-webkit-scrollbar-track {
    background: #0a0a0a;
  }

  ::-webkit-scrollbar-thumb {
    background: #404040;
    border-radius: 4px;
  }

  ::-webkit-scrollbar-thumb:hover {
    background: #525252;
  }
</style>

<script context="module">
  function renderMarkdown(md) {
    if (!md) return '';
    
    // Simple markdown to HTML
    let html = md
      // Headers
      .replace(/^### (.*$)/gim, '<h3>$1</h3>')
      .replace(/^## (.*$)/gim, '<h2>$1</h2>')
      .replace(/^# (.*$)/gim, '<h1>$1</h1>')
      // Bold
      .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
      // Italic
      .replace(/\*(.*?)\*/g, '<em>$1</em>')
      // Code
      .replace(/`([^`]+)`/g, '<code>$1</code>')
      // Links
      .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
      // Line breaks
      .replace(/\n/g, '<br>');
    
    return html;
  }
</script>

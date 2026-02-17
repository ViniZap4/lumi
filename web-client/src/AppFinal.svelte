<script>
  import { onMount } from 'svelte';
  import { getNotes, getFolders, getNote, updateNote } from './lib/api.js';

  let viewMode = 'home'; // 'home', 'tree', 'note'
  let editMode = false; // false = view only, true = edit
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
  let searchType = 'filename'; // 'filename' or 'content'
  let searchResults = [];
  let searchCursor = 0;

  onMount(async () => {
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
        type: 'note',
        name: n.title || n.id,
        id: n.id,
        data: n
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
        type: 'note',
        name: n.title || n.id,
        id: n.id,
        data: n
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
      editMode = false; // Start in view mode
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

  async function handleKey(e) {
    // Allow keys in search modal even when input focused
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
      if (e.key === '/') {
        e.preventDefault();
        showSearch = true;
        performSearch();
      } else if (e.key === 't' || e.key === 'Enter') {
        e.preventDefault();
        viewMode = 'tree';
      } else if (e.key === 'q') {
        e.preventDefault();
        // Could close window
      }
      return;
    }

    // Tree view
    if (viewMode === 'tree') {
      if (e.key === '/') {
        e.preventDefault();
        showSearch = true;
        performSearch();
      } else if (e.key === 'j' || e.key === 'ArrowDown') {
        e.preventDefault();
        if (cursor < displayItems.length - 1) cursor++;
      } else if (e.key === 'k' || e.key === 'ArrowUp') {
        e.preventDefault();
        if (cursor > 0) cursor--;
      } else if (e.key === 'Enter' || e.key === 'l') {
        e.preventDefault();
        if (displayItems[cursor] && displayItems[cursor].type === 'note') {
          openNote(displayItems[cursor]);
        }
      } else if (e.key === 'Escape' || e.key === 'h') {
        e.preventDefault();
        viewMode = 'home';
      }
      return;
    }

    // Note view
    if (viewMode === 'note') {
      if (e.key === 'Escape') {
        e.preventDefault();
        if (editMode) {
          editMode = false; // Exit edit mode first
        } else {
          viewMode = 'tree';
          selectedNote = null;
        }
      } else if (e.key === 'h') {
        e.preventDefault();
        viewMode = 'home';
        selectedNote = null;
      } else if (e.key === 'e' && !editMode) {
        e.preventDefault();
        editMode = true;
      } else if (e.key === 'j' && !editMode) {
        e.preventDefault();
        const content = document.querySelector('.note-view-content');
        if (content) content.scrollBy(0, 40);
      } else if (e.key === 'k' && !editMode) {
        e.preventDefault();
        const content = document.querySelector('.note-view-content');
        if (content) content.scrollBy(0, -40);
      } else if (e.key === '/') {
        e.preventDefault();
        showSearch = true;
        performSearch();
      }
    }
  }

  function renderMarkdown(md) {
    if (!md) return '';
    return md
      .replace(/^### (.*$)/gim, '<h3>$1</h3>')
      .replace(/^## (.*$)/gim, '<h2>$1</h2>')
      .replace(/^# (.*$)/gim, '<h1>$1</h1>')
      .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
      .replace(/\*(.*?)\*/g, '<em>$1</em>')
      .replace(/`([^`]+)`/g, '<code>$1</code>')
      .replace(/\[\[([^\]]+)\]\]/g, '<a href="#" class="wiki-link">$1</a>')
      .replace(/\n/g, '<br>');
  }

  $: if (showSearch) performSearch();
</script>

{#if error}
  <div class="error">{error}</div>
{/if}

<!-- Home View -->
{#if viewMode === 'home'}
  <div class="home-view">
    <div class="ascii-art">
  ██╗     ██╗   ██╗███╗   ███╗██╗
  ██║     ██║   ██║████╗ ████║██║
  ██║     ██║   ██║██╔████╔██║██║
  ██║     ██║   ██║██║╚██╔╝██║██║
  ███████╗╚██████╔╝██║ ╚═╝ ██║██║
  ╚══════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝
    </div>
    <div class="subtitle">Local-first Markdown notes</div>
    <div class="home-keys">
      <div>/  Search notes</div>
      <div>t  Browse tree</div>
    </div>
  </div>
{/if}

<!-- Tree View -->
{#if viewMode === 'tree'}
  <div class="tree-view-3col">
    <!-- Parent Column -->
    <div class="parent-col">
      <div class="col-header">Parent</div>
      <div class="col-content">
        <div class="parent-item">📁 ~</div>
      </div>
    </div>

    <!-- Center Column (Current) -->
    <div class="center-col">
      <div class="col-header">📂 {currentDir}</div>
      <div class="col-content">
        {#each displayItems as item, i}
          <div 
            class="tree-item"
            class:active={i === cursor}
            on:click={() => { cursor = i; if (item.type === 'note') openNote(item); }}
          >
            <span class="icon">{item.type === 'folder' ? '📁' : '📄'}</span>
            <span class="name">{item.name}</span>
          </div>
        {/each}
      </div>
      <div class="col-help">hjkl=move | enter=open | /=search | esc=back</div>
    </div>

    <!-- Preview Column -->
    <div class="preview-col">
      <div class="col-header">Preview</div>
      <div class="col-content">
        {#if previewNote}
          <div class="preview-title">{previewNote.title || previewNote.id}</div>
          <div class="preview-divider"></div>
          <div class="preview-text">
            {@html renderMarkdown(previewNote.content?.split('\n').slice(0, 20).join('\n') || '')}
          </div>
        {:else}
          <div class="no-preview">Select a note to preview</div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Note View -->
{#if viewMode === 'note' && selectedNote}
  {#if editMode}
    <!-- Edit Mode: Split view with preview -->
    <div class="note-view-split">
      <div class="note-editor">
        <div class="note-header">
          <input 
            bind:value={title} 
            placeholder="Title"
            class="note-title-input-header"
          />
          <div class="note-actions">
            <button on:click={save} disabled={saving}>
              {saving ? 'Saving...' : 'Save'}
            </button>
            <button on:click={() => editMode = false}>View (esc)</button>
          </div>
        </div>
        <div class="note-content">
          <textarea 
            bind:value={content}
            placeholder="Write your note..."
          ></textarea>
        </div>
        <div class="note-help">ctrl+s=save | esc=view</div>
      </div>
      
      <div class="note-preview">
        <div class="preview-header-note">Live Preview</div>
        <div class="preview-content-note">
          {@html renderMarkdown(content || '')}
        </div>
      </div>
    </div>
  {:else}
    <!-- View Mode: Full width -->
    <div class="note-view-full">
      <div class="note-header">
        <div class="note-title-display">{title || 'Untitled'}</div>
        <div class="note-actions">
          <button on:click={() => editMode = true}>Edit (e)</button>
          <button on:click={() => viewMode = 'tree'}>Back (esc)</button>
        </div>
      </div>
      <div class="note-view-content">
        {@html renderMarkdown(content || '')}
      </div>
      <div class="note-help">jk=scroll | e=edit | h=home | esc=back</div>
    </div>
  {/if}
{/if}

<!-- Search Modal -->
{#if showSearch}
  <div class="modal-overlay" on:click={() => showSearch = false}>
    <div class="search-modal" on:click|stopPropagation>
      <div class="modal-title">🔍 Find Notes</div>
      
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
              <span class="icon">📄</span>
              <span class="name">{result.name}</span>
            </div>
          {/each}
          {#if searchResults.length === 0}
            <div class="no-results">No results</div>
          {/if}
        </div>

        <div class="search-preview">
          {#if searchResults[searchCursor]?.data}
            <div class="preview-title">{searchResults[searchCursor].name}</div>
            <div class="preview-divider"></div>
            <div class="preview-text">
              {@html renderMarkdown(searchResults[searchCursor].data.content?.split('\n').slice(0, 10).join('\n') || '')}
            </div>
          {:else}
            <div class="no-preview">No preview available</div>
          {/if}
        </div>
      </div>

      <div class="modal-help">ctrl+j/k=navigate | enter=open | esc=close</div>
    </div>
  </div>
{/if}

<style>
  :global(body), :global(html) {
    margin: 0;
    padding: 0;
    height: 100%;
    overflow: hidden;
  }

  * { 
    box-sizing: border-box; 
    margin: 0; 
    padding: 0; 
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes slideUp {
    from { opacity: 0; transform: translateY(20px); }
    to { opacity: 1; transform: translateY(0); }
  }

  /* Home View */
  .home-view {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
    width: 100vw;
    background: #000;
    color: #e5e5e5;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    animation: fadeIn 0.4s ease-out;
  }

  .ascii-art {
    font-family: monospace;
    font-size: 0.75rem;
    line-height: 1.2;
    color: #fbbf24;
    font-weight: bold;
    white-space: pre;
    margin-bottom: 2rem;
    animation: slideUp 0.6s ease-out;
  }

  .subtitle {
    color: #a3a3a3;
    margin-bottom: 3rem;
    animation: slideUp 0.7s ease-out;
  }

  .home-keys {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    color: #d4d4d4;
    font-size: 0.95rem;
    text-align: center;
    animation: slideUp 0.8s ease-out;
  }

  /* Tree View - 3 Column */
  .tree-view-3col {
    display: grid;
    grid-template-columns: 1fr 2fr 2fr;
    height: 100vh;
    width: 100vw;
    background: #000;
    color: #e5e5e5;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    animation: fadeIn 0.4s ease-out;
  }

  .parent-col, .center-col, .preview-col {
    display: flex;
    flex-direction: column;
    border-right: 1px solid rgba(255, 255, 255, 0.08);
    overflow: hidden;
  }

  .preview-col {
    border-right: none;
  }

  .col-header {
    padding: 1.5rem 1.5rem 1rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    font-weight: 600;
    font-size: 0.875rem;
    color: #a3a3a3;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .center-col .col-header {
    background: linear-gradient(135deg, #c084fc 0%, #e879f9 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    font-size: 1rem;
    text-transform: none;
  }

  .col-content {
    flex: 1;
    overflow-y: auto;
    padding: 0.75rem;
  }

  .col-help {
    padding: 1rem 1.5rem;
    border-top: 1px solid rgba(255, 255, 255, 0.08);
    font-size: 0.85rem;
    color: #737373;
    text-align: center;
    font-family: 'SF Mono', monospace;
  }

  .parent-item {
    padding: 0.75rem 1rem;
    color: #a3a3a3;
    font-size: 0.95rem;
  }

  .tree-item {
    display: flex;
    align-items: center;
    gap: 0.875rem;
    padding: 0.875rem 1rem;
    margin-bottom: 0.375rem;
    border-radius: 10px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .tree-item:hover {
    background: rgba(255, 255, 255, 0.05);
    transform: translateX(4px);
  }

  .tree-item.active {
    background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
    color: white;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
    transform: translateX(4px);
  }

  .preview-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: #fbbf24;
    margin-bottom: 1rem;
  }

  .preview-divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.1);
    margin-bottom: 1rem;
  }

  .preview-text {
    color: #d4d4d4;
    line-height: 1.7;
    font-size: 0.95rem;
  }

  .preview-text :global(h1) { display: none; } /* Hide title in tree preview */
  .preview-text :global(h2) { color: #60a5fa; font-size: 1.25em; margin: 1em 0 0.5em; font-weight: 600; }
  .preview-text :global(h3) { color: #34d399; font-size: 1.1em; margin: 1em 0 0.5em; font-weight: 600; }
  .preview-text :global(code) { background: rgba(255, 255, 255, 0.1); padding: 0.2em 0.4em; border-radius: 4px; color: #f87171; }
  .preview-text :global(.wiki-link) { color: #60a5fa; text-decoration: underline; }

  .no-preview {
    padding: 2rem;
    text-align: center;
    color: #737373;
    font-style: italic;
  }

  /* Note View - Split */
  .note-view-split {
    display: grid;
    grid-template-columns: 1fr 1fr;
    height: 100vh;
    width: 100vw;
    background: #000;
    color: #e5e5e5;
    animation: fadeIn 0.4s ease-out;
  }

  /* Note View - Full Width (View Mode) */
  .note-view-full {
    display: flex;
    flex-direction: column;
    height: 100vh;
    width: 100vw;
    background: #000;
    color: #e5e5e5;
    animation: fadeIn 0.4s ease-out;
  }

  .note-editor {
    display: flex;
    flex-direction: column;
    border-right: 1px solid rgba(255, 255, 255, 0.08);
  }

  .note-preview {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .preview-header-note {
    padding: 2rem 2.5rem 1.5rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    font-weight: 600;
    color: #a3a3a3;
    font-size: 0.875rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .preview-content-note {
    flex: 1;
    padding: 2.5rem;
    overflow-y: auto;
    font-size: 1rem;
    line-height: 1.8;
    color: #d4d4d4;
  }

  .preview-content-note :global(h1) { display: none; } /* Hide title in preview */
  .preview-content-note :global(h2) { color: #60a5fa; font-size: 1.5em; margin: 1em 0 0.5em; font-weight: 600; }
  .preview-content-note :global(h3) { color: #34d399; font-size: 1.25em; margin: 1em 0 0.5em; font-weight: 600; }
  .preview-content-note :global(code) { background: rgba(255, 255, 255, 0.1); padding: 0.2em 0.4em; border-radius: 4px; color: #f87171; }
  .preview-content-note :global(.wiki-link) { color: #60a5fa; text-decoration: underline; }

  .note-title-display {
    flex: 1;
    font-size: 1.375rem;
    font-weight: 600;
    color: #c084fc; /* Purple-pink */
  }

  /* Animations for mode transitions */
  .note-view-split, .note-view-full {
    animation: fadeIn 0.3s ease-out;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .note-title-input {
    width: 100%;
    padding: 0.875rem 1.25rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 10px;
    color: #e5e5e5;
    font-size: 1.125rem;
    font-weight: 600;
    margin-bottom: 1rem;
  }

  .note-title-input:focus {
    outline: none;
    border-color: #3b82f6;
    background: rgba(59, 130, 246, 0.1);
  }

  .note-view-content {
    padding: 2.5rem;
    overflow-y: auto;
    line-height: 1.8;
    scroll-behavior: smooth;
  }

  .note-view-content :global(h1) { display: none; } /* Hide title in content */
  .note-view-content :global(h2) { color: #60a5fa; font-size: 1.5em; margin: 1em 0 0.5em; font-weight: 600; }
  .note-view-content :global(h3) { color: #34d399; font-size: 1.25em; margin: 1em 0 0.5em; font-weight: 600; }
  .note-view-content :global(code) { background: rgba(255, 255, 255, 0.1); padding: 0.2em 0.4em; border-radius: 4px; color: #f87171; }
  .note-view-content :global(.wiki-link) { color: #60a5fa; text-decoration: underline; }

  .note-header {
    display: flex;
    gap: 1rem;
    padding: 2rem 2.5rem 1.5rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    align-items: center;
  }

  .note-title-input-header {
    flex: 1;
    padding: 0.875rem 1.25rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 10px;
    color: #e5e5e5;
    font-size: 1.375rem;
    font-weight: 600;
    transition: all 0.2s;
  }

  .note-title-input-header:focus {
    outline: none;
    border-color: #3b82f6;
    background: rgba(59, 130, 246, 0.1);
  }

  .note-title {
    flex: 1;
    padding: 0.875rem 1.25rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 10px;
    color: #e5e5e5;
    font-size: 1.375rem;
    font-weight: 600;
    transition: all 0.2s;
  }

  .note-title:focus {
    outline: none;
    border-color: #3b82f6;
    background: rgba(59, 130, 246, 0.1);
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }

  .note-actions {
    display: flex;
    gap: 0.75rem;
  }

  button {
    padding: 0.875rem 1.75rem;
    background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
    color: white;
    border: none;
    border-radius: 10px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    font-size: 0.95rem;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.2);
  }

  button:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 6px 16px rgba(59, 130, 246, 0.3);
  }

  button:disabled {
    background: rgba(255, 255, 255, 0.1);
    cursor: not-allowed;
    box-shadow: none;
  }

  .note-content {
    flex: 1;
    padding: 2.5rem;
  }

  .note-content textarea {
    width: 100%;
    height: 100%;
    background: transparent;
    border: none;
    color: #e5e5e5;
    font-family: 'SF Mono', monospace;
    font-size: 1rem;
    line-height: 1.8;
    resize: none;
  }

  .note-content textarea:focus {
    outline: none;
  }

  .note-help {
    padding: 1.25rem 2.5rem;
    border-top: 1px solid rgba(255, 255, 255, 0.08);
    font-size: 0.85rem;
    color: #737373;
    text-align: center;
    font-family: 'SF Mono', monospace;
  }

  /* Search Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.8);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    animation: fadeIn 0.2s ease-out;
  }

  .search-modal {
    width: 90%;
    max-width: 900px;
    max-height: 80vh;
    background: linear-gradient(180deg, #0a0a0a 0%, #050505 100%);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 16px;
    padding: 2rem;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    animation: slideUp 0.3s ease-out;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  }

  .modal-title {
    font-size: 1.5rem;
    font-weight: 700;
    background: linear-gradient(135deg, #c084fc 0%, #e879f9 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .search-input-box input {
    width: 100%;
    padding: 1rem 1.25rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 10px;
    color: #e5e5e5;
    font-size: 1rem;
    transition: all 0.2s;
  }

  .search-input-box input:focus {
    outline: none;
    border-color: #3b82f6;
    background: rgba(59, 130, 246, 0.1);
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }

  .search-content {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1.5rem;
    flex: 1;
    min-height: 400px;
    transition: all 0.3s ease-out;
  }

  .search-results {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    overflow-y: auto;
  }

  .results-header {
    font-size: 0.875rem;
    color: #a3a3a3;
    margin-bottom: 0.5rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .result-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .result-item:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  .result-item.active {
    background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
    color: white;
  }

  .search-preview {
    padding: 1.5rem;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 10px;
    overflow-y: auto;
  }

  .preview-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: #fbbf24;
    margin-bottom: 1rem;
  }

  .preview-divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.1);
    margin-bottom: 1rem;
  }

  .preview-text {
    color: #d4d4d4;
    line-height: 1.7;
    font-size: 0.95rem;
  }

  .preview-text :global(h1) { display: none; } /* Hide title in search preview */
  .preview-text :global(h2) { color: #60a5fa; font-size: 1.25em; margin: 1em 0 0.5em; font-weight: 600; }
  .preview-text :global(h3) { color: #34d399; font-size: 1.1em; margin: 1em 0 0.5em; font-weight: 600; }
  .preview-text :global(code) { background: rgba(255, 255, 255, 0.1); padding: 0.2em 0.4em; border-radius: 4px; color: #f87171; }
  .preview-text :global(.wiki-link) { color: #60a5fa; text-decoration: underline; }

  .no-results, .no-preview {
    padding: 2rem;
    text-align: center;
    color: #737373;
    font-style: italic;
  }

  .modal-help {
    font-size: 0.85rem;
    color: #737373;
    text-align: center;
    font-family: 'SF Mono', monospace;
  }

  .icon {
    font-size: 1.25rem;
    flex-shrink: 0;
  }

  .name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 500;
  }

  .error {
    position: fixed;
    top: 1.5rem;
    right: 1.5rem;
    background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
    color: white;
    padding: 1rem 1.5rem;
    border-radius: 12px;
    box-shadow: 0 8px 24px rgba(220, 38, 38, 0.3);
    z-index: 2000;
    max-width: 400px;
    animation: slideUp 0.3s ease-out;
  }

  ::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }

  ::-webkit-scrollbar-track {
    background: transparent;
  }

  ::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.2);
    border-radius: 10px;
  }

  ::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.3);
  }
</style>

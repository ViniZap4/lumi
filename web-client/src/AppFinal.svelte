<script>
  import { onMount } from 'svelte';
  import { getNotes, getFolders, getNote, updateNote } from './lib/api.js';

  let allNotes = [];
  let allFolders = [];
  let displayItems = [];
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
      console.error('Load error:', err);
    }
  }

  function buildItems() {
    displayItems = [];
    
    // Add folders
    allFolders.forEach(f => {
      displayItems.push({
        type: 'folder',
        name: f.name,
        path: f.path,
        data: f
      });
    });
    
    // Add notes
    allNotes.forEach(n => {
      displayItems.push({
        type: 'note',
        name: n.title || n.id,
        id: n.id,
        data: n
      });
    });
    
    cursor = Math.min(cursor, Math.max(0, displayItems.length - 1));
    updatePreview();
  }

  function updatePreview() {
    if (cursor >= 0 && cursor < filteredItems.length) {
      const item = filteredItems[cursor];
      if (item.type === 'note' && item.data) {
        previewContent = item.data.content || '';
      } else if (item.type === 'folder') {
        const folderNotes = allNotes.filter(n => 
          n.path && n.path.includes(item.path)
        );
        previewContent = folderNotes.length > 0
          ? folderNotes.map(n => `📄 ${n.title || n.id}`).join('\n')
          : '(empty folder)';
      }
    }
  }

  async function openItem(item) {
    if (item.type === 'folder') {
      // Could navigate into folder, but for now just show preview
      return;
    }
    
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
        if (filteredItems[cursor]) openItem(filteredItems[cursor]);
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

  function renderMarkdown(md) {
    if (!md) return '';
    return md
      .replace(/^### (.*$)/gim, '<h3>$1</h3>')
      .replace(/^## (.*$)/gim, '<h2>$1</h2>')
      .replace(/^# (.*$)/gim, '<h1>$1</h1>')
      .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
      .replace(/\*(.*?)\*/g, '<em>$1</em>')
      .replace(/`([^`]+)`/g, '<code>$1</code>')
      .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
      .replace(/\n/g, '<br>');
  }

  $: filteredItems = search
    ? displayItems.filter(i => i.name.toLowerCase().includes(search.toLowerCase()))
    : displayItems;
  
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
        <h2>📝 Lumi Notes</h2>
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
            on:click={() => { cursor = i; if (item.type === 'note') openItem(item); }}
          >
            <span class="icon">{item.type === 'folder' ? '📁' : '📄'}</span>
            <span class="name">{item.name}</span>
          </div>
        {/each}
        {#if filteredItems.length === 0}
          <div class="empty">No items found</div>
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
  * { 
    box-sizing: border-box; 
    margin: 0; 
    padding: 0; 
  }

  @keyframes slideIn {
    from { opacity: 0; transform: translateY(-10px); }
    to { opacity: 1; transform: translateY(0); }
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
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
    z-index: 1000;
    max-width: 400px;
    animation: slideIn 0.3s ease-out;
    backdrop-filter: blur(10px);
  }

  .container {
    display: grid;
    grid-template-columns: 380px 1fr;
    height: 100vh;
    background: #000;
    color: #e5e5e5;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Inter', sans-serif;
    animation: fadeIn 0.4s ease-out;
  }

  .sidebar {
    display: flex;
    flex-direction: column;
    border-right: 1px solid rgba(255, 255, 255, 0.08);
    background: linear-gradient(180deg, #0a0a0a 0%, #050505 100%);
    backdrop-filter: blur(20px);
  }

  .header {
    padding: 2rem 1.5rem 1.5rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    animation: slideIn 0.4s ease-out;
  }

  .header h2 {
    margin: 0 0 1.25rem 0;
    font-size: 1.5rem;
    background: linear-gradient(135deg, #fbbf24 0%, #f59e0b 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    font-weight: 700;
    letter-spacing: -0.02em;
  }

  input[type="text"] {
    width: 100%;
    padding: 0.875rem 1rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 10px;
    color: #e5e5e5;
    font-size: 0.95rem;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }

  input[type="text"]:focus {
    outline: none;
    border-color: #3b82f6;
    background: rgba(59, 130, 246, 0.1);
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }

  .items {
    flex: 1;
    overflow-y: auto;
    padding: 0.75rem;
  }

  .item {
    display: flex;
    align-items: center;
    gap: 0.875rem;
    padding: 0.875rem 1rem;
    margin-bottom: 0.375rem;
    border-radius: 10px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    position: relative;
    overflow: hidden;
  }

  .item::before {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.1) 0%, rgba(147, 51, 234, 0.1) 100%);
    opacity: 0;
    transition: opacity 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .item:hover::before {
    opacity: 1;
  }

  .item:hover {
    transform: translateX(4px);
    background: rgba(255, 255, 255, 0.05);
  }

  .item.active {
    background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
    color: white;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
    transform: translateX(4px);
  }

  .item.active::before {
    opacity: 0;
  }

  .icon {
    font-size: 1.25rem;
    flex-shrink: 0;
    position: relative;
    z-index: 1;
  }

  .name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    position: relative;
    z-index: 1;
    font-weight: 500;
  }

  .empty {
    padding: 3rem 2rem;
    text-align: center;
    color: #737373;
    font-style: italic;
  }

  .help {
    padding: 1.25rem 1.5rem;
    border-top: 1px solid rgba(255, 255, 255, 0.08);
    font-size: 0.85rem;
    color: #737373;
    text-align: center;
    font-family: 'SF Mono', monospace;
  }

  .preview {
    display: flex;
    flex-direction: column;
    background: #000;
    animation: fadeIn 0.5s ease-out;
  }

  .preview-header {
    padding: 2rem 2.5rem 1.5rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    font-weight: 600;
    color: #a3a3a3;
    font-size: 0.875rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .preview-content {
    flex: 1;
    padding: 2.5rem;
    overflow-y: auto;
    font-size: 1rem;
    line-height: 1.8;
    color: #d4d4d4;
    animation: fadeIn 0.6s ease-out;
  }

  .preview-content :global(h1) { 
    color: #fbbf24; 
    font-size: 2.25em; 
    margin: 1.5em 0 0.75em; 
    font-weight: 700; 
    letter-spacing: -0.02em;
    line-height: 1.2;
  }
  
  .preview-content :global(h2) { 
    color: #60a5fa; 
    font-size: 1.75em; 
    margin: 1.25em 0 0.5em; 
    font-weight: 600; 
    letter-spacing: -0.01em;
    line-height: 1.3;
  }
  
  .preview-content :global(h3) { 
    color: #34d399; 
    font-size: 1.375em; 
    margin: 1em 0 0.5em; 
    font-weight: 600;
    line-height: 1.4;
  }
  
  .preview-content :global(code) { 
    background: rgba(255, 255, 255, 0.05); 
    padding: 0.25em 0.5em; 
    border-radius: 6px; 
    color: #f87171; 
    font-family: 'SF Mono', 'Monaco', monospace;
    font-size: 0.9em;
  }
  
  .preview-content :global(pre) { 
    background: rgba(255, 255, 255, 0.03); 
    padding: 1.5em; 
    border-radius: 12px; 
    overflow-x: auto;
    border: 1px solid rgba(255, 255, 255, 0.08);
  }
  
  .preview-content :global(pre code) { 
    background: none; 
    color: #e5e5e5;
    padding: 0;
  }
  
  .preview-content :global(a) { 
    color: #60a5fa; 
    text-decoration: none;
    transition: color 0.2s;
  }
  
  .preview-content :global(a:hover) { 
    color: #93c5fd;
    text-decoration: underline;
  }
  
  .preview-content :global(strong) { 
    font-weight: 600; 
    color: #fff; 
  }
  
  .preview-content :global(em) { 
    font-style: italic; 
    color: #d4d4d4; 
  }
  
  .preview-content :global(blockquote) { 
    border-left: 4px solid #3b82f6; 
    padding-left: 1.5em; 
    color: #a3a3a3; 
    margin: 1.5em 0;
    font-style: italic;
  }
  
  .preview-content :global(ul), .preview-content :global(ol) { 
    padding-left: 2em; 
    margin: 1.25em 0; 
  }
  
  .preview-content :global(li) { 
    margin: 0.625em 0; 
  }

  .editor-view {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: #000;
    animation: fadeIn 0.4s ease-out;
  }

  .toolbar {
    display: flex;
    gap: 1rem;
    padding: 2rem 2.5rem 1.5rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    background: linear-gradient(180deg, #0a0a0a 0%, #050505 100%);
    backdrop-filter: blur(20px);
  }

  .title-input {
    flex: 1;
    padding: 0.875rem 1.25rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 10px;
    color: #e5e5e5;
    font-size: 1.375rem;
    font-weight: 600;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .title-input:focus {
    outline: none;
    border-color: #3b82f6;
    background: rgba(59, 130, 246, 0.1);
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }

  .actions {
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

  button:active:not(:disabled) {
    transform: translateY(0);
  }

  button:disabled {
    background: rgba(255, 255, 255, 0.1);
    cursor: not-allowed;
    box-shadow: none;
  }

  .editor {
    flex: 1;
    padding: 2.5rem;
    background: #000;
    color: #e5e5e5;
    border: none;
    font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
    font-size: 1rem;
    line-height: 1.8;
    resize: none;
    transition: opacity 0.2s;
  }

  .editor:focus {
    outline: none;
  }

  ::-webkit-scrollbar {
    width: 10px;
    height: 10px;
  }

  ::-webkit-scrollbar-track {
    background: transparent;
  }

  ::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.2);
    border-radius: 10px;
    transition: background 0.2s;
  }

  ::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.3);
  }
</style>

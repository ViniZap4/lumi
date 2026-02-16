<script>
  import { onMount } from 'svelte';
  import { getNotes, getFolders, getNote, updateNote } from './lib/api.js';

  let folders = [];
  let notes = [];
  let currentPath = '';
  let parentItems = [];
  let currentItems = [];
  let previewContent = '';
  let selectedNote = null;
  let content = '';
  let title = '';
  let search = '';
  let saving = false;
  let error = null;
  let cursor = 0;
  let viewMode = 'tree'; // 'tree' or 'full'

  onMount(async () => {
    await loadAll();
    window.addEventListener('keydown', handleGlobalKey);
    return () => window.removeEventListener('keydown', handleGlobalKey);
  });

  async function loadAll() {
    try {
      error = null;
      const allFolders = await getFolders();
      const allNotes = await getNotes(currentPath);
      
      // Build current items
      currentItems = [];
      
      // Add folders in current path
      allFolders
        .filter(f => {
          const folderDir = f.path.substring(0, f.path.lastIndexOf('/'));
          return folderDir === currentPath || (currentPath === '' && !folderDir.includes('/'));
        })
        .forEach(f => {
          currentItems.push({ type: 'folder', name: f.name, path: f.path, data: f });
        });
      
      // Add notes
      allNotes.forEach(n => {
        currentItems.push({ type: 'note', name: n.title || n.id, data: n });
      });
      
      // Build parent items
      if (currentPath) {
        const parentPath = currentPath.substring(0, currentPath.lastIndexOf('/'));
        parentItems = allFolders
          .filter(f => {
            const folderDir = f.path.substring(0, f.path.lastIndexOf('/'));
            return folderDir === parentPath || (parentPath === '' && !folderDir.includes('/'));
          })
          .map(f => ({ type: 'folder', name: f.name, path: f.path }));
      } else {
        parentItems = [];
      }
      
      cursor = Math.min(cursor, currentItems.length - 1);
      await updatePreview();
    } catch (err) {
      error = 'Failed to load: ' + err.message;
      console.error(err);
    }
  }

  async function updatePreview() {
    if (cursor >= 0 && cursor < currentItems.length) {
      const item = currentItems[cursor];
      if (item.type === 'note') {
        try {
          const note = await getNote(item.data.id);
          previewContent = note.content || '';
        } catch (err) {
          previewContent = 'Failed to load preview';
          console.error('Preview error:', err, 'Note ID:', item.data.id);
        }
      } else if (item.type === 'folder') {
        // Show folder contents
        try {
          const folderNotes = await getNotes(item.path);
          previewContent = folderNotes.map(n => `📄 ${n.title || n.id}`).join('\n') || '(empty folder)';
        } catch (err) {
          previewContent = '(empty folder)';
        }
      }
    }
  }

  async function openItem(item) {
    if (item.type === 'folder') {
      currentPath = item.path;
      cursor = 0;
      await loadAll();
    } else {
      try {
        error = null;
        selectedNote = await getNote(item.data.id);
        title = selectedNote.title || '';
        content = selectedNote.content || '';
        viewMode = 'full';
      } catch (err) {
        error = 'Failed to load note: ' + err.message;
        console.error('Open error:', err, 'Note:', item.data);
      }
    }
  }

  async function goBack() {
    if (currentPath) {
      const parentPath = currentPath.substring(0, currentPath.lastIndexOf('/'));
      currentPath = parentPath;
      cursor = 0;
      await loadAll();
    }
  }

  async function saveNote() {
    if (!selectedNote) return;
    try {
      saving = true;
      error = null;
      await updateNote(selectedNote.id, { title, content });
      selectedNote.content = content;
      selectedNote.title = title;
    } catch (err) {
      error = 'Failed to save: ' + err.message;
    } finally {
      saving = false;
    }
  }

  function handleGlobalKey(e) {
    if (viewMode === 'full') {
      handleFullViewKey(e);
    } else {
      handleTreeKey(e);
    }
  }

  function handleTreeKey(e) {
    if (e.target.tagName === 'INPUT') return;

    switch (e.key) {
      case 'j':
        e.preventDefault();
        if (cursor < currentItems.length - 1) {
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
      case 'h':
        e.preventDefault();
        goBack();
        break;
      case 'l':
      case 'Enter':
        e.preventDefault();
        if (currentItems[cursor]) {
          openItem(currentItems[cursor]);
        }
        break;
      case '/':
        e.preventDefault();
        document.getElementById('search-input')?.focus();
        break;
      case 'Escape':
        e.preventDefault();
        search = '';
        document.getElementById('search-input')?.blur();
        break;
    }
  }

  function handleFullViewKey(e) {
    switch (e.key) {
      case 'Escape':
        e.preventDefault();
        viewMode = 'tree';
        selectedNote = null;
        break;
    }
  }

  $: filteredItems = search 
    ? currentItems.filter(item => item.name?.toLowerCase().includes(search.toLowerCase()))
    : currentItems;
</script>

{#if error}
  <div class="error">{error}</div>
{/if}

{#if viewMode === 'tree'}
  <div class="yazi-layout">
    <!-- Parent column -->
    <div class="column parent">
      <div class="column-title">Parent</div>
      {#each parentItems as item}
        <div class="item">{item.name}</div>
      {/each}
    </div>

    <!-- Current column -->
    <div class="column current">
      <div class="column-title">{currentPath || 'Root'}</div>
      <input 
        id="search-input"
        type="text" 
        bind:value={search} 
        placeholder="Search (or press /)"
        class="search-input"
      />
      {#each filteredItems as item, i}
        <div 
          class="item"
          class:selected={i === cursor}
          on:click={() => { cursor = i; openItem(item); }}
        >
          <span class="icon">{item.type === 'folder' ? '📁' : '📄'}</span>
          <span class="name">{item.name}</span>
        </div>
      {/each}
      {#if filteredItems.length === 0}
        <div class="empty">No items</div>
      {/if}
    </div>

    <!-- Preview column -->
    <div class="column preview">
      <div class="column-title">Preview</div>
      <pre class="preview-content">{previewContent}</pre>
    </div>
  </div>
  <div class="help">hjkl=move | enter/l=open | h=back | /=search | esc=clear</div>
{:else}
  <div class="full-view">
    <div class="note-header">
      <input bind:value={title} class="title-input" placeholder="Title" />
      <button on:click={saveNote} disabled={saving}>
        {saving ? 'Saving...' : 'Save'}
      </button>
      <button on:click={() => viewMode = 'tree'}>Back (esc)</button>
    </div>
    <textarea bind:value={content} class="content-editor"></textarea>
  </div>
{/if}

<style>
  .error {
    background: #ff4444;
    color: white;
    padding: 1rem;
    margin: 1rem;
    border-radius: 4px;
  }

  .yazi-layout {
    display: grid;
    grid-template-columns: 1fr 2fr 2fr;
    height: calc(100vh - 40px);
    gap: 1px;
    background: #333;
  }

  .column {
    background: #1a1a1a;
    overflow-y: auto;
    padding: 0.5rem;
  }

  .column-title {
    font-weight: bold;
    color: #888;
    padding: 0.5rem;
    border-bottom: 1px solid #333;
    margin-bottom: 0.5rem;
  }

  .search-input {
    width: 100%;
    padding: 0.5rem;
    background: #2a2a2a;
    border: 1px solid #444;
    color: #fff;
    margin-bottom: 0.5rem;
    border-radius: 4px;
  }

  .item {
    padding: 0.5rem;
    cursor: pointer;
    border-radius: 4px;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .item:hover {
    background: #2a2a2a;
  }

  .item.selected {
    background: #0066cc;
    color: white;
  }

  .icon {
    font-size: 1.2em;
  }

  .name {
    flex: 1;
  }

  .empty {
    color: #666;
    padding: 1rem;
    text-align: center;
  }

  .preview-content {
    color: #ccc;
    white-space: pre-wrap;
    word-wrap: break-word;
    font-family: monospace;
    font-size: 0.9em;
  }

  .help {
    background: #1a1a1a;
    color: #888;
    padding: 0.5rem 1rem;
    text-align: center;
    border-top: 1px solid #333;
  }

  .full-view {
    height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .note-header {
    display: flex;
    gap: 1rem;
    padding: 1rem;
    background: #1a1a1a;
    border-bottom: 1px solid #333;
  }

  .title-input {
    flex: 1;
    padding: 0.5rem;
    background: #2a2a2a;
    border: 1px solid #444;
    color: #fff;
    font-size: 1.2em;
    border-radius: 4px;
  }

  button {
    padding: 0.5rem 1rem;
    background: #0066cc;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  button:hover {
    background: #0052a3;
  }

  button:disabled {
    background: #444;
    cursor: not-allowed;
  }

  .content-editor {
    flex: 1;
    padding: 2rem;
    background: #1a1a1a;
    color: #fff;
    border: none;
    font-family: monospace;
    font-size: 1em;
    resize: none;
  }
</style>

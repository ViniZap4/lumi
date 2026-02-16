<script>
  import { onMount } from 'svelte';
  import { getNotes, getNote, updateNote } from './lib/api.js';

  let notes = [];
  let selectedNote = null;
  let content = '';
  let title = '';
  let search = '';
  let saving = false;
  let error = null;
  let cursor = 0;

  onMount(async () => {
    await loadNotes();
    window.addEventListener('keydown', handleGlobalKey);
    return () => window.removeEventListener('keydown', handleGlobalKey);
  });

  async function loadNotes() {
    try {
      error = null;
      notes = await getNotes();
      filterNotes();
    } catch (err) {
      error = 'Failed to load notes: ' + err.message;
      console.error(err);
    }
  }

  let filteredNotes = [];
  function filterNotes() {
    if (search) {
      filteredNotes = notes.filter(n => 
        n.title?.toLowerCase().includes(search.toLowerCase()) ||
        n.id?.toLowerCase().includes(search.toLowerCase())
      );
    } else {
      filteredNotes = notes;
    }
    cursor = Math.min(cursor, filteredNotes.length - 1);
  }

  $: if (search !== undefined) filterNotes();

  async function selectNote(note) {
    try {
      error = null;
      selectedNote = await getNote(note.id);
      title = selectedNote.title || '';
      content = selectedNote.content || '';
    } catch (err) {
      error = 'Failed to load note: ' + err.message;
      console.error(err);
    }
  }

  async function save() {
    if (!selectedNote) return;
    saving = true;
    error = null;
    try {
      await updateNote(selectedNote.id, { title, content, tags: selectedNote.tags || [] });
    } catch (err) {
      error = 'Failed to save: ' + err.message;
    } finally {
      saving = false;
    }
  }

  function handleGlobalKey(e) {
    // Vim-like navigation
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
      if ((e.metaKey || e.ctrlKey) && e.key === 's') {
        e.preventDefault();
        save();
      }
      return;
    }

    switch(e.key) {
      case 'j':
        e.preventDefault();
        if (cursor < filteredNotes.length - 1) cursor++;
        break;
      case 'k':
        e.preventDefault();
        if (cursor > 0) cursor--;
        break;
      case 'Enter':
        e.preventDefault();
        if (filteredNotes[cursor]) {
          selectNote(filteredNotes[cursor]);
        }
        break;
      case 'Escape':
        e.preventDefault();
        selectedNote = null;
        break;
      case '/':
        e.preventDefault();
        document.getElementById('search').focus();
        break;
    }
  }
</script>

<main>
  {#if error}
    <div class="error">{error}</div>
  {/if}

  {#if !selectedNote}
    <!-- Tree view -->
    <div class="tree">
      <h1>📂 lumi</h1>
      
      <div class="search-bar">
        <input 
          id="search"
          type="text" 
          bind:value={search} 
          placeholder="🔍 Search notes... (press /)"
        />
      </div>

      <div class="notes-list">
        {#each filteredNotes as note, i}
          <div 
            class="note-item"
            class:selected={i === cursor}
            on:click={() => { cursor = i; selectNote(note); }}
          >
            <span class="icon">📄</span>
            <span class="title">{note.title || note.id}</span>
          </div>
        {/each}
        {#if filteredNotes.length === 0}
          <div class="empty">No notes found</div>
        {/if}
      </div>

      <div class="help">
        <span class="key">j/k</span>=move | 
        <span class="key">enter</span>=open | 
        <span class="key">/</span>=search
      </div>
    </div>
  {:else}
    <!-- Full view -->
    <div class="full-view">
      <div class="header">
        <input 
          type="text" 
          bind:value={title} 
          placeholder="Note title"
          class="title-input"
        />
        <button on:click={save} disabled={saving}>
          {saving ? 'Saving...' : 'Save (Ctrl+S)'}
        </button>
        <button on:click={() => selectedNote = null}>
          Back (Esc)
        </button>
      </div>

      <textarea 
        bind:value={content}
        placeholder="Write your note in Markdown..."
      />

      <div class="help">
        <span class="key">Ctrl+S</span>=save | 
        <span class="key">Esc</span>=back
      </div>
    </div>
  {/if}
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #0a0a0a;
    color: #e0e0e0;
  }

  main {
    height: 100vh;
    overflow: hidden;
  }

  .error {
    position: fixed;
    top: 1rem;
    right: 1rem;
    padding: 1rem;
    background: #ff4444;
    color: white;
    border-radius: 4px;
    z-index: 1000;
  }

  .tree {
    padding: 2rem;
    max-width: 800px;
    margin: 0 auto;
  }

  h1 {
    margin: 0 0 2rem 0;
    font-size: 2rem;
    color: #9d7cd8;
  }

  .search-bar {
    margin-bottom: 2rem;
  }

  .search-bar input {
    width: 100%;
    padding: 0.75rem;
    background: #1a1a1a;
    border: 2px solid #333;
    border-radius: 8px;
    color: inherit;
    font-size: 1rem;
  }

  .search-bar input:focus {
    outline: none;
    border-color: #9d7cd8;
  }

  .notes-list {
    margin-bottom: 2rem;
  }

  .note-item {
    padding: 0.75rem 1rem;
    margin-bottom: 0.5rem;
    background: #1a1a1a;
    border: 2px solid transparent;
    border-radius: 8px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .note-item:hover {
    background: #222;
  }

  .note-item.selected {
    background: #2a2a2a;
    border-color: #9d7cd8;
  }

  .icon {
    font-size: 1.2rem;
  }

  .title {
    font-weight: 500;
  }

  .empty {
    padding: 2rem;
    text-align: center;
    color: #666;
    font-style: italic;
  }

  .full-view {
    height: 100vh;
    display: flex;
    flex-direction: column;
    padding: 2rem;
  }

  .header {
    display: flex;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .title-input {
    flex: 1;
    padding: 0.75rem;
    font-size: 1.5rem;
    font-weight: 600;
    background: #1a1a1a;
    border: 2px solid #333;
    border-radius: 8px;
    color: inherit;
  }

  .title-input:focus {
    outline: none;
    border-color: #9d7cd8;
  }

  button {
    padding: 0.75rem 1.5rem;
    background: #9d7cd8;
    border: none;
    border-radius: 8px;
    color: white;
    cursor: pointer;
    font-weight: 600;
  }

  button:hover:not(:disabled) {
    background: #b392e0;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  textarea {
    flex: 1;
    padding: 1rem;
    background: #1a1a1a;
    border: 2px solid #333;
    border-radius: 8px;
    color: inherit;
    font-family: 'Monaco', 'Menlo', monospace;
    font-size: 1rem;
    line-height: 1.6;
    resize: none;
  }

  textarea:focus {
    outline: none;
    border-color: #9d7cd8;
  }

  .help {
    margin-top: 1rem;
    padding: 0.75rem;
    background: #1a1a1a;
    border-radius: 8px;
    text-align: center;
    color: #888;
    font-size: 0.9rem;
  }

  .key {
    color: #9d7cd8;
    font-weight: 600;
  }
</style>

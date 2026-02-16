<script>
  import { onMount } from 'svelte';
  import { getNote, updateNote } from '../lib/api.js';

  export let noteId = null;

  let note = null;
  let content = '';
  let title = '';
  let saving = false;

  $: if (noteId) {
    loadNote(noteId);
  }

  async function loadNote(id) {
    try {
      note = await getNote(id);
      title = note.title;
      content = note.content;
    } catch (err) {
      console.error('Failed to load note:', err);
    }
  }

  async function save() {
    if (!note) return;
    
    saving = true;
    try {
      await updateNote(note.id, { title, content, tags: note.tags });
    } catch (err) {
      console.error('Failed to save note:', err);
    } finally {
      saving = false;
    }
  }

  function handleKeydown(e) {
    if ((e.metaKey || e.ctrlKey) && e.key === 's') {
      e.preventDefault();
      save();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="editor">
  {#if note}
    <div class="header">
      <input 
        type="text" 
        bind:value={title} 
        placeholder="Note title"
        class="title-input"
      />
      <button on:click={save} disabled={saving}>
        {saving ? 'Saving...' : 'Save'}
      </button>
    </div>
    <textarea 
      bind:value={content}
      placeholder="Write your note in Markdown..."
    />
  {:else}
    <div class="empty">
      <p>Select a note to edit</p>
    </div>
  {/if}
</div>

<style>
  .editor {
    height: 100%;
    display: flex;
    flex-direction: column;
    padding: 1rem;
  }

  .header {
    display: flex;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .title-input {
    flex: 1;
    padding: 0.5rem;
    font-size: 1.2rem;
    font-weight: 500;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 4px;
    color: inherit;
  }

  button {
    padding: 0.5rem 1rem;
    background: #646cff;
    border: none;
    border-radius: 4px;
    color: white;
    cursor: pointer;
    font-weight: 500;
  }

  button:hover:not(:disabled) {
    background: #535bf2;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  textarea {
    flex: 1;
    padding: 1rem;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 4px;
    color: inherit;
    font-family: 'Monaco', 'Menlo', monospace;
    font-size: 0.95rem;
    line-height: 1.6;
    resize: none;
  }

  .empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #666;
    font-style: italic;
  }
</style>

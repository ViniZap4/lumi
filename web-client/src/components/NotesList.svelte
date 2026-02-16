<script>
  import { onMount } from 'svelte';
  import { getNotes } from '../lib/api.js';

  export let currentPath = '';

  let notes = [];
  let selectedNote = null;

  onMount(async () => {
    await loadNotes();
  });

  async function loadNotes() {
    try {
      notes = await getNotes(currentPath);
    } catch (err) {
      console.error('Failed to load notes:', err);
    }
  }

  function selectNote(note) {
    selectedNote = note;
    window.dispatchEvent(new CustomEvent('noteselect', { detail: note }));
  }

  $: if (currentPath !== undefined) {
    loadNotes();
  }

  export function reload() {
    loadNotes();
  }
</script>

<div class="notes-list">
  <h2>Notes</h2>
  {#if notes.length === 0}
    <p class="empty">No notes</p>
  {:else}
    <ul>
      {#each notes as note}
        <li 
          class:selected={selectedNote?.id === note.id}
          on:click={() => selectNote(note)}
        >
          <div class="title">{note.title}</div>
          <div class="id">{note.id}</div>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .notes-list {
    height: 100%;
    overflow-y: auto;
    border-right: 1px solid #333;
    padding: 1rem;
  }

  h2 {
    margin: 0 0 1rem 0;
    font-size: 1.2rem;
  }

  ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  li {
    padding: 0.75rem;
    cursor: pointer;
    border-radius: 4px;
    margin-bottom: 0.5rem;
    border: 1px solid transparent;
  }

  li:hover {
    background: #1a1a1a;
  }

  li.selected {
    background: #2a2a2a;
    border-color: #646cff;
  }

  .title {
    font-weight: 500;
    margin-bottom: 0.25rem;
  }

  .id {
    font-size: 0.85rem;
    color: #888;
  }

  .empty {
    color: #666;
    font-style: italic;
  }
</style>

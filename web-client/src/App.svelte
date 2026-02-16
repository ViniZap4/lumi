<script>
  import { onMount } from 'svelte';
  import NotesList from './components/NotesList.svelte';
  import Editor from './components/Editor.svelte';
  import { connectWebSocket } from './lib/ws.js';

  let currentPath = '';
  let selectedNoteId = null;
  let ws = null;
  let notesListRef;

  onMount(() => {
    window.addEventListener('noteselect', (e) => {
      selectedNoteId = e.detail?.id || null;
    });

    ws = connectWebSocket((msg) => {
      console.log('WebSocket message:', msg);
      if (msg.type === 'note_created' || msg.type === 'note_updated' || msg.type === 'note_deleted') {
        if (notesListRef) {
          notesListRef.reload();
        }
      }
    });

    return () => {
      if (ws) ws.close();
    };
  });
</script>

<main>
  <div class="container">
    <div class="sidebar">
      <h1>lumi</h1>
      <p class="subtitle">Local-first notes</p>
    </div>
    
    <div class="notes">
      <NotesList bind:this={notesListRef} {currentPath} />
    </div>
    
    <div class="editor">
      <Editor noteId={selectedNoteId} />
    </div>
  </div>
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    background: #0a0a0a;
    color: #e0e0e0;
  }

  main {
    height: 100vh;
    overflow: hidden;
  }

  .container {
    display: grid;
    grid-template-columns: 200px 300px 1fr;
    height: 100%;
  }

  .sidebar {
    background: #111;
    padding: 1.5rem;
    border-right: 1px solid #333;
  }

  h1 {
    margin: 0;
    font-size: 1.8rem;
    color: #646cff;
  }

  .subtitle {
    margin: 0.5rem 0 0 0;
    font-size: 0.85rem;
    color: #888;
  }

  .notes {
    background: #0f0f0f;
  }

  .editor {
    background: #0a0a0a;
  }
</style>

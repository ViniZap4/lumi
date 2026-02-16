// web-client/src/lib/api.js
const API_URL = import.meta.env.VITE_LUMI_SERVER_URL || 'http://localhost:8080';
const TOKEN = import.meta.env.VITE_LUMI_TOKEN || 'dev';

const headers = {
  'Content-Type': 'application/json',
  'X-Lumi-Token': TOKEN,
};

export async function getFolders() {
  const res = await fetch(`${API_URL}/api/folders`, { headers });
  if (!res.ok) throw new Error('Failed to fetch folders');
  return res.json();
}

export async function getNotes(path = '') {
  const url = path ? `${API_URL}/api/notes?path=${path}` : `${API_URL}/api/notes`;
  const res = await fetch(url, { headers });
  if (!res.ok) throw new Error('Failed to fetch notes');
  return res.json();
}

export async function getNote(id) {
  // Remove .md extension if present and encode
  const cleanId = id.replace(/\.md$/, '');
  const encodedId = encodeURIComponent(cleanId);
  const res = await fetch(`${API_URL}/api/notes/${encodedId}`, { headers });
  if (!res.ok) throw new Error(`Failed to fetch note: ${res.status} ${res.statusText}`);
  return res.json();
}

export async function createNote(note) {
  const res = await fetch(`${API_URL}/api/notes`, {
    method: 'POST',
    headers,
    body: JSON.stringify(note),
  });
  if (!res.ok) throw new Error('Failed to create note');
  return res.json();
}

export async function updateNote(id, note) {
  const cleanId = id.replace(/\.md$/, '');
  const encodedId = encodeURIComponent(cleanId);
  const res = await fetch(`${API_URL}/api/notes/${encodedId}`, {
    method: 'PUT',
    headers,
    body: JSON.stringify(note),
  });
  if (!res.ok) throw new Error(`Failed to update note: ${res.status}`);
  return res.json();
}

export async function deleteNote(id) {
  const res = await fetch(`${API_URL}/api/notes/${id}`, {
    method: 'DELETE',
    headers,
  });
  if (!res.ok) throw new Error('Failed to delete note');
}

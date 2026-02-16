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
  const res = await fetch(`${API_URL}/api/notes/${id}`, { headers });
  if (!res.ok) throw new Error('Failed to fetch note');
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
  const res = await fetch(`${API_URL}/api/notes/${id}`, {
    method: 'PUT',
    headers,
    body: JSON.stringify(note),
  });
  if (!res.ok) throw new Error('Failed to update note');
  return res.json();
}

export async function deleteNote(id) {
  const res = await fetch(`${API_URL}/api/notes/${id}`, {
    method: 'DELETE',
    headers,
  });
  if (!res.ok) throw new Error('Failed to delete note');
}

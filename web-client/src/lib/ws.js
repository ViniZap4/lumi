// web-client/src/lib/ws.js
const WS_URL = (import.meta.env.VITE_LUMI_SERVER_URL || 'http://localhost:8080').replace('http', 'ws');

let ws = null;
let reconnectTimer = null;

export function connectWebSocket(onMessage) {
  ws = new WebSocket(`${WS_URL}/ws`);

  ws.onopen = () => {
    console.log('WebSocket connected');
    ws.send(JSON.stringify({ type: 'subscribe' }));
  };

  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    onMessage(msg);
  };

  ws.onerror = (error) => {
    console.error('WebSocket error:', error);
  };

  ws.onclose = () => {
    console.log('WebSocket disconnected, reconnecting...');
    reconnectTimer = setTimeout(() => connectWebSocket(onMessage), 3000);
  };

  return ws;
}

export function disconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (ws) {
    ws.onclose = null; // prevent auto-reconnect
    ws.close();
    ws = null;
  }
}

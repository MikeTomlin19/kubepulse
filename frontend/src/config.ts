// Get WebSocket URL from environment or use default development URL
const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsHost = window.location.host;
export const WS_URL = `${wsProtocol}//${wsHost}/ws`; 
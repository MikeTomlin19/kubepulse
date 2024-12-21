import { useState, useEffect, useCallback } from 'react';
import { ClusterData } from '../types/kubernetes';
import { WS_URL } from '../config';

export function useWebSocket() {
  const [clusterData, setClusterData] = useState<ClusterData | null>(null);

  const connect = useCallback(() => {
    console.log('Connecting to WebSocket at:', WS_URL);
    
    const ws = new WebSocket(WS_URL, ['websocket']);

    ws.onopen = () => {
      console.log('WebSocket connected');
    };

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        if (message.type === 'state') {
          const data = message.payload as ClusterData;
          setClusterData(data);
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.onclose = (event) => {
      console.log('WebSocket disconnected:', event.code, event.reason);
      // Attempt to reconnect after 5 seconds
      setTimeout(connect, 5000);
    };

    return () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.close();
      }
    };
  }, []);

  useEffect(() => {
    const cleanup = connect();
    return cleanup;
  }, [connect]);

  return clusterData;
}


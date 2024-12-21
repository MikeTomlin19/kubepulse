import React from 'react';
import { Header } from './components/Header';
import { KubernetesCluster } from './components/KubernetesCluster';
import { useWebSocket } from './hooks/useWebSocket';

const App: React.FC = () => {
  const clusterData = useWebSocket();

  return (
    <div className="min-h-screen bg-gray-100">
      <Header />
      <main className="container mx-auto p-4">
        {clusterData ? (
          <KubernetesCluster data={clusterData} />
        ) : (
          <div className="text-center py-10">
            <p className="text-xl">Connecting to cluster...</p>
          </div>
        )}
      </main>
    </div>
  );
};

export default App;


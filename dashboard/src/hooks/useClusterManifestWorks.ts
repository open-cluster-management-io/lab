import { useState, useEffect } from 'react';
import { fetchManifestWorks } from '../api/manifestWorkService';
import type { ManifestWork } from '../api/manifestWorkService';

/**
 * Custom hook for fetching and managing cluster manifest works
 * @param clusterName The name of the cluster to fetch manifest works for (used as namespace)
 * @returns Object containing manifest works, loading state, and error
 */
export const useClusterManifestWorks = (clusterName: string | null) => {
  const [manifestWorks, setManifestWorks] = useState<ManifestWork[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!clusterName) {
      setLoading(false);
      setManifestWorks([]);
      return;
    }

    const loadManifestWorks = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await fetchManifestWorks(clusterName);
        setManifestWorks(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'An unknown error occurred');
        console.error('Error fetching cluster manifest works:', err);
      } finally {
        setLoading(false);
      }
    };

    loadManifestWorks();
  }, [clusterName]);

  return { manifestWorks, loading, error };
};
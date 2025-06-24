import { useState, useEffect } from 'react';
import { fetchClusterByName, type Cluster } from '../api/clusterService';

interface UseClusterOptions {
  initialData?: Cluster | null;
  skipFetch?: boolean;
}

/**
 * Custom hook for fetching and managing cluster data
 * @param name - Cluster name to fetch
 * @param options - Hook options
 * @returns Object containing cluster data, loading state, error, and refetch function
 */
export function useCluster(name: string | null, options: UseClusterOptions = {}) {
  const { initialData = null, skipFetch = false } = options;

  const [cluster, setCluster] = useState<Cluster | null>(initialData);
  const [loading, setLoading] = useState<boolean>(!initialData && !!name && !skipFetch);
  const [error, setError] = useState<string | null>(null);

  const fetchCluster = async (clusterName: string) => {
    if (!clusterName) {
      setError('Cluster name is required');
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);
      const data = await fetchClusterByName(clusterName);
      setCluster(data);
      setLoading(false);
    } catch (err) {
      console.error('Error fetching cluster details:', err);
      setError('Unable to load cluster details');
      setLoading(false);
    }
  };

  // Refetch function that can be called manually
  const refetch = () => {
    if (name) {
      fetchCluster(name);
    }
  };

  // Fetch data when name changes or when initialData is not provided
  useEffect(() => {
    if (name && !skipFetch && !initialData) {
      fetchCluster(name);
    } else if (initialData) {
      setCluster(initialData);
      setLoading(false);
      setError(null);
    } else if (!name) {
      setCluster(null);
      setLoading(false);
      setError(null);
    }
  }, [name, initialData, skipFetch]);

  return {
    cluster,
    loading,
    error,
    refetch,
    setCluster // Expose setter for cases where we need to update the cluster data
  };
}

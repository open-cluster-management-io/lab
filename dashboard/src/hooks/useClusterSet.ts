import { useState, useEffect } from 'react';
import { fetchClusterSetByName, type ClusterSet } from '../api/clusterSetService';

interface UseClusterSetOptions {
  initialData?: ClusterSet | null;
  skipFetch?: boolean;
}

/**
 * Custom hook for fetching and managing cluster set data
 * @param name - Cluster set name to fetch
 * @param options - Hook options
 * @returns Object containing cluster set data, loading state, error, and refetch function
 */
export function useClusterSet(name: string | null, options: UseClusterSetOptions = {}) {
  const { initialData = null, skipFetch = false } = options;

  const [clusterSet, setClusterSet] = useState<ClusterSet | null>(initialData);
  const [loading, setLoading] = useState<boolean>(!initialData && !!name && !skipFetch);
  const [error, setError] = useState<string | null>(null);

  const fetchClusterSetData = async (clusterSetName: string) => {
    if (!clusterSetName) {
      setError('Cluster set name is required');
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);
      const data = await fetchClusterSetByName(clusterSetName);
      setClusterSet(data);
      setLoading(false);
    } catch (err) {
      console.error('Error fetching cluster set details:', err);
      setError('Unable to load cluster set details');
      setLoading(false);
    }
  };

  // Refetch function that can be called manually
  const refetch = () => {
    if (name) {
      fetchClusterSetData(name);
    }
  };

  // Fetch data when name changes or when initialData is not provided
  useEffect(() => {
    if (name && !skipFetch && !initialData) {
      fetchClusterSetData(name);
    } else if (initialData) {
      setClusterSet(initialData);
      setLoading(false);
      setError(null);
    } else if (!name) {
      setClusterSet(null);
      setLoading(false);
      setError(null);
    }
  }, [name, initialData, skipFetch]);

  return {
    clusterSet,
    loading,
    error,
    refetch,
    setClusterSet // Expose setter for cases where we need to update the cluster set data
  };
}

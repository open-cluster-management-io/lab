import { useState, useEffect } from 'react';
import { fetchPlacementByName, type Placement } from '../api/placementService';

interface UsePlacementOptions {
  initialData?: Placement | null;
  skipFetch?: boolean;
}

/**
 * Custom hook for fetching and managing placement data
 * @param namespace - Placement namespace to fetch
 * @param name - Placement name to fetch
 * @param options - Hook options
 * @returns Object containing placement data, loading state, error, and refetch function
 */
export function usePlacement(namespace: string | null, name: string | null, options: UsePlacementOptions = {}) {
  const { initialData = null, skipFetch = false } = options;

  const [placement, setPlacement] = useState<Placement | null>(initialData);
  const [loading, setLoading] = useState<boolean>(!initialData && !!namespace && !!name && !skipFetch);
  const [error, setError] = useState<string | null>(null);

  const fetchPlacement = async (ns: string, placementName: string) => {
    if (!ns || !placementName) {
      setError('Namespace and placement name are required');
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);
      const data = await fetchPlacementByName(ns, placementName);
      setPlacement(data);
      setLoading(false);
    } catch (err) {
      console.error('Error fetching placement details:', err);
      setError('Unable to load placement details');
      setLoading(false);
    }
  };

  // Refetch function that can be called manually
  const refetch = () => {
    if (namespace && name) {
      fetchPlacement(namespace, name);
    }
  };

  // Fetch data when namespace/name changes or when initialData is not provided
  useEffect(() => {
    if (namespace && name && !skipFetch && !initialData) {
      fetchPlacement(namespace, name);
    } else if (initialData) {
      setPlacement(initialData);
      setLoading(false);
      setError(null);
    } else if (!namespace || !name) {
      setPlacement(null);
      setLoading(false);
      setError(null);
    }
  }, [namespace, name, initialData, skipFetch]);

  return {
    placement,
    loading,
    error,
    refetch,
    setPlacement // Expose setter for cases where we need to update the placement data
  };
}
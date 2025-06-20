import { useState, useEffect } from 'react';
import { fetchClusterAddons } from '../api/addonService';
import type { ManagedClusterAddon } from '../api/addonService';

/**
 * Custom hook for fetching and managing cluster addons
 * @param clusterName The name of the cluster to fetch addons for
 * @returns Object containing addons, loading state, and error
 */
export const useClusterAddons = (clusterName: string | null) => {
  const [addons, setAddons] = useState<ManagedClusterAddon[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!clusterName) {
      setLoading(false);
      setAddons([]);
      return;
    }

    const loadAddons = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await fetchClusterAddons(clusterName);
        setAddons(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'An unknown error occurred');
        console.error('Error fetching cluster addons:', err);
      } finally {
        setLoading(false);
      }
    };

    loadAddons();
  }, [clusterName]);

  return { addons, loading, error };
};
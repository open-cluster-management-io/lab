import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import {
  Box,
  CircularProgress,
  Alert,
} from "@mui/material";
import PageLayout from './layout/PageLayout';
import ManifestWorkDetailContent from './ManifestWorkDetailContent';
import { fetchManifestWorkByName } from '../api/manifestWorkService';
import type { ManifestWork } from '../api/manifestWorkService';

export default function ManifestWorkDetailPage() {
  const { namespace, name } = useParams<{ namespace: string; name: string }>();

  const [mw, setMw] = useState<ManifestWork | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!namespace || !name) return;

    const load = async () => {
      try {
        setLoading(true);
        const data = await fetchManifestWorkByName(namespace, name);
        if (!data) {
          setError('ManifestWork not found');
        }
        setMw(data);
      } catch {
        setError('Failed to load ManifestWork');
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [namespace, name]);

  if (loading) {
    return (
      <PageLayout title="Loading..." backLink="/manifestworks" backLabel="Back to ManifestWorks">
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      </PageLayout>
    );
  }

  if (error || !mw) {
    return (
      <PageLayout title="Error" backLink="/manifestworks" backLabel="Back to ManifestWorks">
        <Alert severity="error">{error || 'ManifestWork not found'}</Alert>
      </PageLayout>
    );
  }

  return (
    <PageLayout
      title={`${mw.namespace}/${mw.name}`}
      backLink="/manifestworks"
      backLabel="Back to ManifestWorks"
    >
      <ManifestWorkDetailContent mw={mw} />
    </PageLayout>
  );
}

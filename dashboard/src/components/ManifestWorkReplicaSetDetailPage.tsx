import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import {
  Box,
  CircularProgress,
  Alert,
} from "@mui/material";
import PageLayout from './layout/PageLayout';
import ManifestWorkReplicaSetDetailContent from './ManifestWorkReplicaSetDetailContent';
import { fetchManifestWorkReplicaSet } from '../api/manifestWorkReplicaSetService';
import type { ManifestWorkReplicaSet } from '../api/manifestWorkReplicaSetService';

export default function ManifestWorkReplicaSetDetailPage() {
  const { namespace, name } = useParams<{ namespace: string; name: string }>();

  const [mwrs, setMwrs] = useState<ManifestWorkReplicaSet | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!namespace || !name) return;

    const load = async () => {
      try {
        setLoading(true);
        const data = await fetchManifestWorkReplicaSet(namespace, name);
        if (!data) {
          setError('ManifestWorkReplicaSet not found');
        }
        setMwrs(data);
      } catch {
        setError('Failed to load ManifestWorkReplicaSet');
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [namespace, name]);

  if (loading) {
    return (
      <PageLayout title="Loading..." backLink="/manifestworkreplicasets" backLabel="Back to WorkReplicaSets">
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      </PageLayout>
    );
  }

  if (error || !mwrs) {
    return (
      <PageLayout title="Error" backLink="/manifestworkreplicasets" backLabel="Back to WorkReplicaSets">
        <Alert severity="error">{error || 'ManifestWorkReplicaSet not found'}</Alert>
      </PageLayout>
    );
  }

  return (
    <PageLayout
      title={`${mwrs.namespace}/${mwrs.name}`}
      backLink="/manifestworkreplicasets"
      backLabel="Back to WorkReplicaSets"
    >
      <ManifestWorkReplicaSetDetailContent mwrs={mwrs} />
    </PageLayout>
  );
}

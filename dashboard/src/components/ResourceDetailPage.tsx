import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { Box, CircularProgress, Alert, Typography } from '@mui/material';
import { fetchManagedResource, type ManagedResource } from '../api/resourceService';
import ResourceDetailContent from './ResourceDetailContent';
import PageLayout from './layout/PageLayout';

export default function ResourceDetailPage() {
  const { cluster, manifestwork, ordinal } = useParams<{
    cluster: string;
    manifestwork: string;
    ordinal: string;
  }>();
  const [resource, setResource] = useState<ManagedResource | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!cluster || !manifestwork || ordinal === undefined) return;
    setLoading(true);
    fetchManagedResource(cluster, manifestwork, parseInt(ordinal, 10))
      .then((r) => {
        if (!r) setError('Resource not found');
        else setResource(r);
      })
      .catch(() => setError('Failed to load resource'))
      .finally(() => setLoading(false));
  }, [cluster, manifestwork, ordinal]);

  const title = resource
    ? `${resource.kind}: ${resource.name}`
    : `${cluster}/${manifestwork}/${ordinal}`;

  return (
    <PageLayout title={title} backLink="/resources" backLabel="Back to Resources">
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
          <CircularProgress />
        </Box>
      ) : error ? (
        <Alert severity="error">{error}</Alert>
      ) : resource ? (
        <>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Cluster: <strong>{resource.cluster}</strong> &nbsp;&middot;&nbsp; ManifestWork: <strong>{resource.manifestWorkName}</strong>
          </Typography>
          <ResourceDetailContent resource={resource} />
        </>
      ) : null}
    </PageLayout>
  );
}

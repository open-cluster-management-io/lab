import { useState } from 'react';
import { useAuth } from '../auth/useAuth';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardHeader,
  CardContent,
  CardActions,
  TextField,
  Button,
  Typography,
  Alert,
  CircularProgress,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Stack,
} from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';

const Login = () => {
  const [token, setToken] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [testing, setTesting] = useState(false);
  const { login, isLoading, error: authError } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();

    if (!token.trim()) {
      setError('Token is required');
      return;
    }

    if (token.trim().length < 10) {
      setError('Token seems too short');
      return;
    }

    setError(null);
    setTesting(true);

    try {
      const testToken = token.startsWith('Bearer ') ? token : `Bearer ${token}`;
      const response = await fetch('/api/clusters', {
        headers: {
          'Authorization': testToken,
          'Content-Type': 'application/json'
        }
      });

      if (response.ok) {
        login(testToken);
        navigate('/overview');
      } else if (response.status === 401) {
        setError('Invalid or expired token. Please check your token and try again.');
      } else {
        setError(`Authentication failed: ${response.status} ${response.statusText}`);
      }
    } catch {
      setError('Failed to test token. Please check your connection and try again.');
    } finally {
      setTesting(false);
    }
  };

  const getTokenInstructions = () => (
    <Stack spacing={2}>
      <Typography variant="body2">
        To get a service account token for OCM Dashboard:
      </Typography>

      <Box component="pre" sx={{
        backgroundColor: '#1a1d21',
        color: '#00ff00',
        p: 2,
        borderRadius: '6px',
        fontSize: '0.75rem',
        overflow: 'auto',
        borderLeft: '4px solid #0066cc',
        fontFamily: 'Monaco, Menlo, monospace',
        lineHeight: 1.5,
      }}>
{`# Create a service account (if not exists)
kubectl create serviceaccount dashboard-user -n default

# Create cluster role binding
kubectl create clusterrolebinding dashboard-user \\
  --clusterrole=cluster-admin \\
  --serviceaccount=default:dashboard-user

# Get the token
kubectl create token dashboard-user --duration=24h`}
      </Box>

      <Typography variant="caption" color="text.secondary">
        Copy the output token and paste it in the field above.
      </Typography>
    </Stack>
  );

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        width: '100%',
        background: 'linear-gradient(135deg, #1a1d21 0%, #002952 100%)',
        p: 2,
      }}
    >
      <Box
        sx={{
          width: '100%',
          maxWidth: '480px',
          margin: '0 auto',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          animation: 'fadeInUp 0.5s ease',
        }}
      >
        <Card
          sx={{
            width: "100%",
            boxShadow: '0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)',
            borderRadius: '12px',
            p: 2,
            position: 'relative',
            overflow: 'hidden',
            '&::before': {
              content: '""',
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              height: '4px',
              background: 'linear-gradient(90deg, #ee0000, #0066cc)',
            },
          }}
        >
          <CardHeader
            sx={{ textAlign: "center", pb: 0 }}
            title={
              <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 2 }}>
                <Box
                  sx={{
                    width: 48,
                    height: 48,
                    bgcolor: '#ee0000',
                    borderRadius: '8px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: 'white',
                    fontSize: 24,
                    fontWeight: 700,
                  }}
                >
                  O
                </Box>
                <Typography
                  variant="h5"
                  component="h1"
                  sx={{
                    fontWeight: 700,
                    fontFamily: "'Red Hat Display', 'Helvetica Neue', Arial, sans-serif",
                  }}
                >
                  OCM Dashboard
                </Typography>
              </Box>
            }
            subheader={
              <Typography variant="body2" sx={{ mt: 1, color: '#495057' }}>
                Sign in with your Kubernetes <strong>Bearer Token</strong>
              </Typography>
            }
          />

          {(error || authError) && (
            <Alert severity="error" sx={{ mx: 3, mb: 2 }}>
              {error || authError}
            </Alert>
          )}

          {isLoading && (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 2 }}>
              <CircularProgress size={24} />
              <Typography variant="body2" sx={{ ml: 2 }}>
                Loading authentication...
              </Typography>
            </Box>
          )}

          <form onSubmit={handleSubmit}>
            <CardContent sx={{ pt: 2 }}>
              <TextField
                multiline
                fullWidth
                rows={4}
                placeholder="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
                variant="outlined"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                error={!!error}
                slotProps={{
                  input: {
                    style: {
                      fontFamily: "JetBrains Mono, monospace",
                      fontSize: "0.75rem",
                    }
                  }
                }}
                sx={{ mb: 2 }}
                disabled={testing}
              />

              <Accordion sx={{ mb: 2, borderRadius: '8px', '&::before': { display: 'none' } }}>
                <AccordionSummary
                  expandIcon={<ExpandMoreIcon />}
                  aria-controls="token-instructions-content"
                  id="token-instructions-header"
                >
                  <Typography variant="subtitle2">
                    How to get a Kubernetes token?
                  </Typography>
                </AccordionSummary>
                <AccordionDetails>
                  {getTokenInstructions()}
                </AccordionDetails>
              </Accordion>

              <Box sx={{ display: "flex", flexDirection: "column", gap: 1.5 }}>
                <Button
                  type="submit"
                  variant="contained"
                  fullWidth
                  disabled={!token.trim() || testing}
                  startIcon={testing ? <CircularProgress size={20} /> : undefined}
                >
                  {testing ? 'Testing Token...' : 'Sign In'}
                </Button>
              </Box>
            </CardContent>
          </form>

          <CardActions sx={{ justifyContent: "center", pt: 0 }}>
            <Typography variant="caption" color="text.secondary">
              Token is stored locally in your browser only.
            </Typography>
          </CardActions>
        </Card>
      </Box>
    </Box>
  );
};

export default Login;

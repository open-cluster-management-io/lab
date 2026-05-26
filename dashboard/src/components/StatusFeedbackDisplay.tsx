import {
  Box,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import type { StatusFeedbackResult, FieldValue } from '../api/manifestWorkService';

export function formatValue(fv: FieldValue): string {
  switch (fv.type) {
    case 'Integer':
      return fv.integer !== undefined ? String(fv.integer) : '-';
    case 'String':
      return fv.string ?? '-';
    case 'Boolean':
      return fv.boolean !== undefined ? String(fv.boolean) : '-';
    case 'JsonRaw':
      return fv.jsonRaw ?? '-';
    default:
      return '-';
  }
}

interface Props {
  feedback: StatusFeedbackResult | undefined;
  variant?: 'table' | 'inline' | 'compact';
  maxItems?: number;
}

export default function StatusFeedbackDisplay({ feedback, variant = 'table', maxItems }: Props) {
  if (!feedback?.values?.length) return null;

  const items = maxItems ? feedback.values.slice(0, maxItems) : feedback.values;

  if (variant === 'compact') {
    const summary = items.map(v => `${v.name}=${formatValue(v.fieldValue)}`).join(', ');
    const extra = maxItems && feedback.values.length > maxItems
      ? ` (+${feedback.values.length - maxItems})`
      : '';
    return (
      <Typography variant="caption" color="text.secondary" noWrap>
        {summary}{extra}
      </Typography>
    );
  }

  if (variant === 'inline') {
    return (
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
        {items.map(v => (
          <Chip
            key={v.name}
            label={`${v.name}: ${formatValue(v.fieldValue)}`}
            size="small"
            variant="outlined"
            color="info"
          />
        ))}
        {maxItems && feedback.values.length > maxItems && (
          <Chip
            label={`+${feedback.values.length - maxItems}`}
            size="small"
            variant="outlined"
          />
        )}
      </Box>
    );
  }

  // table variant (default)
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Name</TableCell>
            <TableCell>Type</TableCell>
            <TableCell>Value</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {items.map(v => (
            <TableRow key={v.name}>
              <TableCell>{v.name}</TableCell>
              <TableCell>
                <Chip label={v.fieldValue.type} size="small" variant="outlined" />
              </TableCell>
              <TableCell sx={{ fontFamily: 'monospace' }}>{formatValue(v.fieldValue)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import StatusFeedbackDisplay, { formatValue } from './StatusFeedbackDisplay';
import type { StatusFeedbackResult, FieldValue } from '../api/manifestWorkService';

// --- formatValue ---

describe('formatValue', () => {
  it('formats Integer values', () => {
    expect(formatValue({ type: 'Integer', integer: 42 } as FieldValue)).toBe('42');
  });

  it('formats String values', () => {
    expect(formatValue({ type: 'String', string: 'hello' } as FieldValue)).toBe('hello');
  });

  it('formats Boolean values', () => {
    expect(formatValue({ type: 'Boolean', boolean: true } as FieldValue)).toBe('true');
    expect(formatValue({ type: 'Boolean', boolean: false } as FieldValue)).toBe('false');
  });

  it('formats JsonRaw values', () => {
    expect(formatValue({ type: 'JsonRaw', jsonRaw: '{"key":"val"}' } as FieldValue)).toBe('{"key":"val"}');
  });

  it('returns dash for undefined integer', () => {
    expect(formatValue({ type: 'Integer' } as FieldValue)).toBe('-');
  });

  it('returns dash for undefined string', () => {
    expect(formatValue({ type: 'String' } as FieldValue)).toBe('-');
  });

  it('returns dash for unknown type', () => {
    expect(formatValue({ type: 'Unknown' } as unknown as FieldValue)).toBe('-');
  });
});

// --- Test data ---

const sampleFeedback: StatusFeedbackResult = {
  values: [
    { name: 'ReadyReplicas', fieldValue: { type: 'Integer', integer: 2 } },
    { name: 'Replicas', fieldValue: { type: 'Integer', integer: 2 } },
    { name: 'clusterIP', fieldValue: { type: 'String', string: '10.96.0.1' } },
  ],
};

// --- Component rendering ---

describe('StatusFeedbackDisplay', () => {
  it('returns null for undefined feedback', () => {
    const { container } = render(<StatusFeedbackDisplay feedback={undefined} />);
    expect(container.innerHTML).toBe('');
  });

  it('returns null for empty values', () => {
    const { container } = render(<StatusFeedbackDisplay feedback={{ values: [] }} />);
    expect(container.innerHTML).toBe('');
  });

  describe('table variant (default)', () => {
    it('renders a table with Name, Type, Value columns', () => {
      render(<StatusFeedbackDisplay feedback={sampleFeedback} />);
      expect(screen.getByText('Name')).toBeInTheDocument();
      expect(screen.getByText('Type')).toBeInTheDocument();
      expect(screen.getByText('Value')).toBeInTheDocument();
    });

    it('renders all feedback values', () => {
      render(<StatusFeedbackDisplay feedback={sampleFeedback} variant="table" />);
      expect(screen.getByText('ReadyReplicas')).toBeInTheDocument();
      expect(screen.getByText('Replicas')).toBeInTheDocument();
      expect(screen.getByText('clusterIP')).toBeInTheDocument();
      expect(screen.getByText('10.96.0.1')).toBeInTheDocument();
    });

    it('respects maxItems', () => {
      render(<StatusFeedbackDisplay feedback={sampleFeedback} variant="table" maxItems={1} />);
      expect(screen.getByText('ReadyReplicas')).toBeInTheDocument();
      expect(screen.queryByText('Replicas')).not.toBeInTheDocument();
    });
  });

  describe('inline variant', () => {
    it('renders chips with name: value format', () => {
      render(<StatusFeedbackDisplay feedback={sampleFeedback} variant="inline" />);
      expect(screen.getByText('ReadyReplicas: 2')).toBeInTheDocument();
      expect(screen.getByText('clusterIP: 10.96.0.1')).toBeInTheDocument();
    });

    it('shows overflow chip when maxItems is exceeded', () => {
      render(<StatusFeedbackDisplay feedback={sampleFeedback} variant="inline" maxItems={1} />);
      expect(screen.getByText('ReadyReplicas: 2')).toBeInTheDocument();
      expect(screen.getByText('+2')).toBeInTheDocument();
    });
  });

  describe('compact variant', () => {
    it('renders comma-separated name=value text', () => {
      render(<StatusFeedbackDisplay feedback={sampleFeedback} variant="compact" />);
      expect(screen.getByText(/ReadyReplicas=2/)).toBeInTheDocument();
      expect(screen.getByText(/clusterIP=10.96.0.1/)).toBeInTheDocument();
    });

    it('shows overflow count when maxItems is exceeded', () => {
      render(<StatusFeedbackDisplay feedback={sampleFeedback} variant="compact" maxItems={1} />);
      expect(screen.getByText(/\(\+2\)/)).toBeInTheDocument();
    });
  });
});

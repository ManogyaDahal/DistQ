import { useState, type FormEvent } from 'react';
import { useTaskLookup } from '../hooks/useApi';
import { formatDateTime } from '../utils';

export default function TaskLookup() {
  const { data, loading, error, lookup } = useTaskLookup();
  const [taskId, setTaskId] = useState('');

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    lookup(taskId);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px', maxWidth: '640px' }}>
      <span style={{ fontSize: '12px', color: 'var(--color-text-muted)' }}>
        Look up a task by ID via GET /api/task/&#123;id&#125;
      </span>

      {/* Search */}
      <form
        onSubmit={handleSubmit}
        style={{ display: 'flex', gap: '8px', alignItems: 'stretch' }}
      >
        <input
          type="text"
          value={taskId}
          onChange={(e) => setTaskId(e.target.value)}
          placeholder="Enter task ID (UUID)"
          style={{
            ...inputStyle,
            flex: 1,
          }}
        />
        <button
          type="submit"
          disabled={loading || !taskId.trim()}
          style={{
            ...buttonStyle,
            opacity: loading || !taskId.trim() ? 0.5 : 1,
            cursor: loading || !taskId.trim() ? 'not-allowed' : 'pointer',
          }}
        >
          {loading ? 'Searching…' : 'Lookup'}
        </button>
      </form>

      {/* Error */}
      {error && (
        <div style={errorBoxStyle}>{error}</div>
      )}

      {/* Result */}
      {data && (
        <div
          style={{
            background: 'var(--color-bg-card)',
            border: '1px solid var(--color-border-default)',
            borderRadius: 'var(--radius-lg)',
            overflow: 'hidden',
            animation: 'slideUp 0.2s ease',
          }}
        >
          {/* Header */}
          <div
            style={{
              padding: '16px 20px',
              borderBottom: '1px solid var(--color-border-default)',
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
            }}
          >
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: '13px',
                color: 'var(--color-text-emphasis)',
                fontWeight: 600,
              }}
            >
              {data.ID}
            </span>
            <span
              style={{
                display: 'inline-block',
                padding: '3px 10px',
                borderRadius: 'var(--radius-sm)',
                fontSize: '11px',
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.04em',
                ...getStatusStyle(data.Status),
              }}
            >
              {data.Status}
            </span>
          </div>

          {/* Fields */}
          <div style={{ padding: '16px 20px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
            <FieldRow label="Type" value={data.Type} />
            <FieldRow label="Priority" value={String(data.Priority)} mono />
            <FieldRow label="Queue" value={data.Queue || '—'} mono />
            <FieldRow label="Worker" value={data.WorkerID || '—'} mono />
            <FieldRow label="Retries" value={`${data.RetryCount} / ${data.MaxRetries}`} mono />
            <FieldRow label="Created" value={formatDateTime(data.CreatedAt)} />
            <FieldRow label="Updated" value={formatDateTime(data.UpdatedAt)} />
            {data.ETA && <FieldRow label="ETA" value={formatDateTime(data.ETA)} />}
            {data.CronExpr && <FieldRow label="Cron" value={data.CronExpr} mono />}
            {data.ErrorMsg && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span style={fieldLabelStyle}>Error</span>
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: '12px',
                    color: 'var(--color-status-danger-text)',
                    lineHeight: 1.5,
                    wordBreak: 'break-word',
                  }}
                >
                  {data.ErrorMsg}
                </span>
              </div>
            )}

            {/* Payload */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
              <span style={fieldLabelStyle}>Payload</span>
              <pre
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: '12px',
                  color: 'var(--color-text-secondary)',
                  lineHeight: 1.6,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  margin: 0,
                  background: 'var(--color-bg-surface)',
                  padding: '12px',
                  borderRadius: 'var(--radius-md)',
                  border: '1px solid var(--color-border-subtle)',
                }}
              >
                {typeof data.Payload === 'string'
                  ? data.Payload
                  : JSON.stringify(data.Payload, null, 2)}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function FieldRow({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div style={{ display: 'flex', alignItems: 'baseline', gap: '12px' }}>
      <span style={fieldLabelStyle}>{label}</span>
      <span
        style={{
          fontSize: '13px',
          color: 'var(--color-text-primary)',
          ...(mono ? { fontFamily: 'var(--font-mono)', fontSize: '12px' } : {}),
        }}
      >
        {value}
      </span>
    </div>
  );
}

function getStatusStyle(status: string): React.CSSProperties {
  switch (status) {
    case 'done':
      return {
        background: 'var(--color-status-ok-bg)',
        color: 'var(--color-status-ok-text)',
      };
    case 'failed':
    case 'dead':
      return {
        background: 'var(--color-status-danger-bg)',
        color: 'var(--color-status-danger-text)',
      };
    case 'running':
    case 'claimed':
      return {
        background: 'var(--color-status-warn-bg)',
        color: 'var(--color-status-warn-text)',
      };
    default:
      return {
        background: 'var(--color-bg-elevated)',
        color: 'var(--color-text-secondary)',
      };
  }
}

const fieldLabelStyle: React.CSSProperties = {
  fontSize: '11px',
  fontWeight: 600,
  color: 'var(--color-text-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  minWidth: '70px',
  flexShrink: 0,
};

const inputStyle: React.CSSProperties = {
  padding: '10px 14px',
  borderRadius: 'var(--radius-md)',
  border: '1px solid var(--color-border-default)',
  background: 'var(--color-bg-input)',
  color: 'var(--color-text-primary)',
  fontSize: '13px',
  fontFamily: 'var(--font-mono)',
  outline: 'none',
  transition: 'border-color var(--transition-fast)',
};

const buttonStyle: React.CSSProperties = {
  padding: '10px 20px',
  borderRadius: 'var(--radius-md)',
  border: '1px solid var(--color-border-hover)',
  background: 'var(--color-bg-elevated)',
  color: 'var(--color-text-primary)',
  fontSize: '13px',
  fontWeight: 500,
  fontFamily: 'var(--font-sans)',
  transition: 'all var(--transition-fast)',
  whiteSpace: 'nowrap',
};

const errorBoxStyle: React.CSSProperties = {
  padding: '12px 16px',
  background: 'var(--color-status-danger-bg)',
  border: '1px solid var(--color-status-danger-border)',
  borderRadius: 'var(--radius-md)',
  color: 'var(--color-status-danger-text)',
  fontSize: '13px',
};

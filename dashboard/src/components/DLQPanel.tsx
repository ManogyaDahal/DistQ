import { useState } from 'react';
import type { TaskBrief } from '../types';
import { retryDLQ, deleteDLQ } from '../hooks/useApi';
import { useToast } from './Toast';
import { formatDateTime } from '../utils';

interface Props {
  tasks: TaskBrief[];
}

export default function DLQPanel({ tasks }: Props) {
  const [retrying, setRetrying] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [retryingAll, setRetryingAll] = useState(false);
  const { showToast } = useToast();

  const handleRetryAll = async () => {
    setRetryingAll(true);
    try {
      const result = await retryDLQ();
      showToast(`Re-enqueued ${result.processed_count} tasks`, 'success');
    } catch (err) {
      showToast(
        err instanceof Error ? err.message : 'Failed to retry DLQ',
        'error'
      );
    } finally {
      setRetryingAll(false);
    }
  };

  const handleRetrySingle = async (id: string) => {
    setRetrying(id);
    try {
      const result = await retryDLQ(id);
      const success = result.results?.some((r) => r.success);
      if (success) {
        showToast(`Task ${id.slice(0, 8)}… re-enqueued`, 'success');
      } else {
        showToast(`Failed to retry task ${id.slice(0, 8)}…`, 'error');
      }
    } catch (err) {
      showToast(
        err instanceof Error ? err.message : 'Retry failed',
        'error'
      );
    } finally {
      setRetrying(null);
    }
  };

  const handleDeleteSingle = async (id: string) => {
    setDeleting(id);
    try {
      await deleteDLQ(id);
      showToast(`Task ${id.slice(0, 8)}… deleted`, 'success');
      // NOTE: DLQ tasks auto-update via WebSocket in real-time,
      // but we optimistically rely on the next WS tick to remove it.
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Delete failed', 'error');
    } finally {
      setDeleting(null);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <div>
          <span
            style={{
              fontSize: '12px',
              color: 'var(--color-text-muted)',
            }}
          >
            Tasks that exceeded maximum retry attempts
          </span>
        </div>
        <button
          onClick={handleRetryAll}
          disabled={tasks.length === 0 || retryingAll}
          style={{
            ...buttonStyle,
            opacity: tasks.length === 0 || retryingAll ? 0.4 : 1,
            cursor: tasks.length === 0 || retryingAll ? 'not-allowed' : 'pointer',
          }}
        >
          {retryingAll ? 'Retrying…' : 'Retry All'}
        </button>
      </div>

      {/* Task list */}
      {tasks.length === 0 ? (
        <div style={emptyStyle}>
          <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>
            DLQ is clean — no failed tasks
          </span>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {tasks.map((t) => (
            <div
              key={t.id}
              style={{
                background: 'var(--color-status-danger-bg)',
                border: '1px solid var(--color-status-danger-border)',
                borderRadius: 'var(--radius-md)',
                padding: '14px 16px',
                display: 'flex',
                flexDirection: 'column',
                gap: '8px',
                transition: 'border-color var(--transition-fast)',
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLElement).style.borderColor =
                  'var(--color-status-danger)';
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLElement).style.borderColor =
                  'var(--color-status-danger-border)';
              }}
            >
              {/* Top row: ID + type + retry button */}
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  gap: '8px',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '10px', minWidth: 0 }}>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: '12px',
                      color: 'var(--color-text-primary)',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {t.name ? (
                      <div>
                        <strong style={{ fontFamily: 'var(--font-sans)' }}>{t.name}</strong>
                        <br />
                        <span style={{ opacity: 0.6 }}>{t.id}</span>
                      </div>
                    ) : (
                      t.id
                    )}
                  </span>
                  <span
                    style={{
                      display: 'inline-block',
                      padding: '1px 8px',
                      borderRadius: 'var(--radius-sm)',
                      fontSize: '11px',
                      fontWeight: 500,
                      background: 'var(--color-bg-elevated)',
                      color: 'var(--color-text-secondary)',
                      flexShrink: 0,
                    }}
                  >
                    {t.type}
                  </span>
                </div>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                  <button
                    onClick={() => handleRetrySingle(t.id)}
                    disabled={retrying === t.id}
                    style={{
                      ...buttonSmallStyle,
                      opacity: retrying === t.id ? 0.5 : 1,
                    }}
                  >
                    {retrying === t.id ? '…' : 'Retry'}
                  </button>
                  <button
                    onClick={() => handleDeleteSingle(t.id)}
                    disabled={deleting === t.id}
                    style={{
                      ...buttonSmallStyle,
                      color: 'var(--color-status-danger-text)',
                      borderColor: 'var(--color-status-danger-border)',
                      opacity: deleting === t.id ? 0.5 : 1,
                    }}
                  >
                    {deleting === t.id ? '…' : 'Delete'}
                  </button>
                </div>
              </div>

              {/* Error message */}
              <div
                style={{
                  fontSize: '12px',
                  color: 'var(--color-status-danger-text)',
                  fontFamily: 'var(--font-mono)',
                  lineHeight: 1.5,
                  wordBreak: 'break-word',
                }}
              >
                {t.error_msg || 'Unknown failure'}
              </div>

              {/* Footer: priority + timestamp */}
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  fontSize: '11px',
                  color: 'var(--color-text-muted)',
                }}
              >
                <span>Priority {t.priority}</span>
                <span>{formatDateTime(t.created_at)}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

const buttonStyle: React.CSSProperties = {
  padding: '6px 16px',
  borderRadius: 'var(--radius-md)',
  border: '1px solid var(--color-border-hover)',
  background: 'var(--color-bg-elevated)',
  color: 'var(--color-text-primary)',
  fontSize: '12px',
  fontWeight: 500,
  fontFamily: 'var(--font-sans)',
  transition: 'all var(--transition-fast)',
};

const buttonSmallStyle: React.CSSProperties = {
  padding: '3px 10px',
  borderRadius: 'var(--radius-sm)',
  border: '1px solid var(--color-border-hover)',
  background: 'var(--color-bg-elevated)',
  color: 'var(--color-text-secondary)',
  fontSize: '11px',
  fontWeight: 500,
  fontFamily: 'var(--font-sans)',
  cursor: 'pointer',
  transition: 'all var(--transition-fast)',
  flexShrink: 0,
};

const emptyStyle: React.CSSProperties = {
  padding: '40px',
  textAlign: 'center',
};

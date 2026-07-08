import { useScheduled, deleteScheduled } from '../hooks/useApi';
import { formatDateTime } from '../utils';
import { useToast } from './Toast';
import type { ScheduledEntry } from '../types';

export default function ScheduledTasks() {
  const { data, loading, error, refetch } = useScheduled();
  const { showToast } = useToast();

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this scheduled task?')) return;
    try {
      await deleteScheduled(id);
      showToast('Task deleted successfully', 'success');
      refetch();
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Failed to delete task', 'error');
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <span style={{ fontSize: '12px', color: 'var(--color-text-muted)' }}>
          Tasks with a future ETA in the scheduled sorted set
        </span>
        <button onClick={refetch} disabled={loading} style={buttonStyle}>
          {loading ? 'Loading…' : 'Refresh'}
        </button>
      </div>

      {error && (
        <div style={errorBoxStyle}>{error}</div>
      )}

      {!error && (!data || data.length === 0) && !loading && (
        <div style={emptyStyle}>
          <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>
            No scheduled tasks
          </span>
        </div>
      )}

      {loading && !data && (
        <div style={emptyStyle}>
          <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>Loading…</span>
        </div>
      )}

      {data && data.length > 0 && (
        <div style={{ overflow: 'auto' }}>
          <table style={tableStyle}>
            <thead>
              <tr>
                <th style={thStyle}>Task ID</th>
                <th style={thStyle}>Type</th>
                <th style={thStyle}>Priority</th>
                <th style={thStyle}>Status</th>
                <th style={thStyle}>ETA</th>
                <th style={thStyle}>Countdown</th>
                <th style={thStyle}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {data.map((entry: ScheduledEntry) => {
                const etaDate = new Date(entry.eta * 1000);
                const now = Date.now();
                const diffMs = etaDate.getTime() - now;
                const countdown =
                  diffMs > 0
                    ? formatCountdown(diffMs)
                    : 'Ready';

                return (
                  <tr
                    key={entry.task.ID}
                    style={trStyle}
                    onMouseEnter={(e) => {
                      (e.currentTarget as HTMLElement).style.backgroundColor =
                        'var(--color-bg-hover)';
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLElement).style.backgroundColor =
                        'transparent';
                    }}
                  >
                    <td style={{ ...tdStyle, fontSize: '12px' }}>
                      {entry.task.Name ? (
                        <div>
                          <strong>{entry.task.Name}</strong>
                          <br />
                          <span style={{ opacity: 0.6, fontFamily: 'var(--font-mono)' }}>
                            {entry.task.ID}
                          </span>
                        </div>
                      ) : (
                        <span style={{ fontFamily: 'var(--font-mono)' }}>{entry.task.ID}</span>
                      )}
                    </td>
                    <td style={tdStyle}>
                      <span style={badgeStyle}>{entry.task.Type}</span>
                    </td>
                    <td style={{ ...tdStyle, fontFamily: 'var(--font-mono)', textAlign: 'center' }}>
                      {entry.task.Priority}
                    </td>
                    <td style={tdStyle}>
                      <span style={statusBadgeStyle}>{entry.task.Status}</span>
                    </td>
                    <td style={{ ...tdStyle, color: 'var(--color-text-secondary)' }}>
                      {formatDateTime(etaDate.toISOString())}
                    </td>
                    <td
                      style={{
                        ...tdStyle,
                        fontFamily: 'var(--font-mono)',
                        fontSize: '12px',
                        color:
                          diffMs > 0
                            ? 'var(--color-status-warn-text)'
                            : 'var(--color-status-ok-text)',
                      }}
                    >
                      {countdown}
                    </td>
                    <td style={tdStyle}>
                      <button
                        onClick={() => handleDelete(entry.task.ID)}
                        style={{ ...buttonStyle, padding: '4px 8px', color: 'var(--color-status-danger-text)' }}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function formatCountdown(ms: number): string {
  const totalSecs = Math.floor(ms / 1000);
  const h = Math.floor(totalSecs / 3600);
  const m = Math.floor((totalSecs % 3600) / 60);
  const s = totalSecs % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

const tableStyle: React.CSSProperties = {
  width: '100%',
  borderCollapse: 'collapse',
  fontSize: '13px',
};

const thStyle: React.CSSProperties = {
  textAlign: 'left',
  padding: '10px 16px',
  fontSize: '11px',
  fontWeight: 600,
  color: 'var(--color-text-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  borderBottom: '1px solid var(--color-border-default)',
};

const tdStyle: React.CSSProperties = {
  padding: '10px 16px',
  borderBottom: '1px solid var(--color-border-subtle)',
  color: 'var(--color-text-primary)',
};

const trStyle: React.CSSProperties = {
  transition: 'background var(--transition-fast)',
};

const badgeStyle: React.CSSProperties = {
  display: 'inline-block',
  padding: '1px 8px',
  borderRadius: 'var(--radius-sm)',
  fontSize: '11px',
  fontWeight: 500,
  background: 'var(--color-bg-elevated)',
  color: 'var(--color-text-secondary)',
};

const statusBadgeStyle: React.CSSProperties = {
  display: 'inline-block',
  padding: '2px 10px',
  borderRadius: 'var(--radius-sm)',
  fontSize: '11px',
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.04em',
  background: 'var(--color-status-warn-bg)',
  color: 'var(--color-status-warn-text)',
};

const buttonStyle: React.CSSProperties = {
  padding: '6px 16px',
  borderRadius: 'var(--radius-md)',
  border: '1px solid var(--color-border-hover)',
  background: 'var(--color-bg-elevated)',
  color: 'var(--color-text-primary)',
  fontSize: '12px',
  fontWeight: 500,
  fontFamily: 'var(--font-sans)',
  cursor: 'pointer',
  transition: 'all var(--transition-fast)',
};

const errorBoxStyle: React.CSSProperties = {
  padding: '12px 16px',
  background: 'var(--color-status-danger-bg)',
  border: '1px solid var(--color-status-danger-border)',
  borderRadius: 'var(--radius-md)',
  color: 'var(--color-status-danger-text)',
  fontSize: '13px',
};

const emptyStyle: React.CSSProperties = {
  padding: '40px',
  textAlign: 'center',
};

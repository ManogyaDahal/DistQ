import { useOngoing } from '../hooks/useApi';
import { formatDuration } from '../utils';
import type { OngoingTask } from '../types';

export default function OngoingTasks() {
  const { data, loading, error, refetch } = useOngoing();

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
          Tasks currently being processed by workers (pending acknowledgement)
        </span>
        <button onClick={refetch} disabled={loading} style={buttonStyle}>
          {loading ? 'Loading…' : 'Refresh'}
        </button>
      </div>

      {error && <div style={errorBoxStyle}>{error}</div>}

      {!error && (!data || data.length === 0) && !loading && (
        <div style={emptyStyle}>
          <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>
            No tasks currently being processed
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
                <th style={thStyle}>Source</th>
                <th style={thStyle}>Type</th>
                <th style={thStyle}>Worker</th>
                <th style={thStyle}>Stream ID</th>
                <th style={thStyle}>Idle Time</th>
                <th style={thStyle}>Retries</th>
              </tr>
            </thead>
            <tbody>
              {data.map((entry: OngoingTask, idx: number) => (
                <tr
                  key={`${entry.task.ID}-${idx}`}
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
                  <td
                    style={{
                      ...tdStyle,
                      fontFamily: 'var(--font-mono)',
                      fontSize: '12px',
                    }}
                  >
                    {entry.task.ID}
                  </td>
                  <td style={tdStyle}>
                    <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{entry.task.Source ?? '-'}</span>
                  </td>
                  <td style={tdStyle}>
                    <span style={badgeStyle}>{entry.task.Type}</span>
                  </td>
                  <td
                    style={{
                      ...tdStyle,
                      fontFamily: 'var(--font-mono)',
                      fontSize: '12px',
                      color: 'var(--color-text-secondary)',
                    }}
                  >
                    {entry.worker_id}
                  </td>
                  <td
                    style={{
                      ...tdStyle,
                      fontFamily: 'var(--font-mono)',
                      fontSize: '11px',
                      color: 'var(--color-text-muted)',
                    }}
                  >
                    {entry.stream_id}
                  </td>
                  <td
                    style={{
                      ...tdStyle,
                      fontFamily: 'var(--font-mono)',
                      fontSize: '12px',
                      color:
                        entry.idle_ms > 30000
                          ? 'var(--color-status-warn-text)'
                          : 'var(--color-text-secondary)',
                    }}
                  >
                    {formatDuration(entry.idle_ms)}
                  </td>
                  <td
                    style={{
                      ...tdStyle,
                      fontFamily: 'var(--font-mono)',
                      textAlign: 'center',
                      color:
                        entry.retries > 0
                          ? 'var(--color-status-warn-text)'
                          : 'var(--color-text-secondary)',
                    }}
                  >
                    {entry.retries}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
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

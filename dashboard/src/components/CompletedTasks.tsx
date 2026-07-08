import { useState } from 'react';
import { useCompleted } from '../hooks/useApi';
import type { Task } from '../types';
import { formatDateTime } from '../utils';

export default function CompletedTasks() {
  const { data, loading, error, refetch } = useCompleted();
  const [expandedId, setExpandedId] = useState<string | null>(null);

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
        <span style={{ fontSize: '12px', color: 'var(--color-text-muted)' }}>
          Tasks with status "done"
        </span>
        <button onClick={refetch} disabled={loading} style={buttonStyle}>
          {loading ? 'Loading…' : 'Refresh'}
        </button>
      </div>

      {error && (
        <div
          style={{
            padding: '12px 16px',
            background: 'var(--color-status-danger-bg)',
            border: '1px solid var(--color-status-danger-border)',
            borderRadius: 'var(--radius-md)',
            color: 'var(--color-status-danger-text)',
            fontSize: '13px',
          }}
        >
          {error}
        </div>
      )}

      {!error && (!data || data.length === 0) && !loading && (
        <div style={emptyStyle}>
          <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>
            No completed tasks found
          </span>
        </div>
      )}

      {loading && !data && (
        <div style={emptyStyle}>
          <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>
            Loading…
          </span>
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
                <th style={thStyle}>Created</th>
                <th style={thStyle}>Updated</th>
              </tr>
            </thead>
            <tbody>
              {data.map((t: Task) => (
                <TaskRow
                  key={t.ID}
                  task={t}
                  expanded={expandedId === t.ID}
                  onToggle={() =>
                    setExpandedId(expandedId === t.ID ? null : t.ID)
                  }
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function TaskRow({
  task,
  expanded,
  onToggle,
}: {
  task: Task;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <>
      <tr
        onClick={onToggle}
        style={{
          cursor: 'pointer',
          transition: 'background var(--transition-fast)',
        }}
        onMouseEnter={(e) => {
          (e.currentTarget as HTMLElement).style.backgroundColor =
            'var(--color-bg-hover)';
        }}
        onMouseLeave={(e) => {
          (e.currentTarget as HTMLElement).style.backgroundColor = 'transparent';
        }}
      >
        <td style={{ ...tdStyle, fontFamily: 'var(--font-mono)', fontSize: '12px' }}>
          <span style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
            <span
              style={{
                display: 'inline-block',
                width: '12px',
                fontSize: '10px',
                color: 'var(--color-text-muted)',
                transition: 'transform var(--transition-fast)',
                transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)',
              }}
            >
              ▶
            </span>
            {task.Name ? (
              <div>
                <strong>{task.Name}</strong>
                <br />
                <span style={{ opacity: 0.6 }}>{task.ID}</span>
              </div>
            ) : (
              task.ID
            )}
          </span>
        </td>
        <td style={tdStyle}>
          <span style={badgeStyle}>{task.Type}</span>
        </td>
        <td
          style={{
            ...tdStyle,
            fontFamily: 'var(--font-mono)',
            textAlign: 'center',
          }}
        >
          {task.Priority}
        </td>
        <td style={{ ...tdStyle, color: 'var(--color-text-secondary)' }}>
          {formatDateTime(task.CreatedAt)}
        </td>
        <td style={{ ...tdStyle, color: 'var(--color-text-secondary)' }}>
          {formatDateTime(task.UpdatedAt)}
        </td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={5} style={{ padding: 0 }}>
            <div
              style={{
                padding: '16px 24px',
                background: 'var(--color-bg-surface)',
                borderTop: '1px solid var(--color-border-subtle)',
                borderBottom: '1px solid var(--color-border-subtle)',
                animation: 'slideUp 0.2s ease',
              }}
            >
              <pre
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: '12px',
                  color: 'var(--color-text-secondary)',
                  lineHeight: 1.6,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  margin: 0,
                }}
              >
                {JSON.stringify(task, null, 2)}
              </pre>
            </div>
          </td>
        </tr>
      )}
    </>
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

const emptyStyle: React.CSSProperties = {
  padding: '40px',
  textAlign: 'center',
};

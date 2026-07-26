import { useState } from 'react';
import { useEnqueuedTasks } from '../hooks/useApi';
import { formatDateTime } from '../utils';

export default function EnqueuedTasks() {
  const { data: tasks, loading, error, refetch } = useEnqueuedTasks();
  const [expandedId, setExpandedId] = useState<string | null>(null);

  if (loading) return <div style={{ color: 'var(--color-text-muted)' }}>Loading enqueued tasks...</div>;
  if (error) return <div style={{ color: 'var(--color-status-danger-text)' }}>Error: {error}</div>;
  if (!tasks || tasks.length === 0) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        <span style={{ color: 'var(--color-text-muted)' }}>No enqueued tasks.</span>
        <button onClick={refetch} style={refreshButtonStyle}>Refresh</button>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <button onClick={refetch} style={refreshButtonStyle}>Refresh</button>
      </div>
      
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
          <thead>
            <tr>
              <th style={thStyle}>ID</th>
              <th style={thStyle}>Source</th>
              <th style={thStyle}>Type</th>
              <th style={thStyle}>Priority</th>
              <th style={thStyle}>Created At</th>
            </tr>
          </thead>
          <tbody>
            {tasks.map((task) => {
              const isExpanded = expandedId === task.ID;
              return (
                <React.Fragment key={task.ID}>
                  <tr
                    onClick={() => setExpandedId(isExpanded ? null : task.ID)}
                    style={{
                      borderBottom: '1px solid var(--color-border-subtle)',
                      cursor: 'pointer',
                      background: isExpanded ? 'var(--color-bg-hover)' : 'transparent',
                      transition: 'background var(--transition-fast)',
                    }}
                  >
                    <td style={{ ...tdStyle, fontFamily: 'var(--font-mono)' }}>
                      {task.ID.substring(0, 8)}...
                    </td>
                    <td style={tdStyle}>{(task as any).Source ?? '-'}</td>
                    <td style={tdStyle}>{task.Type}</td>
                    <td style={tdStyle}>
                      <span
                        style={{
                          display: 'inline-block',
                          padding: '2px 8px',
                          borderRadius: 'var(--radius-sm)',
                          fontSize: '11px',
                          fontWeight: 600,
                          background: 'var(--color-bg-elevated)',
                          color: 'var(--color-text-secondary)',
                        }}
                      >
                        P{task.Priority}
                      </span>
                    </td>
                    <td style={{ ...tdStyle, color: 'var(--color-text-muted)' }}>
                      {formatDateTime(task.CreatedAt)}
                    </td>
                  </tr>
                  {isExpanded && (
                    <tr>
                      <td colSpan={4} style={{ padding: 0 }}>
                        <div
                          style={{
                            padding: '16px',
                            background: 'var(--color-bg-surface)',
                            borderBottom: '1px solid var(--color-border-subtle)',
                          }}
                        >
                          <div style={{ marginBottom: '12px' }}>
                            <span style={labelStyle}>Full ID:</span>
                            <span style={{ fontFamily: 'var(--font-mono)', fontSize: '13px' }}>
                              {task.ID}
                            </span>
                          </div>
                          <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                            <span style={labelStyle}>Payload:</span>
                            <pre
                              style={{
                                fontFamily: 'var(--font-mono)',
                                fontSize: '12px',
                                color: 'var(--color-text-secondary)',
                                background: 'var(--color-bg-base)',
                                padding: '12px',
                                borderRadius: 'var(--radius-md)',
                                border: '1px solid var(--color-border-default)',
                                overflowX: 'auto',
                                margin: 0,
                              }}
                            >
                              {typeof task.Payload === 'string'
                                ? task.Payload
                                : JSON.stringify(task.Payload, null, 2)}
                            </pre>
                          </div>
                        </div>
                      </td>
                    </tr>
                  )}
                </React.Fragment>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

import React from 'react';

const thStyle: React.CSSProperties = {
  padding: '12px 16px',
  borderBottom: '1px solid var(--color-border-default)',
  color: 'var(--color-text-muted)',
  fontSize: '12px',
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.04em',
};

const tdStyle: React.CSSProperties = {
  padding: '12px 16px',
  color: 'var(--color-text-primary)',
  fontSize: '13px',
};

const labelStyle: React.CSSProperties = {
  display: 'inline-block',
  width: '80px',
  fontSize: '12px',
  fontWeight: 600,
  color: 'var(--color-text-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.04em',
};

const refreshButtonStyle: React.CSSProperties = {
  padding: '6px 12px',
  borderRadius: 'var(--radius-sm)',
  border: '1px solid var(--color-border-default)',
  background: 'var(--color-bg-elevated)',
  color: 'var(--color-text-secondary)',
  fontSize: '12px',
  cursor: 'pointer',
  transition: 'all var(--transition-fast)',
  alignSelf: 'flex-start',
};

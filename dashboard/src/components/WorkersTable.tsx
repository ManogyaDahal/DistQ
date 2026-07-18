import type { WorkerStatus } from '../types';
import { formatRelativeTime } from '../utils';

interface Props {
  workers: WorkerStatus[];
}

export default function WorkersTable({ workers }: Props) {
  const sorted = [...workers].sort((a, b) => a.id.localeCompare(b.id));

  return (
    <div style={{ overflow: 'auto' }}>
      {sorted.length === 0 ? (
        <div style={emptyStyle}>
          <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>
            No workers detected
          </span>
        </div>
      ) : (
        <table style={tableStyle}>
          <thead>
            <tr>
              <th style={thStyle}>Worker ID</th>
              <th style={thStyle}>Status</th>
              <th style={thStyle}>Slots In Use</th>
              <th style={thStyle}>Last Heartbeat</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((w) => (
              <tr
                key={w.id}
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
                <td style={{ ...tdStyle, fontFamily: 'var(--font-mono)', fontSize: '12px' }}>
                  {w.id}
                </td>
                <td style={tdStyle}>
                  <span
                    style={{
                      display: 'inline-block',
                      padding: '2px 10px',
                      borderRadius: 'var(--radius-sm)',
                      fontSize: '11px',
                      fontWeight: 600,
                      textTransform: 'uppercase',
                      letterSpacing: '0.04em',
                      ...(w.status === 'active'
                        ? {
                            background: 'var(--color-status-ok-bg)',
                            color: 'var(--color-status-ok-text)',
                          }
                        : {
                            background: 'var(--color-status-danger-bg)',
                            color: 'var(--color-status-danger-text)',
                          }),
                    }}
                  >
                    {w.status}
                  </span>
                </td>
                <td style={{ ...tdStyle }}>
                  <SlotUsage ongoing={w.ongoing_tasks} total={w.total_slots} />
                </td>
                <td style={{ ...tdStyle, color: 'var(--color-text-secondary)' }}>
                  {formatRelativeTime(w.last_seen)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

// ── SlotUsage sub-component ─────────────────────────────────────────────────

function SlotUsage({ ongoing, total }: { ongoing: number; total: number }) {
  // total=0 means the worker hasn't sent concurrency info yet (legacy)
  if (total === 0) {
    return (
      <span style={{ fontFamily: 'var(--font-mono)', fontSize: '12px' }}>
        {ongoing} running
      </span>
    );
  }

  const pct = Math.min(ongoing / total, 1);
  const isFull = ongoing >= total;
  const isEmpty = ongoing === 0;

  const barColor = isEmpty
    ? 'var(--color-status-ok-text)'
    : isFull
    ? 'var(--color-status-danger-text)'
    : 'var(--color-status-warn-text, #f59e0b)';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', minWidth: '120px' }}>
      {/* Label: "2 / 4 slots" */}
      <div style={{ display: 'flex', alignItems: 'baseline', gap: '4px' }}>
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: '13px',
            fontWeight: 600,
            color: barColor,
          }}
        >
          {ongoing}
        </span>
        <span style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>
          / {total} slots
        </span>
        {isFull && (
          <span
            style={{
              fontSize: '10px',
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
              color: 'var(--color-status-danger-text)',
              marginLeft: '4px',
            }}
          >
            full
          </span>
        )}
        {isEmpty && (
          <span
            style={{
              fontSize: '10px',
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
              color: 'var(--color-status-ok-text)',
              marginLeft: '4px',
            }}
          >
            idle
          </span>
        )}
      </div>
      {/* Progress bar */}
      <div
        style={{
          height: '4px',
          borderRadius: '2px',
          background: 'var(--color-bg-elevated)',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            height: '100%',
            width: `${pct * 100}%`,
            background: barColor,
            borderRadius: '2px',
            transition: 'width 0.4s ease, background 0.4s ease',
          }}
        />
      </div>
    </div>
  );
}

// ── Styles ──────────────────────────────────────────────────────────────────

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

const emptyStyle: React.CSSProperties = {
  padding: '32px',
  textAlign: 'center',
};

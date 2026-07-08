import { useCron } from '../hooks/useApi';
import { formatEpoch } from '../utils';
import type { CronJob } from '../types';

export default function CronJobs() {
  const { data, loading, error, refetch } = useCron();

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
          Recurring task schedules registered in Redis
        </span>
        <button onClick={refetch} disabled={loading} style={buttonStyle}>
          {loading ? 'Loading…' : 'Refresh'}
        </button>
      </div>

      {error && <div style={errorBoxStyle}>{error}</div>}

      {!error && (!data || data.length === 0) && !loading && (
        <div style={emptyStyle}>
          <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>
            No cron jobs registered
          </span>
        </div>
      )}

      {loading && !data && (
        <div style={emptyStyle}>
          <span style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>Loading…</span>
        </div>
      )}

      {data && data.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
          {data.map((job: CronJob) => (
            <div
              key={job.id}
              style={{
                background: 'var(--color-bg-card)',
                border: '1px solid var(--color-border-default)',
                borderRadius: 'var(--radius-md)',
                padding: '16px',
                display: 'flex',
                flexDirection: 'column',
                gap: '10px',
                transition: 'border-color var(--transition-fast)',
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLElement).style.borderColor =
                  'var(--color-border-hover)';
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLElement).style.borderColor =
                  'var(--color-border-default)';
              }}
            >
              {/* Top row: job ID + expression */}
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  gap: '12px',
                }}
              >
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: '12px',
                    color: 'var(--color-text-primary)',
                  }}
                >
                  {job.id}
                </span>
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: '13px',
                    fontWeight: 600,
                    color: 'var(--color-text-emphasis)',
                    background: 'var(--color-bg-elevated)',
                    padding: '3px 10px',
                    borderRadius: 'var(--radius-sm)',
                  }}
                >
                  {job.expr}
                </span>
              </div>

              {/* Template info */}
              <div
                style={{
                  display: 'flex',
                  gap: '16px',
                  flexWrap: 'wrap',
                  fontSize: '12px',
                }}
              >
                <div>
                  <span style={{ color: 'var(--color-text-muted)' }}>Type: </span>
                  <span style={{ color: 'var(--color-text-secondary)' }}>
                    {job.task_template?.type || '—'}
                  </span>
                </div>
                <div>
                  <span style={{ color: 'var(--color-text-muted)' }}>Priority: </span>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--color-text-secondary)',
                    }}
                  >
                    {job.task_template?.priority ?? '—'}
                  </span>
                </div>
                <div>
                  <span style={{ color: 'var(--color-text-muted)' }}>Max Retries: </span>
                  <span
                    style={{
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--color-text-secondary)',
                    }}
                  >
                    {job.task_template?.max_retries ?? '—'}
                  </span>
                </div>
                <div>
                  <span style={{ color: 'var(--color-text-muted)' }}>Last Run: </span>
                  <span style={{ color: 'var(--color-text-secondary)' }}>
                    {job.last_run_unix ? formatEpoch(job.last_run_unix) : 'Never'}
                  </span>
                </div>
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

import type { Metrics } from '../types';

interface Props {
  metrics: Metrics;
  queueDepths: Record<string, number>;
}

interface StatCardData {
  label: string;
  value: string | number;
  subtitle: string;
  danger?: boolean;
}

export default function StatsGrid({ metrics, queueDepths }: Props) {
  const totalQueueDepth = Object.values(queueDepths).reduce((a, b) => a + b, 0);

  const idlePct =
    metrics.total_workers > 0
      ? Math.round((metrics.free_workers / metrics.total_workers) * 100)
      : 0;

  const cards: StatCardData[] = [
    {
      label: 'Ongoing Tasks',
      value: metrics.ongoing_tasks,
      subtitle: 'Active executions',
    },
    {
      label: 'Workers',
      value: `${metrics.free_workers} / ${metrics.total_workers}`,
      subtitle: `${idlePct}% idle capacity`,
    },
    {
      label: 'Dead-Letter Queue',
      value: metrics.dlq_count,
      subtitle: 'Failed executions',
      danger: metrics.dlq_count > 0,
    },
    {
      label: 'Scheduled',
      value: metrics.scheduled_count,
      subtitle: 'Deferred tasks',
    },
    {
      label: 'Cron Jobs',
      value: metrics.cron_count,
      subtitle: 'Recurring schedules',
    },
    {
      label: 'Queue Depth',
      value: totalQueueDepth,
      subtitle: `Across ${Object.keys(queueDepths).length} priority levels`,
    },
  ];

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
        gap: '16px',
        animation: 'fadeIn 0.4s ease',
      }}
    >
      {cards.map((card) => (
        <div
          key={card.label}
          style={{
            background: card.danger
              ? 'var(--color-status-danger-bg)'
              : 'var(--color-bg-card)',
            border: `1px solid ${
              card.danger
                ? 'var(--color-status-danger-border)'
                : 'var(--color-border-default)'
            }`,
            borderRadius: 'var(--radius-lg)',
            padding: '20px',
            display: 'flex',
            flexDirection: 'column',
            gap: '8px',
            transition: 'border-color var(--transition-normal), background var(--transition-normal)',
          }}
          onMouseEnter={(e) => {
            if (!card.danger)
              (e.currentTarget as HTMLElement).style.borderColor =
                'var(--color-border-hover)';
          }}
          onMouseLeave={(e) => {
            if (!card.danger)
              (e.currentTarget as HTMLElement).style.borderColor =
                'var(--color-border-default)';
          }}
        >
          <span
            style={{
              fontSize: '11px',
              fontWeight: 600,
              color: 'var(--color-text-muted)',
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
            }}
          >
            {card.label}
          </span>
          <span
            style={{
              fontSize: '28px',
              fontWeight: 700,
              fontFamily: 'var(--font-mono)',
              color: card.danger
                ? 'var(--color-status-danger-text)'
                : 'var(--color-text-emphasis)',
              lineHeight: 1,
            }}
          >
            {card.value}
          </span>
          <span
            style={{
              fontSize: '12px',
              color: 'var(--color-text-muted)',
            }}
          >
            {card.subtitle}
          </span>
        </div>
      ))}
    </div>
  );
}

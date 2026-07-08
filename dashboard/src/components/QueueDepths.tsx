interface Props {
  queueDepths: Record<string, number>;
}

export default function QueueDepths({ queueDepths }: Props) {
  const entries = Object.entries(queueDepths).sort(
    ([a], [b]) => parseInt(b) - parseInt(a)
  );

  const maxDepth = Math.max(...entries.map(([, d]) => d), 1);

  if (entries.length === 0) {
    return (
      <div style={emptyStyle}>
        <span style={{ fontSize: '13px', color: 'var(--color-text-muted)' }}>
          No queues configured
        </span>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
      {entries.map(([priority, depth]) => {
        const pct = (depth / maxDepth) * 100;
        const shade = getShadeForPriority(parseInt(priority));

        return (
          <div key={priority} style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span
                  style={{
                    width: '6px',
                    height: '6px',
                    borderRadius: '50%',
                    backgroundColor: shade,
                    flexShrink: 0,
                  }}
                />
                <span
                  style={{
                    fontSize: '12px',
                    fontWeight: 500,
                    color: 'var(--color-text-secondary)',
                  }}
                >
                  Priority {priority}
                </span>
              </div>
              <span
                style={{
                  fontSize: '13px',
                  fontWeight: 600,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--color-text-emphasis)',
                }}
              >
                {depth}
              </span>
            </div>
            <div
              style={{
                width: '100%',
                height: '4px',
                borderRadius: '2px',
                backgroundColor: 'var(--color-bg-elevated)',
                overflow: 'hidden',
              }}
            >
              <div
                style={{
                  width: `${pct}%`,
                  height: '100%',
                  borderRadius: '2px',
                  backgroundColor: shade,
                  transition: 'width var(--transition-slow)',
                  minWidth: depth > 0 ? '3px' : '0',
                }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

function getShadeForPriority(priority: number): string {
  // Higher priority → brighter grey, lower → darker grey. Fully neutral.
  if (priority >= 10) return '#b0b0b0';
  if (priority >= 5) return '#808080';
  return '#505050';
}

const emptyStyle: React.CSSProperties = {
  padding: '32px',
  textAlign: 'center',
};

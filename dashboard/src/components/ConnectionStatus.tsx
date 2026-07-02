import type { ConnectionStatus } from '../hooks/useWebSocket';

interface Props {
  status: ConnectionStatus;
}

const statusConfig: Record<
  ConnectionStatus,
  { label: string; dotColor: string; animation: string }
> = {
  connected: {
    label: 'Live',
    dotColor: 'var(--color-status-ok)',
    animation: 'pulse-connected 2s ease-in-out infinite',
  },
  connecting: {
    label: 'Connecting…',
    dotColor: 'var(--color-status-warn)',
    animation: 'pulse-connecting 1.2s ease-in-out infinite',
  },
  disconnected: {
    label: 'Disconnected',
    dotColor: 'var(--color-status-danger)',
    animation: 'none',
  },
  error: {
    label: 'Error',
    dotColor: 'var(--color-status-danger)',
    animation: 'none',
  },
};

export default function ConnectionStatus({ status }: Props) {
  const config = statusConfig[status];

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
      }}
    >
      <span
        style={{
          width: '8px',
          height: '8px',
          borderRadius: '50%',
          backgroundColor: config.dotColor,
          animation: config.animation,
          flexShrink: 0,
          boxShadow: status === 'connected' ? `0 0 8px ${config.dotColor}` : 'none',
        }}
      />
      <span
        style={{
          fontSize: '12px',
          fontWeight: 500,
          color: 'var(--color-text-secondary)',
          letterSpacing: '0.02em',
          textTransform: 'uppercase' as const,
        }}
      >
        {config.label}
      </span>
    </div>
  );
}

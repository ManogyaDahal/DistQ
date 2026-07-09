import { useState } from 'react';
import ConnectionStatusIndicator from './ConnectionStatus';
import type { ConnectionStatus } from '../hooks/useWebSocket';
import type { Section } from '../types';

interface Props {
  connectionStatus: ConnectionStatus;
  activeSection: Section;
  onSectionChange: (section: Section) => void;
  children: React.ReactNode;
}

interface NavItem {
  id: Section;
  label: string;
  icon: string;
}

const navItems: NavItem[] = [
  { id: 'overview', label: 'Overview', icon: '◉' },
  { id: 'workers', label: 'Workers', icon: '⚙' },
  { id: 'ongoing', label: 'Ongoing', icon: '▶' },
  { id: 'enqueued', label: 'Enqueued', icon: '☷' },
  { id: 'completed', label: 'Completed', icon: '✓' },
  { id: 'scheduled', label: 'Scheduled', icon: '◷' },
  { id: 'cron', label: 'Cron Jobs', icon: '↻' },
  { id: 'dlq', label: 'Dead-Letter', icon: '✕' },
  { id: 'submit', label: 'Submit Task', icon: '+' },
  { id: 'lookup', label: 'Task Lookup', icon: '⌕' },
];

const sectionTitles: Record<Section, string> = {
  overview: 'Overview',
  workers: 'Active Workers',
  ongoing: 'Ongoing Tasks',
  enqueued: 'Enqueued Tasks',
  completed: 'Completed Tasks',
  scheduled: 'Scheduled Tasks',
  cron: 'Cron Jobs',
  dlq: 'Dead-Letter Queue',
  submit: 'Submit Task',
  lookup: 'Task Lookup',
};

export default function Layout({
  connectionStatus,
  activeSection,
  onSectionChange,
  children,
}: Props) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  return (
    <div
      style={{
        display: 'flex',
        width: '100%',
        height: '100vh',
        background: 'var(--color-bg-base)',
      }}
    >
      {/* ── Sidebar ──────────────────────────────────────────────── */}
      <aside
        style={{
          width: sidebarCollapsed ? '60px' : '220px',
          background: 'var(--color-bg-surface)',
          borderRight: '1px solid var(--color-border-default)',
          display: 'flex',
          flexDirection: 'column',
          transition: 'width var(--transition-normal)',
          flexShrink: 0,
          overflow: 'hidden',
        }}
      >
        {/* Brand */}
        <div
          style={{
            padding: sidebarCollapsed ? '20px 12px' : '20px 20px',
            borderBottom: '1px solid var(--color-border-default)',
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            minHeight: '64px',
          }}
        >
          <span
            style={{
              width: '10px',
              height: '10px',
              borderRadius: '50%',
              background: 'var(--color-text-emphasis)',
              flexShrink: 0,
            }}
          />
          {!sidebarCollapsed && (
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span
                style={{
                  fontSize: '16px',
                  fontWeight: 700,
                  color: 'var(--color-text-emphasis)',
                  letterSpacing: '-0.02em',
                }}
              >
                DistQ
              </span>
              <span
                style={{
                  fontSize: '10px',
                  fontWeight: 500,
                  color: 'var(--color-text-muted)',
                  padding: '1px 6px',
                  border: '1px solid var(--color-border-default)',
                  borderRadius: 'var(--radius-sm)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                }}
              >
                Telemetry
              </span>
            </div>
          )}
        </div>

        {/* Navigation */}
        <nav
          style={{
            flex: 1,
            padding: '12px 8px',
            display: 'flex',
            flexDirection: 'column',
            gap: '2px',
            overflowY: 'auto',
          }}
        >
          {navItems.map((item) => {
            const isActive = activeSection === item.id;
            return (
              <button
                key={item.id}
                onClick={() => onSectionChange(item.id)}
                title={sidebarCollapsed ? item.label : undefined}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '10px',
                  padding: sidebarCollapsed ? '10px 12px' : '8px 12px',
                  borderRadius: 'var(--radius-md)',
                  border: 'none',
                  background: isActive
                    ? 'var(--color-bg-active)'
                    : 'transparent',
                  color: isActive
                    ? 'var(--color-text-emphasis)'
                    : 'var(--color-text-secondary)',
                  fontSize: '13px',
                  fontWeight: isActive ? 600 : 400,
                  fontFamily: 'var(--font-sans)',
                  cursor: 'pointer',
                  transition: 'all var(--transition-fast)',
                  textAlign: 'left',
                  width: '100%',
                  justifyContent: sidebarCollapsed ? 'center' : 'flex-start',
                }}
                onMouseEnter={(e) => {
                  if (!isActive)
                    (e.currentTarget as HTMLElement).style.background =
                      'var(--color-bg-hover)';
                }}
                onMouseLeave={(e) => {
                  if (!isActive)
                    (e.currentTarget as HTMLElement).style.background =
                      'transparent';
                }}
              >
                <span
                  style={{
                    width: '18px',
                    textAlign: 'center',
                    fontSize: '14px',
                    flexShrink: 0,
                    opacity: isActive ? 1 : 0.6,
                  }}
                >
                  {item.icon}
                </span>
                {!sidebarCollapsed && <span>{item.label}</span>}
              </button>
            );
          })}
        </nav>

        {/* Collapse toggle */}
        <div
          style={{
            padding: '12px 8px',
            borderTop: '1px solid var(--color-border-default)',
          }}
        >
          <button
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            style={{
              width: '100%',
              padding: '8px 12px',
              borderRadius: 'var(--radius-md)',
              border: 'none',
              background: 'transparent',
              color: 'var(--color-text-muted)',
              fontSize: '12px',
              fontFamily: 'var(--font-sans)',
              cursor: 'pointer',
              transition: 'color var(--transition-fast)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: sidebarCollapsed ? 'center' : 'flex-start',
              gap: '8px',
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLElement).style.color =
                'var(--color-text-primary)';
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLElement).style.color =
                'var(--color-text-muted)';
            }}
          >
            <span style={{ fontSize: '14px' }}>
              {sidebarCollapsed ? '→' : '←'}
            </span>
            {!sidebarCollapsed && <span>Collapse</span>}
          </button>
        </div>
      </aside>

      {/* ── Main Content ─────────────────────────────────────────── */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        {/* Top bar */}
        <header
          style={{
            height: '64px',
            borderBottom: '1px solid var(--color-border-default)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '0 28px',
            background: 'var(--color-bg-surface)',
            flexShrink: 0,
          }}
        >
          <h2
            style={{
              fontSize: '15px',
              fontWeight: 600,
              color: 'var(--color-text-emphasis)',
              letterSpacing: '-0.01em',
            }}
          >
            {sectionTitles[activeSection]}
          </h2>
          <ConnectionStatusIndicator status={connectionStatus} />
        </header>

        {/* Content */}
        <main
          style={{
            flex: 1,
            padding: '28px',
            overflowY: 'auto',
          }}
        >
          {children}
        </main>
      </div>
    </div>
  );
}

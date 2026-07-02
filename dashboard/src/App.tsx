import { useState } from 'react';
import { useWebSocket } from './hooks/useWebSocket';
import { ToastProvider } from './components/Toast';
import Layout from './components/Layout';
import StatsGrid from './components/StatsGrid';
import QueueDepths from './components/QueueDepths';
import WorkersTable from './components/WorkersTable';
import DLQPanel from './components/DLQPanel';
import EnqueuedTasks from './components/EnqueuedTasks';
import CompletedTasks from './components/CompletedTasks';
import ScheduledTasks from './components/ScheduledTasks';
import CronJobs from './components/CronJobs';
import OngoingTasks from './components/OngoingTasks';
import SubmitTaskForm from './components/SubmitTaskForm';
import TaskLookup from './components/TaskLookup';
import type { Section } from './types';

function App() {
  const { data, status } = useWebSocket();
  const [activeSection, setActiveSection] = useState<Section>('overview');

  const metrics = data?.metrics ?? {
    ongoing_tasks: 0,
    total_workers: 0,
    free_workers: 0,
    dlq_count: 0,
    scheduled_count: 0,
    cron_count: 0,
  };

  const queueDepths = data?.queue_depths ?? {};
  const workers = data?.workers ?? [];
  const dlqTasks = data?.dlq_tasks ?? [];

  return (
    <ToastProvider>
      <Layout
        connectionStatus={status}
        activeSection={activeSection}
        onSectionChange={setActiveSection}
      >
        {activeSection === 'overview' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '28px' }}>
            <StatsGrid metrics={metrics} queueDepths={queueDepths} />

            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(380px, 1fr))',
                gap: '20px',
              }}
            >
              {/* Queue Depths Panel */}
              <Panel title="Queue Depths">
                <QueueDepths queueDepths={queueDepths} />
              </Panel>

              {/* Workers Mini */}
              <Panel title="Workers" badge={String(workers.length)}>
                <WorkersTable workers={workers} />
              </Panel>
            </div>

            {/* DLQ Panel */}
            {dlqTasks.length > 0 && (
              <Panel title="Dead-Letter Queue" badge={String(dlqTasks.length)}>
                <DLQPanel tasks={dlqTasks} />
              </Panel>
            )}
          </div>
        )}

        {activeSection === 'workers' && (
          <Panel title="Active Workers" badge={String(workers.length)}>
            <WorkersTable workers={workers} />
          </Panel>
        )}

        {activeSection === 'dlq' && (
          <Panel title="Dead-Letter Queue" badge={String(dlqTasks.length)}>
            <DLQPanel tasks={dlqTasks} />
          </Panel>
        )}

        {activeSection === 'completed' && (
          <Panel title="Completed Tasks">
            <CompletedTasks />
          </Panel>
        )}

        {activeSection === 'scheduled' && (
          <Panel title="Scheduled Tasks">
            <ScheduledTasks />
          </Panel>
        )}

        {activeSection === 'cron' && (
          <Panel title="Cron Jobs">
            <CronJobs />
          </Panel>
        )}

        {activeSection === 'ongoing' && (
          <Panel title="Ongoing Tasks">
            <OngoingTasks />
          </Panel>
        )}

        {activeSection === 'enqueued' && (
          <Panel title="Enqueued Tasks">
            <EnqueuedTasks />
          </Panel>
        )}

        {activeSection === 'submit' && (
          <Panel title="Submit Task">
            <SubmitTaskForm />
          </Panel>
        )}

        {activeSection === 'lookup' && (
          <Panel title="Task Lookup">
            <TaskLookup />
          </Panel>
        )}
      </Layout>
    </ToastProvider>
  );
}

// ── Reusable Panel wrapper ──────────────────────────────────────────────

function Panel({
  title,
  badge,
  children,
}: {
  title: string;
  badge?: string;
  children: React.ReactNode;
}) {
  return (
    <section
      style={{
        background: 'var(--color-bg-card)',
        border: '1px solid var(--color-border-default)',
        borderRadius: 'var(--radius-lg)',
        overflow: 'hidden',
        animation: 'fadeIn 0.3s ease',
      }}
    >
      <div
        style={{
          padding: '16px 20px',
          borderBottom: '1px solid var(--color-border-default)',
          display: 'flex',
          alignItems: 'center',
          gap: '10px',
        }}
      >
        <h3
          style={{
            fontSize: '13px',
            fontWeight: 600,
            color: 'var(--color-text-emphasis)',
            letterSpacing: '-0.01em',
          }}
        >
          {title}
        </h3>
        {badge && (
          <span
            style={{
              fontSize: '11px',
              fontWeight: 600,
              fontFamily: 'var(--font-mono)',
              color: 'var(--color-text-muted)',
              background: 'var(--color-bg-elevated)',
              padding: '1px 8px',
              borderRadius: 'var(--radius-sm)',
            }}
          >
            {badge}
          </span>
        )}
      </div>
      <div style={{ padding: '20px' }}>{children}</div>
    </section>
  );
}

export default App;

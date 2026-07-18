// ── WebSocket payload (pushed every 2s by the hub) ─────────────────────────

export interface StatsPayload {
  timestamp: number;
  metrics: Metrics;
  queue_depths: Record<string, number>;
  workers: WorkerStatus[];
  dlq_tasks: TaskBrief[];
}

export interface Metrics {
  ongoing_tasks: number;
  total_workers: number;
  free_workers: number;
  dlq_count: number;
  scheduled_count: number;
  cron_count: number;
}

export interface WorkerSlotStatus {
  id: string;
  status: 'idle' | 'busy';
}

export interface WorkerStatus {
  id: string;
  status: 'active' | 'stale';
  last_seen: number; // unix epoch seconds
  ongoing_tasks: number;
  /** goroutine concurrency the worker was started with (0 = legacy/unknown) */
  total_slots: number;
  workers?: WorkerSlotStatus[];
}

export interface TaskBrief {
  id: string;
  name?: string;
  type: string;
  priority: number;
  status: string;
  created_at: string; // ISO 8601
  error_msg: string;
}

// ── Full Task model (GET /api/task/{id}, GET /api/completed) ────────────────

export interface Task {
  ID: string;
  Name?: string;
  Type: string;
  Payload: unknown;
  Priority: number;
  Status: TaskStatus;
  MaxRetries: number;
  RetryCount: number;
  ETA: string | null;
  CronExpr: string;
  WorkerID: string;
  Queue: string;
  CreatedAt: string;
  UpdatedAt: string;
  ErrorMsg: string;
}

export type TaskStatus =
  | 'pending'
  | 'claimed'
  | 'running'
  | 'done'
  | 'failed'
  | 'retrying'
  | 'dead';

// ── GET /api/scheduled ──────────────────────────────────────────────────────

export interface ScheduledEntry {
  task: Task;
  eta: number; // unix epoch as float score
}

// ── GET /api/cron ───────────────────────────────────────────────────────────

export interface CronJob {
  id: string;
  expr: string;
  task_template: {
    name?: string;
    type: string;
    priority: number;
    payload: unknown;
    max_retries: number;
  };
  last_run_unix: number;
}

// ── GET /api/ongoing ────────────────────────────────────────────────────────

export interface OngoingTask {
  task: Task;
  worker_id: string;
  stream_id: string;
  idle_ms: number;
  retries: number;
}

// ── POST /api/task request ──────────────────────────────────────────────────

export interface SubmitTaskRequest {
  name?: string;
  type: string;
  payload: unknown;
  priority: number;
  max_retries?: number;
  eta?: string;
  cron_expr?: string;
}

export interface SubmitTaskResponse {
  id: string;
  name?: string;
  kind: 'immediate' | 'scheduled' | 'cron';
  status?: string;
  priority?: number;
  stream_id?: string;
  queue?: string;
  eta?: string;
  cron_expr?: string;
}

// ── POST /api/dlq/retry response ───────────────────────────────────────────

export interface RetryDLQResponse {
  processed_count: number;
  results: {
    id: string;
    success: boolean;
    error?: string;
  }[];
}

// ── Navigation ──────────────────────────────────────────────────────────────

export type Section =
  | 'overview'
  | 'workers'
  | 'ongoing'
  | 'enqueued'
  | 'dlq'
  | 'completed'
  | 'scheduled'
  | 'cron'
  | 'submit'
  | 'lookup';

import { useState, useEffect, useCallback } from 'react';
import type {
  Task,
  ScheduledEntry,
  CronJob,
  OngoingTask,
  SubmitTaskRequest,
  SubmitTaskResponse,
  RetryDLQResponse,
} from '../types';

// ── Generic fetch helper ────────────────────────────────────────────────

interface FetchState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  refetch: () => void;
}

function useFetch<T>(url: string, autoFetch = true): FetchState<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(url);
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `HTTP ${res.status}`);
      }
      const json = await res.json();
      setData(json);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [url]);

  useEffect(() => {
    if (autoFetch) {
      fetchData();
    }
  }, [autoFetch, fetchData]);

  return { data, loading, error, refetch: fetchData };
}

// ── Endpoint-specific hooks ─────────────────────────────────────────────

export function useCompleted() {
  return useFetch<Task[]>('/api/completed');
}

export function useScheduled() {
  return useFetch<ScheduledEntry[]>('/api/scheduled');
}

export function useCron() {
  return useFetch<CronJob[]>('/api/cron');
}

export function useOngoing() {
  return useFetch<OngoingTask[]>('/api/ongoing');
}

export function useTaskLookup() {
  const [data, setData] = useState<Task | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const lookup = useCallback(async (id: string) => {
    if (!id.trim()) return;
    setLoading(true);
    setError(null);
    setData(null);
    try {
      const res = await fetch(`/api/task/${encodeURIComponent(id)}`);
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `HTTP ${res.status}`);
      }
      const json = await res.json();
      setData(json);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, []);

  return { data, loading, error, lookup };
}

// ── Action functions ────────────────────────────────────────────────────

export async function submitTask(
  req: SubmitTaskRequest
): Promise<SubmitTaskResponse> {
  const res = await fetch('/api/task', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body.error || 'Failed to submit task');
  }
  return body;
}

export async function retryDLQ(id?: string): Promise<RetryDLQResponse> {
  const url = id ? `/api/dlq/retry/${encodeURIComponent(id)}` : '/api/dlq/retry';
  const res = await fetch(url, { method: 'POST' });
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body.error || 'Failed to retry DLQ');
  }
  return body;
}

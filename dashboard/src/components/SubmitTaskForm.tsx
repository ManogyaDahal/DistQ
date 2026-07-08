import { useState, type FormEvent } from 'react';
import { submitTask } from '../hooks/useApi';
import { useToast } from './Toast';
import type { SubmitTaskRequest, SubmitTaskResponse } from '../types';

export default function SubmitTaskForm() {
  const { showToast } = useToast();
  const [name, setName] = useState('');
  const [type, setType] = useState('');
  const [payload, setPayload] = useState('{}');
  const [priority, setPriority] = useState(5);
  const [maxRetries, setMaxRetries] = useState('');
  const [eta, setEta] = useState('');
  const [minDateTime] = useState(() => {
    const now = new Date();
    const pad = (n: number) => n.toString().padStart(2, '0');
    return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}T${pad(now.getHours())}:${pad(now.getMinutes())}`;
  });
  const [cronExpr, setCronExpr] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<SubmitTaskResponse | null>(null);
  const [payloadError, setPayloadError] = useState<string | null>(null);

  const validatePayload = (val: string): boolean => {
    try {
      JSON.parse(val);
      setPayloadError(null);
      return true;
    } catch (e) {
      setPayloadError(e instanceof Error ? e.message : 'Invalid JSON');
      return false;
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    if (!type.trim()) {
      showToast('Task type is required', 'error');
      return;
    }

    if (!validatePayload(payload)) {
      showToast('Invalid JSON payload', 'error');
      return;
    }

    setSubmitting(true);
    setResult(null);

    try {
      const req: SubmitTaskRequest = {
        name: name.trim() || undefined,
        type: type.trim(),
        payload: JSON.parse(payload),
        priority,
      };

      if (maxRetries.trim()) {
        req.max_retries = parseInt(maxRetries, 10);
      }
      if (eta.trim()) {
        const etaDate = new Date(eta);
        if (etaDate < new Date()) {
          showToast('ETA cannot be in the past', 'error');
          setSubmitting(false);
          return;
        }
        req.eta = etaDate.toISOString();
      }
      if (cronExpr.trim()) {
        const parts = cronExpr.trim().split(/\s+/);
        if (parts.length !== 5) {
          showToast('Invalid cron expression. Expected 5 fields (e.g. "*/5 * * * *")', 'error');
          setSubmitting(false);
          return;
        }
        req.cron_expr = cronExpr.trim();
      }

      const response = await submitTask(req);
      setResult(response);
      showToast(`Task submitted: ${response.kind} (${response.id.slice(0, 8)}…)`, 'success');
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Submit failed', 'error');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px', maxWidth: '640px' }}>
      <span style={{ fontSize: '12px', color: 'var(--color-text-muted)' }}>
        Submit a new task to the DistQ queue via POST /api/task
      </span>

      <form
        onSubmit={handleSubmit}
        style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
      >
        {/* Name */}
        <div style={fieldStyle}>
          <label style={labelStyle}>Task Name (Optional)</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Daily Data Sync"
            style={inputStyle}
          />
        </div>

        {/* Type */}
        <div style={fieldStyle}>
          <label style={labelStyle}>
            Task Type <span style={{ color: 'var(--color-status-danger-text)' }}>*</span>
          </label>
          <input
            type="text"
            value={type}
            onChange={(e) => setType(e.target.value)}
            placeholder="e.g. send_email"
            style={inputStyle}
          />
        </div>

        {/* Payload */}
        <div style={fieldStyle}>
          <label style={labelStyle}>Payload (JSON)</label>
          <textarea
            value={payload}
            onChange={(e) => {
              setPayload(e.target.value);
              if (payloadError) validatePayload(e.target.value);
            }}
            onBlur={() => validatePayload(payload)}
            rows={5}
            style={{
              ...inputStyle,
              fontFamily: 'var(--font-mono)',
              fontSize: '12px',
              resize: 'vertical',
              minHeight: '100px',
            }}
          />
          {payloadError && (
            <span
              style={{
                fontSize: '11px',
                color: 'var(--color-status-danger-text)',
                marginTop: '4px',
              }}
            >
              {payloadError}
            </span>
          )}
        </div>

        {/* Priority */}
        <div style={fieldStyle}>
          <label style={labelStyle}>Priority</label>
          <select
            value={priority}
            onChange={(e) => setPriority(parseInt(e.target.value, 10))}
            style={inputStyle}
          >
            {[1, 5, 10].map((p) => (
              <option key={p} value={p}>
                {p} {p === 1 ? '(Low)' : p === 5 ? '(Default)' : '(High)'}
              </option>
            ))}
          </select>
        </div>

        {/* Max Retries */}
        <div style={fieldStyle}>
          <label style={labelStyle}>Max Retries (optional)</label>
          <input
            type="number"
            value={maxRetries}
            onChange={(e) => setMaxRetries(e.target.value)}
            placeholder="Default from config"
            min={0}
            style={inputStyle}
          />
        </div>

        {/* ETA */}
        <div style={fieldStyle}>
          <label style={labelStyle}>ETA (optional)</label>
          <input
            type="datetime-local"
            value={eta}
            min={minDateTime}
            onChange={(e) => setEta(e.target.value)}
            style={inputStyle}
          />
        </div>

        {/* Cron Expression */}
        <div style={fieldStyle}>
          <label style={labelStyle}>Cron Expression (optional)</label>
          <input
            type="text"
            value={cronExpr}
            onChange={(e) => setCronExpr(e.target.value)}
            placeholder="e.g. */5 * * * *"
            style={inputStyle}
          />
          <span style={{ fontSize: '11px', color: 'var(--color-text-faint)' }}>
            If set, creates a recurring cron job instead of a one-time task
          </span>
        </div>

        {/* Submit */}
        <button
          type="submit"
          disabled={submitting}
          style={{
            ...submitButtonStyle,
            opacity: submitting ? 0.6 : 1,
            cursor: submitting ? 'not-allowed' : 'pointer',
          }}
        >
          {submitting ? 'Submitting…' : 'Submit Task'}
        </button>
      </form>

      {/* Result */}
      {result && (
        <div
          style={{
            background: 'var(--color-bg-card)',
            border: '1px solid var(--color-border-default)',
            borderRadius: 'var(--radius-md)',
            padding: '16px',
            animation: 'slideUp 0.2s ease',
          }}
        >
          <div
            style={{
              fontSize: '11px',
              fontWeight: 600,
              color: 'var(--color-text-muted)',
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              marginBottom: '10px',
            }}
          >
            Result
          </div>
          <pre
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: '12px',
              color: 'var(--color-text-secondary)',
              lineHeight: 1.6,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              margin: 0,
            }}
          >
            {JSON.stringify(result, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

const fieldStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: '6px',
};

const labelStyle: React.CSSProperties = {
  fontSize: '12px',
  fontWeight: 500,
  color: 'var(--color-text-secondary)',
};

const inputStyle: React.CSSProperties = {
  padding: '10px 14px',
  borderRadius: 'var(--radius-md)',
  border: '1px solid var(--color-border-default)',
  background: 'var(--color-bg-input)',
  color: 'var(--color-text-primary)',
  fontSize: '13px',
  fontFamily: 'var(--font-sans)',
  outline: 'none',
  transition: 'border-color var(--transition-fast)',
  width: '100%',
};

const submitButtonStyle: React.CSSProperties = {
  padding: '10px 24px',
  borderRadius: 'var(--radius-md)',
  border: '1px solid var(--color-border-hover)',
  background: 'var(--color-bg-elevated)',
  color: 'var(--color-text-emphasis)',
  fontSize: '13px',
  fontWeight: 600,
  fontFamily: 'var(--font-sans)',
  transition: 'all var(--transition-fast)',
  alignSelf: 'flex-start',
};



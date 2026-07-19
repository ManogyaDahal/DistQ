import { useEffect, useRef, useState } from 'react';
import { useToast } from './Toast';

type Mode = 'normal' | 'network' | 'broker' | 'worker' | 'timeout';
type Phase = 'idle' | 'running' | 'paused' | 'success' | 'error';
type Log = { id: number; time: string; message: string; tone: 'info' | 'success' | 'error' };
type Task = { Status?: string; WorkerID?: string; Queue?: string; Priority?: number; CreatedAt?: string };
type Result = { task_ids?: string[]; task?: Task; execution_ms?: number; latency_ms?: number; message?: string };
type Output = { task_id: string; status: string; queue: string; priority: number; worker_id: string; task_type: string; created_at: string; execution_ms: number; latency_ms: number };

const code = ['sdk := client.NewSDK("http://localhost:8080")', '', 'task := client.Task{', '    Type: "demo.print",', '    Payload: map[string]any{', '        "message": "Hello DistQ",', '    },', '    Priority: 5,', '}', '', 'resp, err := sdk.Enqueue(ctx, task)'];
const modeCode: Record<Mode, string[]> = {
  normal: code,
  network: [...code, '', 'if err != nil { // network error', '    return fmt.Errorf("enqueue failed: %w", err)', '}'],
  broker: [...code, '', 'if err != nil { // broker unavailable', '    return err', '}'],
  worker: [...code, '', 'if resp.Status == "failed" {', '    return errors.New("worker execution failed")', '}'],
  timeout: ['ctx, cancel := context.WithTimeout(ctx, 3*time.Second)', 'defer cancel()', '', ...code.slice(0, -1), 'resp, err := sdk.Enqueue(ctx, task)', 'if errors.Is(err, context.DeadlineExceeded) {', '    return err', '}'],
};
const stages = ['Go SDK', 'Broker', 'Redis Stream', 'Worker', 'Completed'];
const labels: Record<Mode, string> = { normal: 'Normal', network: 'Network Failure', broker: 'Broker Offline', worker: 'Worker Failure', timeout: 'Timeout' };
const now = () => new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
const ready = (): Log => ({ id: Date.now(), time: now(), message: 'SDK Playground ready.', tone: 'info' });

export default function SDKPlayground({ onViewDashboard }: { onViewDashboard?: (taskId: string) => void }) {
  const { showToast } = useToast();
  const [phase, setPhase] = useState<Phase>('idle');
  const [mode, setMode] = useState<Mode>('normal');
  const [logs, setLogs] = useState<Log[]>(() => [ready()]);
  const [line, setLine] = useState(-1);
  const [stage, setStage] = useState(-1);
  const [request, setRequest] = useState<object | null>(null);
  const [response, setResponse] = useState<Output | null>(null);
  const [elapsed, setElapsed] = useState<number | null>(null);
  const logRef = useRef<HTMLDivElement>(null);
  const paused = useRef(false);
  const token = useRef(0);
  const controller = useRef<AbortController | null>(null);
  const timers = useRef<number[]>([]);
  const liveCode = modeCode[mode];
  const sdkLine = liveCode.findIndex(value => value.startsWith('sdk :='));
  const taskLine = liveCode.findIndex(value => value.startsWith('task :='));
  const payloadLine = liveCode.findIndex(value => value.trimStart().startsWith('Payload:'));
  const enqueueLine = liveCode.findIndex(value => value.startsWith('resp, err :='));

  const cancel = () => { token.current += 1; controller.current?.abort(); controller.current = null; timers.current.forEach(clearTimeout); timers.current = []; paused.current = false; };
  const reset = () => { cancel(); setPhase('idle'); setLogs([ready()]); setLine(-1); setStage(-1); setRequest(null); setResponse(null); setElapsed(null); };
  useEffect(() => { logRef.current?.scrollTo({ top: logRef.current.scrollHeight, behavior: 'smooth' }); }, [logs]);
  useEffect(() => () => { cancel(); }, []);
  const add = (message: string, tone: Log['tone'] = 'info') => setLogs(items => [...items, { id: Date.now() + Math.random(), time: now(), message, tone }]);
  const copy = async (value: string, message: string) => { try { await navigator.clipboard.writeText(value); showToast('✓ ' + message, 'success'); } catch { showToast('Unable to copy to clipboard.', 'error'); } };
  const sleep = (ms: number, id: number) => new Promise<boolean>(resolve => {
    const check = () => { if (id !== token.current) return resolve(false); if (paused.current) { timers.current.push(window.setTimeout(check, 80)); return; } timers.current.push(window.setTimeout(() => resolve(id === token.current), ms)); };
    check();
  });
  const step = async (message: string, codeLine: number, arch: number, id: number, tone: Log['tone'] = 'info') => {
    if (!await sleep(480, id)) return false;
    setLine(codeLine); setStage(arch); add(message, tone); return true;
  };
  const finishError = (message: string, recovery: string, at: number, started: number) => { setLine(at); add(message, 'error'); add(recovery); setResponse(null); setElapsed(Math.round(performance.now() - started)); setPhase('error'); };

  const run = async () => {
    const id = ++token.current; const started = performance.now(); controller.current = new AbortController(); paused.current = false;
    setPhase('running'); setLogs([]); setLine(-1); setStage(-1); setRequest(null); setResponse(null); setElapsed(null);
    const payload = { type: 'demo.print', payload: { message: 'Hello DistQ' }, priority: 5 };
    try {
      if (!await step('Initializing Go Client SDK...', sdkLine, 0, id)) return;
      if (mode === 'network') return finishError('SDK request failed: dial tcp localhost:8080: network is unreachable.', 'Recovery: check the API endpoint and retry.', sdkLine, started);
      if (!await step('Connected to DistQ API through pkg/client.', sdkLine, 0, id, 'success')) return;
      if (!await step('Creating demo task...', taskLine, 0, id)) return;
      if (!await step('Serializing task payload...', payloadLine, 0, id)) return;
      setRequest(payload);
      if (!await step('Sending POST request via SDK...', enqueueLine, 0, id)) return;
      if (mode === 'timeout') return finishError('context deadline exceeded while waiting for POST /tasks.', 'Recovery: SDK cancelled the request safely; increase the timeout and retry.', enqueueLine, started);
      if (mode === 'broker') return finishError('SDK enqueue failed: broker is unavailable (connection refused).', 'Recovery: reconnect the broker, then replay this demonstration.', enqueueLine, started);
      const api = await fetch('/api/sdk/submit', { method: 'POST', signal: controller.current.signal });
      const data: Result & { error?: string; details?: string } = await api.json();
      if (!api.ok) throw new Error(data.details || data.error || 'SDK request was rejected.');
      if (!await step('Broker accepted task.', enqueueLine, 1, id, 'success')) return;
      if (!await step('Task published to Redis Stream.', enqueueLine, 2, id, 'success')) return;
      if (!await step('Worker picked up task.', enqueueLine, 3, id, 'success')) return;
      if (mode === 'worker') return finishError('Worker returned an execution error: demo.print handler failed.', 'Recovery: task marked for retry; inspect worker logs, then replay.', enqueueLine, started);
      if (!await step('Worker completed task successfully.', enqueueLine, 4, id, 'success')) return;
      const task = data.task || {};
      // latency_ms is returned by the API and is kept as the single source of truth for both views.
      const output: Output = { task_id: data.task_ids?.[0] || 'pending', status: 'completed', queue: task.Queue || 'default', priority: task.Priority || 5, worker_id: task.WorkerID || 'homie-44841', task_type: 'demo.print', created_at: task.CreatedAt || new Date().toISOString(), execution_ms: 312, latency_ms: data.latency_ms ?? data.execution_ms ?? 0 };
      setResponse(output);
      if (!await step('SDK received successful response.', enqueueLine, 4, id, 'success')) return;
      add('SDK demo completed successfully.', 'success'); setElapsed(Math.round(performance.now() - started)); setPhase('success');
    } catch (error) {
      if (id !== token.current || (error instanceof DOMException && error.name === 'AbortError')) return;
      finishError('SDK request failed: ' + (error instanceof Error ? error.message : 'Unknown error.'), 'Recovery: verify API and broker availability, then retry.', enqueueLine, started);
    } finally { if (id === token.current) controller.current = null; }
  };

  const togglePause = () => { if (phase === 'running') { paused.current = true; setPhase('paused'); add('Demo paused.'); } else if (phase === 'paused') { paused.current = false; setPhase('running'); add('Demo resumed.'); } };
  const busy = phase === 'running' || phase === 'paused';
  const formatDuration = (ms: number) => ms >= 1000 ? (ms / 1000).toFixed(1) + ' s' : ms + ' ms';
  const errorStatus = mode === 'timeout' ? '🟠 Timeout' : '🔴 Failed';

  return <div style={s.root}>
    <style>{'@keyframes sdk-in{from{opacity:0;transform:translateY(6px)}to{opacity:1}}.sdk-log{animation:sdk-in .25s ease both}.sdk-code-line{display:block;padding:0 12px;min-height:20px;white-space:pre;transition:background .28s ease,color .28s ease,box-shadow .28s ease}.sdk-code-line.active{background:#2e2a1a;color:#fff;border-left:2px solid #c9b86c;box-shadow:inset 0 0 20px rgba(201,184,108,.08)}.sdk-s{color:#a8d7a8}.sdk-k{color:#d5b3ee}.sdk-f{color:#8fc6e8}.sdk-c{color:#747474}.sdk-v{color:#e8c785}'}</style>
    <div style={s.hero}><div><div style={s.eyebrow}>Developer integration</div><h2 style={s.title}>Go Client SDK Playground</h2><p style={s.muted}>Follow a real Go SDK request from initialization to worker completion.</p></div><span style={statusBadge(phase, mode === 'timeout')}>{phase === 'success' ? '🟢 COMPLETED' : phase === 'error' ? (mode === 'timeout' ? '🟠 TIMEOUT' : '🔴 FAILED') : phase === 'paused' ? 'Ⅱ PAUSED' : busy ? '🟡 RUNNING' : '● READY'}</span></div>
    <div style={s.controls}><button disabled={busy} onClick={run} style={s.primary}>▶ Run Demo</button><button disabled={!busy} onClick={togglePause} style={s.button}>{phase === 'paused' ? '▶ Resume' : 'Ⅱ Pause'}</button><button disabled={busy} onClick={run} style={s.button}>↻ Replay</button><button onClick={reset} style={s.button}>↺ Reset</button><button disabled={busy} onClick={() => setLogs([])} style={s.button}>⌫ Clear Logs</button><label style={s.mode}>Demo mode <select disabled={busy} value={mode} onChange={e => setMode(e.target.value as Mode)} style={s.select}>{(Object.keys(labels) as Mode[]).map(key => <option key={key} value={key}>{labels[key]}</option>)}</select></label></div>
    <div style={s.progress}><div style={{ ...s.fill, width: (Math.max(0, stage + 1) / stages.length * 100) + '%' }} /></div>
    <div style={s.grid}><Panel title="Execution timeline"><div ref={logRef} style={s.timeline}>{logs.length ? logs.map((entry, index) => <div className="sdk-log" key={entry.id} style={s.event}><i style={{ ...s.dot, background: entry.tone === 'error' ? 'var(--color-status-danger-text)' : entry.tone === 'success' ? 'var(--color-status-ok-text)' : 'var(--color-status-warn-text)' }} /><div><time style={s.time}>[{entry.time}]</time><div style={{ color: entry.tone === 'error' ? 'var(--color-status-danger-text)' : entry.tone === 'success' ? 'var(--color-status-ok-text)' : 'var(--color-text-primary)' }}>{entry.message}</div>{index < logs.length - 1 && <div style={s.arrow}>↓</div>}</div></div>) : <div style={s.empty}>Logs cleared. Run a demo to begin.</div>}</div></Panel><Panel title="Live SDK code"><div style={s.codeToolbar}><span>go</span><button style={s.copy} onClick={() => copy(liveCode.join('\n'), 'SDK code copied')}>Copy SDK Code</button></div><pre style={s.code}>{liveCode.map((value, index) => <code key={index} className={'sdk-code-line ' + (line === index ? 'active' : '')}>{highlightGo(value) || ' '}</code>)}</pre></Panel></div>
    <Panel title="Request & response"><div className="sdk-request" style={s.request}><Json title="Request JSON" data={request} onCopy={() => copy(JSON.stringify(request, null, 2), 'Request JSON copied')} /><div className="sdk-endpoint" style={s.endpoint}>POST<strong>/tasks</strong><small>via pkg/client</small></div><Json title="Response JSON" data={response} onCopy={() => copy(JSON.stringify(response, null, 2), 'Response JSON copied')} /></div></Panel>
    <Panel title="Execution architecture"><div style={s.arch}>{stages.map((name, index) => <div key={name} style={{ display: 'contents' }}><div style={{ ...s.node, ...(index <= stage ? s.nodeOn : {}) }}>{index < stage ? '✓ ' : index === stage ? '◉ ' : '○ '}{name}</div>{index < stages.length - 1 && <span style={s.archArrow}>→</span>}</div>)}</div></Panel>
    {(phase === 'success' || phase === 'error') && <section style={{ ...s.done, borderColor: phase === 'success' ? 'var(--color-status-ok)' : mode === 'timeout' ? 'var(--color-status-warn)' : 'var(--color-status-danger-border)' }}><b style={{ color: phase === 'success' ? 'var(--color-status-ok-text)' : mode === 'timeout' ? 'var(--color-status-warn-text)' : 'var(--color-status-danger-text)' }}>{phase === 'success' ? '✓ SDK Demo Completed Successfully' : 'SDK Demo Did Not Complete'}</b><div style={s.summaryLead}><span style={statusBadge(phase, mode === 'timeout')}>{phase === 'success' ? '🟢 Completed' : errorStatus}</span><span style={s.duration}><span>Workflow Duration</span>{formatDuration(elapsed || 0)}</span></div>{phase === 'success' && response && <><div style={s.details}><Detail label="Task ID" value={response.task_id} onCopy={() => copy(response.task_id, 'Task ID copied')} /><Detail label="Status" value="🟢 Completed" /><Detail label="Queue" value={response.queue} /><Detail label="Priority" value={response.priority} /><Detail label="Worker" value={response.worker_id} /><Detail label="Task Type" value={response.task_type} /><Detail label="Execution Time" value={response.execution_ms + ' ms'} /><Detail label="API Latency" value={response.latency_ms + ' ms'} /><Detail label="Workflow Duration" value={formatDuration(elapsed || 0)} /></div><button style={{ ...s.button, ...s.dashboardButton }} onClick={() => onViewDashboard?.(response.task_id)}>View in Dashboard →</button></>}</section>}
  </div>;
}

function highlightGo(value: string) { const matcher = /(\/\/.*|"(?:\\.|[^"\\])*"|\b(?:if|return|defer|nil)\b|\b(?:sdk|client|context|fmt|errors)\b|\b(?:NewSDK|Enqueue|Task|WithTimeout|DeadlineExceeded|Errorf|New)\b|\b(?:Type|Payload|Priority)\b)/g; const parts = value.split(matcher); return <>{parts.map((part, index) => <span key={index} className={part.startsWith('//') ? 'sdk-c' : part.startsWith('"') ? 'sdk-s' : /^(if|return|defer|nil)$/.test(part) ? 'sdk-k' : /^(NewSDK|Enqueue|Task|WithTimeout|DeadlineExceeded|Errorf|New)$/.test(part) ? 'sdk-f' : /^(Type|Payload|Priority)$/.test(part) ? 'sdk-v' : ''}>{part}</span>)}</>; }
function Panel({ title, children }: { title: string; children: React.ReactNode }) { return <section style={s.panel}><header style={s.panelHead}>{title}</header>{children}</section>; }
function Json({ title, data, onCopy }: { title: string; data: object | null; onCopy: () => void }) { return <div style={s.jsonCard}><div style={{ ...s.jsonLabel, display: 'flex', justifyContent: 'space-between' }}><span>{title}</span><button style={s.copy} disabled={!data} onClick={onCopy}>Copy</button></div><pre style={s.json}>{data ? JSON.stringify(data, null, 2) : '// Waiting for demo…'}</pre></div>; }
function Detail({ label, value, onCopy }: { label: string; value: string | number; onCopy?: () => void }) { return <div><div style={s.detailLabel}>{label}</div><div style={s.detailValue}>{value}{onCopy && <button style={{ ...s.copy, marginLeft: '7px' }} onClick={onCopy}>Copy</button>}</div></div>; }
function statusBadge(phase: Phase, timeout = false) { const color = phase === 'success' ? 'var(--color-status-ok-text)' : phase === 'error' ? (timeout ? 'var(--color-status-warn-text)' : 'var(--color-status-danger-text)') : phase === 'running' ? 'var(--color-status-warn-text)' : 'var(--color-text-muted)'; const bg = phase === 'success' ? 'var(--color-status-ok-bg)' : phase === 'error' ? (timeout ? 'var(--color-status-warn-bg)' : 'var(--color-status-danger-bg)') : phase === 'running' ? 'var(--color-status-warn-bg)' : 'var(--color-bg-elevated)'; return { color, background: bg, fontFamily: 'var(--font-mono)', fontSize: '10px', padding: '4px 8px', borderRadius: '999px' }; }

const s = { copy: { background: 'transparent', border: '1px solid var(--color-border-hover)', color: 'var(--color-text-secondary)', borderRadius: 'var(--radius-sm)', padding: '3px 7px', fontSize: '10px', cursor: 'pointer' }, root: { display: 'flex', flexDirection: 'column', gap: '16px' }, hero: { display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px', flexWrap: 'wrap' }, eyebrow: { color: 'var(--color-text-muted)', fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '.1em' }, title: { margin: '3px 0', color: 'var(--color-text-emphasis)', fontSize: '24px', letterSpacing: '-.03em' }, muted: { color: 'var(--color-text-secondary)', fontSize: '12px' }, controls: { display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'center' }, button: { background: 'var(--color-bg-elevated)', border: '1px solid var(--color-border-hover)', color: 'var(--color-text-primary)', borderRadius: 'var(--radius-sm)', padding: '7px 10px', cursor: 'pointer', fontSize: '12px' }, primary: { background: 'var(--color-status-warn-bg)', border: '1px solid var(--color-status-warn)', color: 'var(--color-text-emphasis)', borderRadius: 'var(--radius-sm)', padding: '7px 10px', cursor: 'pointer', fontSize: '12px' }, mode: { color: 'var(--color-text-secondary)', display: 'flex', gap: '7px', alignItems: 'center', fontSize: '12px', marginLeft: '4px' }, select: { background: 'var(--color-bg-elevated)', border: '1px solid var(--color-border-hover)', color: 'var(--color-text-primary)', borderRadius: 'var(--radius-sm)', padding: '6px 8px' }, progress: { height: '3px', borderRadius: '3px', background: 'var(--color-bg-elevated)', overflow: 'hidden' }, fill: { height: '100%', background: 'var(--color-status-warn-text)', transition: 'width .35s ease' }, grid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(310px, 1fr))', gap: '16px' }, panel: { background: 'var(--color-bg-surface)', border: '1px solid var(--color-border-default)', borderRadius: 'var(--radius-md)', overflow: 'hidden' }, panelHead: { padding: '10px 13px', borderBottom: '1px solid var(--color-border-default)', color: 'var(--color-text-secondary)', fontSize: '10px', fontWeight: 600, letterSpacing: '.08em', textTransform: 'uppercase' }, timeline: { height: '292px', overflowY: 'auto', padding: '15px', background: 'var(--color-bg-input)', fontFamily: 'var(--font-mono)', fontSize: '12px' }, event: { display: 'grid', gridTemplateColumns: '13px 1fr', gap: '8px', minHeight: '38px' }, dot: { width: '7px', height: '7px', borderRadius: '50%', marginTop: '6px' }, time: { color: 'var(--color-text-muted)', fontSize: '10px' }, arrow: { color: 'var(--color-text-faint)', height: '16px' }, empty: { color: 'var(--color-text-muted)', paddingTop: '110px', textAlign: 'center' }, codeToolbar: { padding: '7px 12px', borderBottom: '1px solid var(--color-border-default)', display: 'flex', justifyContent: 'space-between', color: 'var(--color-text-muted)', fontFamily: 'var(--font-mono)', fontSize: '10px' }, code: { margin: 0, height: '292px', overflow: 'auto', padding: '15px 0', background: '#090909', color: '#d2d2d2', fontSize: '12px', lineHeight: 1.65, fontFamily: 'var(--font-mono)' }, request: { padding: '14px', display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) 120px minmax(0, 1fr)', gap: '12px', alignItems: 'stretch' }, jsonCard: { minWidth: 0, background: 'var(--color-bg-input)', border: '1px solid var(--color-border-default)', borderRadius: 'var(--radius-sm)', overflow: 'hidden' }, jsonLabel: { padding: '7px 10px', color: 'var(--color-text-muted)', borderBottom: '1px solid var(--color-border-default)', fontSize: '10px', textTransform: 'uppercase', letterSpacing: '.07em' }, json: { margin: 0, padding: '11px', minHeight: '130px', overflow: 'auto', color: '#b8d6ef', fontSize: '11px', lineHeight: 1.55 }, endpoint: { display: 'flex', flexDirection: 'column', justifyContent: 'center', alignItems: 'center', gap: '2px', textAlign: 'center', color: 'var(--color-status-warn-text)', fontFamily: 'var(--font-mono)', fontSize: '12px', background: 'var(--color-status-warn-bg)', borderRadius: 'var(--radius-sm)' }, arch: { padding: '16px', display: 'flex', gap: '8px', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap' }, node: { padding: '9px 11px', border: '1px solid var(--color-border-default)', borderRadius: 'var(--radius-sm)', color: 'var(--color-text-muted)', fontFamily: 'var(--font-mono)', fontSize: '11px', transition: 'all .25s ease' }, nodeOn: { borderColor: 'var(--color-status-ok)', background: 'var(--color-status-ok-bg)', color: 'var(--color-status-ok-text)', animation: 'sdk-glow 1.2s ease-in-out' }, archArrow: { color: 'var(--color-text-faint)' }, done: { padding: '15px', background: 'var(--color-bg-surface)', border: '1px solid', borderRadius: 'var(--radius-md)', display: 'grid', gap: '10px' }, summaryLead: { display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }, duration: { display: 'flex', gap: '6px', color: 'var(--color-text-primary)', fontFamily: 'var(--font-mono)', fontSize: '12px' }, details: { marginTop: '3px', paddingTop: '12px', borderTop: '1px solid var(--color-border-default)', display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(120px, 1fr))', gap: '12px' }, detailLabel: { color: 'var(--color-text-muted)', fontSize: '10px', textTransform: 'uppercase', letterSpacing: '.06em' }, detailValue: { color: 'var(--color-text-primary)', fontFamily: 'var(--font-mono)', fontSize: '11px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }, dashboardButton: { justifySelf: 'start', marginTop: '2px', borderColor: 'var(--color-status-ok)', color: 'var(--color-status-ok-text)' } } as const;

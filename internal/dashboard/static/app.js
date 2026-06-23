// DistQ Telemetry Dashboard script

(function() {
    let ws = null;
    let reconnectTimer = null;
    
    // UI elements
    const connectionStatusEl = document.getElementById('connection-status');
    const connectionTextEl = document.getElementById('connection-text');
    
    const statOngoingEl = document.getElementById('stat-ongoing');
    const statFreeWorkersEl = document.getElementById('stat-free-workers');
    const statTotalWorkersEl = document.getElementById('stat-total-workers');
    const activePercentageEl = document.getElementById('active-percentage');
    const statDlqEl = document.getElementById('stat-dlq');
    const cardDlqEl = document.getElementById('card-dlq');
    const statScheduledEl = document.getElementById('stat-scheduled');
    const statCronEl = document.getElementById('stat-cron');
    
    const queueDepthsContainer = document.getElementById('queue-depths-container');
    const workersCountEl = document.getElementById('workers-count');
    const workersTableBody = document.getElementById('workers-table-body');
    const dlqTaskListEl = document.getElementById('dlq-task-list');
    const btnReprocess = document.getElementById('btn-reprocess');
    
    const toastEl = document.getElementById('toast');

    // Initialize connection
    function connect() {
        if (ws) {
            ws.close();
        }
        
        clearTimeout(reconnectTimer);
        
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/ws`;
        
        setConnectionState('connecting', 'Connecting...');
        
        ws = new WebSocket(wsUrl);
        
        ws.onopen = function() {
            setConnectionState('connected', 'Live');
            showToast('Connected to telemetry stream');
        };
        
        ws.onmessage = function(event) {
            try {
                const data = JSON.parse(event.data);
                updateUI(data);
            } catch (e) {
                console.error('Failed to parse websocket message', e);
            }
        };
        
        ws.onclose = function() {
            setConnectionState('disconnected', 'Disconnected');
            scheduleReconnect();
        };
        
        ws.onerror = function() {
            setConnectionState('error', 'Error');
        };
    }

    function setConnectionState(state, text) {
        connectionTextEl.textContent = text;
        connectionStatusEl.className = 'pulse-indicator';
        
        if (state === 'connected') {
            connectionStatusEl.classList.add('connected');
        } else if (state === 'error' || state === 'disconnected') {
            connectionStatusEl.classList.add('error');
        }
    }

    function scheduleReconnect() {
        clearTimeout(reconnectTimer);
        reconnectTimer = setTimeout(connect, 3000);
    }

    // Toast utility
    let toastTimeout = null;
    function showToast(message) {
        toastEl.textContent = message;
        toastEl.classList.add('show');
        clearTimeout(toastTimeout);
        toastTimeout = setTimeout(() => {
            toastEl.classList.remove('show');
        }, 3000);
    }

    // Relative time formatter
    function formatRelativeTime(epochSecs) {
        const diff = Math.floor(Date.now() / 1000) - epochSecs;
        if (diff < 0) return 'just now';
        if (diff < 5) return 'just now';
        if (diff < 60) return `${diff}s ago`;
        const mins = Math.floor(diff / 60);
        if (mins < 60) return `${mins}m ago`;
        const hrs = Math.floor(mins / 60);
        return `${hrs}h ago`;
    }

    // Update UI components
    function updateUI(payload) {
        const metrics = payload.metrics || {};
        
        // Stats
        statOngoingEl.textContent = metrics.ongoing_tasks || 0;
        statFreeWorkersEl.textContent = metrics.free_workers || 0;
        statTotalWorkersEl.textContent = metrics.total_workers || 0;
        statDlqEl.textContent = metrics.dlq_count || 0;
        statScheduledEl.textContent = metrics.scheduled_count || 0;
        statCronEl.textContent = metrics.cron_count || 0;
        
        // Free capacity calculations
        if (metrics.total_workers > 0) {
            const pct = Math.round((metrics.free_workers / metrics.total_workers) * 100);
            activePercentageEl.textContent = `${pct}%`;
        } else {
            activePercentageEl.textContent = '0%';
        }
        
        // Highlight DLQ card if dirty
        if (metrics.dlq_count > 0) {
            cardDlqEl.classList.add('active-danger');
            btnReprocess.disabled = false;
        } else {
            cardDlqEl.classList.remove('active-danger');
            btnReprocess.disabled = true;
        }
        
        // Render queue depths
        renderQueueDepths(payload.queue_depths || {});
        
        // Render workers
        renderWorkers(payload.workers || []);
        
        // Render DLQ tasks
        renderDLQ(payload.dlq_tasks || []);
    }

    function renderQueueDepths(depths) {
        const entries = Object.entries(depths);
        if (entries.length === 0) {
            queueDepthsContainer.innerHTML = '<div class="empty-state">No queues configured</div>';
            return;
        }
        
        // Sort by priority level (descending)
        entries.sort((a, b) => parseInt(b[0]) - parseInt(a[0]));
        
        queueDepthsContainer.innerHTML = entries.map(([priority, depth]) => {
            let priorityClass = 'priority-low';
            const prioInt = parseInt(priority);
            if (prioInt >= 10) {
                priorityClass = 'priority-high';
            } else if (prioInt >= 5) {
                priorityClass = 'priority-medium';
            }
            
            return `
                <div class="queue-item">
                    <div class="queue-priority">
                        <span class="priority-indicator ${priorityClass}"></span>
                        <span class="queue-label">Priority ${priority}</span>
                    </div>
                    <span class="queue-depth">${depth}</span>
                </div>
            `;
        }).join('');
    }

    function renderWorkers(workers) {
        workersCountEl.textContent = workers.length;
        
        if (workers.length === 0) {
            workersTableBody.innerHTML = '<tr><td colspan="4" class="empty-state">No workers detected</td></tr>';
            return;
        }
        
        // Sort by ID
        workers.sort((a, b) => a.id.localeCompare(b.id));
        
        workersTableBody.innerHTML = workers.map(w => {
            const lastSeenText = formatRelativeTime(w.last_seen);
            const statusClass = w.status === 'active' ? 'active' : 'stale';
            
            return `
                <tr>
                    <td class="worker-id-cell">${escapeHtml(w.id)}</td>
                    <td>
                        <span class="status-tag ${statusClass}">${w.status}</span>
                    </td>
                    <td class="font-mono">${w.ongoing_tasks || 0}</td>
                    <td class="text-secondary">${lastSeenText}</td>
                </tr>
            `;
        }).join('');
    }

    function renderDLQ(tasks) {
        if (tasks.length === 0) {
            dlqTaskListEl.innerHTML = '<div class="empty-state">DLQ is clean</div>';
            return;
        }
        
        dlqTaskListEl.innerHTML = tasks.map(t => {
            const timeText = new Date(t.created_at).toLocaleString();
            return `
                <div class="dlq-task-item">
                    <div class="task-meta-row">
                        <span class="task-id">${escapeHtml(t.id)}</span>
                        <span class="task-type-badge">${escapeHtml(t.type)}</span>
                    </div>
                    <div class="task-error">${escapeHtml(t.error_msg || 'Unknown failure')}</div>
                    <div class="task-footer">
                        <span>Priority ${t.priority}</span>
                        <span>${timeText}</span>
                    </div>
                </div>
            `;
        }).join('');
    }

    function escapeHtml(str) {
        if (!str) return '';
        return str
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }

    // Wire up reprocessing button
    btnReprocess.addEventListener('click', async function() {
        btnReprocess.disabled = true;
        const text = btnReprocess.textContent;
        btnReprocess.textContent = 'Reprocessing...';
        
        try {
            const response = await fetch('/api/dlq/retry', { method: 'POST' });
            const data = await response.json();
            
            if (response.ok) {
                showToast(`Success: re-enqueued ${data.processed_count} tasks`);
            } else {
                showToast(`Error: ${data.error || 'Failed to reprocess'}`);
            }
        } catch (e) {
            showToast('Network error while reprocessing');
        } finally {
            btnReprocess.textContent = text;
            btnReprocess.disabled = false;
        }
    });

    // Start
    connect();
})();

// DistQ Telemetry Dashboard script

(function() {
    let ws = null;
    let reconnectTimer = null;
    
    // Cached telemetry data
    let cachedDLQTasks = [];
    let dlqSearchQuery = "";
    const expandedTaskIds = new Set();
    
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
    const dlqSearchInput = document.getElementById('dlq-search');
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
        
        setConnectionState('connecting', 'Connecting');
        
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
            setConnectionState('error', 'Connection Error');
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
        }, 4000);
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
        
        // Update cached DLQ tasks and re-render
        cachedDLQTasks = payload.dlq_tasks || [];
        renderDLQ(cachedDLQTasks);
    }

    function renderQueueDepths(depths) {
        const entries = Object.entries(depths);
        if (entries.length === 0) {
            queueDepthsContainer.innerHTML = '<div class="empty-state">No queues configured</div>';
            return;
        }
        
        // Sort by priority level (descending)
        entries.sort((a, b) => parseInt(b[0]) - parseInt(a[0]));
        
        // Find max depth for relative bar scaling
        const maxDepth = Math.max(...entries.map(([, depth]) => depth), 0);
        
        queueDepthsContainer.innerHTML = entries.map(([priority, depth]) => {
            let priorityClass = 'priority-low';
            const prioInt = parseInt(priority);
            if (prioInt >= 10) {
                priorityClass = 'priority-high';
            } else if (prioInt >= 5) {
                priorityClass = 'priority-medium';
            }
            
            // Calculate percentage fill
            const pct = maxDepth > 0 ? (depth / maxDepth) * 100 : 0;
            
            return `
                <div class="queue-item">
                    <div class="queue-priority">
                        <span class="priority-indicator ${priorityClass}"></span>
                        <span class="queue-label">Priority ${priority}</span>
                    </div>
                    <div class="queue-track">
                        <div class="queue-fill ${priorityClass}" style="width: ${pct}%"></div>
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
                    <td class="text-secondary font-mono">${lastSeenText}</td>
                </tr>
            `;
        }).join('');
    }

    function formatPayload(payloadStr) {
        if (!payloadStr) return 'No payload data';
        try {
            const parsed = JSON.parse(payloadStr);
            return JSON.stringify(parsed, null, 2);
        } catch (e) {
            return payloadStr;
        }
    }

    function renderDLQ(tasks) {
        // Apply filter query
        const filteredTasks = tasks.filter(t => {
            if (!dlqSearchQuery) return true;
            return t.id.toLowerCase().includes(dlqSearchQuery) || 
                   t.type.toLowerCase().includes(dlqSearchQuery) ||
                   (t.error_msg && t.error_msg.toLowerCase().includes(dlqSearchQuery));
        });

        if (filteredTasks.length === 0) {
            dlqTaskListEl.innerHTML = `<div class="empty-state">${tasks.length === 0 ? 'DLQ is clean' : 'No tasks match filter'}</div>`;
            return;
        }
        
        dlqTaskListEl.innerHTML = filteredTasks.map(t => {
            const isExpanded = expandedTaskIds.has(t.id);
            const timeText = new Date(t.created_at).toLocaleString();
            const formattedPayload = formatPayload(t.payload);
            
            return `
                <div class="dlq-task-item ${isExpanded ? 'expanded' : ''}" data-id="${escapeHtml(t.id)}">
                    <div class="task-meta-row">
                        <div class="task-id-group">
                            <span class="task-chevron">▶</span>
                            <span class="task-id">${escapeHtml(t.id)}</span>
                        </div>
                        <span class="task-type-badge">${escapeHtml(t.type)}</span>
                    </div>
                    <div class="task-error-summary">${escapeHtml(t.error_msg || 'Unknown failure')}</div>
                    
                    <div class="task-details-wrapper">
                        <div class="task-details-content">
                            <div class="task-details-inner">
                                <div class="detail-label-group">
                                    <div class="detail-item">
                                        <span class="detail-title">Priority</span>
                                        <span class="detail-value">${t.priority}</span>
                                    </div>
                                    <div class="detail-item">
                                        <span class="detail-title">Retries</span>
                                        <span class="detail-value">${t.retry_count} / ${t.max_retries}</span>
                                    </div>
                                </div>
                                
                                <div class="detail-item">
                                    <span class="detail-title">Error Message</span>
                                    <pre class="task-error-block">${escapeHtml(t.error_msg || 'No error details available')}</pre>
                                </div>

                                <div class="detail-item">
                                    <span class="detail-title">Payload (Parameters)</span>
                                    <pre class="task-payload-block">${escapeHtml(formattedPayload)}</pre>
                                </div>

                                <div class="task-details-actions">
                                    <button class="btn-single-reprocess" data-id="${escapeHtml(t.id)}">Reprocess This Task</button>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="task-footer">
                        <span>Failed at ${timeText}</span>
                        <span>Click to ${isExpanded ? 'collapse' : 'inspect'}</span>
                    </div>
                </div>
            `;
        }).join('');
    }

    // Set up search filter listener
    dlqSearchInput.addEventListener('input', function(e) {
        dlqSearchQuery = e.target.value.toLowerCase().trim();
        renderDLQ(cachedDLQTasks);
    });

    // Expand/Collapse task & Reprocess single task click handler using event delegation
    dlqTaskListEl.addEventListener('click', async function(e) {
        // Check if single task reprocess button was clicked
        const reprocessBtn = e.target.closest('.btn-single-reprocess');
        if (reprocessBtn) {
            e.stopPropagation();
            const taskId = reprocessBtn.dataset.id;
            if (!taskId) return;
            
            reprocessBtn.disabled = true;
            const originalText = reprocessBtn.textContent;
            reprocessBtn.textContent = 'Reprocessing...';
            
            try {
                const response = await fetch(`/api/dlq/retry?id=${encodeURIComponent(taskId)}`, { method: 'POST' });
                const data = await response.json();
                
                if (response.ok && data.processed_count > 0) {
                    showToast(`Success: re-enqueued task ${taskId}`);
                    expandedTaskIds.delete(taskId);
                } else {
                    showToast(`Error: ${data.error || 'Failed to reprocess task'}`);
                    reprocessBtn.disabled = false;
                    reprocessBtn.textContent = originalText;
                }
            } catch (err) {
                showToast('Network error while reprocessing task');
                reprocessBtn.disabled = false;
                reprocessBtn.textContent = originalText;
            }
            return;
        }

        // Prevent toggling expansion if clicking inside code pre blocks or other details
        if (e.target.closest('pre') || e.target.closest('.task-details-inner')) {
            return;
        }

        const taskItem = e.target.closest('.dlq-task-item');
        if (!taskItem) return;
        
        const taskId = taskItem.dataset.id;
        if (!taskId) return;

        if (expandedTaskIds.has(taskId)) {
            expandedTaskIds.delete(taskId);
            taskItem.classList.remove('expanded');
        } else {
            expandedTaskIds.add(taskId);
            taskItem.classList.add('expanded');
        }
        
        // Re-render task footer helper text for expand/collapse
        const footerSpan = taskItem.querySelector('.task-footer span:last-child');
        if (footerSpan) {
            footerSpan.textContent = expandedTaskIds.has(taskId) ? 'Click to collapse' : 'Click to inspect';
        }
    });

    function escapeHtml(str) {
        if (!str) return '';
        return str
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }

    // Wire up reprocessing button for all tasks
    btnReprocess.addEventListener('click', async function() {
        if (!confirm('Are you sure you want to re-enqueue all failed tasks in the DLQ?')) {
            return;
        }
        
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

    // Handle ESC and Cmd+K key shortcuts for search focus
    window.addEventListener('keydown', function(e) {
        if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
            e.preventDefault();
            dlqSearchInput.focus();
            dlqSearchInput.select();
        } else if (e.key === 'Escape' && document.activeElement === dlqSearchInput) {
            dlqSearchInput.blur();
        }
    });

    // Start
    connect();
})();

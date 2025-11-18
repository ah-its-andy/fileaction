// FileAction Frontend Application
// Fluent Design System Implementation

const API_BASE = '/api';

// Application State
const state = {
    workflows: [],
    currentWorkflowId: null,
    tasks: [],
    tasksTotal: 0,
    tasksPage: 1,
    tasksPageSize: 20,
    tasksStatus: 'all', // 'all', 'running', 'pending', 'completed', 'failed'
    tasksAutoRefresh: null,
    logWebSocket: null, // WebSocket connection for real-time logs
    currentTaskId: null,
    editingWorkflowId: null,
    currentTab: 'workflows', // 'workflows' or 'monitoring'
    monitoringAutoRefresh: null,
};

// Initialize application
document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
});

async function initializeApp() {
    // Load workflows
    await loadWorkflows();
    
    // Setup event listeners
    setupEventListeners();
    
    // Select first workflow if available
    if (state.workflows.length > 0) {
        selectWorkflow(state.workflows[0].id);
    } else {
        showEmptyState();
    }
}

function setupEventListeners() {
    // New workflow button
    document.getElementById('btnNewWorkflow').addEventListener('click', () => {
        openWorkflowModal();
    });
    
    // Workflow form submission
    document.getElementById('workflowForm').addEventListener('submit', handleWorkflowSubmit);
    
    // Close modal on background click
    document.querySelectorAll('.modal').forEach(modal => {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                closeModal(modal.id);
            }
        });
    });
}

// ============== Workflow Management ==============

async function loadWorkflows() {
    try {
        const workflows = await apiRequest('/workflows');
        state.workflows = workflows;
        renderWorkflowList();
    } catch (error) {
        console.error('Failed to load workflows:', error);
    }
}

function renderWorkflowList() {
    const container = document.getElementById('workflowList');
    
    if (state.workflows.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">📋</div>
                <p>No workflows yet</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = state.workflows.map(workflow => `
        <div class="workflow-item ${workflow.id === state.currentWorkflowId ? 'active' : ''}" 
             onclick="selectWorkflow('${workflow.id}')">
            <span class="workflow-icon ${workflow.enabled ? 'enabled' : 'disabled'}">
                ${workflow.enabled ? '●' : '○'}
            </span>
            <span class="workflow-name">${escapeHtml(workflow.name)}</span>
        </div>
    `).join('');
}

async function selectWorkflow(workflowId) {
    state.currentWorkflowId = workflowId;
    renderWorkflowList();
    await loadWorkflowDetail(workflowId);
    startTasksAutoRefresh();
}

async function loadWorkflowDetail(workflowId) {
    try {
        const workflow = await apiRequest(`/workflows/${workflowId}`);
        renderWorkflowHeader(workflow);
        await loadTasks(workflowId);
    } catch (error) {
        console.error('Failed to load workflow detail:', error);
    }
}

function renderWorkflowHeader(workflow) {
    const header = document.getElementById('contentHeader');
    header.innerHTML = `
        <div class="header-left">
            <span class="workflow-status-icon ${workflow.enabled ? 'enabled' : 'disabled'}">
                ${workflow.enabled ? '●' : '○'}
            </span>
            <div class="header-info">
                <h1>${escapeHtml(workflow.name)}</h1>
                <p>${escapeHtml(workflow.description || 'No description')}</p>
            </div>
        </div>
        <div class="header-actions">
            <button class="btn btn-primary btn-small" onclick="scanWorkflow('${workflow.id}')">
                🔍 Scan Files
            </button>
            <button class="btn btn-warning btn-small" onclick="clearAndRescan('${workflow.id}', '${escapeHtml(workflow.name)}')">
                🔄 Clear & Rescan
            </button>
            <button class="btn ${workflow.enabled ? 'btn-secondary' : 'btn-success'} btn-small" 
                    onclick="toggleWorkflow('${workflow.id}')">
                ${workflow.enabled ? '⏸ Disable' : '▶️ Enable'}
            </button>
            <button class="btn btn-secondary btn-small" onclick="editWorkflow('${workflow.id}')">
                ✏️ Edit
            </button>
            <button class="btn btn-danger btn-small" onclick="deleteWorkflow('${workflow.id}', '${escapeHtml(workflow.name)}')">
                🗑️ Delete
            </button>
        </div>
    `;
    
    // Render task status tabs
    renderTaskStatusTabs();
}

async function loadTasks(workflowId, page = 1, status = state.tasksStatus) {
    try {
        state.tasksPage = page;
        state.tasksStatus = status;
        const offset = (page - 1) * state.tasksPageSize;
        
        // Build query parameters
        let queryParams = `workflow_id=${workflowId}&limit=${state.tasksPageSize}&offset=${offset}`;
        
        // Add status filter based on active tab
        if (status && status !== 'all') {
            queryParams += `&status=${status}`;
        }
        
        const response = await apiRequest(`/tasks?${queryParams}`);
        state.tasks = response.tasks || [];
        state.tasksTotal = response.total || 0;
        
        renderTaskList();
    } catch (error) {
        console.error('Failed to load tasks:', error);
    }
}

function renderTaskStatusTabs() {
    const container = document.getElementById('contentBody');
    
    const tabs = `
        <div class="task-status-tabs">
            <button class="tab-button ${state.tasksStatus === 'all' ? 'active' : ''}" 
                onclick="switchTaskStatus('all')">
                All
            </button>
            <button class="tab-button ${state.tasksStatus === 'pending' ? 'active' : ''}" 
                onclick="switchTaskStatus('pending')">
                <span class="tab-icon">○</span> Pending
            </button>
            <button class="tab-button ${state.tasksStatus === 'running' ? 'active' : ''}" 
                onclick="switchTaskStatus('running')">
                <span class="tab-icon">▸</span> Running
            </button>
            <button class="tab-button ${state.tasksStatus === 'completed' ? 'active' : ''}" 
                onclick="switchTaskStatus('completed')">
                <span class="tab-icon">●</span> Completed
            </button>
            <button class="tab-button ${state.tasksStatus === 'failed' ? 'active' : ''}" 
                onclick="switchTaskStatus('failed')">
                <span class="tab-icon">✕</span> Failed
            </button>
        </div>
    `;
    
    // Insert tabs before task list
    const existingTabs = container.querySelector('.task-status-tabs');
    if (existingTabs) {
        existingTabs.remove();
    }
    container.insertAdjacentHTML('afterbegin', tabs);
}

function switchTaskStatus(status) {
    state.tasksStatus = status;
    loadTasks(state.currentWorkflowId, 1, status);
}

function renderTaskList() {
    const container = document.getElementById('contentBody');
    
    // Always render tabs to update active state
    renderTaskStatusTabs();
    
    // Find or create task list container
    let taskListContainer = container.querySelector('.task-list-container');
    if (!taskListContainer) {
        taskListContainer = document.createElement('div');
        taskListContainer.className = 'task-list-container';
        container.appendChild(taskListContainer);
    }
    
    if (state.tasks.length === 0) {
        taskListContainer.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">📋</div>
                <h3>No ${state.tasksStatus === 'all' ? '' : state.tasksStatus} tasks</h3>
                <p>${state.tasksStatus === 'all' ? 'Scan files to create tasks' : `No tasks with status: ${state.tasksStatus}`}</p>
            </div>
        `;
        return;
    }
    
    const totalPages = Math.ceil(state.tasksTotal / state.tasksPageSize);
    const startItem = (state.tasksPage - 1) * state.tasksPageSize + 1;
    const endItem = Math.min(state.tasksPage * state.tasksPageSize, state.tasksTotal);
    
    taskListContainer.innerHTML = `
        <div class="task-list">
            ${state.tasks.map(task => renderTaskCard(task)).join('')}
        </div>
        ${totalPages > 1 ? `
        <div class="pagination">
            <div class="pagination-info">
                Showing ${startItem}-${endItem} of ${state.tasksTotal} ${state.tasksStatus === 'all' ? '' : state.tasksStatus} tasks
            </div>
            <div class="pagination-controls">
                <button class="btn btn-secondary btn-small" 
                    onclick="loadTasks('${state.currentWorkflowId}', ${state.tasksPage - 1}, '${state.tasksStatus}')"
                    ${state.tasksPage === 1 ? 'disabled' : ''}>
                    ← Previous
                </button>
                <span class="pagination-pages">
                    Page ${state.tasksPage} of ${totalPages}
                </span>
                <button class="btn btn-secondary btn-small" 
                    onclick="loadTasks('${state.currentWorkflowId}', ${state.tasksPage + 1}, '${state.tasksStatus}')"
                    ${state.tasksPage === totalPages ? 'disabled' : ''}>
                    Next →
                </button>
            </div>
        </div>
        ` : ''}
    `;
}

function renderTaskCard(task) {
    const fileName = task.input_path.split('/').pop();
    const startTime = task.started_at ? formatDate(task.started_at) : '-';
    
    // Calculate duration
    let duration = '-';
    if (task.started_at) {
        const start = new Date(task.started_at);
        const end = task.completed_at ? new Date(task.completed_at) : new Date();
        const durationMs = end - start;
        const seconds = Math.floor(durationMs / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        
        if (hours > 0) {
            duration = `${hours}h ${minutes % 60}m`;
        } else if (minutes > 0) {
            duration = `${minutes}m ${seconds % 60}s`;
        } else {
            duration = `${seconds}s`;
        }
    }
    
    return `
        <div class="task-card">
            <div class="task-header">
                <span class="task-status ${task.status}">${task.status}</span>
                <div class="task-title" title="${escapeHtml(fileName)}">${escapeHtml(fileName)}</div>
                <div class="task-actions">
                    <button class="btn btn-secondary btn-small" onclick="viewTaskLog('${task.id}', '${task.status}')">
                        View Log
                    </button>
                    ${task.status === 'failed' || task.status === 'cancelled' ? `
                    <button class="btn btn-success btn-small" onclick="retryTask('${task.id}')">
                        Retry
                    </button>
                    ` : ''}
                    ${task.status === 'running' ? `
                    <button class="btn btn-danger btn-small" onclick="cancelTask('${task.id}')">
                        Cancel
                    </button>
                    ` : ''}
                </div>
            </div>
            <div class="task-info">
                ${task.error_message ? `
                <div class="task-info-item error">
                    <strong>Error:</strong>
                    <span title="${escapeHtml(task.error_message)}">${escapeHtml(task.error_message)}</span>
                </div>
                ` : `
                <div class="task-info-item file-path">
                    <strong>Path:</strong>
                    <span title="${escapeHtml(task.input_path)}">${escapeHtml(task.input_path)}</span>
                </div>
                <div class="task-info-item">
                    <strong>Started:</strong>
                    <span>${startTime}</span>
                </div>
                <div class="task-info-item">
                    <strong>Duration:</strong>
                    <span>${duration}</span>
                </div>
                `}
            </div>
        </div>
    `;
}

function showEmptyState() {
    const header = document.getElementById('contentHeader');
    const body = document.getElementById('contentBody');
    
    header.innerHTML = `
        <div class="header-left">
            <div class="header-info">
                <h1>Welcome to FileAction</h1>
                <p>Create your first workflow to get started</p>
            </div>
        </div>
    `;
    
    body.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">⚙️</div>
            <h3>No workflows yet</h3>
            <p>Click the + button to create your first workflow</p>
        </div>
    `;
}

// ============== Workflow Actions ==============

async function scanWorkflow(workflowId) {
    try {
        await apiRequest(`/workflows/${workflowId}/scan`, { method: 'POST' });
        showNotification('Scan started! Tasks will appear shortly.');
        setTimeout(() => loadTasks(workflowId), 2000);
    } catch (error) {
        console.error('Failed to scan workflow:', error);
        showNotification('Failed to start scan', 'error');
    }
}

async function clearAndRescan(workflowId, workflowName) {
    if (!confirm(`Clear all tasks and files for "${workflowName}" and rescan?\n\nThis will delete all existing tasks and file records for this workflow.`)) {
        return;
    }
    
    try {
        await apiRequest(`/workflows/${workflowId}/clear-index`, { method: 'POST' });
        showNotification('Index cleared! Rescanning files...');
        // Clear current tasks display
        state.tasks = [];
        renderTaskList();
        // Reload tasks after a delay
        setTimeout(() => loadTasks(workflowId), 2000);
    } catch (error) {
        console.error('Failed to clear and rescan:', error);
        showNotification('Failed to clear index', 'error');
    }
}

async function toggleWorkflow(workflowId) {
    try {
        const workflow = await apiRequest(`/workflows/${workflowId}/toggle`, { method: 'PUT' });
        await loadWorkflows();
        renderWorkflowHeader(workflow);
        showNotification(`Workflow ${workflow.enabled ? 'enabled' : 'disabled'}`);
    } catch (error) {
        console.error('Failed to toggle workflow:', error);
        showNotification('Failed to toggle workflow', 'error');
    }
}

async function deleteWorkflow(workflowId, workflowName) {
    if (!confirm(`Are you sure you want to delete "${workflowName}"?`)) {
        return;
    }
    
    try {
        await apiRequest(`/workflows/${workflowId}`, { method: 'DELETE' });
        await loadWorkflows();
        if (state.workflows.length > 0) {
            selectWorkflow(state.workflows[0].id);
        } else {
            showEmptyState();
        }
        showNotification('Workflow deleted');
    } catch (error) {
        console.error('Failed to delete workflow:', error);
        showNotification('Failed to delete workflow', 'error');
    }
}

// ============== Task Actions ==============

async function viewTaskLog(taskId, taskStatus) {
    state.currentTaskId = taskId;
    const modal = document.getElementById('logModal');
    const titleEl = document.getElementById('logModalTitle');
    const contentEl = document.getElementById('logContent');
    
    titleEl.textContent = 'Task Log';
    contentEl.innerHTML = '<div class="loading"></div> Loading log...';
    modal.classList.add('active');
    
    // Use WebSocket for running tasks, HTTP for completed tasks
    if (taskStatus === 'running') {
        connectWebSocket(taskId);
    } else {
        await loadTaskLog(taskId);
    }
}

function connectWebSocket(taskId) {
    // Close existing WebSocket if any
    if (state.logWebSocket) {
        state.logWebSocket.close();
        state.logWebSocket = null;
    }

    const contentEl = document.getElementById('logContent');
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/ws/logs`;
    
    try {
        const ws = new WebSocket(wsUrl);
        state.logWebSocket = ws;
        
        ws.onopen = () => {
            console.log('WebSocket connected for task:', taskId);
            contentEl.textContent = ''; // Clear loading message
            
            // Send subscribe message
            ws.send(JSON.stringify({
                action: 'subscribe',
                task_id: taskId
            }));
        };
        
        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                console.log('WebSocket message:', message.type);
                
                if (message.type === 'subscribed') {
                    console.log('Subscribed to task:', message.task_id);
                } else if (message.type === 'log') {
                    // Append new log content
                    contentEl.textContent += message.content;
                    // Auto-scroll to bottom
                    contentEl.scrollTop = contentEl.scrollHeight;
                } else if (message.type === 'complete') {
                    // Task completed
                    console.log('Task completed');
                    // Refresh task list to show updated status
                    setTimeout(() => loadTasks(state.currentWorkflowId), 500);
                } else if (message.type === 'close') {
                    // Server closing connection
                    console.log('Server closing WebSocket');
                    ws.close();
                } else if (message.type === 'pong') {
                    // Ping response
                }
            } catch (error) {
                console.error('Failed to parse WebSocket message:', error);
            }
        };
        
        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            contentEl.textContent = 'WebSocket connection error. Falling back to HTTP polling...';
            // Fallback to HTTP polling
            setTimeout(() => loadTaskLog(taskId), 1000);
        };
        
        ws.onclose = () => {
            console.log('WebSocket closed');
            state.logWebSocket = null;
        };
        
        // Send ping every 20 seconds to keep connection alive
        const pingInterval = setInterval(() => {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ action: 'ping' }));
            } else {
                clearInterval(pingInterval);
            }
        }, 20000);
        
    } catch (error) {
        console.error('Failed to create WebSocket:', error);
        // Fallback to HTTP polling
        loadTaskLog(taskId);
    }
}

async function loadTaskLog(taskId, offset = 0) {
    try {
        const response = await apiRequest(`/tasks/${taskId}/log/tail?offset=${offset}`);
        const contentEl = document.getElementById('logContent');
        
        if (response.content) {
            contentEl.textContent = response.content;
            contentEl.scrollTop = contentEl.scrollHeight;
        } else {
            contentEl.textContent = 'No log available';
        }
        
        // Continue polling if not completed
        if (!response.completed && state.currentTaskId === taskId) {
            setTimeout(() => loadTaskLog(taskId, response.offset), 1000);
        }
    } catch (error) {
        console.error('Failed to load task log:', error);
        document.getElementById('logContent').textContent = 'Failed to load log';
    }
}

async function retryTask(taskId) {
    try {
        await apiRequest(`/tasks/${taskId}/retry`, { method: 'POST' });
        showNotification('Task retry initiated');
        setTimeout(() => loadTasks(state.currentWorkflowId), 1000);
    } catch (error) {
        console.error('Failed to retry task:', error);
        showNotification('Failed to retry task', 'error');
    }
}

async function cancelTask(taskId) {
    try {
        await apiRequest(`/tasks/${taskId}/cancel`, { method: 'POST' });
        showNotification('Task cancelled');
        setTimeout(() => loadTasks(state.currentWorkflowId), 1000);
    } catch (error) {
        console.error('Failed to cancel task:', error);
        showNotification('Failed to cancel task', 'error');
    }
}

// ============== Auto Refresh ==============

function startTasksAutoRefresh() {
    stopTasksAutoRefresh();
    state.tasksAutoRefresh = setInterval(() => {
        if (state.currentWorkflowId) {
            loadTasks(state.currentWorkflowId);
        }
    }, 3000);
}

function stopTasksAutoRefresh() {
    if (state.tasksAutoRefresh) {
        clearInterval(state.tasksAutoRefresh);
        state.tasksAutoRefresh = null;
    }
}

// ============== Workflow Modal ==============

function openWorkflowModal(workflowId = null) {
    const modal = document.getElementById('workflowModal');
    const title = document.getElementById('modalTitle');
    const form = document.getElementById('workflowForm');
    
    form.reset();
    state.editingWorkflowId = workflowId;
    
    if (workflowId) {
        title.textContent = 'Edit Workflow';
        const workflow = state.workflows.find(w => w.id === workflowId);
        if (workflow) {
            document.getElementById('workflowName').value = workflow.name;
            document.getElementById('workflowDescription').value = workflow.description || '';
            document.getElementById('workflowYaml').value = workflow.yaml_content;
            document.getElementById('workflowEnabled').checked = workflow.enabled;
        }
    } else {
        title.textContent = 'New Workflow';
        document.getElementById('workflowYaml').value = getDefaultWorkflowYAML();
    }
    
    modal.classList.add('active');
}

function editWorkflow(workflowId) {
    openWorkflowModal(workflowId);
}

async function handleWorkflowSubmit(e) {
    e.preventDefault();
    
    const name = document.getElementById('workflowName').value;
    const description = document.getElementById('workflowDescription').value;
    const yamlContent = document.getElementById('workflowYaml').value;
    const enabled = document.getElementById('workflowEnabled').checked;
    
    const data = {
        name,
        description,
        yaml_content: yamlContent,
        enabled
    };
    
    try {
        if (state.editingWorkflowId) {
            await apiRequest(`/workflows/${state.editingWorkflowId}`, {
                method: 'PUT',
                body: JSON.stringify(data)
            });
            showNotification('Workflow updated');
        } else {
            const workflow = await apiRequest('/workflows', {
                method: 'POST',
                body: JSON.stringify(data)
            });
            showNotification('Workflow created');
            state.editingWorkflowId = workflow.id;
        }
        
        closeModal('workflowModal');
        await loadWorkflows();
        
        if (state.editingWorkflowId) {
            selectWorkflow(state.editingWorkflowId);
        }
    } catch (error) {
        console.error('Failed to save workflow:', error);
        showNotification(error.message || 'Failed to save workflow', 'error');
    }
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    modal.classList.remove('active');
    
    if (modalId === 'logModal') {
        // Close WebSocket if open
        if (state.logWebSocket) {
            state.logWebSocket.close();
            state.logWebSocket = null;
        }
        state.currentTaskId = null;
    }
}

// ============== Utility Functions ==============

async function apiRequest(url, options = {}) {
    const defaultOptions = {
        headers: {
            'Content-Type': 'application/json'
        }
    };
    
    const response = await fetch(API_BASE + url, { ...defaultOptions, ...options });
    
    if (!response.ok) {
        const error = await response.json().catch(() => ({ error: 'Request failed' }));
        throw new Error(error.error || `HTTP ${response.status}`);
    }
    
    return response.json();
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString();
}

function showNotification(message, type = 'info') {
    // Simple alert for now, can be enhanced with toast notifications
    if (type === 'error') {
        alert('Error: ' + message);
    } else {
        console.log(message);
    }
}

function getDefaultWorkflowYAML() {
    return `name: my-workflow
description: Example workflow
on:
  paths:
    - ./input
convert:
  from: jpg
  to: png
steps:
  - name: convert-image
    run: magick "$\{\{ input_path \}\}" "$\{\{ output_path \}\}"
options:
  concurrency: 2
  include_subdirs: true
  file_glob: "*.jpg"
  skip_on_nochange: true`;
}

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    stopTasksAutoRefresh();
    stopLogAutoRefresh();
    stopMonitoringAutoRefresh();
});

// ============== Tab Switching ==============

function switchTab(tabName) {
    state.currentTab = tabName;
    
    // Update nav tabs
    document.querySelectorAll('.nav-tab').forEach(tab => {
        tab.classList.remove('active');
        if (tab.dataset.tab === tabName) {
            tab.classList.add('active');
        }
    });
    
    // Update tab content
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.remove('active');
    });
    
    if (tabName === 'workflows') {
        document.getElementById('workflowsTab').classList.add('active');
        document.querySelector('.sidebar').style.display = 'flex';
        stopMonitoringAutoRefresh();
        startTasksAutoRefresh();
    } else if (tabName === 'monitoring') {
        document.getElementById('monitoringTab').classList.add('active');
        document.querySelector('.sidebar').style.display = 'none';
        stopTasksAutoRefresh();
        loadMonitoringData();
        startMonitoringAutoRefresh();
    }
}

// ============== Monitoring Functions ==============

async function loadMonitoringData() {
    try {
        // Load executor pool stats
        const stats = await apiRequest('/scheduler/stats');
        updateMonitoringStats(stats);
        
        // Load executor status
        const executors = await apiRequest('/scheduler/executors');
        renderExecutorCards(executors);
    } catch (error) {
        console.error('Failed to load monitoring data:', error);
    }
}

function updateMonitoringStats(stats) {
    document.getElementById('statTotalExecutors').textContent = stats.total || 0;
    document.getElementById('statAvailableExecutors').textContent = stats.available || 0;
    document.getElementById('statBusyExecutors').textContent = stats.busy || 0;
}

function renderExecutorCards(executors) {
    const container = document.getElementById('executorCards');
    
    if (!executors || executors.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">⚙️</div>
                <h3>No executors</h3>
                <p>Executor pool is not initialized</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = executors.map(executor => {
        const isBusy = executor.busy;
        const statusClass = isBusy ? 'busy' : 'idle';
        const statusText = isBusy ? 'Busy' : 'Idle';
        
        return `
            <div class="executor-card ${statusClass}">
                <div class="executor-header">
                    <div class="executor-id">Executor #${executor.id}</div>
                    <div class="executor-status-badge ${statusClass}">${statusText}</div>
                </div>
                <div class="executor-task-info">
                    ${isBusy && executor.current_task ? `
                        <div><strong>Status:</strong> Executing task</div>
                        <div class="executor-task-id">${executor.current_task}</div>
                    ` : `
                        <div><strong>Status:</strong> Waiting for tasks</div>
                    `}
                </div>
            </div>
        `;
    }).join('');
}

function startMonitoringAutoRefresh() {
    stopMonitoringAutoRefresh();
    state.monitoringAutoRefresh = setInterval(() => {
        if (state.currentTab === 'monitoring') {
            loadMonitoringData();
        }
    }, 2000);
}

function stopMonitoringAutoRefresh() {
    if (state.monitoringAutoRefresh) {
        clearInterval(state.monitoringAutoRefresh);
        state.monitoringAutoRefresh = null;
    }
}

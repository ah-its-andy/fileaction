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
    currentTab: 'workflows', // 'workflows', 'plugins', or 'monitoring'
    monitoringAutoRefresh: null,
    plugins: [],
    currentPluginId: null,
    editingPluginId: null,
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

// ============== Workflow Slide Panel ==============

async function openWorkflowModal(workflowId = null) {
    const panel = document.getElementById('workflowPanel');
    const title = document.getElementById('workflowPanelTitle');
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
    
    // Load plugins for inserter
    await loadPluginsForInserter();
    
    panel.classList.add('active');
}

function closeWorkflowPanel() {
    const panel = document.getElementById('workflowPanel');
    panel.classList.remove('active');
}

async function loadPluginsForInserter() {
    try {
        const plugins = await apiRequest('/plugins');
        renderPluginInserter(plugins);
    } catch (error) {
        console.error('Failed to load plugins for inserter:', error);
        document.getElementById('pluginInserterList').innerHTML = `
            <div class="plugin-inserter-empty">
                <div class="plugin-inserter-empty-icon">⚠️</div>
                <p>Failed to load plugins</p>
            </div>
        `;
    }
}

function renderPluginInserter(plugins) {
    const container = document.getElementById('pluginInserterList');
    
    if (plugins.length === 0) {
        container.innerHTML = `
            <div class="plugin-inserter-empty">
                <div class="plugin-inserter-empty-icon">🧩</div>
                <p>No plugins available</p>
                <small>Create plugins in the Plugins tab</small>
            </div>
        `;
        return;
    }
    
    container.innerHTML = plugins.map(plugin => `
        <div class="plugin-inserter-item" onclick="insertPluginIntoWorkflow('${escapeHtml(plugin.name)}', '${escapeHtml(plugin.current_version || '')}')">
            <div class="plugin-inserter-item-header">
                <span class="plugin-inserter-icon">🧩</span>
                <span class="plugin-inserter-name">${escapeHtml(plugin.name)}</span>
                <span class="plugin-inserter-version">v${escapeHtml(plugin.current_version || '0.0.0')}</span>
            </div>
            <div class="plugin-inserter-description">${escapeHtml(plugin.description || 'No description')}</div>
        </div>
    `).join('');
}

function filterPluginInserter() {
    const searchTerm = document.getElementById('pluginInserterSearch').value.toLowerCase();
    const items = document.querySelectorAll('.plugin-inserter-item');
    
    items.forEach(item => {
        const text = item.textContent.toLowerCase();
        if (text.includes(searchTerm)) {
            item.style.display = 'block';
        } else {
            item.style.display = 'none';
        }
    });
}

function insertPluginIntoWorkflow(pluginName, version) {
    const yamlTextarea = document.getElementById('workflowYaml');
    const currentYaml = yamlTextarea.value;
    
    // Generate plugin step YAML
    const pluginStep = `
  - name: ${pluginName}
    uses: ${pluginName}@v${version}
    with:
      # Configure plugin inputs here
      input1: value1`;
    
    // Try to insert after "steps:" if it exists
    if (currentYaml.includes('steps:')) {
        // Find the position after "steps:"
        const stepsIndex = currentYaml.indexOf('steps:');
        const afterStepsIndex = currentYaml.indexOf('\n', stepsIndex) + 1;
        
        const before = currentYaml.substring(0, afterStepsIndex);
        const after = currentYaml.substring(afterStepsIndex);
        
        yamlTextarea.value = before + pluginStep + '\n' + after;
    } else {
        // Append at the end
        yamlTextarea.value = currentYaml + '\n\nsteps:' + pluginStep;
    }
    
    // Scroll to the inserted position
    yamlTextarea.focus();
    yamlTextarea.scrollTop = yamlTextarea.scrollHeight;
    
    showNotification(`Plugin "${pluginName}" inserted`, 'success');
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
        
        closeWorkflowPanel();
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
    } else if (tabName === 'plugins') {
        document.getElementById('pluginsTab').classList.add('active');
        document.querySelector('.sidebar').style.display = 'none';
        stopMonitoringAutoRefresh();
        stopTasksAutoRefresh();
        loadPlugins();
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
                        ${executor.current_workflow ? `<div><strong>Workflow:</strong> ${escapeHtml(executor.current_workflow)}</div>` : ''}
                        ${executor.current_file ? `<div><strong>File:</strong> ${escapeHtml(executor.current_file)}</div>` : ''}
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

// ============== Plugin Management ==============

async function loadPlugins() {
    try {
        const plugins = await apiRequest('/plugins');
        state.plugins = plugins;
        renderPluginsList();
    } catch (error) {
        console.error('Failed to load plugins:', error);
        showNotification('Failed to load plugins', 'error');
    }
}

function renderPluginsList() {
    const container = document.getElementById('pluginsList');
    
    if (state.plugins.length === 0) {
        container.innerHTML = `
            <div class="empty-state" style="grid-column: 1 / -1;">
                <div class="empty-state-icon">🧩</div>
                <p>No plugins yet</p>
                <small>Add your first plugin to extend workflow capabilities</small>
            </div>
        `;
        return;
    }
    
    container.innerHTML = state.plugins.map(plugin => `
        <div class="plugin-card" onclick="viewPlugin('${plugin.id}')">
            <div class="plugin-card-header">
                <div class="plugin-icon">🧩</div>
                <div class="plugin-badge">${escapeHtml(plugin.source)}</div>
            </div>
            <div class="plugin-card-body">
                <h3 class="plugin-card-title">${escapeHtml(plugin.name)}</h3>
                <p class="plugin-card-description">${escapeHtml(plugin.description || 'No description')}</p>
                <div class="plugin-card-meta">
                    <span class="plugin-version">v${escapeHtml(plugin.current_version || '0.0.0')}</span>
                    <span class="plugin-date">${formatDate(plugin.updated_at)}</span>
                </div>
            </div>
            <div class="plugin-card-actions">
                <button class="btn-icon" onclick="event.stopPropagation(); editPlugin('${plugin.id}')" title="Edit">
                    ✏️
                </button>
                <button class="btn-icon" onclick="event.stopPropagation(); deletePlugin('${plugin.id}')" title="Delete">
                    🗑️
                </button>
            </div>
        </div>
    `).join('');
}

function filterPlugins() {
    const searchTerm = document.getElementById('pluginSearchInput').value.toLowerCase();
    const sourceFilter = document.getElementById('pluginSourceFilter').value;
    
    const filtered = state.plugins.filter(plugin => {
        const matchesSearch = !searchTerm || 
            plugin.name.toLowerCase().includes(searchTerm) ||
            (plugin.description || '').toLowerCase().includes(searchTerm);
        
        const matchesSource = !sourceFilter || plugin.source === sourceFilter;
        
        return matchesSearch && matchesSource;
    });
    
    const container = document.getElementById('pluginsList');
    if (filtered.length === 0) {
        container.innerHTML = `
            <div class="empty-state" style="grid-column: 1 / -1;">
                <div class="empty-state-icon">🔍</div>
                <p>No plugins found</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = filtered.map(plugin => `
        <div class="plugin-card" onclick="viewPlugin('${plugin.id}')">
            <div class="plugin-card-header">
                <div class="plugin-icon">🧩</div>
                <div class="plugin-badge">${escapeHtml(plugin.source)}</div>
            </div>
            <div class="plugin-card-body">
                <h3 class="plugin-card-title">${escapeHtml(plugin.name)}</h3>
                <p class="plugin-card-description">${escapeHtml(plugin.description || 'No description')}</p>
                <div class="plugin-card-meta">
                    <span class="plugin-version">v${escapeHtml(plugin.current_version || '0.0.0')}</span>
                    <span class="plugin-date">${formatDate(plugin.updated_at)}</span>
                </div>
            </div>
            <div class="plugin-card-actions">
                <button class="btn-icon" onclick="event.stopPropagation(); editPlugin('${plugin.id}')" title="Edit">
                    ✏️
                </button>
                <button class="btn-icon" onclick="event.stopPropagation(); deletePlugin('${plugin.id}')" title="Delete">
                    🗑️
                </button>
            </div>
        </div>
    `).join('');
}

function openPluginModal(pluginId = null) {
    state.editingPluginId = pluginId;
    const modal = document.getElementById('pluginModal');
    const form = document.getElementById('pluginForm');
    const title = document.getElementById('pluginModalTitle');
    
    if (pluginId) {
        title.textContent = 'Edit Plugin';
        // Load plugin data
        loadPluginForEdit(pluginId);
    } else {
        title.textContent = 'Add Plugin';
        form.reset();
        // Set default template
        const defaultYaml = `name: my-plugin
description: Plugin description
version: 1.0.0
dependencies: []
inputs:
  input1:
    type: string
    default: "value"
    required: true
    description: "Input description"
steps:
  - name: Step 1
    run: echo "Hello from plugin"
tags:
  - utility`;
        document.getElementById('pluginYaml').value = defaultYaml;
    }
    
    modal.classList.add('active');
}

async function loadPluginForEdit(pluginId) {
    try {
        const data = await apiRequest(`/plugins/${pluginId}`);
        const plugin = data.plugin;
        const versions = data.versions;
        
        // Get current version YAML
        const currentVersion = versions.find(v => v.id === plugin.current_version_id);
        
        document.getElementById('pluginName').value = plugin.name;
        document.getElementById('pluginName').disabled = true; // Can't change name
        document.getElementById('pluginDescription').value = plugin.description || '';
        document.getElementById('pluginYaml').value = currentVersion ? currentVersion.yaml_content : '';
    } catch (error) {
        console.error('Failed to load plugin:', error);
        showNotification('Failed to load plugin', 'error');
    }
}

async function handlePluginSubmit(e) {
    e.preventDefault();
    
    const form = e.target;
    const formData = {
        name: form.name.value.trim(),
        description: form.description.value.trim(),
        yaml_content: form.yaml_content.value.trim(),
    };
    
    try {
        if (state.editingPluginId) {
            // Update existing plugin (creates new version)
            await apiRequest(`/plugins/${state.editingPluginId}`, {
                method: 'PUT',
                body: JSON.stringify({
                    description: formData.description,
                    yaml_content: formData.yaml_content,
                }),
            });
            showNotification('Plugin updated successfully', 'success');
        } else {
            // Create new plugin
            await apiRequest('/plugins', {
                method: 'POST',
                body: JSON.stringify(formData),
            });
            showNotification('Plugin created successfully', 'success');
        }
        
        closeModal('pluginModal');
        await loadPlugins();
    } catch (error) {
        console.error('Failed to save plugin:', error);
        showNotification(error.message || 'Failed to save plugin', 'error');
    }
}

async function editPlugin(pluginId) {
    openPluginModal(pluginId);
}

async function deletePlugin(pluginId) {
    const plugin = state.plugins.find(p => p.id === pluginId);
    if (!plugin) return;
    
    if (!confirm(`Are you sure you want to delete plugin "${plugin.name}"?\n\nThis will remove all versions and cannot be undone.`)) {
        return;
    }
    
    try {
        await apiRequest(`/plugins/${pluginId}`, {
            method: 'DELETE',
        });
        showNotification('Plugin deleted successfully', 'success');
        await loadPlugins();
    } catch (error) {
        console.error('Failed to delete plugin:', error);
        showNotification(error.message || 'Failed to delete plugin', 'error');
    }
}

async function viewPlugin(pluginId) {
    try {
        const data = await apiRequest(`/plugins/${pluginId}`);
        const plugin = data.plugin;
        const versions = data.versions;
        
        const modal = document.getElementById('pluginDetailModal');
        const title = document.getElementById('pluginDetailTitle');
        const content = document.getElementById('pluginDetailContent');
        
        title.textContent = plugin.name;
        
        content.innerHTML = `
            <div class="plugin-detail">
                <div class="plugin-detail-header">
                    <div>
                        <h4>${escapeHtml(plugin.name)}</h4>
                        <p>${escapeHtml(plugin.description || 'No description')}</p>
                    </div>
                    <div class="plugin-detail-meta">
                        <div><strong>Source:</strong> ${escapeHtml(plugin.source)}</div>
                        <div><strong>Current Version:</strong> v${escapeHtml(plugin.current_version || '0.0.0')}</div>
                        <div><strong>Last Updated:</strong> ${formatDate(plugin.updated_at)}</div>
                    </div>
                </div>
                
                <div class="plugin-versions">
                    <h5>Version History</h5>
                    <div class="versions-list">
                        ${versions.map(v => `
                            <div class="version-item ${v.id === plugin.current_version_id ? 'active' : ''}">
                                <div class="version-info">
                                    <span class="version-number">v${escapeHtml(v.version)}</span>
                                    ${v.id === plugin.current_version_id ? '<span class="version-badge">Current</span>' : ''}
                                    <span class="version-date">${formatDate(v.created_at)}</span>
                                </div>
                                ${v.id !== plugin.current_version_id ? `
                                    <button class="btn btn-sm" onclick="activatePluginVersion('${plugin.id}', '${v.id}')">
                                        Activate
                                    </button>
                                ` : ''}
                            </div>
                        `).join('')}
                    </div>
                </div>
                
                <div class="form-actions">
                    <button class="btn btn-secondary" onclick="closeModal('pluginDetailModal')">Close</button>
                    <button class="btn btn-primary" onclick="closeModal('pluginDetailModal'); editPlugin('${plugin.id}')">Edit</button>
                </div>
            </div>
        `;
        
        modal.classList.add('active');
    } catch (error) {
        console.error('Failed to load plugin details:', error);
        showNotification('Failed to load plugin details', 'error');
    }
}

async function activatePluginVersion(pluginId, versionId) {
    if (!confirm('Are you sure you want to activate this version?\n\nThis will make it the current version for all new workflow executions.')) {
        return;
    }
    
    try {
        await apiRequest(`/plugins/${pluginId}/versions/${versionId}/activate`, {
            method: 'PUT',
        });
        showNotification('Version activated successfully', 'success');
        closeModal('pluginDetailModal');
        await loadPlugins();
    } catch (error) {
        console.error('Failed to activate version:', error);
        showNotification(error.message || 'Failed to activate version', 'error');
    }
}

// Setup plugin form submission
document.addEventListener('DOMContentLoaded', () => {
    const pluginForm = document.getElementById('pluginForm');
    if (pluginForm) {
        pluginForm.addEventListener('submit', handlePluginSubmit);
    }
});

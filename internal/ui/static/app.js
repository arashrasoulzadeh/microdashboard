// microdashboard UI Application

const API_BASE = '/api';
const WIDGET_TYPES = [
    { value: 'gauge', label: 'Gauge', icon: '📊' },
    { value: 'sparkline', label: 'Sparkline', icon: '📈' },
    { value: 'status', label: 'Status Indicator', icon: '🔴' },
    { value: 'numeric', label: 'Numeric', icon: '🔢' },
    { value: 'progress', label: 'Progress Bar', icon: '📏' },
    { value: 'text', label: 'Text Label', icon: '📝' }
];

// --- Section Navigation ---
function showSection(name) {
    document.querySelectorAll('.section').forEach(s => s.classList.remove('active'));
    document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('section-' + name).classList.add('active');
    document.querySelector('.nav-btn[onclick*="' + name + '"]').classList.add('active');

    // Load data for section
    if (name === 'dashboards') loadDashboards();
    else if (name === 'devices') { loadDevices(); loadDashboardsForSelect(); }
    else if (name === 'monitors') loadMonitors();
}

// --- Dashboard Management ---
async function loadDashboards() {
    try {
        const res = await fetch(API_BASE + '/dashboards');
        const data = await res.json();
        renderDashboardTable(data);
    } catch (e) {
        console.error('Failed to load dashboards:', e);
    }
}

function renderDashboardTable(dashboards) {
    const tbody = document.querySelector('#dashboards-table tbody');
    if (!dashboards.length) {
        tbody.innerHTML = '<tr><td colspan="7" class="empty-state"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M9 9h6v6H9z"/></svg>No dashboards yet. Click "New Dashboard" to create one.</td></tr>';
        return;
    }
    tbody.innerHTML = dashboards.map(d => `
        <tr>
            <td><code>${escapeHtml(d.id)}</code></td>
            <td>${escapeHtml(d.name)}</td>
            <td>${d.width}×${d.height}</td>
            <td>${d.widget_count}</td>
            <td>${(d.refresh_interval / 1000).toFixed(1)}s</td>
            <td class="timestamp">${formatTimestamp(d.created_at)}</td>
            <td>
                <button class="btn btn-sm btn-secondary" onclick="openDashboardModal('${escapeHtml(d.id)}')">Edit</button>
                <button class="btn btn-sm btn-danger" onclick="deleteDashboard('${escapeHtml(d.id)}')">Delete</button>
            </td>
        </tr>
    `).join('');
}

function openDashboardModal(id = null) {
    const modal = document.getElementById('dashboard-modal');
    const form = document.getElementById('dashboard-form');
    form.reset();
    document.getElementById('widgets-container').innerHTML = '';
    document.getElementById('dash-result').innerHTML = '';

    if (id) {
        document.getElementById('dash-modal-title').textContent = 'Edit Dashboard';
        fetch(API_BASE + '/dashboards/' + encodeURIComponent(id))
            .then(r => r.json())
            .then(d => {
                document.getElementById('dash-id').value = d.id;
                document.getElementById('dash-id-input').value = d.id;
                document.getElementById('dash-id-input').readOnly = true;
                document.getElementById('dash-name').value = d.name;
                document.getElementById('dash-width').value = d.width;
                document.getElementById('dash-height').value = d.height;
                document.getElementById('dash-refresh').value = d.refresh_interval;

                // Parse widgets from json_definition
                try {
                    const widgets = JSON.parse(d.json_definition);
                    widgets.forEach(w => addWidgetRow(w));
                } catch (e) {
                    addWidgetRow();
                }
            });
    } else {
        document.getElementById('dash-modal-title').textContent = 'New Dashboard';
        document.getElementById('dash-id-input').readOnly = false;
        addWidgetRow();
    }
    modal.hidden = false;
}

function closeDashboardModal() {
    document.getElementById('dashboard-modal').hidden = true;
}

function addWidgetRow(widget = null) {
    const container = document.getElementById('widgets-container');
    const div = document.createElement('div');
    div.className = 'widget-row';
    div.innerHTML = `
        <input type="text" name="widgets[${Date.now()}][id]" placeholder="Widget ID" class="input" value="${widget?.id || ''}" required>
        <select name="widgets[${Date.now()}][type]" class="input" required>
            ${WIDGET_TYPES.map(t => `<option value="${t.value}" ${widget?.type === t.value ? 'selected' : ''}>${t.icon} ${t.label}</option>`).join('')}
        </select>
        <input type="number" name="widgets[${Date.now()}][x]" placeholder="X" class="input" value="${widget?.x || 0}" min="0">
        <input type="number" name="widgets[${Date.now()}][y]" placeholder="Y" class="input" value="${widget?.y || 0}" min="0">
        <input type="number" name="widgets[${Date.now()}][width]" placeholder="W" class="input" value="${widget?.width || 32}" min="8">
        <input type="number" name="widgets[${Date.now()}][height]" placeholder="H" class="input" value="${widget?.height || 16}" min="8">
        <input type="text" name="widgets[${Date.now()}][expression]" placeholder="${{latency:monitor1}}" class="input" value="${widget?.expression || ''}" required>
        <button type="button" class="btn btn-sm btn-danger" onclick="this.closest('.widget-row').remove()">✕</button>
    `;
    container.appendChild(div);
}

async function deleteDashboard(id) {
    if (!confirm('Delete dashboard "' + id + '"?')) return;
    try {
        await fetch(API_BASE + '/dashboards/' + encodeURIComponent(id), { method: 'DELETE' });
        loadDashboards();
    } catch (e) {
        alert('Failed to delete: ' + e.message);
    }
}

// --- Device Management ---
async function loadDevices() {
    try {
        const res = await fetch(API_BASE + '/devices');
        const data = await res.json();
        renderDeviceTable(data);
    } catch (e) {
        console.error('Failed to load devices:', e);
    }
}

function renderDeviceTable(devices) {
    const tbody = document.querySelector('#devices-table tbody');
    if (!devices.length) {
        tbody.innerHTML = '<tr><td colspan="4" class="empty-state"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 17h8"/></svg>No devices registered.</td></tr>';
        return;
    }
    tbody.innerHTML = devices.map(d => `
        <tr>
            <td><code>${escapeHtml(d.device_id)}</code></td>
            <td>${d.dashboard_id ? '<code>' + escapeHtml(d.dashboard_id) + '</code>' : '<span class="muted">Unassigned</span>'}</td>
            <td class="timestamp">${formatTimestamp(d.created_at)}</td>
            <td>
                <select class="input" onchange="assignDashboard('${escapeHtml(d.device_id)}', this.value)" style="width: auto;">
                    <option value="">-- Assign --</option>
                </select>
                <button class="btn btn-sm btn-danger" onclick="deleteDevice('${escapeHtml(d.device_id)}')">Delete</button>
            </td>
        </tr>
    `).join('');
    // Populate dashboard options
    fetch(API_BASE + '/dashboards').then(r => r.json()).then(dashboards => {
        document.querySelectorAll('#devices-table select').forEach(sel => {
            const current = sel.value;
            sel.innerHTML = '<option value="">-- Assign --</option>' + dashboards.map(d => `<option value="${escapeHtml(d.id)}" ${d.id === current ? 'selected' : ''}>${escapeHtml(d.name)} (${d.id})</option>`).join('');
        });
    });
}

async function loadDashboardsForSelect() {
    try {
        const res = await fetch(API_BASE + '/dashboards');
        const dashboards = await res.json();
        const select = document.getElementById('device-dashboard-select');
        if (select) {
            select.innerHTML = '<option value="">-- None --</option>' + dashboards.map(d => `<option value="${escapeHtml(d.id)}">${escapeHtml(d.name)} (${d.id})</option>`).join('');
        }
    } catch (e) {}
}

function openDeviceModal() {
    loadDashboardsForSelect();
    document.getElementById('device-form').reset();
    document.getElementById('device-result').innerHTML = '';
    document.getElementById('device-modal').hidden = false;
}

function closeDeviceModal() {
    document.getElementById('device-modal').hidden = true;
}

async function assignDashboard(deviceId, dashboardId) {
    try {
        await fetch(API_BASE + '/devices/' + encodeURIComponent(deviceId) + '/dashboard', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ dashboard_id: dashboardId })
        });
        loadDevices();
    } catch (e) {
        alert('Failed to assign: ' + e.message);
    }
}

async function deleteDevice(id) {
    if (!confirm('Delete device "' + id + '"?')) return;
    try {
        await fetch(API_BASE + '/devices/' + encodeURIComponent(id), { method: 'DELETE' });
        loadDevices();
    } catch (e) {
        alert('Failed to delete: ' + e.message);
    }
}

// --- Monitor Management ---
async function loadMonitors() {
    try {
        const res = await fetch(API_BASE + '/monitors');
        const data = await res.json();
        renderMonitorTable(data);
    } catch (e) {
        console.error('Failed to load monitors:', e);
    }
}

function renderMonitorTable(monitors) {
    const tbody = document.querySelector('#monitors-table tbody');
    if (!monitors.length) {
        tbody.innerHTML = '<tr><td colspan="8" class="empty-state"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><circle cx="12" cy="12" r="10"/><line x1="12" y1="6" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>No monitors configured.</td></tr>';
        return;
    }
    tbody.innerHTML = monitors.map(m => {
        let statusClass = 'status-pending';
        if (m.last_status >= 200 && m.last_status < 300) statusClass = 'status-ok';
        else if (m.last_status >= 400 && m.last_status < 500) statusClass = 'status-warn';
        else if (m.last_status >= 500) statusClass = 'status-crit';
        return `
        <tr>
            <td><code>${escapeHtml(m.id)}</code></td>
            <td>${escapeHtml(m.url)}</td>
            <td>${m.method}</td>
            <td>${m.timeout}ms</td>
            <td>${m.last_elapsed_ms ? m.last_elapsed_ms + 'ms' : '<span class="muted">--</span>'}</td>
            <td><span class="status-badge ${statusClass}">${statusLabel(m.last_status)}</span></td>
            <td class="timestamp">${m.last_checked ? formatTimestamp(m.last_checked) : '<span class="muted">Never</span>'}</td>
            <td>
                <button class="btn btn-sm btn-danger" onclick="deleteMonitor('${escapeHtml(m.id)}')">Delete</button>
            </td>
        </tr>
        `;
    }).join('');
}

function statusLabel(status) {
    if (!status) return 'Pending';
    if (status >= 200 && status < 300) return 'OK';
    if (status >= 300 && status < 400) return 'Redirect';
    if (status >= 400 && status < 500) return 'Client Err';
    if (status >= 500) return 'Server Err';
    return 'Unknown';
}

function openMonitorModal() {
    document.getElementById('monitor-form').reset();
    document.getElementById('monitor-result').innerHTML = '';
    document.getElementById('monitor-modal').hidden = false;
}

function closeMonitorModal() {
    document.getElementById('monitor-modal').hidden = true;
}

async function deleteMonitor(id) {
    if (!confirm('Delete monitor "' + id + '"?')) return;
    try {
        await fetch(API_BASE + '/monitors/' + encodeURIComponent(id), { method: 'DELETE' });
        loadMonitors();
    } catch (e) {
        alert('Failed to delete: ' + e.message);
    }
}

// --- HTMX Response Handling ---
document.body.addEventListener('htmx:afterRequest', function(evt) {
    if (evt.detail.successful) {
        // Close modals on successful form submit
        const modal = evt.target.closest('.modal');
        if (modal) modal.hidden = true;
        // Reload relevant section
        const activeSection = document.querySelector('.section.active');
        if (activeSection) {
            if (activeSection.id === 'section-dashboards') loadDashboards();
            else if (activeSection.id === 'section-devices') loadDevices();
            else if (activeSection.id === 'section-monitors') loadMonitors();
        }
    }
});

// --- Utils ---
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatTimestamp(ts) {
    if (!ts) return '<span class="muted">--</span>';
    const date = new Date(ts * 1000);
    return date.toLocaleString();
}

// --- Init ---
document.addEventListener('DOMContentLoaded', () => {
    showSection('dashboards');
});

// Expose functions globally for inline onclick
window.showSection = showSection;
window.openDashboardModal = openDashboardModal;
window.closeDashboardModal = closeDashboardModal;
window.addWidgetRow = addWidgetRow;
window.deleteDashboard = deleteDashboard;
window.openDeviceModal = openDeviceModal;
window.closeDeviceModal = closeDeviceModal;
window.deleteDevice = deleteDevice;
window.assignDashboard = assignDashboard;
window.openMonitorModal = openMonitorModal;
window.closeMonitorModal = closeMonitorModal;
window.deleteMonitor = deleteMonitor;
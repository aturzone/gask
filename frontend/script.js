const API_URL = 'http://localhost:7887/api';
const AUTH = btoa('admin:securepass'); // Basic Auth header

async function fetchWithAuth(url, options = {}) {
  options.headers = {
    ...options.headers,
    'Authorization': `Basic ${AUTH}`,
    'Content-Type': 'application/json',
  };
  const res = await fetch(url, options);
  if (!res.ok) throw new Error('API error');
  return res.json();
}

async function loadTasks() {
  const tasks = await fetchWithAuth(`${API_URL}/tasks`);
  const list = document.getElementById('task-list');
  list.innerHTML = '';
  tasks.forEach(task => {
    const div = document.createElement('div');
    div.className = 'task-box';
    div.innerHTML = `
      <h3>${task.title}</h3>
      <p>${task.description}</p>
      <p>Due: ${task.due_date} (Shamsi: ${toShamsi(new Date(task.due_date))})</p>
    `;
    list.appendChild(div);
  });
}

function toShamsi(date) {
  // Simple JS Jalali conversion (implement or use library like moment-jalaali)
  // For now, placeholder; in production, add moment.js with jalaali
  return date.toISOString().split('T')[0]; // Replace with actual conversion
}

async function createTask() {
  const title = prompt('Task Title');
  const desc = prompt('Description');
  const due = prompt('Due Date (YYYY-MM-DD Shamsi)');
  // Convert Shamsi to Miladi here if needed
  await fetchWithAuth(`${API_URL}/tasks`, {
    method: 'POST',
    body: JSON.stringify({ title, description: desc, due_date: due }),
  });
  loadTasks();
}

// Similar functions for notes, boxes, loadNotes, loadBoxes...

// Initial load
loadTasks();
// loadNotes(); loadBoxes();

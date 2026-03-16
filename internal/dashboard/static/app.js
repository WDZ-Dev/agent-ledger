(function () {
  "use strict";

  const $ = (sel) => document.querySelector(sel);
  const fmt = (n) => (n >= 1 ? n.toFixed(2) : n.toFixed(4));

  // Tenant filtering
  let currentTenant = "";

  function tenantQS(prefix) {
    if (!currentTenant) return prefix;
    const sep = prefix.includes("?") ? "&" : "?";
    return prefix + sep + "tenant=" + encodeURIComponent(currentTenant);
  }

  // Admin token
  let adminToken = localStorage.getItem("agentledger_admin_token") || "";
  $("#admin-token").value = adminToken;

  $("#save-token").addEventListener("click", () => {
    adminToken = $("#admin-token").value.trim();
    localStorage.setItem("agentledger_admin_token", adminToken);
    loadRules();
  });

  async function adminFetch(url, opts = {}) {
    if (!adminToken) return null;
    opts.headers = { ...opts.headers, Authorization: "Bearer " + adminToken };
    const resp = await fetch(url, opts);
    if (!resp.ok) return null;
    return resp.json();
  }

  async function fetchJSON(url) {
    const resp = await fetch(url);
    if (!resp.ok) throw new Error(resp.statusText);
    return resp.json();
  }

  // Summary cards
  async function loadSummary() {
    try {
      const d = await fetchJSON(tenantQS("/api/dashboard/summary"));
      $("#today-spend").textContent = "$" + fmt(d.today_spend_usd);
      $("#month-spend").textContent = "$" + fmt(d.month_spend_usd);
      $("#today-requests").textContent = d.today_requests.toLocaleString();
      $("#active-sessions").textContent = d.active_sessions;
    } catch (e) {
      console.error("summary:", e);
    }
  }

  // Timeseries chart (simple canvas bar chart — no external deps)
  async function loadTimeseries() {
    const interval = $("#timeseries-interval").value;
    const hours = $("#timeseries-hours").value;
    try {
      const points = await fetchJSON(
        tenantQS(`/api/dashboard/timeseries?interval=${interval}&hours=${hours}`)
      );
      drawBarChart(
        "timeseries-chart",
        points || [],
        (p) => p.Timestamp.slice(5, 16),
        (p) => p.CostUSD
      );
    } catch (e) {
      console.error("timeseries:", e);
    }
  }

  function drawBarChart(canvasId, data, labelFn, valueFn) {
    const canvas = document.getElementById(canvasId);
    const ctx = canvas.getContext("2d");
    const dpr = window.devicePixelRatio || 1;
    const rect = canvas.parentElement.getBoundingClientRect();
    canvas.width = rect.width * dpr;
    canvas.height = rect.height * dpr;
    ctx.scale(dpr, dpr);

    const w = rect.width;
    const h = rect.height;
    const pad = { top: 20, right: 20, bottom: 40, left: 60 };
    const cw = w - pad.left - pad.right;
    const ch = h - pad.top - pad.bottom;

    ctx.clearRect(0, 0, w, h);

    if (!data.length) {
      ctx.fillStyle = "#8b949e";
      ctx.font = "14px sans-serif";
      ctx.textAlign = "center";
      ctx.fillText("No data", w / 2, h / 2);
      return;
    }

    const values = data.map(valueFn);
    const maxVal = Math.max(...values, 0.001);

    // Grid lines
    ctx.strokeStyle = "#21262d";
    ctx.lineWidth = 1;
    for (let i = 0; i <= 4; i++) {
      const y = pad.top + (ch / 4) * i;
      ctx.beginPath();
      ctx.moveTo(pad.left, y);
      ctx.lineTo(pad.left + cw, y);
      ctx.stroke();

      ctx.fillStyle = "#8b949e";
      ctx.font = "11px sans-serif";
      ctx.textAlign = "right";
      const label = "$" + fmt(maxVal * (1 - i / 4));
      ctx.fillText(label, pad.left - 8, y + 4);
    }

    // Bars
    const barW = Math.max(2, (cw / data.length) * 0.7);
    const gap = cw / data.length;
    ctx.fillStyle = "#58a6ff";
    for (let i = 0; i < data.length; i++) {
      const barH = (values[i] / maxVal) * ch;
      const x = pad.left + gap * i + (gap - barW) / 2;
      const y = pad.top + ch - barH;
      ctx.fillRect(x, y, barW, barH);
    }

    // X-axis labels (show subset to avoid overlap)
    ctx.fillStyle = "#8b949e";
    ctx.font = "10px sans-serif";
    ctx.textAlign = "center";
    const step = Math.max(1, Math.floor(data.length / 8));
    for (let i = 0; i < data.length; i += step) {
      const x = pad.left + gap * i + gap / 2;
      ctx.save();
      ctx.translate(x, pad.top + ch + 12);
      ctx.rotate(-0.5);
      ctx.fillText(labelFn(data[i]), 0, 0);
      ctx.restore();
    }
  }

  // Costs table
  async function loadCosts() {
    const groupBy = $("#costs-group").value;
    try {
      const entries = await fetchJSON(
        tenantQS(`/api/dashboard/costs?group_by=${groupBy}&hours=24`)
      );
      const tbody = $("#costs-body");
      tbody.innerHTML = "";
      if (!entries || !entries.length) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#8b949e">No data</td></tr>';
        return;
      }
      for (const e of entries) {
        let name = e.Model || e.Provider || e.AgentID || e.SessionID || "(unknown)";
        if (groupBy === "provider") name = e.Provider;
        if (groupBy === "agent") name = e.AgentID || "(none)";
        if (groupBy === "session") name = e.SessionID || "(none)";

        const tr = document.createElement("tr");
        tr.innerHTML = `
          <td>${esc(name)}</td>
          <td>${e.Requests}</td>
          <td>${e.InputTokens.toLocaleString()}</td>
          <td>${e.OutputTokens.toLocaleString()}</td>
          <td>$${fmt(e.TotalCostUSD)}</td>
        `;
        tbody.appendChild(tr);
      }
    } catch (e) {
      console.error("costs:", e);
    }
  }

  // Sessions table
  async function loadSessions() {
    try {
      const sessions = await fetchJSON("/api/dashboard/sessions");
      const tbody = $("#sessions-body");
      tbody.innerHTML = "";
      if (!sessions || !sessions.length) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#8b949e">No active sessions</td></tr>';
        return;
      }
      for (const s of sessions) {
        const tr = document.createElement("tr");
        const statusClass = "status-" + s.Status;
        const started = new Date(s.StartedAt).toLocaleTimeString();
        tr.innerHTML = `
          <td>${esc(s.ID.slice(0, 12))}...</td>
          <td>${esc(s.AgentID || "(none)")}</td>
          <td>${esc(s.UserID || "(none)")}</td>
          <td>${s.CallCount}</td>
          <td>$${fmt(s.TotalCostUSD)}</td>
          <td class="${statusClass}">${s.Status}</td>
          <td>${started}</td>
        `;
        tbody.appendChild(tr);
      }
    } catch (e) {
      console.error("sessions:", e);
    }
  }

  function esc(s) {
    const el = document.createElement("span");
    el.textContent = s;
    return el.innerHTML;
  }

  // API Keys table
  async function loadAPIKeys() {
    try {
      const keys = await fetchJSON("/api/admin/api-keys");
      const tbody = $("#apikeys-body");
      tbody.innerHTML = "";
      if (!keys || !keys.length) {
        tbody.innerHTML = '<tr><td colspan="3" style="text-align:center;color:#8b949e">No data</td></tr>';
        return;
      }
      for (const k of keys) {
        const tr = document.createElement("tr");
        tr.innerHTML = `<td>${esc(k.api_key_hash)}</td><td>${k.requests}</td><td>$${fmt(k.total_cost_usd)}</td>`;
        tbody.appendChild(tr);
      }
    } catch (e) { /* admin API may not be enabled */ }
  }

  // Budget Rules table
  async function loadRules() {
    try {
      const rules = await adminFetch("/api/admin/budgets/rules");
      const tbody = $("#rules-body");
      tbody.innerHTML = "";
      if (!rules || !rules.length) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#8b949e">No rules configured</td></tr>';
        return;
      }
      for (const r of rules) {
        const tr = document.createElement("tr");
        tr.innerHTML = `
          <td>${esc(r.api_key_pattern || r.APIKeyPattern || "")}</td>
          <td>${r.daily_limit_usd || r.DailyLimitUSD || 0}</td>
          <td>${r.monthly_limit_usd || r.MonthlyLimitUSD || 0}</td>
          <td>${r.action || r.Action || ""}</td>
          <td><button class="btn-delete" data-pattern="${esc(r.api_key_pattern || r.APIKeyPattern || "")}">&#215;</button></td>
        `;
        tbody.appendChild(tr);
      }
      // Wire delete buttons
      for (const btn of tbody.querySelectorAll(".btn-delete")) {
        btn.addEventListener("click", async () => {
          await adminFetch("/api/admin/budgets/rules?pattern=" + encodeURIComponent(btn.dataset.pattern), { method: "DELETE" });
          loadRules();
        });
      }
    } catch (e) { /* admin API may not be enabled */ }
  }

  // Add Rule button
  $("#add-rule-btn").addEventListener("click", async () => {
    const rule = {
      api_key_pattern: $("#rule-pattern").value,
      daily_limit_usd: parseFloat($("#rule-daily").value) || 0,
      monthly_limit_usd: parseFloat($("#rule-monthly").value) || 0,
      action: $("#rule-action").value,
    };
    await adminFetch("/api/admin/budgets/rules", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rule),
    });
    $("#rule-pattern").value = "";
    $("#rule-daily").value = "";
    $("#rule-monthly").value = "";
    loadRules();
  });

  // Tenant filter Apply button
  $("#apply-tenant").addEventListener("click", () => {
    currentTenant = $("#tenant-filter").value.trim();
    loadAll();
  });

  function loadAll() {
    loadSummary();
    loadTimeseries();
    loadCosts();
    loadSessions();
    loadAPIKeys();
    loadRules();
  }

  // Event listeners for controls
  $("#timeseries-interval").addEventListener("change", loadTimeseries);
  $("#timeseries-hours").addEventListener("change", loadTimeseries);
  $("#costs-group").addEventListener("change", loadCosts);

  // Initial load + auto-refresh every 30s
  loadAll();
  setInterval(loadAll, 30000);
})();

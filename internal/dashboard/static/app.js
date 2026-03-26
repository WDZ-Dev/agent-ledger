(function () {
  "use strict";

  Chart.defaults.color = "#e1e4e8";

  const $ = (sel) => document.querySelector(sel);

  function fmtCost(n) {
    if (n >= 1000) return "$" + (n / 1000).toFixed(1) + "k";
    if (n >= 100) return "$" + n.toFixed(0);
    if (n >= 1) return "$" + n.toFixed(2);
    if (n >= 0.01) return "$" + n.toFixed(3);
    return "$" + n.toFixed(4);
  }

  function fmtAxis(n) {
    if (n >= 1000) return "$" + (n / 1000).toFixed(1) + "k";
    if (n >= 100) return "$" + n.toFixed(0);
    if (n >= 10) return "$" + n.toFixed(1);
    return "$" + n.toFixed(2);
  }

  function fmtPct(n) {
    if (n === 0) return "0%";
    if (n < 0.001) return "<0.1%";
    return (n * 100).toFixed(1) + "%";
  }

  function fmtDuration(startedAt, endedAt) {
    var start = new Date(startedAt).getTime();
    var end = endedAt ? new Date(endedAt).getTime() : Date.now();
    var ms = end - start;
    if (ms < 0) return "--";
    var secs = Math.floor(ms / 1000);
    if (secs < 60) return secs + "s";
    var mins = Math.floor(secs / 60);
    secs = secs % 60;
    if (mins < 60) return mins + "m " + secs + "s";
    var hrs = Math.floor(mins / 60);
    mins = mins % 60;
    return hrs + "h " + mins + "m";
  }

  function fmtTokenCount(n) {
    if (n >= 1000000) return (n / 1000000).toFixed(1) + "M";
    if (n >= 1000) return (n / 1000).toFixed(1) + "k";
    return String(n);
  }

  const DONUT_COLORS = [
    "#388bfd", "#3fb950", "#d29922", "#f85149", "#a371f7",
    "#79c0ff", "#56d364", "#e3b341", "#ff7b72", "#bc8cff",
    "#7ee787", "#ffa657", "#ff9bce", "#8b949e",
  ];

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
    loadBudgetStatus();
    loadBlocked();
  });

  async function adminFetch(url, opts = {}) {
    if (!adminToken) return null;
    opts.headers = {
      ...opts.headers,
      Authorization: "Bearer " + adminToken,
      "X-Requested-With": "XMLHttpRequest",
    };
    const resp = await fetch(url, opts);
    if (!resp.ok) return null;
    return resp.json();
  }

  async function fetchJSON(url) {
    const resp = await fetch(url);
    if (!resp.ok) throw new Error(resp.statusText);
    return resp.json();
  }

  // ── Summary cards ──
  async function loadSummary() {
    try {
      const d = await fetchJSON(tenantQS("/api/dashboard/summary"));
      $("#today-spend").textContent = fmtCost(d.today_spend_usd);
      $("#month-spend").textContent = fmtCost(d.month_spend_usd);
      $("#today-requests").textContent = d.today_requests.toLocaleString();
      $("#active-sessions").textContent = d.active_sessions;
    } catch (e) {
      console.error("summary:", e);
    }
  }

  // ── Error stats + avg cost cards ──
  async function loadStats() {
    try {
      const s = await fetchJSON(tenantQS("/api/dashboard/stats?hours=24"));
      $("#error-rate").textContent = fmtPct(s.error_rate);
      if (s.error_rate > 0.05) {
        $("#error-rate").classList.add("card-value-error");
      } else {
        $("#error-rate").classList.remove("card-value-error");
      }
      $("#avg-cost").textContent = fmtCost(s.avg_cost_per_request);

      // Error breakdown panel
      $("#stat-total").textContent = s.total_requests.toLocaleString();
      $("#stat-errors").textContent = s.error_requests.toLocaleString();
      $("#stat-429").textContent = s.count_429.toLocaleString();
      $("#stat-5xx").textContent = s.count_5xx.toLocaleString();
      $("#stat-latency").textContent = s.avg_duration_ms.toFixed(0) + "ms";
      $("#stat-avg-cost").textContent = fmtCost(s.avg_cost_per_request);
    } catch (e) {
      console.error("stats:", e);
    }
  }

  // ── Timeseries chart ──
  let timeseriesChart = null;
  let currentTimeseriesTab = "cost";

  function formatLabel(ts, interval) {
    const d = new Date(ts);
    if (interval === "day") {
      return d.toLocaleDateString("en-US", { weekday: "short", month: "short", day: "numeric" });
    }
    if (interval === "minute") {
      return d.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" });
    }
    const now = new Date();
    if (d.toDateString() === now.toDateString()) {
      return d.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" });
    }
    return d.toLocaleDateString("en-US", { month: "short", day: "numeric" }) +
      " " + d.toLocaleTimeString("en-US", { hour: "numeric" });
  }

  async function loadTimeseries() {
    const hours = parseFloat($("#timeseries-hours").value);
    let interval = "hour";
    if (hours <= 6) interval = "minute";
    else if (hours > 24) interval = "day";

    try {
      const points = await fetchJSON(
        tenantQS(`/api/dashboard/timeseries?interval=${interval}&hours=${hours}`)
      );
      const data = points || [];
      const labels = data.map((p) => formatLabel(p.Timestamp, interval));
      const values = data.map((p) => p.CostUSD);
      const ctx = document.getElementById("timeseries-chart").getContext("2d");

      if (timeseriesChart) timeseriesChart.destroy();

      timeseriesChart = new Chart(ctx, {
        type: "line",
        data: {
          labels,
          datasets: [{
            label: "Cost (USD)",
            data: values,
            backgroundColor: "rgba(56, 139, 253, 0.12)",
            borderColor: "rgba(56, 139, 253, 1)",
            borderWidth: 2,
            fill: true,
            tension: 0.35,
            pointRadius: data.length > 60 ? 0 : 3,
            pointBackgroundColor: "rgba(56, 139, 253, 1)",
            pointBorderColor: "#161b22",
            pointBorderWidth: 2,
            pointHoverRadius: 6,
          }],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          interaction: { intersect: false, mode: "index" },
          plugins: {
            legend: { display: false },
            tooltip: {
              backgroundColor: "#1c2128", borderColor: "#30363d", borderWidth: 1,
              titleColor: "#e1e4e8", bodyColor: "#c9d1d9",
              titleFont: { size: 13 }, bodyFont: { size: 13 },
              padding: 12, cornerRadius: 8,
              callbacks: { label: (ctx) => " " + fmtCost(ctx.parsed.y) },
            },
          },
          scales: {
            x: {
              grid: { color: "rgba(33,38,45,0.5)", drawBorder: false },
              ticks: { color: "#8b949e", font: { size: 11 }, maxRotation: 0, autoSkip: true, maxTicksLimit: 10 },
            },
            y: {
              beginAtZero: true,
              grid: { color: "rgba(33,38,45,0.5)", drawBorder: false },
              ticks: { color: "#8b949e", font: { size: 11 }, maxTicksLimit: 6, callback: (v) => fmtAxis(v) },
            },
          },
        },
      });
    } catch (e) {
      console.error("timeseries:", e);
    }
  }

  // ── Token timeseries chart ──
  let tokenChart = null;

  async function loadTokenTimeseries() {
    const hours = parseFloat($("#timeseries-hours").value);
    let interval = "hour";
    if (hours <= 6) interval = "minute";
    else if (hours > 24) interval = "day";

    try {
      const points = await fetchJSON(
        tenantQS(`/api/dashboard/timeseries/tokens?interval=${interval}&hours=${hours}`)
      );
      const data = points || [];
      const labels = data.map((p) => formatLabel(p.Timestamp, interval));
      const ctx = document.getElementById("token-chart").getContext("2d");

      if (tokenChart) tokenChart.destroy();

      tokenChart = new Chart(ctx, {
        type: "line",
        data: {
          labels,
          datasets: [
            {
              label: "Input Tokens",
              data: data.map((p) => p.InputTokens),
              backgroundColor: "rgba(56, 139, 253, 0.15)",
              borderColor: "rgba(56, 139, 253, 1)",
              borderWidth: 2,
              fill: true,
              tension: 0.35,
              pointRadius: data.length > 60 ? 0 : 3,
            },
            {
              label: "Output Tokens",
              data: data.map((p) => p.OutputTokens),
              backgroundColor: "rgba(63, 185, 80, 0.15)",
              borderColor: "rgba(63, 185, 80, 1)",
              borderWidth: 2,
              fill: true,
              tension: 0.35,
              pointRadius: data.length > 60 ? 0 : 3,
            },
          ],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          interaction: { intersect: false, mode: "index" },
          plugins: {
            legend: { display: true, position: "top", labels: { color: "#8b949e", font: { size: 11 } } },
            tooltip: {
              backgroundColor: "#1c2128", borderColor: "#30363d", borderWidth: 1,
              titleColor: "#e1e4e8", bodyColor: "#c9d1d9",
              padding: 12, cornerRadius: 8,
              callbacks: { label: (ctx) => " " + ctx.dataset.label + ": " + fmtTokenCount(ctx.parsed.y) },
            },
          },
          scales: {
            x: {
              grid: { color: "rgba(33,38,45,0.5)", drawBorder: false },
              ticks: { color: "#8b949e", font: { size: 11 }, maxRotation: 0, autoSkip: true, maxTicksLimit: 10 },
            },
            y: {
              beginAtZero: true,
              stacked: true,
              grid: { color: "rgba(33,38,45,0.5)", drawBorder: false },
              ticks: { color: "#8b949e", font: { size: 11 }, maxTicksLimit: 6, callback: (v) => fmtTokenCount(v) },
            },
          },
        },
      });
    } catch (e) {
      console.error("token timeseries:", e);
    }
  }

  // Cost/Token tab switching
  document.querySelectorAll("[data-tab]").forEach((btn) => {
    btn.addEventListener("click", () => {
      document.querySelectorAll("[data-tab]").forEach((b) => b.classList.remove("tab-active"));
      btn.classList.add("tab-active");
      currentTimeseriesTab = btn.dataset.tab;
      if (currentTimeseriesTab === "cost") {
        document.getElementById("timeseries-chart").parentElement.style.display = "";
        document.getElementById("token-chart-container").style.display = "none";
        loadTimeseries();
      } else {
        document.getElementById("timeseries-chart").parentElement.style.display = "none";
        document.getElementById("token-chart-container").style.display = "";
        loadTokenTimeseries();
      }
    });
  });

  // ── Provider donut chart ──
  let providerChart = null;

  async function loadProviderChart() {
    try {
      const entries = await fetchJSON(tenantQS("/api/dashboard/costs?group_by=provider&hours=720"));
      if (!entries || !entries.length) return;

      const labels = entries.map((e) => e.Provider);
      const values = entries.map((e) => e.TotalCostUSD);
      const ctx = document.getElementById("provider-chart").getContext("2d");

      if (providerChart) providerChart.destroy();

      const capLabels = labels.map((l) => l.charAt(0).toUpperCase() + l.slice(1));

      providerChart = new Chart(ctx, {
        type: "doughnut",
        data: {
          labels: capLabels,
          datasets: [{
            data: values,
            backgroundColor: DONUT_COLORS.slice(0, labels.length),
            borderColor: "#161b22",
            borderWidth: 2,
          }],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          cutout: "55%",
          layout: { padding: 8 },
          plugins: {
            legend: {
              position: "bottom",
              labels: {
                color: "#e1e4e8",
                font: { size: 13, weight: "500" },
                padding: 16,
                usePointStyle: true,
                pointStyleWidth: 10,
                generateLabels: function (chart) {
                  const data = chart.data;
                  return data.labels.map((label, i) => ({
                    text: label + "  " + fmtCost(data.datasets[0].data[i]),
                    fillStyle: data.datasets[0].backgroundColor[i],
                    fontColor: "#e1e4e8",
                    strokeStyle: "transparent",
                    index: i,
                    pointStyle: "rectRounded",
                  }));
                },
              },
            },
            tooltip: {
              backgroundColor: "#1c2128", borderColor: "#30363d", borderWidth: 1,
              titleColor: "#e1e4e8", bodyColor: "#c9d1d9",
              padding: 12, cornerRadius: 8,
              callbacks: {
                label: function (ctx) {
                  const total = ctx.dataset.data.reduce((a, b) => a + b, 0);
                  const pct = ((ctx.parsed / total) * 100).toFixed(1);
                  return " " + fmtCost(ctx.parsed) + " (" + pct + "%)";
                },
              },
            },
          },
        },
      });
    } catch (e) {
      console.error("provider chart:", e);
    }
  }

  // ── Agent cost leaderboard ──
  let agentChart = null;

  async function loadAgentChart() {
    try {
      const entries = await fetchJSON(tenantQS("/api/dashboard/costs?group_by=agent&hours=168"));
      if (!entries || !entries.length) return;

      const top10 = entries.slice(0, 10);
      const labels = top10.map((e) => e.AgentID || "(none)");
      const values = top10.map((e) => e.TotalCostUSD);
      const ctx = document.getElementById("agent-chart").getContext("2d");

      if (agentChart) agentChart.destroy();

      agentChart = new Chart(ctx, {
        type: "bar",
        data: {
          labels,
          datasets: [{
            label: "Cost (USD)",
            data: values,
            backgroundColor: DONUT_COLORS.slice(0, labels.length),
            borderColor: "#161b22",
            borderWidth: 1,
            borderRadius: 4,
          }],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          indexAxis: "y",
          plugins: {
            legend: { display: false },
            tooltip: {
              backgroundColor: "#1c2128", borderColor: "#30363d", borderWidth: 1,
              titleColor: "#e1e4e8", bodyColor: "#c9d1d9",
              padding: 12, cornerRadius: 8,
              callbacks: { label: (ctx) => " " + fmtCost(ctx.parsed.x) },
            },
          },
          scales: {
            x: {
              beginAtZero: true,
              grid: { color: "rgba(33,38,45,0.5)", drawBorder: false },
              ticks: { color: "#8b949e", font: { size: 11 }, callback: (v) => fmtAxis(v) },
            },
            y: {
              grid: { display: false },
              ticks: { color: "#c9d1d9", font: { size: 11 } },
            },
          },
        },
      });
    } catch (e) {
      console.error("agent chart:", e);
    }
  }

  // ── Model usage chart ──
  let modelChart = null;

  async function loadModelChart() {
    try {
      const entries = await fetchJSON(tenantQS("/api/dashboard/costs?group_by=model&hours=168"));
      if (!entries || !entries.length) return;

      const top10 = entries.slice(0, 10);
      const labels = top10.map((e) => e.Model || "(unknown)");
      const values = top10.map((e) => e.TotalCostUSD);
      const ctx = document.getElementById("model-chart").getContext("2d");

      if (modelChart) modelChart.destroy();

      modelChart = new Chart(ctx, {
        type: "bar",
        data: {
          labels,
          datasets: [{
            label: "Cost (USD)",
            data: values,
            backgroundColor: DONUT_COLORS.slice(0, labels.length),
            borderColor: "#161b22",
            borderWidth: 1,
            borderRadius: 4,
          }],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          indexAxis: "y",
          plugins: {
            legend: { display: false },
            tooltip: {
              backgroundColor: "#1c2128", borderColor: "#30363d", borderWidth: 1,
              titleColor: "#e1e4e8", bodyColor: "#c9d1d9",
              padding: 12, cornerRadius: 8,
              callbacks: { label: (ctx) => " " + fmtCost(ctx.parsed.x) },
            },
          },
          scales: {
            x: {
              beginAtZero: true,
              grid: { color: "rgba(33,38,45,0.5)", drawBorder: false },
              ticks: { color: "#8b949e", font: { size: 11 }, callback: (v) => fmtAxis(v) },
            },
            y: {
              grid: { display: false },
              ticks: { color: "#c9d1d9", font: { size: 11 } },
            },
          },
        },
      });
    } catch (e) {
      console.error("model chart:", e);
    }
  }

  // ── Cost breakdown table ──
  async function loadCosts() {
    const groupBy = $("#costs-group").value;
    try {
      const entries = await fetchJSON(tenantQS(`/api/dashboard/costs?group_by=${groupBy}&hours=168`));
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
          <td>${e.Requests.toLocaleString()}</td>
          <td>${e.InputTokens.toLocaleString()}</td>
          <td>${e.OutputTokens.toLocaleString()}</td>
          <td>${fmtCost(e.TotalCostUSD)}</td>
        `;
        tbody.appendChild(tr);
      }
    } catch (e) {
      console.error("costs:", e);
    }
  }

  // ── Most expensive requests ──
  async function loadExpensive() {
    try {
      const items = await fetchJSON(tenantQS("/api/dashboard/expensive?hours=168&limit=10"));
      const tbody = $("#expensive-body");
      tbody.innerHTML = "";
      if (!items || !items.length) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#8b949e">No data</td></tr>';
        return;
      }
      for (const r of items) {
        const t = new Date(r.timestamp).toLocaleString("en-US", {
          month: "short", day: "numeric", hour: "numeric", minute: "2-digit",
        });
        const tokens = (r.input_tokens + r.output_tokens).toLocaleString();
        const tr = document.createElement("tr");
        tr.innerHTML = `
          <td>${t}</td>
          <td>${esc(r.agent_id || "(none)")}</td>
          <td>${esc(r.model)}</td>
          <td>${tokens}</td>
          <td>${fmtCost(r.cost_usd)}</td>
        `;
        tbody.appendChild(tr);
      }
    } catch (e) {
      console.error("expensive:", e);
    }
  }

  // ── Sessions table ──
  let currentSessionTab = "active";

  function renderSessionRows(sessions, tbody) {
    tbody.innerHTML = "";
    if (!sessions || !sessions.length) {
      tbody.innerHTML = '<tr><td colspan="10" style="text-align:center;color:#8b949e">No sessions</td></tr>';
      return;
    }

    // Anomaly detection: compute mean cost and calls
    var totalCost = 0, totalCalls = 0;
    for (var i = 0; i < sessions.length; i++) {
      totalCost += (sessions[i].TotalCostUSD || sessions[i].total_cost_usd || 0);
      totalCalls += (sessions[i].CallCount || sessions[i].call_count || 0);
    }
    var meanCost = totalCost / sessions.length;
    var meanCalls = totalCalls / sessions.length;

    for (const s of sessions) {
      var id = s.ID || s.id || "";
      var agentID = s.AgentID || s.agent_id || "(none)";
      var userID = s.UserID || s.user_id || "(none)";
      var task = s.Task || s.task || "";
      var callCount = s.CallCount || s.call_count || 0;
      var costUSD = s.TotalCostUSD || s.total_cost_usd || 0;
      var totalTokens = s.TotalTokens || s.total_tokens || 0;
      var status = s.Status || s.status || "";
      var startedAt = s.StartedAt || s.started_at || "";
      var endedAt = s.EndedAt || s.ended_at || null;

      var isAnomaly = (meanCost > 0 && costUSD > meanCost * 3) ||
                      (meanCalls > 0 && callCount > meanCalls * 3);

      const tr = document.createElement("tr");
      if (isAnomaly) tr.classList.add("session-anomaly");
      const statusClass = "status-" + status;
      const started = new Date(startedAt).toLocaleString("en-US", {
        month: "short", day: "numeric", hour: "numeric", minute: "2-digit",
      });
      var duration = fmtDuration(startedAt, endedAt);
      var taskDisplay = task ? task.substring(0, 40) : "";
      if (task && task.length > 40) taskDisplay += "...";

      tr.innerHTML = `
        <td><code>${esc(id.slice(0, 12))}</code></td>
        <td>${esc(agentID)}</td>
        <td>${esc(userID)}</td>
        <td title="${esc(task)}">${esc(taskDisplay)}</td>
        <td>${callCount}</td>
        <td>${fmtCost(costUSD)}</td>
        <td>${totalTokens.toLocaleString()}</td>
        <td class="${statusClass}">${status}</td>
        <td>${duration}</td>
        <td>${started}</td>
      `;
      tbody.appendChild(tr);
    }
  }

  async function loadSessions() {
    try {
      const sessions = await fetchJSON("/api/dashboard/sessions");
      if (currentSessionTab === "active") {
        renderSessionRows(sessions, $("#sessions-body"));
      }
    } catch (e) {
      console.error("sessions:", e);
    }
  }

  async function loadSessionHistory(hours) {
    try {
      const sessions = await fetchJSON(`/api/dashboard/sessions/history?hours=${hours}&limit=50`);
      renderSessionRows(sessions, $("#sessions-body"));
    } catch (e) {
      console.error("session history:", e);
    }
  }

  // Session tab switching
  document.querySelectorAll("[data-session-tab]").forEach((btn) => {
    btn.addEventListener("click", () => {
      document.querySelectorAll("[data-session-tab]").forEach((b) => b.classList.remove("tab-active"));
      btn.classList.add("tab-active");
      currentSessionTab = btn.dataset.sessionTab;
      if (currentSessionTab === "active") {
        loadSessions();
      } else {
        loadSessionHistory(parseInt(currentSessionTab, 10));
      }
    });
  });

  function esc(s) {
    const el = document.createElement("span");
    el.textContent = s;
    return el.innerHTML;
  }

  // ── Latency stats + chart ──
  let latencyChart = null;

  async function loadLatency() {
    try {
      const data = await fetchJSON(tenantQS("/api/dashboard/latency?hours=24"));
      if (!data) return;

      // Update stat items
      $("#stat-p50").textContent = data.p50_ms.toFixed(0) + "ms";
      $("#stat-p90").textContent = data.p90_ms.toFixed(0) + "ms";
      $("#stat-p99").textContent = data.p99_ms.toFixed(0) + "ms";

      // Render bucket chart
      var buckets = data.buckets || [];
      if (!buckets.length) return;

      var labels = buckets.map(function(b) { return b.label; });
      var values = buckets.map(function(b) { return b.count; });
      var ctx = document.getElementById("latency-chart").getContext("2d");

      if (latencyChart) latencyChart.destroy();

      latencyChart = new Chart(ctx, {
        type: "bar",
        data: {
          labels: labels,
          datasets: [{
            label: "Requests",
            data: values,
            backgroundColor: "rgba(163, 113, 247, 0.6)",
            borderColor: "#a371f7",
            borderWidth: 1,
            borderRadius: 4,
          }],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: {
            legend: { display: false },
            tooltip: {
              backgroundColor: "#1c2128", borderColor: "#30363d", borderWidth: 1,
              titleColor: "#e1e4e8", bodyColor: "#c9d1d9",
              padding: 12, cornerRadius: 8,
            },
          },
          scales: {
            x: {
              grid: { display: false },
              ticks: { color: "#8b949e", font: { size: 10 } },
            },
            y: {
              beginAtZero: true,
              grid: { color: "rgba(33,38,45,0.5)", drawBorder: false },
              ticks: { color: "#8b949e", font: { size: 10 } },
            },
          },
        },
      });
    } catch (e) {
      console.error("latency:", e);
    }
  }

  // ── API Keys table ──
  async function loadAPIKeys() {
    try {
      const keys = await adminFetch("/api/admin/api-keys");
      const tbody = $("#apikeys-body");
      tbody.innerHTML = "";
      if (!keys || !keys.length) {
        tbody.innerHTML = '<tr><td colspan="3" style="text-align:center;color:#8b949e">No data</td></tr>';
        return;
      }
      for (const k of keys) {
        const tr = document.createElement("tr");
        tr.innerHTML = `<td><code>${esc(k.api_key_hash)}</code></td><td>${k.requests}</td><td>${fmtCost(k.total_cost_usd)}</td>`;
        tbody.appendChild(tr);
      }
    } catch (e) { /* admin API may not be enabled */ }
  }

  // ── Budget Rules table ──
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
          <td><code>${esc(r.api_key_pattern || r.APIKeyPattern || "")}</code></td>
          <td>${fmtCost(r.daily_limit_usd || r.DailyLimitUSD || 0)}</td>
          <td>${fmtCost(r.monthly_limit_usd || r.MonthlyLimitUSD || 0)}</td>
          <td>${r.action || r.Action || ""}</td>
          <td><button class="btn-delete" data-pattern="${esc(r.api_key_pattern || r.APIKeyPattern || "")}">&#215;</button></td>
        `;
        tbody.appendChild(tr);
      }
      for (const btn of tbody.querySelectorAll(".btn-delete")) {
        btn.addEventListener("click", async () => {
          await adminFetch("/api/admin/budgets/rules?pattern=" + encodeURIComponent(btn.dataset.pattern), { method: "DELETE" });
          loadRules();
        });
      }
    } catch (e) { /* admin API may not be enabled */ }
  }

  // ── Budget Status gauges ──
  async function loadBudgetStatus() {
    try {
      const statuses = await adminFetch("/api/admin/budgets/status");
      const container = $("#budget-status-body");
      container.innerHTML = "";
      if (!statuses || !statuses.length) {
        container.innerHTML = '<div style="color:#8b949e;font-size:0.85rem;padding:0.5rem">No budget data</div>';
        return;
      }
      for (const s of statuses) {
        var item = document.createElement("div");
        item.className = "budget-item";

        var dailyPct = s.daily_limit > 0 ? Math.min((s.daily_spent / s.daily_limit) * 100, 100) : 0;
        var monthlyPct = s.monthly_limit > 0 ? Math.min((s.monthly_spent / s.monthly_limit) * 100, 100) : 0;

        var dailyClass = "budget-fill";
        if (dailyPct > 90) dailyClass += " budget-fill-danger";
        else if (dailyPct > 70) dailyClass += " budget-fill-warn";

        var monthlyClass = "budget-fill";
        if (monthlyPct > 90) monthlyClass += " budget-fill-danger";
        else if (monthlyPct > 70) monthlyClass += " budget-fill-warn";

        var html = '<div class="budget-item-label"><span>' + esc(s.pattern) + '</span><span>' + esc(s.action) + '</span></div>';

        if (s.daily_limit > 0) {
          html += '<div class="budget-bar-label"><span>Daily</span><span>' + fmtCost(s.daily_spent) + ' / ' + fmtCost(s.daily_limit) + '</span></div>';
          html += '<div class="budget-bar"><div class="' + dailyClass + '" style="width:' + dailyPct + '%"></div></div>';
        }
        if (s.monthly_limit > 0) {
          html += '<div class="budget-bar-label"><span>Monthly</span><span>' + fmtCost(s.monthly_spent) + ' / ' + fmtCost(s.monthly_limit) + '</span></div>';
          html += '<div class="budget-bar"><div class="' + monthlyClass + '" style="width:' + monthlyPct + '%"></div></div>';
        }
        if (s.daily_limit <= 0 && s.monthly_limit <= 0) {
          html += '<div class="budget-bar-label"><span>Spend: ' + fmtCost(s.daily_spent) + ' today / ' + fmtCost(s.monthly_spent) + ' month</span></div>';
        }

        item.innerHTML = html;
        container.appendChild(item);
      }
    } catch (e) { /* admin API may not be enabled */ }
  }

  // ── Blocked Keys table ──
  async function loadBlocked() {
    try {
      const patterns = await adminFetch("/api/admin/api-keys/blocked");
      const tbody = $("#blocked-body");
      tbody.innerHTML = "";
      if (!patterns || !patterns.length) {
        tbody.innerHTML = '<tr><td colspan="2" style="text-align:center;color:#8b949e">No blocked keys</td></tr>';
        return;
      }
      for (const p of patterns) {
        const tr = document.createElement("tr");
        tr.innerHTML = `<td><code>${esc(p)}</code></td><td><button class="btn-delete" data-pattern="${esc(p)}">&#215;</button></td>`;
        tbody.appendChild(tr);
      }
      for (const btn of tbody.querySelectorAll(".btn-delete")) {
        btn.addEventListener("click", async () => {
          await adminFetch("/api/admin/api-keys/block?pattern=" + encodeURIComponent(btn.dataset.pattern), { method: "DELETE" });
          loadBlocked();
        });
      }
    } catch (e) { /* admin API may not be enabled */ }
  }

  // Block key button
  $("#block-btn").addEventListener("click", async () => {
    var pattern = $("#block-pattern").value.trim();
    if (!pattern) return;
    await adminFetch("/api/admin/api-keys/block", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ pattern: pattern }),
    });
    $("#block-pattern").value = "";
    loadBlocked();
  });

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
    loadBudgetStatus();
  });

  // Tenant filter
  $("#apply-tenant").addEventListener("click", () => {
    currentTenant = $("#tenant-filter").value.trim();
    loadAll();
  });

  function loadAll() {
    loadSummary();
    loadStats();
    if (currentTimeseriesTab === "cost") {
      loadTimeseries();
    } else {
      loadTokenTimeseries();
    }
    loadProviderChart();
    loadAgentChart();
    loadModelChart();
    loadCosts();
    loadExpensive();
    if (currentSessionTab === "active") {
      loadSessions();
    } else {
      loadSessionHistory(parseInt(currentSessionTab, 10));
    }
    loadLatency();
    loadAPIKeys();
    loadRules();
    loadBudgetStatus();
    loadBlocked();
  }

  $("#timeseries-hours").addEventListener("change", () => {
    if (currentTimeseriesTab === "cost") {
      loadTimeseries();
    } else {
      loadTokenTimeseries();
    }
  });
  $("#costs-group").addEventListener("change", loadCosts);

  loadAll();
  setInterval(loadAll, 30000);
})();

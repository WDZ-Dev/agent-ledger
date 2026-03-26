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
        const started = new Date(s.StartedAt).toLocaleString("en-US", {
          month: "short", day: "numeric", hour: "numeric", minute: "2-digit",
        });
        tr.innerHTML = `
          <td><code>${esc(s.ID.slice(0, 12))}</code></td>
          <td>${esc(s.AgentID || "(none)")}</td>
          <td>${esc(s.UserID || "(none)")}</td>
          <td>${s.CallCount}</td>
          <td>${fmtCost(s.TotalCostUSD)}</td>
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

  // Tenant filter
  $("#apply-tenant").addEventListener("click", () => {
    currentTenant = $("#tenant-filter").value.trim();
    loadAll();
  });

  function loadAll() {
    loadSummary();
    loadStats();
    loadTimeseries();
    loadProviderChart();
    loadCosts();
    loadExpensive();
    loadSessions();
    loadAPIKeys();
    loadRules();
  }

  $("#timeseries-hours").addEventListener("change", loadTimeseries);
  $("#costs-group").addEventListener("change", loadCosts);

  loadAll();
  setInterval(loadAll, 30000);
})();

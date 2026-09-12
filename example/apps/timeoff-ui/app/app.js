(async function () {
  const $ = (id) => document.getElementById(id);
  const state = { config: null, token: null, actor: null, info: null };

  const cfg = await fetch("./config.json", { cache: "no-store" }).then((r) => r.json());
  state.config = cfg;
  $("namespace").textContent = cfg.namespace || "local";
  if (cfg.uiVersion) $("ver-ui").textContent = "ui: " + cfg.uiVersion;

  const timeoff = api(cfg.timeoffApi);
  const payroll = api(cfg.payrollApi);

  function api(base) {
    base = base.replace(/\/+$/, "");
    return async function (path, opts = {}) {
      const headers = { "Content-Type": "application/json", ...(opts.headers || {}) };
      if (state.token) headers.Authorization = "Bearer " + state.token;
      const res = await fetch(base + path, { ...opts, headers });
      const body = res.status === 204 ? null : await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(body && body.error ? body.error : res.status + " " + res.statusText);
      return body;
    };
  }

  const fmtDate = (s) => new Date(s + "T00:00:00").toLocaleDateString(undefined, { month: "short", day: "numeric" });
  const fmtMoney = (cents) => (cents / 100).toLocaleString(undefined, { style: "currency", currency: "USD" });
  const fmtTime = (iso) => new Date(iso).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });

  // --- timeoff-api ---

  async function loadInfo() {
    try {
      const info = await timeoff("/info");
      state.info = info;
      $("company").textContent = (info.company_name || "Time Off") + " · Time Off";
      $("ver-timeoff").textContent = "timeoff-api: " + info.version;
      $("kind").innerHTML = info.kinds.map((k) => `<option value="${k}">${k}</option>`).join("");
      $("policy").textContent = info.auto_approve
        ? `Auto-approved, up to ${info.max_days_per_request} days per request`
        : `Up to ${info.max_days_per_request} days per request, decisions expected within ${info.approval_window_days} days`;
      $("requests-error").hidden = true;
    } catch (e) {
      $("ver-timeoff").textContent = "timeoff-api: unreachable";
      showError("requests-error", "timeoff-api is not reachable yet: " + e.message);
    }
  }

  async function loadEmployees() {
    const emps = await timeoff("/employees").catch(() => []);
    const sel = $("actor");
    sel.innerHTML = `<option value="">choose…</option>` + emps.map((e) => `<option value="${e.ID}">${e.Name} · ${e.Team}</option>`).join("");
    const saved = sessionStorage.getItem("actor");
    if (saved && emps.some((e) => String(e.ID) === saved)) {
      sel.value = saved;
      await startSession(Number(saved));
    }
  }

  async function startSession(employeeId) {
    if (!employeeId) { state.token = null; state.actor = null; sessionStorage.removeItem("actor"); return renderRequests(); }
    const s = await timeoff("/session", { method: "POST", body: JSON.stringify({ employee_id: employeeId }) });
    state.token = s.token;
    state.actor = s.employee;
    sessionStorage.setItem("actor", String(employeeId));
    renderRequests();
  }
  $("actor").addEventListener("change", (e) => startSession(Number(e.target.value)).catch((err) => showError("form-error", err.message)));

  let requests = [];
  async function loadRequests() {
    try {
      requests = await timeoff("/requests");
      $("requests-error").hidden = true;
    } catch (e) {
      showError("requests-error", e.message);
      return;
    }
    renderRequests();
  }

  function renderRequests() {
    const tbody = $("requests").querySelector("tbody");
    $("request-count").textContent = requests.length ? `${requests.filter((r) => r.status === "pending").length} pending of ${requests.length}` : "";
    $("requests-empty").hidden = requests.length > 0;
    tbody.innerHTML = requests.map((r) => `
      <tr>
        <td>${esc(r.employee)}<div class="hint">${esc(r.team)}</div></td>
        <td>${r.kind}</td>
        <td>${fmtDate(r.start_date)} – ${fmtDate(r.end_date)}${r.note ? `<div class="hint">${esc(r.note)}</div>` : ""}</td>
        <td class="num">${r.days}</td>
        <td><span class="status ${r.status}">${r.status}</span>${r.overdue ? `<span class="overdue">overdue</span>` : ""}${r.decided_by ? `<div class="hint">by ${esc(r.decided_by)}</div>` : ""}</td>
        <td>${r.status === "pending" ? `
          <div class="decide">
            <button class="ghost" data-act="approve" data-id="${r.id}" ${state.token ? "" : "disabled"}>Approve</button>
            <button class="ghost" data-act="deny" data-id="${r.id}" ${state.token ? "" : "disabled"}>Deny</button>
          </div>` : ""}</td>
      </tr>`).join("");
  }

  $("requests").addEventListener("click", async (e) => {
    const btn = e.target.closest("button[data-act]");
    if (!btn) return;
    btn.disabled = true;
    try {
      await timeoff(`/requests/${btn.dataset.id}/${btn.dataset.act}`, { method: "POST" });
      await Promise.all([loadRequests(), loadPayroll()]);
    } catch (err) {
      showError("requests-error", err.message);
      btn.disabled = false;
    }
  });

  $("request-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    $("form-error").hidden = true;
    if (!state.token) return showError("form-error", "Pick who you are acting as first.");
    const f = new FormData(e.target);
    try {
      await timeoff("/requests", { method: "POST", body: JSON.stringify({
        kind: f.get("kind"), start_date: f.get("start_date"), end_date: f.get("end_date"), note: f.get("note"),
      }) });
      e.target.reset();
      await Promise.all([loadRequests(), loadPayroll()]);
    } catch (err) {
      showError("form-error", err.message);
    }
  });

  // --- payroll-api ---

  async function loadPayroll() {
    let info;
    try {
      info = await payroll("/info");
    } catch (e) {
      $("payroll-offline").hidden = false;
      $("payroll").hidden = true;
      $("ver-payroll").textContent = "payroll-api: not deployed";
      $("events-empty").hidden = false;
      $("events").innerHTML = "";
      return;
    }
    $("payroll-offline").hidden = true;
    $("payroll").hidden = false;
    $("ver-payroll").textContent = "payroll-api: " + info.version;
    $("provider").innerHTML = `<span class="dot"></span> Payments provider connected, key ${esc(info.payments_provider.key_hint)} · ${info.events_consumed} events consumed`;

    const ledger = await payroll("/ledger");
    $("pay-period").textContent = `${ledger.pay_period}, ${fmtDate(ledger.period_start)} – ${fmtDate(ledger.period_end)}`;
    const tbody = $("ledger").querySelector("tbody");
    $("ledger-empty").hidden = ledger.employees.length > 0;
    tbody.innerHTML = ledger.employees.map((e) => `
      <tr>
        <td>${esc(e.employee)}<div class="hint">${esc(e.team)}</div></td>
        <td class="num">${e.paid_days}</td>
        <td class="num">${e.unpaid_days}</td>
        <td class="num ${e.adjustment_cents < 0 ? "neg" : ""}">${fmtMoney(e.adjustment_cents)}</td>
      </tr>`).join("");

    const events = await payroll("/events");
    $("events-empty").hidden = events.length > 0;
    $("events").innerHTML = events.map((ev) => `
      <li><time>${fmtTime(ev.occurred_at)}</time>
        <span>${esc(ev.employee)} · ${ev.kind} · ${ev.days}d · <strong>${ev.type === "timeoff.requested" ? "requested" : ev.status}</strong>${ev.decided_by ? ` by ${esc(ev.decided_by)}` : ""}</span></li>`).join("");
  }

  // --- utils ---

  function showError(id, msg) { const el = $(id); el.textContent = msg; el.hidden = false; }
  function esc(s) { return String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c])); }

  await loadInfo();
  await loadEmployees();
  await Promise.all([loadRequests(), loadPayroll()]);
  setInterval(() => { loadRequests(); loadPayroll(); }, 5000);
})();

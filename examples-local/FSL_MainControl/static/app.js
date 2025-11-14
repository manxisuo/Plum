let currentTaskId = null;
let statusTimer = null;
let map = null;
let markersLayer = null;
let tracksLayer = null;
let lastStage = null;
let mapBoundsLocked = false;
let workflowList = [];
let selectedWorkflowId = null;
let activeRunId = null;
const tingHeadings = new Map();
const lastPositions = new Map();
const MAP_CENTER = [30.664554, 122.510268];
const DEFAULT_ZOOM = 13;
const OFFLINE_TILE_URL = "/static/tiles/{z}/{x}/{y}.png";
const ONLINE_TILE_URL = "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png";
const TILE_ATTRIBUTION =
  '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';
let configState = null;
let activeTingTab = 0;

function createTingIcon(angleDeg = 0, label = "") {
  return L.divIcon({
    className: "ting-icon",
    iconSize: [32, 32],
    iconAnchor: [16, 28],
    popupAnchor: [0, -22],
    html: `
      <div class="ting-wrapper">
        <div class="ting-triangle" style="transform: rotate(${angleDeg}deg);"></div>
        <div class="ting-label">${label}</div>
      </div>
    `,
  });
}

function clamp(value, min, max) {
  if (!Number.isFinite(value)) return min;
  return Math.min(Math.max(value, min), max);
}

function ensureNumber(value, fallback = 0) {
  const num = Number(value);
  return Number.isFinite(num) ? num : fallback;
}

function cloneConfig(data) {
  return JSON.parse(JSON.stringify(data || {}));
}

function toPointObject(point) {
  if (!Array.isArray(point)) {
    return { lat: point.lat, lon: point.lon };
  }
  return { lat: point[0], lon: point[1] };
}

function toArrayPoint(point) {
  return [point.lat, point.lon];
}

function catmullRomInterpolate(p0, p1, p2, p3, t) {
  const t2 = t * t;
  const t3 = t2 * t;
  const calc = (k0, k1, k2, k3) =>
    0.5 *
    ((2 * k1) +
      (-k0 + k2) * t +
      (2 * k0 - 5 * k1 + 4 * k2 - k3) * t2 +
      (-k0 + 3 * k1 - 3 * k2 + k3) * t3);
  return {
    lat: calc(p0.lat, p1.lat, p2.lat, p3.lat),
    lon: calc(p0.lon, p1.lon, p2.lon, p3.lon),
  };
}

function smoothTrackPoints(points, segments = 6) {
  if (!points || points.length <= 2) {
    return points ? points.map(toPointObject) : [];
  }

  const result = [];
  const objs = points.map(toPointObject);
  for (let i = 0; i < objs.length - 1; i += 1) {
    const p0 = objs[i - 1] || objs[i];
    const p1 = objs[i];
    const p2 = objs[i + 1];
    const p3 = objs[i + 2] || p2;
    result.push(p1);
    for (let j = 1; j < segments; j += 1) {
      const t = j / segments;
      result.push(catmullRomInterpolate(p0, p1, p2, p3, t));
    }
  }
  result.push(objs[objs.length - 1]);
  return result;
}

function createDefaultTing(index, taskArea) {
  // 如果没有提供任务区域，使用默认值
  if (!taskArea) {
    taskArea = {
      top_left: { lat: MAP_CENTER[0] + 0.01, lon: MAP_CENTER[1] - 0.01 },
      bottom_right: { lat: MAP_CENTER[0] - 0.01, lon: MAP_CENTER[1] + 0.01 },
    };
  }
  
  // 计算任务区域的中心经度
  const centerLon = (taskArea.top_left.lon + taskArea.bottom_right.lon) / 2;
  // 计算任务区域上方中心位置（在区域外，上方约0.0045度，约500米）
  const topCenterLat = taskArea.top_left.lat + 0.0045;
  
  // 在中心位置附近稍微分散，形成一个小队形
  const offsetLat = (index % 2 === 0 ? 0.0005 : -0.0005) * Math.floor(index / 2);
  const offsetLon = (index % 2 === 0 ? -0.001 : 0.001) * (index % 2 === 0 ? 1 : -1);
  
  return {
    id: `usv-${index + 1}`,
    name: `USV ${index + 1}`,
    position: {
      lat: topCenterLat + offsetLat,
      lon: centerLon + offsetLon,
    },
    speed_mps: 40,
    sonar_range_m: 300,
    suspect_prob: 0.4,
    confirm_prob: 0.6,
  };
}

function hydrateConfig(data) {
  const cfg = cloneConfig(data);
  if (!Array.isArray(cfg.tings)) {
    cfg.tings = [];
  }
  cfg.ting_count = Math.max(1, Math.round(ensureNumber(cfg.ting_count, cfg.tings.length || 1)));

  if (!cfg.task_area) {
    cfg.task_area = {
      top_left: { lat: MAP_CENTER[0] + 0.01, lon: MAP_CENTER[1] - 0.01 },
      bottom_right: { lat: MAP_CENTER[0] - 0.01, lon: MAP_CENTER[1] + 0.01 },
    };
  } else {
    cfg.task_area.top_left = {
      lat: ensureNumber(cfg.task_area?.top_left?.lat, MAP_CENTER[0] + 0.01),
      lon: ensureNumber(cfg.task_area?.top_left?.lon, MAP_CENTER[1] - 0.01),
    };
    cfg.task_area.bottom_right = {
      lat: ensureNumber(cfg.task_area?.bottom_right?.lat, MAP_CENTER[0] - 0.01),
      lon: ensureNumber(cfg.task_area?.bottom_right?.lon, MAP_CENTER[1] + 0.01),
    };
  }

  if (cfg.tings.length < cfg.ting_count) {
    for (let i = cfg.tings.length; i < cfg.ting_count; i += 1) {
      cfg.tings.push(createDefaultTing(i, cfg.task_area));
    }
  } else if (cfg.tings.length > cfg.ting_count) {
    cfg.tings = cfg.tings.slice(0, cfg.ting_count);
  }

  cfg.tings = cfg.tings.map((ting, idx) => ({
    id: (ting.id || `usv-${idx + 1}`).trim(),
    name: ting.name ? ting.name.trim() : (ting.id || `USV ${idx + 1}`),
    position: {
      lat: ensureNumber(ting.position?.lat, MAP_CENTER[0]),
      lon: ensureNumber(ting.position?.lon, MAP_CENTER[1]),
    },
    speed_mps: ensureNumber(ting.speed_mps, 30),
    sonar_range_m: ensureNumber(ting.sonar_range_m ?? ting.sonar_range, 200),
    suspect_prob: clamp(ensureNumber(ting.suspect_prob, 0.4), 0, 1),
    confirm_prob: clamp(ensureNumber(ting.confirm_prob, 0.6), 0, 1),
  }));

  cfg.ting_count = cfg.tings.length;
  if (cfg.ting_count <= 0) {
    activeTingTab = 0;
  } else if (activeTingTab >= cfg.ting_count) {
    activeTingTab = cfg.ting_count - 1;
  }

  return cfg;
}

function onTingCountChange(event) {
  if (!configState) return;
  let value = Number(event.target.value);
  if (!Number.isFinite(value) || value < 1) {
    value = configState.ting_count;
  }
  value = Math.min(Math.round(value), 12);
  event.target.value = value;

  const currentLength = configState.tings.length;
  if (value > currentLength) {
    for (let i = currentLength; i < value; i += 1) {
      configState.tings.push(createDefaultTing(i, configState.task_area));
    }
  } else if (value < currentLength) {
    configState.tings = configState.tings.slice(0, value);
  }
  configState.ting_count = configState.tings.length;
  if (configState.ting_count === 0) {
    activeTingTab = 0;
  } else if (activeTingTab >= configState.ting_count) {
    activeTingTab = configState.ting_count - 1;
  }
  renderConfigForm();
}

function handleTaskAreaInput(event) {
  if (!configState) return;
  const area = event.target.dataset.area;
  const axis = event.target.dataset.axis;
  if (!area || !axis) {
    return;
  }
  const value = parseFloat(event.target.value);
  if (!Number.isFinite(value)) {
    return;
  }
  if (!configState.task_area[area]) {
    configState.task_area[area] = { lat: MAP_CENTER[0], lon: MAP_CENTER[1] };
  }
  configState.task_area[area][axis] = value;
  renderPreviewFromConfig();
}

function updateTingTabLabel(index) {
  const tabButton = document.querySelector(`.tab-button[data-ting-index="${index}"]`);
  if (!tabButton || !configState) return;
  const ting = configState.tings[index];
  tabButton.textContent = ting.name || ting.id || `USV ${index + 1}`;
}

function handleTingFieldInput(event) {
  if (!configState) return;
  const input = event.target;
  const index = Number(input.dataset.tingIndex);
  if (!Number.isFinite(index) || !configState.tings[index]) {
    return;
  }
  const ting = configState.tings[index];
  const field = input.dataset.field;
  const subfield = input.dataset.subfield;

  if (field === "position" && subfield) {
    const value = parseFloat(input.value);
    if (Number.isFinite(value)) {
      ting.position[subfield] = value;
    }
  } else if (field === "speed_mps" || field === "sonar_range_m") {
    const value = parseFloat(input.value);
    if (Number.isFinite(value)) {
      ting[field] = value;
    }
  } else if (field === "suspect_prob" || field === "confirm_prob") {
    let value = parseFloat(input.value);
    if (!Number.isFinite(value)) {
      return;
    }
    value = clamp(value, 0, 1);
    ting[field] = value;
    input.value = value.toString();
  } else if (field === "id" || field === "name") {
    ting[field] = input.value;
  }

  updateTingTabLabel(index);
  renderPreviewFromConfig();
}

function createTingInput(index, labelText, value, options = {}) {
  const wrapper = document.createElement("div");
  wrapper.className = "form-field";

  const label = document.createElement("label");
  label.textContent = labelText;

  let input;
  if (options.component === "range") {
    const container = document.createElement("div");
    container.className = "range-field";

    input = document.createElement("input");
    input.type = "range";
    const min = Number(options.min ?? "0");
    const max = Number(options.max ?? "10");
    const step = Number(options.step ?? "0.1");
    const initial = Number.isFinite(value) ? value : min;
    input.min = String(min);
    input.max = String(max);
    input.step = String(step);
    input.value = initial;

    const number = document.createElement("input");
    number.type = "number";
    number.min = String(min);
    number.max = String(max);
    number.step = String(step);
    number.value = initial;

    input.addEventListener("input", (e) => {
      const raw = Number(e.target.value);
      if (!Number.isFinite(raw)) {
        return;
      }
      const clamped = clamp(raw, min, max);
      if (clamped !== raw) {
        input.value = clamped;
      }
      if (Number(number.value) !== clamped) {
        number.value = clamped;
      }
      const synthetic = new Event("input", { bubbles: true });
      number.dispatchEvent(synthetic);
    });

    number.dataset.tingIndex = String(index);
    number.dataset.field = options.field;
    if (options.subfield) {
      number.dataset.subfield = options.subfield;
    }
    number.addEventListener("input", (e) => {
      const raw = Number(e.target.value);
      if (!Number.isFinite(raw)) {
        return;
      }
      const clamped = clamp(raw, min, max);
      if (clamped !== raw) {
        number.value = clamped;
      }
      if (Number(input.value) !== clamped) {
        input.value = clamped;
      }
      handleTingFieldInput(e);
    });

    container.appendChild(input);
    container.appendChild(number);
    wrapper.appendChild(label);
    wrapper.appendChild(container);
    return wrapper;
  }

  input = document.createElement("input");
  input.type = options.type || "text";
  if (options.step !== undefined) input.step = options.step;
  if (options.min !== undefined) input.min = options.min;
  if (options.max !== undefined) input.max = options.max;
  input.value =
    input.type === "number" ? (Number.isFinite(value) ? value : "") : value ?? "";
  input.dataset.tingIndex = String(index);
  input.dataset.field = options.field;
  if (options.subfield) {
    input.dataset.subfield = options.subfield;
  }
  input.addEventListener("input", handleTingFieldInput);

  wrapper.appendChild(label);
  wrapper.appendChild(input);
  return wrapper;
}

function renderConfigForm() {
  if (!configState) return;

  const tingCountInput = document.getElementById("input-ting-count");
  if (!tingCountInput) {
    return;
  }
  tingCountInput.value = configState.ting_count;
  tingCountInput.onchange = onTingCountChange;

  const topLat = document.getElementById("input-top-left-lat");
  const topLon = document.getElementById("input-top-left-lon");
  const bottomLat = document.getElementById("input-bottom-right-lat");
  const bottomLon = document.getElementById("input-bottom-right-lon");

  const taskArea = configState.task_area || {
    top_left: { lat: "", lon: "" },
    bottom_right: { lat: "", lon: "" },
  };

  if (topLat) {
    topLat.value = taskArea.top_left?.lat ?? "";
    topLat.dataset.area = "top_left";
    topLat.dataset.axis = "lat";
    topLat.oninput = handleTaskAreaInput;
  }
  if (topLon) {
    topLon.value = taskArea.top_left?.lon ?? "";
    topLon.dataset.area = "top_left";
    topLon.dataset.axis = "lon";
    topLon.oninput = handleTaskAreaInput;
  }
  if (bottomLat) {
    bottomLat.value = taskArea.bottom_right?.lat ?? "";
    bottomLat.dataset.area = "bottom_right";
    bottomLat.dataset.axis = "lat";
    bottomLat.oninput = handleTaskAreaInput;
  }
  if (bottomLon) {
    bottomLon.value = taskArea.bottom_right?.lon ?? "";
    bottomLon.dataset.area = "bottom_right";
    bottomLon.dataset.axis = "lon";
    bottomLon.oninput = handleTaskAreaInput;
  }

  const tabList = document.getElementById("ting-tab-list");
  const tabPanels = document.getElementById("ting-tab-panels");
  if (!tabList || !tabPanels) return;
  tabList.innerHTML = "";
  tabPanels.innerHTML = "";

  if (!configState.tings || configState.tings.length === 0) {
    const hint = document.createElement("div");
    hint.className = "empty-hint";
    hint.textContent = "暂无 USV 配置";
    tabPanels.appendChild(hint);
    renderPreviewFromConfig();
    return;
  }

  if (activeTingTab >= configState.ting_count) {
    activeTingTab = Math.max(0, configState.ting_count - 1);
  }

  configState.tings.forEach((ting, index) => {
    const tabButton = document.createElement("button");
    tabButton.type = "button";
    tabButton.className = "tab-button";
    tabButton.dataset.tingIndex = String(index);
    tabButton.textContent = ting.name || ting.id || `USV ${index + 1}`;
    if (index === activeTingTab) {
      tabButton.classList.add("active");
    }
    tabButton.addEventListener("click", () => {
      activeTingTab = index;
      renderConfigForm();
    });
    tabList.appendChild(tabButton);

    const panel = document.createElement("div");
    panel.className = "tab-panel";
    if (index === activeTingTab) {
      panel.classList.add("active");
    }

    const row1 = document.createElement("div");
    row1.className = "form-row";
    row1.appendChild(createTingInput(index, "USV ID", ting.id, { field: "id" }));
    row1.appendChild(createTingInput(index, "名称", ting.name, { field: "name" }));
    panel.appendChild(row1);

    const row2 = document.createElement("div");
    row2.className = "form-row";
    row2.appendChild(
      createTingInput(index, "初始纬度", ting.position.lat, {
        field: "position",
        subfield: "lat",
        type: "number",
        step: "0.000001",
      })
    );
    row2.appendChild(
      createTingInput(index, "初始经度", ting.position.lon, {
        field: "position",
        subfield: "lon",
        type: "number",
        step: "0.000001",
      })
    );
    panel.appendChild(row2);

    const row3 = document.createElement("div");
    row3.className = "form-row";
    row3.appendChild(
      createTingInput(index, "速度 (m/s)", ting.speed_mps, {
        field: "speed_mps",
        component: "range",
        step: "0.1",
        min: "0",
        max: "50",
      })
    );
    row3.appendChild(
      createTingInput(index, "声呐范围 (m)", ting.sonar_range_m, {
        field: "sonar_range_m",
        type: "number",
        step: "1",
        min: "0",
      })
    );
    panel.appendChild(row3);

    const row4 = document.createElement("div");
    row4.className = "form-row";
    row4.appendChild(
      createTingInput(index, "疑似概率", ting.suspect_prob, {
        field: "suspect_prob",
        type: "number",
        step: "0.01",
        min: "0",
        max: "1",
      })
    );
    row4.appendChild(
      createTingInput(index, "确认概率", ting.confirm_prob, {
        field: "confirm_prob",
        type: "number",
        step: "0.01",
        min: "0",
        max: "1",
      })
    );
    panel.appendChild(row4);

    tabPanels.appendChild(panel);
  });
  renderPreviewFromConfig();
}

function showConfigModal() {
  const modal = document.getElementById("config-modal");
  if (!modal) return;
  modal.classList.add("show");
  document.body.classList.add("modal-open");
}

function hideConfigModal() {
  const modal = document.getElementById("config-modal");
  if (!modal) return;
  if (!modal.classList.contains("show")) return;
  modal.classList.remove("show");
  document.body.classList.remove("modal-open");
}

function renderPreviewFromConfig() {
  if (!configState) return;
  const previewStatus = {
    stage: "config_preview",
    config: {
      task_area: cloneConfig(configState.task_area),
    },
    plan: { work_zones: [] },
    tings: configState.tings.map((ting, idx) => ({
      id: ting.id,
      label: String(idx + 1).padStart(3, "0"),
      name: ting.name,
      position: {
        lat: ting.position.lat,
        lon: ting.position.lon,
      },
      speed_mps: ting.speed_mps,
      sonar_range_m: ting.sonar_range_m,
      suspect_prob: ting.suspect_prob,
      confirm_prob: ting.confirm_prob,
    })),
    suspect_mines: [],
    confirmed_mines: [],
    destroyed_mines: [],
    evaluated_mines: [],
    tracks: [],
    timeline: [],
  };
  renderMap(previewStatus);
}

function buildConfigPayload() {
  if (!configState) return null;
  const payload = cloneConfig(configState);
  payload.ting_count = Math.max(
    1,
    Math.round(ensureNumber(payload.ting_count, payload.tings.length || 1))
  );
  payload.tings = payload.tings.slice(0, payload.ting_count).map((ting, index) => ({
    id: (ting.id || `usv-${index + 1}`).trim(),
    label: String(index + 1).padStart(3, "0"),
    name: ting.name ? ting.name.trim() : (ting.id || `USV ${index + 1}`),
    position: {
      lat: ensureNumber(ting.position?.lat, MAP_CENTER[0]),
      lon: ensureNumber(ting.position?.lon, MAP_CENTER[1]),
    },
    speed_mps: ensureNumber(ting.speed_mps, 30),
    sonar_range_m: ensureNumber(ting.sonar_range_m ?? ting.sonar_range, 200),
    suspect_prob: clamp(ensureNumber(ting.suspect_prob, 0.4), 0, 1),
    confirm_prob: clamp(ensureNumber(ting.confirm_prob, 0.6), 0, 1),
  }));

  payload.task_area = payload.task_area || {};
  payload.task_area.top_left = {
    lat: ensureNumber(payload.task_area?.top_left?.lat, MAP_CENTER[0] + 0.01),
    lon: ensureNumber(payload.task_area?.top_left?.lon, MAP_CENTER[1] - 0.01),
  };
  payload.task_area.bottom_right = {
    lat: ensureNumber(payload.task_area?.bottom_right?.lat, MAP_CENTER[0] - 0.01),
    lon: ensureNumber(payload.task_area?.bottom_right?.lon, MAP_CENTER[1] + 0.01),
  };

  payload.ting_count = payload.tings.length;

  return payload;
}

function logStatus(message) {
  const logElem = document.getElementById("status-log");
  const now = new Date().toLocaleTimeString();
  logElem.textContent += `[${now}] ${message}\n`;
  logElem.scrollTop = logElem.scrollHeight;
}

async function loadDefaults() {
  try {
    const resp = await fetch("/api/config/defaults");
    if (!resp.ok) {
      throw new Error(`HTTP ${resp.status}`);
    }
    const data = await resp.json();
    activeTingTab = 0;
    configState = hydrateConfig(data);
    tingHeadings.clear();
    lastPositions.clear();
    renderConfigForm();
    logStatus("已加载默认参数");
  } catch (err) {
    logStatus(`加载默认参数失败：${err.message}`);
  }
}

async function startTask() {
  try {
    if (!configState) {
      logStatus("参数尚未加载，请先加载默认参数");
      return;
    }
    const payload = buildConfigPayload();
    if (!payload) {
      logStatus("参数无效，请检查后再试。");
      return;
    }
    const payloadWithWorkflow = { ...payload, workflow_id: selectedWorkflowId || null };
    const resp = await fetch("/api/task/start", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payloadWithWorkflow),
    });
    if (!resp.ok) {
      let detail = "启动任务失败";
      try {
        const err = await resp.json();
        detail = err.detail || JSON.stringify(err);
      } catch (_) {
        detail = await resp.text();
      }
      throw new Error(detail);
    }
    const data = await resp.json();
    currentTaskId = data.task_id;
    document.getElementById("current-task").textContent = `任务：${currentTaskId}`;
    logStatus(`任务已启动，阶段：${data.stage}`);
    const workflowPayload = {
      taskId: data.task_id,
      sweepPayload: data.扫雷_payload,  // 属性名保持英文，值使用中文阶段名称
      config: payload,
    };
    if (!selectedWorkflowId) {
      logStatus("⚠️ 尚未选择工作流，后续阶段需手工触发。");
    } else {
      await triggerWorkflow(selectedWorkflowId, workflowPayload);
    }
    mapBoundsLocked = false;
    tingHeadings.clear();
    lastPositions.clear();
    configState = hydrateConfig(payload);
    renderConfigForm();
    hideConfigModal();
    startPolling();
  } catch (err) {
    logStatus(`启动任务失败：${err.message}`);
  }
}

function startPolling() {
  if (statusTimer) {
    clearInterval(statusTimer);
  }
  if (!currentTaskId) {
    return;
  }
  refreshStatus();
  statusTimer = setInterval(refreshStatus, 1000);
}

async function refreshStatus() {
  if (!currentTaskId) {
    return;
  }
  try {
    const resp = await fetch(`/api/status?task_id=${currentTaskId}`);
    if (!resp.ok) {
      throw new Error("获取状态失败");
    }
    const data = await resp.json();
    renderMap(data);
    updateInfoPanel(data);
    if (selectedWorkflowId) {
      await refreshWorkflowRuns(selectedWorkflowId);
    }
  } catch (err) {
    logStatus(`刷新状态失败：${err.message}`);
  }
}
async function refreshWorkflows() {
  try {
    const resp = await fetch("/api/workflows");
    if (!resp.ok) {
      throw new Error("无法获取工作流");
    }
    const list = await resp.json();
    workflowList = Array.isArray(list?.items) ? list.items : Array.isArray(list) ? list : [];
    const select = document.getElementById("workflow-list");
    select.innerHTML = "";
    const emptyOption = document.createElement("option");
    emptyOption.value = "";
    emptyOption.textContent = workflowList.length ? "请选择工作流" : "暂无工作流";
    select.appendChild(emptyOption);
    workflowList.forEach((wf) => {
      const opt = document.createElement("option");
      const id = wf.WorkflowID || wf.workflowId || wf.id || wf.ID;
      opt.value = id || "";
      const name = wf.name || wf.WorkflowName || wf.workflowName || wf.Name;
      opt.textContent = name ? `${name} (${id || "未命名"})` : id || "未命名工作流";
      select.appendChild(opt);
    });
  } catch (err) {
    logStatus(`刷新工作流列表失败：${err.message}`);
  }
}

async function refreshWorkflowRuns(workflowId) {
  try {
    const resp = await fetch(`/api/workflows/${workflowId}/runs`);
    if (!resp.ok) {
      throw new Error("无法获取工作流运行状态");
    }
    const runs = await resp.json();
    const latest = Array.isArray(runs) ? runs[0] : runs?.items?.[0];
    if (latest) {
      if (latest.runId && latest.runId !== activeRunId) {
        activeRunId = latest.runId;
      }
      if (latest.status) {
        logStatus(`工作流 ${workflowId} 状态：${latest.status}`);
      }
      if (activeRunId) {
        await refreshWorkflowRunStatus(workflowId, activeRunId);
      }
    }
  } catch (err) {
    logStatus(`获取工作流运行状态失败：${err.message}`);
  }
}

async function triggerWorkflow(workflowId, payload) {
  try {
    logStatus(`触发工作流 ${workflowId} 运行...`);
    // 添加 stageControlBase，指向当前 FSL_MainControl 服务地址
    const workflowPayload = {
      taskPayload: {
        ...(payload || {}),
        stageControlBase: window.location.origin, // 使用当前页面的 origin 作为 FSL_MainControl 地址
      },
    };
    const resp = await fetch(`/api/workflows/${workflowId}/run`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(workflowPayload),
    });
    if (!resp.ok) {
      let detail = `触发工作流失败 (HTTP ${resp.status})`;
      try {
        const err = await resp.json();
        detail = err.detail || JSON.stringify(err);
      } catch (_) {
        detail = await resp.text();
      }
      throw new Error(detail);
    }
    const result = await resp.json();
    if (result.runId) {
      activeRunId = result.runId;
    }
    logStatus(`工作流已触发：${result.runId || JSON.stringify(result)}`);
  } catch (err) {
    logStatus(`触发工作流失败：${err.message}`);
  }
}

async function refreshWorkflowRunStatus(workflowId, runId) {
  try {
    const resp = await fetch(`/api/workflows/${workflowId}/runs/${runId}/status`);
    if (!resp.ok) {
      throw new Error("无法获取运行节点状态");
    }
    const detail = await resp.json();
    if (detail.nodes) {
      const entries = Object.entries(detail.nodes)
        .map(([nodeId, state]) => `${nodeId}: ${state}`)
        .join(", ");
      logStatus(`运行 ${runId} 节点状态：${entries || "暂无"}`);
    }
  } catch (err) {
    logStatus(`获取运行节点状态失败：${err.message}`);
  }
}

function stageLabel(stage) {
  const mapping = {
    sweep_pending: "待扫雷",
    sweep_running: "扫雷中",
    investigate_pending: "待查证",
    investigate_running: "查证中",
    destroy_pending: "待灭雷",
    destroy_running: "灭雷中",
    evaluate_pending: "待评估",
    evaluate_running: "评估中",
    completed: "已完成",
  };
  return mapping[stage] || stage || "未启动";
}

function updateInfoPanel(status) {
  const stageText = stageLabel(status.stage);
  document.getElementById("status-stage").textContent = stageText;

  const suspectCount = status.suspect_mines?.length || 0;
  const confirmedCount = status.confirmed_mines?.length || 0;
  const destroyedCount = status.destroyed_mines?.length || 0;

  document.getElementById("status-suspect").textContent = suspectCount;
  document.getElementById("status-confirmed").textContent = confirmedCount;
  document.getElementById("status-destroyed").textContent = destroyedCount;

  renderTimeline(status.timeline || []);
  renderServiceCalls(status.service_calls || []);

  if (lastStage !== status.stage) {
    lastStage = status.stage;
  }
}

function renderTimeline(timeline) {
  const list = document.getElementById("timeline");
  list.innerHTML = "";
  timeline
    .sort((a, b) => a.timestamp - b.timestamp)
    .forEach((item) => {
      const li = document.createElement("li");
      const time = formatTimestamp(item.timestamp);
      li.textContent = `[${time}] ${item.message || item.stage}`;
      list.appendChild(li);
    });
}

function formatTimestamp(value) {
  if (!value) {
    return "--:--:--";
  }
  if (typeof value === "number") {
    return new Date(value * 1000).toLocaleTimeString();
  }
  return new Date(value).toLocaleTimeString();
}

function renderServiceCalls(serviceCalls) {
  const container = document.getElementById("service-calls-list");
  if (!container) return;
  
  container.innerHTML = "";
  
  if (!serviceCalls || serviceCalls.length === 0) {
    container.innerHTML = '<div class="service-call-empty">暂无服务调用记录</div>';
    return;
  }
  
  serviceCalls.forEach((call, index) => {
    const callDiv = document.createElement("div");
    callDiv.className = "service-call-item";
    
    const statusClass = call.error ? "service-call-error" : 
                       call.status_code >= 300 ? "service-call-warning" : 
                       "service-call-success";
    
    const time = formatTimestamp(call.timestamp);
    const duration = call.duration_ms ? `${call.duration_ms.toFixed(2)} ms` : "N/A";
    
    // 构建 endpoint 地址显示（只显示 nodeId，不显示 instanceId）
    let endpointAddr = "";
    if (call.endpoint_info) {
      const ep = call.endpoint_info;
      endpointAddr = `${ep.ip || "N/A"}:${ep.port || "N/A"}`;
      if (ep.nodeId) {
        endpointAddr += ` (${ep.nodeId})`;
      }
    } else {
      // 如果没有 endpoint_info，尝试从 endpoint 字段解析（向后兼容）
      endpointAddr = call.endpoint || "";
    }
    
    callDiv.innerHTML = `
      <div class="service-call-header ${statusClass}">
        <span class="service-call-endpoint">${call.endpoint || ""}</span>
        <span class="service-call-address">${endpointAddr}</span>
        <span class="service-call-time">${time}</span>
        <span class="service-call-duration">${duration}</span>
      </div>
    `;
    
    container.appendChild(callDiv);
  });
}

function computeHeading(prev, curr) {
  if (!prev || !curr) return null;
  const dLat = curr.lat - prev.lat;
  const dLon = curr.lon - prev.lon;
  if (Math.abs(dLat) < 1e-7 && Math.abs(dLon) < 1e-7) {
    return null;
  }
  const angle = Math.atan2(dLon, dLat) * (180 / Math.PI);
  return angle;
}

function renderMap(status) {
  if (!map) {
    map = L.map("map").setView(MAP_CENTER, DEFAULT_ZOOM);
    let fallbackAttached = false;
    const offlineLayer = L.tileLayer(OFFLINE_TILE_URL, {
      maxZoom: 18,
      minZoom: 5,
      errorTileUrl: "/static/tiles/blank.png",
    });
    offlineLayer.on("tileerror", () => {
      if (!fallbackAttached) {
        fallbackAttached = true;
        L.tileLayer(ONLINE_TILE_URL, {
          attribution: TILE_ATTRIBUTION,
          maxZoom: 19,
        }).addTo(map);
      }
    });
    offlineLayer.addTo(map);
    map.on("movestart", () => {
      mapBoundsLocked = true;
    });
  }

  if (markersLayer) {
    markersLayer.clearLayers();
  } else {
    markersLayer = L.layerGroup().addTo(map);
  }
  if (tracksLayer) {
    tracksLayer.clearLayers();
  } else {
    tracksLayer = L.layerGroup().addTo(map);
  }

  const taskArea = status.plan?.work_zones?.length
    ? status.plan.summary.task_area
    : status.config.task_area;

  if (taskArea) {
    const bounds = [
      [taskArea.top_left.lat, taskArea.top_left.lon],
      [taskArea.bottom_right.lat, taskArea.bottom_right.lon],
    ];
    L.rectangle(bounds, { color: "#0277bd", weight: 1 }).addTo(markersLayer);
    if (!mapBoundsLocked) {
      map.fitBounds(bounds, { padding: [20, 20] });
    }
  }

  const workZones = status.plan?.work_zones || [];
  workZones.forEach((zone) => {
    const rect = [
      [zone.top_left.lat, zone.top_left.lon],
      [zone.bottom_right.lat, zone.bottom_right.lon],
    ];
    L.rectangle(rect, { color: "#26a69a", weight: 1, dashArray: "4,3" }).addTo(markersLayer);
  });

  (status.suspect_mines || []).forEach((mine) => {
    const marker = L.circleMarker([mine.position.lat, mine.position.lon], {
      radius: 6,
      color: "#fdd835",
      fillColor: "#f9a825",
      fillOpacity: 0.9,
    });
    marker.bindTooltip(`疑似水雷 ${mine.id}`);
    marker.addTo(markersLayer);
  });

  (status.confirmed_mines || []).forEach((mine) => {
    const marker = L.circleMarker([mine.position.lat, mine.position.lon], {
      radius: 6,
      color: "#d81b60",
      fillColor: "#f48fb1",
      fillOpacity: 0.9,
    });
    marker.bindTooltip(`确认水雷 ${mine.id}`);
    marker.addTo(markersLayer);
  });

  const evaluationMap = {};
  (status.evaluated_mines || []).forEach((mine) => {
    if (mine.id) {
      evaluationMap[mine.id] = mine.evaluation_score ?? mine.score ?? mine.percent;
    }
  });

  (status.destroyed_mines || []).forEach((mine) => {
    const marker = L.circleMarker([mine.position.lat, mine.position.lon], {
      radius: 7,
      color: "#4caf50",
      fillColor: "#81c784",
      fillOpacity: 0.9,
    });
    const score =
      evaluationMap[mine.id] ?? mine.evaluation_score ?? mine.score ?? mine.percent;
    if (score !== undefined) {
      const formatted = Number(score).toFixed(0);
      marker.bindTooltip(
        `毁伤程度: ${formatted}%`,
        { permanent: true, direction: "right", offset: [12, 0], opacity: 0.85 }
      );
    } else {
      marker.bindTooltip(`已销毁 ${mine.id}`);
    }
    marker.addTo(markersLayer);
  });

  const tracksByTing = {};
  (status.tracks || []).forEach((track) => {
    const id = track.ting_id;
    if (!tracksByTing[id]) {
      tracksByTing[id] = [];
    }
    tracksByTing[id].push([track.position.lat, track.position.lon]);
  });
  const smoothedTracksByTing = {};
  Object.entries(tracksByTing).forEach(([id, polyline]) => {
    const smoothed = smoothTrackPoints(polyline, 6).map(toArrayPoint);
    smoothedTracksByTing[id] = smoothed;
    if (smoothed.length > 1) {
      L.polyline(smoothed, { color: "#7e57c2", weight: 2, opacity: 0.7 }).addTo(tracksLayer);
    }
  });

  const trackHeadings = {};
  Object.entries(smoothedTracksByTing).forEach(([id, polyline]) => {
    if (polyline.length >= 2) {
      const prevPoint = polyline[polyline.length - 2];
      const currPoint = polyline[polyline.length - 1];
      const heading = computeHeading(
        { lat: prevPoint[0], lon: prevPoint[1] },
        { lat: currPoint[0], lon: currPoint[1] }
      );
      if (heading !== null) {
        trackHeadings[id] = heading;
      }
    }
  });

  const activeIds = new Set((status.tings || []).map((ting) => ting.id));

  (status.tings || []).forEach((ting, index) => {
    const currentPos = { lat: ting.position.lat, lon: ting.position.lon };
    let heading = trackHeadings[ting.id];
    if (heading === undefined || heading === null) {
      const previousPos = lastPositions.get(ting.id);
      const computed = computeHeading(previousPos, currentPos);
      if (computed !== null) {
        heading = computed;
      }
    }
    if (heading === undefined || heading === null) {
      heading = tingHeadings.get(ting.id) ?? 0;
    } else {
      tingHeadings.set(ting.id, heading);
    }
    lastPositions.set(ting.id, currentPos);

    const displayLabel = ting.label && /^\d+$/.test(ting.label)
      ? String(ting.label).padStart(3, "0")
      : String(index + 1).padStart(3, "0");

    const sonarRange =
      Number.isFinite(ting.sonar_range_m)
        ? ting.sonar_range_m
        : Number.isFinite(ting.sonar_range)
        ? ting.sonar_range
        : 0;
    if (sonarRange > 0) {
      L.circle([ting.position.lat, ting.position.lon], {
        radius: sonarRange,
        color: "#1e90ff",
        weight: 1,
        opacity: 0.5,
        fillColor: "#1e90ff",
        fillOpacity: 0.08,
        interactive: false,
      }).addTo(markersLayer);
    }

    const marker = L.marker([ting.position.lat, ting.position.lon], {
      icon: createTingIcon(heading, displayLabel),
    });
    marker.bindTooltip(`${ting.name || ting.id}`, { direction: "top", offset: [0, -12] });
    marker.addTo(markersLayer);
  });

  [...tingHeadings.keys()].forEach((id) => {
    if (!activeIds.has(id)) {
      tingHeadings.delete(id);
      lastPositions.delete(id);
    }
  });
}

document.getElementById("btn-load-defaults").addEventListener("click", loadDefaults);
document.getElementById("btn-start-task").addEventListener("click", startTask);
document.getElementById("btn-refresh-workflows").addEventListener("click", refreshWorkflows);
document.getElementById("workflow-list").addEventListener("change", (e) => {
  selectedWorkflowId = e.target.value || null;
});

const openConfigBtn = document.getElementById("btn-open-config");
if (openConfigBtn) {
  openConfigBtn.addEventListener("click", async () => {
    if (!configState) {
      await loadDefaults();
    } else {
      renderConfigForm();
    }
    if (configState) {
      showConfigModal();
    }
  });
}

const modalCloseBtn = document.getElementById("btn-config-close");
if (modalCloseBtn) {
  modalCloseBtn.addEventListener("click", hideConfigModal);
}

const modalCloseFooterBtn = document.getElementById("btn-config-close-footer");
if (modalCloseFooterBtn) {
  modalCloseFooterBtn.addEventListener("click", hideConfigModal);
}

const configModal = document.getElementById("config-modal");
if (configModal) {
  configModal.addEventListener("click", (event) => {
    if (event.target.classList.contains("modal-overlay")) {
      hideConfigModal();
    }
  });
}

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    hideConfigModal();
  }
});

loadDefaults();
refreshWorkflows();


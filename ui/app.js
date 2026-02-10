const input = document.getElementById("roomInput");
const generateBtn = document.getElementById("generateBtn");
const fillExampleBtn = document.getElementById("fillExampleBtn");
const resultCard = document.getElementById("resultCard");
const resultInput = document.getElementById("resultInput");
const copyBtn = document.getElementById("copyBtn");
const openLink = document.getElementById("openLink");
const statusText = document.getElementById("statusText");
const resultState = document.getElementById("resultState");
const infoName = document.getElementById("infoName");
const infoVersion = document.getElementById("infoVersion");
const infoAuthor = document.getElementById("infoAuthor");
const infoRepo = document.getElementById("infoRepo");

const exampleURL = "https://live.bilibili.com/12345";
let watchSource;
let pollTimer;

function setStatus(text, type) {
  statusText.textContent = text || "";
  resultState.textContent = text || "等待中";
  resultState.className = "badge";
  if (type === "ready") {
    resultState.classList.add("ready");
  }
  if (type === "error") {
    resultState.classList.add("error");
  }
}

function toRoomID(raw) {
  const value = raw.trim();
  if (!value) {
    return { error: "请输入直播间链接或房间号" };
  }
  if (/^\d+$/.test(value)) {
    return { roomID: value };
  }
  try {
    const normalized = /^https?:\/\//i.test(value) ? value : `https://${value}`;
    const parsed = new URL(normalized);
    const host = parsed.hostname.toLowerCase();
    if (host === "b23.tv") {
      return { error: "暂不支持短链，请使用完整直播间链接" };
    }
    if (!host.endsWith("bilibili.com")) {
      return { error: "仅支持B站直播间URL或房间号" };
    }
    const parts = parsed.pathname.split("/").filter(Boolean);
    const candidates = parts.filter((part) => /^\d+$/.test(part));
    if (candidates.length === 0) {
      return { error: "无法解析房间号" };
    }
    return { roomID: candidates[0] };
  } catch (error) {
    return { error: "链接格式不正确" };
  }
}

function buildLink(roomID) {
  const origin = window.location.origin;
  const url = new URL("/live.m3u8", origin);
  url.searchParams.set("room_id", roomID);
  return url.toString();
}

function showResult(link) {
  resultInput.value = link;
  openLink.href = link;
  resultCard.classList.add("active");
}

function hideResult() {
  resultInput.value = "";
  openLink.href = "#";
  resultCard.classList.remove("active");
}

function setInfo(data) {
  if (!data) {
    return;
  }
  if (infoName) {
    infoName.textContent = data.name || "-";
  }
  if (infoVersion) {
    infoVersion.textContent = data.version || "-";
  }
  if (infoAuthor) {
    infoAuthor.textContent = data.author || "-";
  }
  if (infoRepo) {
    const repo = data.repo || "";
    infoRepo.textContent = repo || "-";
    infoRepo.href = repo || "#";
  }
}

function applyStreamState(state) {
  if (!state) {
    return;
  }
  let type = "";
  if (state.state === "ready") {
    type = "ready";
  }
  if (["error", "locked", "hidden", "offline", "loop"].includes(state.state)) {
    type = "error";
  }
  setStatus(state.message || "等待中", type);
}

function stopWatch() {
  if (watchSource) {
    watchSource.close();
    watchSource = null;
  }
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

function watchStream(roomID) {
  stopWatch();
  if (window.EventSource) {
    watchSource = new EventSource(`/api/watch?room_id=${encodeURIComponent(roomID)}`);
    watchSource.addEventListener("status", (event) => {
      try {
        const payload = JSON.parse(event.data);
        applyStreamState(payload);
      } catch (error) {
        setStatus("状态解析失败", "error");
      }
    });
    watchSource.addEventListener("ready", () => {
      stopWatch();
    });
    watchSource.addEventListener("stop", () => {
      stopWatch();
    });
    watchSource.onerror = () => {
      stopWatch();
    };
    return;
  }

  pollTimer = setInterval(async () => {
    const state = await requestStatus(roomID, true);
    if (state && state.state === "ready") {
      stopWatch();
    }
  }, 2000);
}

async function requestStatus(roomID, silent) {
  try {
    const resp = await fetch(`/api/status?room_id=${encodeURIComponent(roomID)}`);
    let data;
    try {
      data = await resp.json();
    } catch (error) {
      data = { state: "error", message: resp.statusText || "状态获取失败" };
    }
    if (!silent) {
      applyStreamState(data);
    }
    if (data.state === "waiting" || data.state === "loading") {
      watchStream(roomID);
    }
    return data;
  } catch (error) {
    if (!silent) {
      setStatus("状态获取失败", "error");
    }
    return null;
  }
}

async function loadInfo() {
  try {
    const resp = await fetch("/api");
    if (!resp.ok) {
      return;
    }
    const data = await resp.json();
    setInfo(data);
  } catch (error) {
  }
}

generateBtn.addEventListener("click", async () => {
  generateBtn.disabled = true;
  setStatus("解析中");
  const { roomID, error } = toRoomID(input.value);
  if (error) {
    setStatus(error, "error");
    generateBtn.disabled = false;
    hideResult();
    return;
  }
  setStatus("检查中");
  const state = await requestStatus(roomID, false);
  if (!state) {
    hideResult();
    generateBtn.disabled = false;
    return;
  }
  if (["waiting", "loading", "ready"].includes(state.state)) {
    const link = buildLink(roomID);
    showResult(link);
  } else {
    hideResult();
  }
  generateBtn.disabled = false;
});

fillExampleBtn.addEventListener("click", () => {
  input.value = exampleURL;
  setStatus("");
});

copyBtn.addEventListener("click", async () => {
  const value = resultInput.value;
  if (!value) {
    setStatus("没有可复制的链接", "error");
    return;
  }
  try {
    await navigator.clipboard.writeText(value);
    setStatus("已复制", "ready");
  } catch (error) {
    setStatus("复制失败", "error");
  }
});

loadInfo();

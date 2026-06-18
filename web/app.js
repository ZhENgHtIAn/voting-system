const topicContainer = document.getElementById("topics");
const errorContainer = document.getElementById("error");

async function fetchResults() {
  const resp = await fetch("/api/results", { method: "GET" });
  if (!resp.ok) {
    throw new Error(`获取结果失败: HTTP ${resp.status}`);
  }
  return resp.json();
}

async function submitVote(topicName) {
  const resp = await fetch("/api/vote", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ topic_name: topicName }),
  });
  if (!resp.ok) {
    const errorBody = await resp.json().catch(() => ({ error: "投票失败" }));
    throw new Error(errorBody.error || `投票失败: HTTP ${resp.status}`);
  }
  return resp.json();
}

function renderResults(results) {
  topicContainer.innerHTML = "";
  const topicNames = Object.keys(results).sort();
  topicNames.forEach((name) => {
    const row = document.createElement("div");
    row.className = "topic-row";
    row.innerHTML = `
      <div>
        <strong>${name}</strong>
        <div>当前票数: ${results[name]}</div>
      </div>
    `;

    const button = document.createElement("button");
    button.textContent = `投票 ${name}`;
    button.addEventListener("click", async () => {
      clearError();
      try {
        const data = await submitVote(name);
        renderResults(data.results || {});
      } catch (err) {
        showError(err.message);
      }
    });
    row.appendChild(button);
    topicContainer.appendChild(row);
  });
}

function showError(message) {
  errorContainer.textContent = message;
}

function clearError() {
  errorContainer.textContent = "";
}

async function bootstrap() {
  clearError();
  try {
    const data = await fetchResults();
    renderResults(data.results || {});
  } catch (err) {
    showError(err.message);
  }
}

bootstrap();

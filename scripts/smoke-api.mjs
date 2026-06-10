import { mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

const baseUrl = process.env.API_BASE_URL ?? "http://localhost:8080/api/v1";
const teamleadLogin = process.env.SMOKE_TEAMLEAD_LOGIN ?? "teamlead";
const teamleadPassword = process.env.SMOKE_TEAMLEAD_PASSWORD ?? "demo123";
const startedAt = Date.now();

let cookie = "";

async function main() {
  await login(teamleadLogin, teamleadPassword);

  const happyTrader = await createTrader(`smoke_trader_${startedAt}`, `Smoke_OP_${startedAt}`, "demo123");
  const happyRequisite = await createRequisite(`+7999${String(startedAt).slice(-7)}`, happyTrader.id);

  await login(happyTrader.login, "demo123");
  const happyShiftRequisite = await takeRequisite(happyRequisite.id);
  await addTurnover(happyShiftRequisite.id, 100000);
  const payout = await createPayout(50000);
  await addTransfer(payout.id, happyShiftRequisite.id, 50000);
  await uploadCSV("trader", "inbound", traderCSV(happyTrader.externalWorkerName, "smoke-in-ok", 1000, "hand_success"));
  await uploadCSV("trader", "outbound", traderCSV(happyTrader.externalWorkerName, "smoke-out-ok", 500, "hand_success"));
  const closedShift = await closeShift();
  assert(closedShift.status === "closed", `expected closed shift, got ${closedShift.status}`);

  await login(teamleadLogin, teamleadPassword);
  const mismatchTrader = await createTrader(`smoke_mismatch_${startedAt}`, `Smoke_MIS_${startedAt}`, "demo123");
  const mismatchRequisite = await createRequisite(`+7988${String(startedAt).slice(-7)}`, mismatchTrader.id);

  await login(mismatchTrader.login, "demo123");
  const mismatchShiftRequisite = await takeRequisite(mismatchRequisite.id);
  await addTurnover(mismatchShiftRequisite.id, 200000);
  const mismatchPayout = await createPayout(50000);
  await addTransfer(mismatchPayout.id, mismatchShiftRequisite.id, 50000);
  await uploadCSV("trader", "inbound", traderCSV(mismatchTrader.externalWorkerName, "smoke-in-mismatch", 1000, "hand_success"));
  await uploadCSV("trader", "outbound", traderCSV(mismatchTrader.externalWorkerName, "smoke-out-matched", 500, "hand_success"));

  const inboundRun = await latestReconciliation("trader", "inbound");
  assert(inboundRun.status === "mismatch", `expected inbound mismatch, got ${inboundRun.status}`);
  await acceptMismatch("inbound", inboundRun.id, "Smoke mismatch accepted with required comment");
  const discrepancyShift = await closeShift();
  assert(
    discrepancyShift.status === "closed_with_discrepancy",
    `expected closed_with_discrepancy, got ${discrepancyShift.status}`,
  );

  console.log("smoke ok");
}

async function login(loginName, password) {
  const response = await request("/auth/login", {
    method: "POST",
    body: { login: loginName, password },
    anonymous: true,
  });
  const setCookie = response.headers.get("set-cookie");
  if (setCookie) {
    cookie = setCookie.split(";")[0];
  }
  return response.json();
}

async function createTrader(loginName, externalWorkerName, password) {
  const response = await request("/teamlead/traders", {
    method: "POST",
    body: {
      login: loginName,
      password,
      salaryRateBps: 75,
      externalWorkerName,
    },
  });
  const payload = await response.json();
  return payload.trader;
}

async function createRequisite(phone, assignedTraderId) {
  const response = await request("/teamlead/requisites", {
    method: "POST",
    body: {
      phone,
      methodType: "SBP",
      proxy: "127.0.0.1:9000",
      assignedTraderId,
    },
  });
  const payload = await response.json();
  return payload.requisite;
}

async function takeRequisite(requisiteId) {
  const response = await request(`/trader/requisites/${requisiteId}/take`, {
    method: "POST",
    body: {
      cardNumber: "4111111111111111",
      holderName: "Smoke Trader",
    },
  });
  const payload = await response.json();
  return payload.shiftRequisite;
}

async function addTurnover(shiftRequisiteId, amountMinor) {
  await request("/trader/shift/current/turnovers", {
    method: "POST",
    body: {
      shiftRequisiteId,
      amountMinor,
      comment: "Smoke turnover",
    },
  });
}

async function createPayout(amountMinor) {
  const response = await request("/trader/payouts", {
    method: "POST",
    body: {
      destinationBank: "Smoke Bank",
      destinationRequisite: "2200000000000000",
      amountMinor,
    },
  });
  const payload = await response.json();
  return payload.payout;
}

async function addTransfer(payoutId, sourceShiftRequisiteId, amountMinor) {
  await request(`/trader/payouts/${payoutId}/transfers`, {
    method: "POST",
    body: {
      sourceShiftRequisiteId,
      amountMinor,
      comment: "Smoke transfer",
    },
  });
}

async function uploadCSV(scope, direction, content) {
  const dir = await mkdtemp(join(tmpdir(), "astra-crm-smoke-"));
  const filePath = join(dir, `${scope}-${direction}.csv`);
  await writeFile(filePath, content);
  const form = new FormData();
  form.append("file", new Blob([content], { type: "text/csv" }), `${scope}-${direction}.csv`);
  await request(`/${scope}/${direction}/import`, {
    method: "POST",
    form,
  });
}

async function latestReconciliation(scope, direction) {
  const response = await request(`/${scope}/${direction}/reconciliation/latest`);
  const payload = await response.json();
  return payload.run;
}

async function acceptMismatch(direction, runId, comment) {
  await request(`/trader/${direction}/reconciliation/${runId}/accept`, {
    method: "POST",
    body: { comment },
  });
}

async function closeShift() {
  const response = await request("/trader/shift/current/close", {
    method: "POST",
    body: {},
  });
  const payload = await response.json();
  return payload.shift;
}

async function request(path, options = {}) {
  const headers = {
    Accept: "application/json",
    ...(options.headers ?? {}),
  };
  let body;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  const response = await fetch(`${baseUrl}${path}`, {
    method: options.method ?? "GET",
    headers: cookie && !options.anonymous ? { ...headers, Cookie: cookie } : headers,
    body,
  });

  if (!response.ok) {
    let payload;
    try {
      payload = await response.json();
    } catch {
      payload = await response.text();
    }
    throw new Error(`${options.method ?? "GET"} ${path} failed with ${response.status}: ${JSON.stringify(payload)}`);
  }

  return response;
}

function traderCSV(workerName, innerId, amount, status) {
  return [
    "id|innerId|requisite|requisitePhone|deviceName|methodType|methodName|amount|courseWorker|currency|status|createdAt|closedAt|updatedAt|oldAmount|receipt|orderComment|requisiteId|workerName|workerAmount|workerProfit|ordered|counted|initials",
    `${innerId}|${innerId}|79991111111|79991111111|device-1|СБП|sbp|${amount}.0|91.20|RUB|${status}|28.05.2026 14:00:00|28.05.2026 14:05:00|28.05.2026 14:05:00|None|None|smoke|req-1|${workerName}|${amount}.0|0.0|true|true|SM`,
    "",
  ].join("\n");
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});

import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

const apiBaseURL = process.env.E2E_API_BASE_URL ?? "http://localhost:8080/api/v1";
const teamleadLogin = process.env.E2E_TEAMLEAD_LOGIN ?? "teamlead";
const teamleadPassword = process.env.E2E_TEAMLEAD_PASSWORD ?? "demo123";
const traderPassword = "demo123";

test.describe.configure({ mode: "serial" });

test("teamlead creates trader and requisite, trader closes matched shift", async ({ page }) => {
  const id = uniqueId();
  const traderLogin = `e2e_trader_${id}`;
  const workerName = `E2E_OP_${id}`;
  const phone = `+7977${id.slice(-7)}`;

  await loginUI(page, teamleadLogin, teamleadPassword, /\/teamlead\/dashboard/);
  await page.goto("/teamlead/traders");
  await expect(page.getByRole("heading", { name: "Сотрудники" })).toBeVisible();

  await createTraderUI(page, { login: traderLogin, workerName });
  await expect(page.getByText(traderLogin)).toBeVisible();

  await page.goto("/teamlead/requisites");
  await expect(page.getByRole("heading", { name: "Реквизиты" })).toBeVisible();
  await createRequisiteUI(page, { phone, traderLogin });
  await expect(page.getByText(phone)).toBeVisible();

  await logoutUI(page, teamleadLogin);
  await loginUI(page, traderLogin, traderPassword, /\/trader\/requisites/);
  await expect(page.getByRole("heading", { name: "Мои реквизиты" })).toBeVisible();

  await takeRequisiteUI(page);
  await addTurnoverUI(page, "1000", "E2E turnover");
  await createPaidPayoutUI(page, "500", "E2E Bank", "2200000000000000");

  await page.goto("/trader/inbound");
  await importCsvUI(page, traderCSV(workerName, `e2e-in-ok-${id}`, 1000, "hand_success"), `inbound-${id}.csv`);
  await expect(page.getByText(`e2e-in-ok-${id}`)).toBeVisible();

  await page.goto("/trader/outbound");
  await importCsvUI(page, traderCSV(workerName, `e2e-out-ok-${id}`, 500, "hand_success"), `outbound-${id}.csv`);
  await expect(page.getByText(`e2e-out-ok-${id}`)).toBeVisible();

  await page.goto("/trader/requisites");
  await closeShiftUI(page);
  await expect(page.getByText("Закрыта").first()).toBeVisible();
});

test("trader accepts inbound mismatch with comment and closes shift with discrepancy", async ({ page, request }) => {
  const id = uniqueId();
  const setup = await prepareMismatchShift(request, id);

  await loginUI(page, setup.traderLogin, traderPassword, /\/trader\/requisites/);
  await page.goto("/trader/inbound");
  await expect(page.getByText("Есть расхождение сверки")).toBeVisible();

  await page.getByRole("button", { name: "Подтвердить расхождение" }).click();
  const dialog = page.getByRole("dialog", { name: "Подтвердить расхождение" });
  await dialog.locator("textarea").fill("E2E accepted mismatch with required comment");
  await dialog.getByRole("button", { name: "Подтвердить" }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByText("Есть расхождение сверки")).toBeHidden();

  await page.goto("/trader/requisites");
  await closeShiftUI(page);
  await expect(page.getByText("С расхождением").first()).toBeVisible();
});

async function loginUI(page: Page, login: string, password: string, targetURL: RegExp) {
  await page.goto("/login");
  await page.getByLabel("Логин").fill(login);
  await page.getByLabel("Пароль").fill(password);
  await page.getByRole("button", { name: "Войти" }).click();
  await expect(page).toHaveURL(targetURL);
}

async function logoutUI(page: Page, login: string) {
  await page.getByRole("button", { name: new RegExp(`${escapeRegExp(login)}.*Выйти`) }).click();
  await expect(page).toHaveURL(/\/login/);
}

async function createTraderUI(page: Page, input: { login: string; workerName: string }) {
  await page.getByRole("button", { name: "Добавить трейдера" }).click();
  const dialog = page.getByRole("dialog", { name: "Добавить трейдера" });
  const fields = dialog.locator("input");
  await fields.nth(0).fill(input.login);
  await fields.nth(1).fill(traderPassword);
  await fields.nth(2).fill(input.workerName);
  await fields.nth(3).fill("0.75");
  await dialog.getByRole("button", { name: "Сохранить" }).click();
  await expect(dialog).toBeHidden();
}

async function createRequisiteUI(page: Page, input: { phone: string; traderLogin: string }) {
  await page.getByRole("button", { name: "Добавить реквизит" }).click();
  const dialog = page.getByRole("dialog", { name: "Добавить реквизит" });
  await dialog.locator("input").nth(0).fill(input.phone);
  await dialog.locator("input").nth(1).fill("127.0.0.1:9000");
  await dialog.locator("select").nth(1).selectOption({ label: input.traderLogin });
  await dialog.getByRole("button", { name: "Сохранить" }).click();
  await expect(dialog).toBeHidden();
}

async function takeRequisiteUI(page: Page) {
  await page.getByRole("button", { name: "В работу" }).first().click();
  const dialog = page.getByRole("dialog", { name: "Взять реквизит в работу" });
  await dialog.locator("input").nth(0).fill("4111111111111111");
  await dialog.locator("input").nth(1).fill("E2E Trader");
  await dialog.getByRole("button", { name: "Сохранить" }).click();
  await expect(dialog).toBeHidden();
}

async function addTurnoverUI(page: Page, amount: string, comment: string) {
  await page.getByRole("button", { name: "Оборот" }).first().click();
  const dialog = page.getByRole("dialog", { name: "Добавить оборот" });
  await dialog.locator("input").fill(amount);
  await dialog.locator("textarea").fill(comment);
  await dialog.getByRole("button", { name: "Добавить" }).click();
  await expect(dialog).toBeHidden();
}

async function createPaidPayoutUI(page: Page, amount: string, bank: string, destination: string) {
  await page.goto("/trader/payouts");
  await page.getByRole("button", { name: "Добавить выплату" }).click();
  const createDialog = page.getByRole("dialog", { name: "Ручная выплата" });
  await createDialog.locator("input").nth(0).fill(bank);
  await createDialog.locator("input").nth(1).fill(destination);
  await createDialog.locator("input").nth(2).fill(amount);
  await createDialog.getByRole("button", { name: "Создать" }).click();
  await expect(createDialog).toBeHidden();

  await expect(page.getByText(bank)).toBeVisible();
  await page.getByRole("button", { name: "Действия" }).first().click();
  await page.getByRole("menuitem", { name: "Детали" }).click();
  const detailsDialog = page.getByRole("dialog", { name: "Детали выплаты" });
  await detailsDialog.locator("input").fill(amount);
  await detailsDialog.locator("textarea").fill("E2E transfer");
  await detailsDialog.getByRole("button", { name: "Добавить перевод" }).click();
  await expect(detailsDialog.getByText("E2E transfer")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(detailsDialog).toBeHidden();
}

async function importCsvUI(page: Page, content: string, filename: string) {
  await page.getByRole("button", { name: "Импорт CSV" }).click();
  const dialog = page.getByRole("dialog", { name: "Импорт CSV" });
  await dialog.locator('input[type="file"]').setInputFiles({
    name: filename,
    mimeType: "text/csv",
    buffer: Buffer.from(content),
  });
  await dialog.getByRole("button", { name: "Загрузить" }).click();

  const result = page.getByRole("dialog", { name: "Результат импорта" });
  await expect(result).toBeVisible();
  await result.getByRole("button", { name: "Закрыть" }).first().click();
  await expect(result).toBeHidden();

  if (await dialog.isVisible().catch(() => false)) {
    await dialog.getByRole("button", { name: "Отмена" }).click();
    await expect(dialog).toBeHidden();
  }
}

async function closeShiftUI(page: Page) {
  await page.getByRole("button", { name: "Закрыть смену" }).first().click();
  const dialog = page.getByRole("dialog", { name: "Чеклист закрытия смены" });
  await expect(dialog.getByText("Все проверки пройдены.")).toBeVisible();
  await dialog.getByRole("button", { name: "Закрыть смену" }).first().click();
  await expect(dialog).toBeHidden();
}

async function prepareMismatchShift(request: APIRequestContext, id: string) {
  const cookieJar = new CookieJar();
  await apiLogin(request, cookieJar, teamleadLogin, teamleadPassword);

  const traderLogin = `e2e_mismatch_${id}`;
  const workerName = `E2E_MIS_${id}`;
  const trader = await apiJSON<{ trader: { id: number } }>(request, cookieJar, "/teamlead/traders", {
    method: "POST",
    data: {
      login: traderLogin,
      password: traderPassword,
      salaryRateBps: 75,
      externalWorkerName: workerName,
    },
  });
  const requisite = await apiJSON<{ requisite: { id: number } }>(request, cookieJar, "/teamlead/requisites", {
    method: "POST",
    data: {
      phone: `+7988${id.slice(-7)}`,
      methodType: "SBP",
      proxy: "127.0.0.1:9000",
      assignedTraderId: trader.trader.id,
    },
  });

  await apiLogin(request, cookieJar, traderLogin, traderPassword);
  const shiftRequisite = await apiJSON<{ shiftRequisite: { id: number } }>(
    request,
    cookieJar,
    `/trader/requisites/${requisite.requisite.id}/take`,
    {
      method: "POST",
      data: {
        cardNumber: "4111111111111111",
        holderName: "E2E Mismatch",
      },
    },
  );
  await apiJSON(request, cookieJar, "/trader/shift/current/turnovers", {
    method: "POST",
    data: {
      shiftRequisiteId: shiftRequisite.shiftRequisite.id,
      amountMinor: 200000,
      comment: "E2E mismatch turnover",
    },
  });
  const payout = await apiJSON<{ payout: { id: number } }>(request, cookieJar, "/trader/payouts", {
    method: "POST",
    data: {
      destinationBank: "E2E Bank",
      destinationRequisite: "2200000000000000",
      amountMinor: 50000,
    },
  });
  await apiJSON(request, cookieJar, `/trader/payouts/${payout.payout.id}/transfers`, {
    method: "POST",
    data: {
      sourceShiftRequisiteId: shiftRequisite.shiftRequisite.id,
      amountMinor: 50000,
      comment: "E2E mismatch transfer",
    },
  });
  await apiUpload(request, cookieJar, "/trader/inbound/import", traderCSV(workerName, `e2e-in-mis-${id}`, 1000, "hand_success"));
  await apiUpload(request, cookieJar, "/trader/outbound/import", traderCSV(workerName, `e2e-out-mis-${id}`, 500, "hand_success"));

  return { traderLogin };
}

async function apiLogin(request: APIRequestContext, cookieJar: CookieJar, login: string, password: string) {
  const response = await request.post(`${apiBaseURL}/auth/login`, {
    data: { login, password },
  });
  await expect(response, await response.text()).toBeOK();
  cookieJar.setFromResponse(response.headers()["set-cookie"]);
}

async function apiJSON<T = unknown>(
  request: APIRequestContext,
  cookieJar: CookieJar,
  path: string,
  options: { method?: "GET" | "POST" | "PATCH" | "DELETE"; data?: unknown } = {},
) {
  const response = await request.fetch(`${apiBaseURL}${path}`, {
    method: options.method ?? "GET",
    data: options.data,
    headers: cookieJar.headers(),
  });
  await expect(response, await response.text()).toBeOK();
  if (response.status() === 204) return undefined as T;
  return response.json() as Promise<T>;
}

async function apiUpload(request: APIRequestContext, cookieJar: CookieJar, path: string, content: string) {
  const response = await request.post(`${apiBaseURL}${path}`, {
    headers: cookieJar.headers(),
    multipart: {
      file: {
        name: "orders.csv",
        mimeType: "text/csv",
        buffer: Buffer.from(content),
      },
    },
  });
  await expect(response, await response.text()).toBeOK();
}

class CookieJar {
  private cookie = "";

  setFromResponse(setCookie?: string) {
    if (!setCookie) return;
    this.cookie = setCookie.split(";")[0];
  }

  headers() {
    return this.cookie ? { Cookie: this.cookie } : {};
  }
}

function traderCSV(workerName: string, innerId: string, amount: number, status: string) {
  return [
    "id|innerId|requisite|requisitePhone|deviceName|methodType|methodName|amount|courseWorker|currency|status|createdAt|closedAt|updatedAt|oldAmount|receipt|orderComment|requisiteId|workerName|workerAmount|workerProfit|ordered|counted|initials",
    `${innerId}|${innerId}|79991111111|79991111111|device-1|СБП|sbp|${amount}.0|91.20|RUB|${status}|28.05.2026 14:00:00|28.05.2026 14:05:00|28.05.2026 14:05:00|None|None|e2e|req-1|${workerName}|${amount}.0|0.0|true|true|EE`,
    "",
  ].join("\n");
}

function uniqueId() {
  return `${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

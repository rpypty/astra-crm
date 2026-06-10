import { apiClient, queryString } from "@/lib/api-client";
import type {
  AccountingPeriod,
  AssignmentHistoryItem,
  AuditLogEntry,
  CurrentUser,
  ImportIssue,
  ImportResult,
  Order,
  OrderDirection,
  Payout,
  PayoutTransfer,
  ReconciliationSummary,
  Requisite,
  ShiftRequisite,
  Trader,
  TurnoverEntry,
  UserStatus,
} from "@/lib/domain";

type ListResponse<T> = { items: T[]; page?: number; pageSize?: number; total?: number };
type AuthResponse = { user: CurrentUser };
type TraderResponse = { trader: BackendTrader };
type ResetPasswordResponse = { temporaryPassword: string };
type RequisiteResponse = { requisite: BackendRequisite };
type AssignmentResponse = { assignment: BackendAssignment };
type CurrentShiftResponse = { shift?: BackendShift; checklist?: BackendChecklist };
type ChecklistResponse = { checklist: BackendChecklist };
type TakeRequisiteResponse = { shiftRequisite: BackendShiftRequisite };
type ShiftRequisiteResponse = { shiftRequisite: BackendShiftRequisite };
type TurnoverResponse = { turnover: BackendTurnover };
type PayoutResponse = { payout: BackendPayout };
type PayoutDetailsResponse = { payout: BackendPayout; transfers: BackendTransfer[] };
type TransferResponse = { transfer: BackendTransfer };
type ImportResponse = { result: BackendImportResult };
type ReconciliationResponse = { run: BackendReconciliationRun };

type BackendTrader = Trader & { teamId: number; role: "trader"; createdAt: string; updatedAt: string };

type BackendRequisite = Omit<Requisite, "proxy" | "methodType"> & {
  teamId: number;
  methodType: string;
  proxy?: string;
  createdAt: string;
};

type BackendAssignment = {
  id: number;
  requisiteId: number;
  traderId: number;
  assignedBy: number;
  assignedAt: string;
  unassignedAt?: string;
  comment?: string;
};

type BackendShift = {
  id: number;
  status: "open" | "closing" | "closed" | "closed_with_discrepancy";
  startedAt: string;
  inboundReconciliationStatus: string;
  outboundReconciliationStatus: string;
};

type BackendChecklist = {
  inboundImported: boolean;
  inboundOk: boolean;
  outboundImported: boolean;
  outboundOk: boolean;
  allPayoutsFullyPaid: boolean;
  canClose: boolean;
};

type BackendAssignedRequisite = {
  id: number;
  phone: string;
  methodType: string;
  proxy?: string;
  shiftRequisiteId?: number;
  cardNumber?: string;
  holderName?: string;
  inWork: boolean;
};

type BackendShiftRequisite = {
  id: number;
  requisiteId: number;
  cardNumber: string;
  holderName: string;
  status: string;
};

type BackendTurnover = {
  id: number;
  shiftRequisiteId: number;
  amountMinor: number;
  comment?: string;
  createdAt: string;
};

type BackendPayout = {
  id: number;
  destinationBank: string;
  destinationRequisite: string;
  amountMinor: number;
  paidAmountMinor: number;
  status: string;
  createdAt: string;
};

type BackendTransfer = {
  id: number;
  manualPayoutOrderId: number;
  sourceShiftRequisiteId: number;
  amountMinor: number;
  comment?: string;
  createdAt: string;
};

type BackendOrder = {
  id: number;
  externalId: string;
  externalInnerId: string;
  workerName: string;
  traderLogin?: string;
  requisiteRaw?: string;
  requisitePhone?: string;
  methodType?: string;
  methodName?: string;
  amountMinor: number;
  rawStatus: string;
  normalizedStatus: string;
  createdAtExternal: string;
};

type BackendImportResult = {
  status: string;
  rowsCount: number;
  rawStatusCounts?: Record<string, number>;
  normalizedStatusCounts?: Record<string, number>;
  unknownStatuses?: string[];
};

type BackendReconciliationRun = {
  id: number;
  status: "matched" | "mismatch" | "accepted_with_comment";
  expectedAmountMinor: number;
  actualAmountMinor: number;
  diffAmountMinor: number;
  comment?: string;
};

export const api = {
  auth: {
    login: (input: { login: string; password: string }) => apiClient.post<AuthResponse>("/auth/login", input),
    me: () => apiClient.get<AuthResponse>("/auth/me"),
    logout: () => apiClient.post<void>("/auth/logout"),
  },

  traders: {
    async list(filters?: { search?: string; status?: string }) {
      const response = await apiClient.get<ListResponse<BackendTrader>>("/teamlead/traders");
      return response.items.map(toTrader).filter((trader) => filterTrader(trader, filters));
    },
    async save(input: {
      id?: number;
      login: string;
      password?: string;
      externalWorkerName: string;
      salaryRateBps: number;
      status: UserStatus;
    }) {
      if (input.id) {
        await apiClient.patch<TraderResponse>(`/teamlead/traders/${input.id}`, {
          externalWorkerName: input.externalWorkerName,
          salaryRateBps: input.salaryRateBps,
          status: input.status,
        });
        return;
      }

      await apiClient.post<TraderResponse>("/teamlead/traders", {
        login: input.login,
        password: input.password,
        externalWorkerName: input.externalWorkerName,
        salaryRateBps: input.salaryRateBps,
      });
    },
    async resetPassword(traderId: number) {
      const response = await apiClient.post<ResetPasswordResponse>(`/teamlead/traders/${traderId}/reset-password`);
      return { password: response.temporaryPassword };
    },
    archive: (traderId: number) => apiClient.delete<void>(`/teamlead/traders/${traderId}`),
  },

  requisites: {
    async list(filters?: { search?: string; methodType?: string; status?: string; traderId?: string }) {
      const response = await apiClient.get<ListResponse<BackendRequisite>>("/teamlead/requisites");
      return response.items.map(toRequisite).filter((requisite) => filterRequisite(requisite, filters));
    },
    async save(input: {
      id?: number;
      phone: string;
      methodType: "SBP" | "C2C";
      proxy: string;
      assignedTraderId?: number;
      status: "active" | "archived";
    }) {
      if (input.id) {
        await apiClient.patch<RequisiteResponse>(`/teamlead/requisites/${input.id}`, {
          phone: input.phone,
          methodType: input.methodType,
          proxy: input.proxy,
          status: input.status,
        });
        if (input.assignedTraderId) {
          await apiClient.post<AssignmentResponse>(`/teamlead/requisites/${input.id}/assign`, {
            traderId: input.assignedTraderId,
            comment: "Изменение назначения из CRM",
          });
        }
        return;
      }

      await apiClient.post<RequisiteResponse>("/teamlead/requisites", {
        phone: input.phone,
        methodType: input.methodType,
        proxy: input.proxy,
        assignedTraderId: input.assignedTraderId,
      });
    },
    async history(requisiteId: number) {
      const response = await apiClient.get<ListResponse<BackendAssignment>>(
        `/teamlead/requisites/${requisiteId}/assignment-history`,
      );
      return response.items.map(toAssignmentHistory);
    },
    archive: (requisiteId: number) => apiClient.delete<void>(`/teamlead/requisites/${requisiteId}`),
  },

  traderShift: {
    async current() {
      const [shiftResponse, checklistResponse] = await Promise.all([
        apiClient.get<CurrentShiftResponse>("/trader/shift/current"),
        apiClient.get<ChecklistResponse>("/trader/shift/current/checklist").catch(() => undefined),
      ]);

      return {
        shift: shiftResponse.shift,
        checklist: checklistResponse?.checklist,
      };
    },
    async requisites() {
      const [assignedResponse, turnoversResponse] = await Promise.all([
        apiClient.get<ListResponse<BackendAssignedRequisite>>("/trader/requisites"),
        apiClient.get<ListResponse<BackendTurnover>>("/trader/shift/current/turnovers").catch(() => ({ items: [] })),
      ]);
      const latestTurnovers = latestTurnoverByShiftRequisite(turnoversResponse.items);
      return assignedResponse.items.map((item) => toShiftRequisite(item, latestTurnovers));
    },
    async takeRequisite(input: { shiftRequisiteId: number; cardNumber: string; holderName: string }) {
      await apiClient.post<TakeRequisiteResponse>(`/trader/requisites/${input.shiftRequisiteId}/take`, {
        cardNumber: input.cardNumber,
        holderName: input.holderName,
      });
    },
    async updateDetails(input: { shiftRequisiteId: number; cardNumber: string; holderName: string }) {
      await apiClient.patch<ShiftRequisiteResponse>(`/trader/shift-requisites/${input.shiftRequisiteId}`, {
        cardNumber: input.cardNumber,
        holderName: input.holderName,
      });
    },
    async addTurnover(input: { shiftRequisiteId: number; amountMinor: number; comment?: string }) {
      await apiClient.post<TurnoverResponse>("/trader/shift/current/turnovers", input);
    },
    async turnovers(shiftRequisiteId: number) {
      const response = await apiClient.get<ListResponse<BackendTurnover>>(
        `/trader/shift-requisites/${shiftRequisiteId}/turnovers`,
      );
      return response.items.map(toTurnover);
    },
    async close() {
      await apiClient.post<void>("/trader/shift/current/close", {});
    },
  },

  payouts: {
    async list() {
      const response = await apiClient.get<ListResponse<BackendPayout>>("/trader/payouts");
      return response.items.map(toPayout);
    },
    async transfers(payoutId: number) {
      const response = await apiClient.get<PayoutDetailsResponse>(`/trader/payouts/${payoutId}`);
      return response.transfers.map(toTransfer);
    },
    async create(input: { destinationBank: string; destinationRequisite: string; amountMinor: number }) {
      await apiClient.post<PayoutResponse>("/trader/payouts", input);
    },
    async addTransfer(input: { payoutId: number; sourceShiftRequisiteId: number; amountMinor: number; comment?: string }) {
      await apiClient.post<TransferResponse>(`/trader/payouts/${input.payoutId}/transfers`, {
        sourceShiftRequisiteId: input.sourceShiftRequisiteId,
        amountMinor: input.amountMinor,
        comment: input.comment,
      });
    },
    async deleteTransfer(input: { payoutId: number; transferId: number } | number) {
      if (typeof input === "number") {
        throw new Error("Для удаления перевода нужен payoutId");
      }
      await apiClient.delete<void>(`/trader/payouts/${input.payoutId}/transfers/${input.transferId}`);
    },
  },

  imports: {
    async upload(input: {
      file: File;
      scope: "teamlead" | "trader";
      direction: OrderDirection;
      accountingPeriodId?: number;
    }) {
      const formData = new FormData();
      formData.set("file", input.file);
      if (input.accountingPeriodId) {
        formData.set("accountingPeriodId", String(input.accountingPeriodId));
      }

      const periodId = input.accountingPeriodId ?? Number(import.meta.env.VITE_DEMO_ACCOUNTING_PERIOD_ID ?? 1);
      const path =
        input.scope === "teamlead"
          ? `/teamlead/${input.direction}/import?accountingPeriodId=${periodId}`
          : `/trader/${input.direction}/import`;
      const response = await apiClient.upload<ImportResponse>(path, formData);
      return toImportResult(response.result);
    },
    async acceptMismatch(input: { scope: "teamlead" | "trader"; direction: OrderDirection; runId: number; comment: string }) {
      if (input.scope !== "trader") {
        throw new Error("Подтверждение расхождения периода тимлида пока не реализовано в backend.");
      }
      await apiClient.post<ReconciliationResponse>(`/trader/${input.direction}/reconciliation/${input.runId}/accept`, {
        comment: input.comment,
      });
    },
  },

  orders: {
    async list(scope: "teamlead" | "trader", direction: OrderDirection, filters?: { search?: string; status?: string }) {
      const response = await apiClient.get<ListResponse<BackendOrder>>(
        `/${scope}/${direction}/orders${queryString({ status: filters?.status })}`,
      );
      return response.items.map(toOrder).filter((order) => filterOrder(order, filters));
    },
    async reconciliation(scope: "teamlead" | "trader", direction: OrderDirection) {
      const response = await apiClient.get<ReconciliationResponse>(`/${scope}/${direction}/reconciliation/latest`);
      return toReconciliation(response.run);
    },
  },

  periods: {
    async list(): Promise<AccountingPeriod[]> {
      return apiClient.get<ListResponse<AccountingPeriod>>("/teamlead/periods").then((response) => response.items);
    },
  },

  audit: {
    async list(): Promise<AuditLogEntry[]> {
      return apiClient.get<ListResponse<AuditLogEntry>>("/teamlead/audit").then((response) => response.items);
    },
  },
};

function toTrader(trader: BackendTrader): Trader {
  return {
    id: trader.id,
    login: trader.login,
    externalWorkerName: trader.externalWorkerName,
    salaryRateBps: trader.salaryRateBps,
    assignedRequisitesCount: trader.assignedRequisitesCount ?? 0,
    currentShiftStatus: trader.currentShiftStatus ?? "none",
    status: trader.status,
  };
}

function toRequisite(requisite: BackendRequisite): Requisite {
  return {
    id: requisite.id,
    phone: requisite.phone,
    methodType: requisite.methodType as Requisite["methodType"],
    proxy: requisite.proxy ?? "",
    assignedTraderId: requisite.assignedTraderId,
    assignedTraderLogin: requisite.assignedTraderLogin,
    status: requisite.status as Requisite["status"],
    updatedAt: requisite.updatedAt,
  };
}

function toAssignmentHistory(assignment: BackendAssignment): AssignmentHistoryItem {
  return {
    id: assignment.id,
    changedAt: assignment.assignedAt,
    newTrader: String(assignment.traderId),
    changedBy: String(assignment.assignedBy),
    comment: assignment.comment ?? "",
  };
}

function toShiftRequisite(item: BackendAssignedRequisite, latestTurnovers: Map<number, number>): ShiftRequisite {
  const shiftRequisiteId = item.shiftRequisiteId ?? item.id;
  return {
    id: shiftRequisiteId,
    requisiteId: item.id,
    phone: item.phone,
    methodType: item.methodType as ShiftRequisite["methodType"],
    proxy: item.proxy ?? "",
    cardNumber: item.cardNumber,
    holderName: item.holderName,
    latestTurnoverMinor: latestTurnovers.get(shiftRequisiteId) ?? 0,
    status: item.inWork ? "in_work" : "assigned",
  };
}

function toTurnover(item: BackendTurnover): TurnoverEntry {
  return {
    id: item.id,
    shiftRequisiteId: item.shiftRequisiteId,
    amountMinor: item.amountMinor,
    comment: item.comment,
    createdAt: item.createdAt,
  };
}

function toPayout(item: BackendPayout): Payout {
  return {
    id: item.id,
    destinationBank: item.destinationBank,
    destinationRequisite: item.destinationRequisite,
    amountMinor: item.amountMinor,
    paidMinor: item.paidAmountMinor,
    status: item.status === "paid" ? "paid" : item.status === "cancelled" ? "cancelled" : "open",
    createdAt: item.createdAt,
  };
}

function toTransfer(item: BackendTransfer): PayoutTransfer {
  return {
    id: item.id,
    payoutId: item.manualPayoutOrderId,
    sourceShiftRequisiteId: item.sourceShiftRequisiteId,
    amountMinor: item.amountMinor,
    comment: item.comment,
    createdAt: item.createdAt,
  };
}

function toImportResult(result: BackendImportResult): ImportResult {
  const normalized = result.normalizedStatusCounts ?? {};
  const raw = result.rawStatusCounts ?? {};
  const successCount = (normalized.success ?? 0) + (normalized.corrected ?? 0);
  const failedCount = normalized.failed ?? 0;
  const issues: ImportIssue[] = (result.unknownStatuses ?? []).map((status) => ({
    row: 0,
    message: `Неизвестный статус CSV: ${status}`,
  }));

  return {
    status: result.status === "failed" ? "failed" : "matched",
    importedRows: result.rowsCount,
    successCount,
    failedCount,
    duplicateCount: raw.duplicate ?? 0,
    expectedMinor: 0,
    actualMinor: 0,
    issues,
  };
}

function toOrder(item: BackendOrder): Order {
  return {
    id: String(item.externalId || item.id),
    createdAt: item.createdAtExternal,
    trader: item.traderLogin ?? item.workerName,
    workerName: item.workerName,
    requisite: item.requisitePhone ?? item.requisiteRaw ?? "",
    method: item.methodName ?? item.methodType ?? "",
    bankName: item.methodName ?? "",
    amountMinor: item.amountMinor,
    status: item.rawStatus || item.normalizedStatus,
    innerId: item.externalInnerId,
  };
}

function toReconciliation(run: BackendReconciliationRun): ReconciliationSummary {
  return {
    status: run.status,
    expectedMinor: run.expectedAmountMinor,
    actualMinor: run.actualAmountMinor,
    diffMinor: run.diffAmountMinor,
    comment: run.comment,
    runId: run.id,
  };
}

function latestTurnoverByShiftRequisite(items: BackendTurnover[]) {
  const result = new Map<number, number>();
  for (const item of items) {
    if (!result.has(item.shiftRequisiteId)) {
      result.set(item.shiftRequisiteId, item.amountMinor);
    }
  }
  return result;
}

function filterTrader(trader: Trader, filters?: { search?: string; status?: string }) {
  const search = filters?.search?.trim().toLowerCase();
  const matchesSearch =
    !search || trader.login.toLowerCase().includes(search) || trader.externalWorkerName.toLowerCase().includes(search);
  const matchesStatus = !filters?.status || filters.status === "all" || trader.status === filters.status;
  return matchesSearch && matchesStatus;
}

function filterRequisite(requisite: Requisite, filters?: { search?: string; methodType?: string; status?: string; traderId?: string }) {
  const search = filters?.search?.trim().toLowerCase();
  const matchesSearch = !search || requisite.phone.toLowerCase().includes(search);
  const matchesMethod = !filters?.methodType || filters.methodType === "all" || requisite.methodType === filters.methodType;
  const matchesStatus = !filters?.status || filters.status === "all" || requisite.status === filters.status;
  const matchesTrader =
    !filters?.traderId ||
    filters.traderId === "all" ||
    String(requisite.assignedTraderId ?? "unassigned") === filters.traderId;
  return matchesSearch && matchesMethod && matchesStatus && matchesTrader;
}

function filterOrder(order: Order, filters?: { search?: string; status?: string }) {
  const search = filters?.search?.trim().toLowerCase();
  const matchesSearch =
    !search ||
    [order.id, order.trader, order.workerName, order.requisite, order.innerId].some((value) =>
      value.toLowerCase().includes(search),
    );
  const matchesStatus = !filters?.status || filters.status === "all" || order.status === filters.status;
  return matchesSearch && matchesStatus;
}

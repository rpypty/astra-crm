import { ApiError, apiClient, queryString } from "@/shared/api/api-client";
import type { ApiSchema } from "@/shared/api/generated/openapi";
import type {
  AccountingPeriod,
  AssignmentHistoryItem,
  AuditLogEntry,
  Bank,
  CurrentUser,
  ImportIssue,
  ImportResult,
  Order,
  OrderDashboard,
  OrderDirection,
  OrderFilters,
  Payout,
  PayoutTransfer,
  ReconciliationItem,
  ReconciliationSummary,
  Requisite,
  RequisiteAssignmentEvent,
  RequisiteAssignmentWorkRow,
  RequisiteReport,
  RequisiteReportShift,
  RequisiteReportSummary,
  ShiftReportDetails,
  ShiftReportReconciliation,
  ShiftReportRow,
  ShiftReport,
  ShiftRequisite,
  Trader,
  TraderProfile,
  TurnoverEntry,
  UserStatus,
} from "@/shared/model/domain";
import type { PeriodFilter } from "@/shared/lib/period-filter";

type AuthResponse = ApiSchema<"AuthResponse">;
type BanksListResponse = ApiSchema<"BanksListResponse">;
type TradersListResponse = ApiSchema<"TradersListResponse">;
type TraderResponse = ApiSchema<"TraderResponse">;
type ResetPasswordResponse = ApiSchema<"ResetTraderPasswordResponse">;
type RequisitesListResponse = ApiSchema<"RequisitesListResponse">;
type RequisiteResponse = ApiSchema<"RequisiteResponse">;
type AssignmentResponse = ApiSchema<"AssignmentResponse">;
type AssignmentHistoryResponse = ApiSchema<"AssignmentHistoryResponse">;
type AssignmentRowsResponse = ApiSchema<"AssignmentRowsResponse">;
type AssignmentEventsResponse = ApiSchema<"AssignmentEventsResponse">;
type RequisiteReportResponse = ApiSchema<"RequisiteReportResponse">;
type CurrentShiftResponse = ApiSchema<"CurrentShiftResponse">;
type ShiftHistoryResponse = ApiSchema<"ShiftHistoryResponse">;
type ChecklistResponse = ApiSchema<"ChecklistResponse">;
type CloseShiftResponse = ApiSchema<"CloseShiftResponse">;
type TakeRequisiteResponse = ApiSchema<"TakeRequisiteResponse">;
type ShiftRequisiteResponse = ApiSchema<"ShiftRequisiteResponse">;
type TurnoversResponse = ApiSchema<"TurnoversResponse">;
type TurnoverResponse = ApiSchema<"TurnoverResponse">;
type PayoutsResponse = ApiSchema<"PayoutsResponse">;
type PayoutDetailsResponse = ApiSchema<"PayoutDetailsResponse">;
type PayoutResponse = ApiSchema<"PayoutResponse">;
type TransferResponse = ApiSchema<"TransferResponse">;
type ImportResponse = ApiSchema<"ImportResponse">;
type ReconciliationResponse = ApiSchema<"ReconciliationResponse">;
type ReconciliationItemsResponse = ApiSchema<"ReconciliationItemsResponse">;
type DashboardResponse = ApiSchema<"DashboardResponse">;
type OrdersListResponse = ApiSchema<"OrdersListResponse">;
type AccountingPeriodsResponse = ApiSchema<"AccountingPeriodsResponse">;
type AuditLogResponse = ApiSchema<"AuditLogResponse">;
type TraderProfileResponse = ApiSchema<"TraderProfileResponse">;

type BackendTrader = ApiSchema<"Trader">;
type BackendRequisite = ApiSchema<"Requisite">;
type BackendAssignment = ApiSchema<"RequisiteAssignment">;
type BackendAssignmentWorkRow = ApiSchema<"RequisiteAssignmentWorkRow">;
type BackendAssignmentEvent = ApiSchema<"RequisiteAssignmentEvent">;
type BackendRequisiteReportSummary = ApiSchema<"RequisiteReportSummary">;
type BackendRequisiteReportShift = ApiSchema<"RequisiteReportShift">;
type BackendAssignedRequisite = ApiSchema<"AssignedRequisite">;
type BackendTurnover = ApiSchema<"TurnoverEntry">;
type BackendPayout = ApiSchema<"Payout">;
type BackendTransfer = ApiSchema<"PayoutTransfer">;
type BackendOrder = ApiSchema<"Order">;
type BackendImportResult = ApiSchema<"ImportResult">;
type BackendReconciliationRun = ApiSchema<"ReconciliationRun">;

type BackendShiftReportReconciliation = {
  id: number;
  status: "matched" | "mismatch" | "accepted_with_comment";
  expectedAmountMinor: number;
  actualAmountMinor: number;
  diffAmountMinor: number;
  comment?: string;
  createdAt: string;
};

type BackendShiftReportRow = {
  rowKey: string;
  shiftRequisiteId?: number;
  requisiteId?: number;
  phone: string;
  methodType: string;
  bankCode: string;
  bankName: string;
  proxy?: string;
  employeeComment?: string;
  cardNumber?: string;
  holderName?: string;
  status: string;
  inboundTurnoverMinor: number;
  outboundTurnoverMinor: number;
  closingBalanceMinor: number;
  targetTurnoverMinor: number;
  csvInboundMinor: number;
  csvOutboundMinor: number;
  inboundDiffMinor: number;
  outboundDiffMinor: number;
  hasMismatch: boolean;
  csvOnly: boolean;
};

type ShiftReportDetailsResponse = {
  report: {
    shift: ApiSchema<"Shift">;
    inbound?: BackendShiftReportReconciliation;
    outbound?: BackendShiftReportReconciliation;
    rows: BackendShiftReportRow[];
  };
};

export const api = {
  auth: {
    async login(input: { login: string; password: string }) {
      const response = await apiClient.post<AuthResponse>("/auth/login", input);
      return { user: toCurrentUser(response.user) };
    },
    async me() {
      const response = await apiClient.get<AuthResponse>("/auth/me");
      return { user: toCurrentUser(response.user) };
    },
    logout: () => apiClient.post<void>("/auth/logout"),
  },

  banks: {
    async list(): Promise<Bank[]> {
      const response = await apiClient.get<BanksListResponse>("/banks");
      return response.items;
    },
  },

  traders: {
    async list(filters?: { search?: string; status?: string }) {
      const response = await apiClient.get<TradersListResponse>("/teamlead/traders");
      return response.items.map(toTrader).filter((trader) => filterTrader(trader, filters));
    },
    async save(input: {
      id?: number;
      login: string;
      password?: string;
      salaryRateBps: number;
      status: UserStatus;
    }) {
      if (input.id) {
        await apiClient.patch<TraderResponse>(`/teamlead/traders/${input.id}`, {
          salaryRateBps: input.salaryRateBps,
          status: input.status,
        });
        return;
      }

      await apiClient.post<TraderResponse>("/teamlead/traders", {
        login: input.login,
        password: input.password,
        externalWorkerName: input.login,
        salaryRateBps: input.salaryRateBps,
      });
    },
    async resetPassword(traderId: number) {
      const response = await apiClient.post<ResetPasswordResponse>(`/teamlead/traders/${traderId}/reset-password`);
      return { password: response.temporaryPassword };
    },
    archive: (traderId: number) =>
      apiClient.patch<TraderResponse>(`/teamlead/traders/${traderId}`, {
        status: "disabled",
      }),
  },

  requisites: {
    async list(filters?: { search?: string; bankCode?: string; status?: string; traderId?: string }) {
      const response = await apiClient.get<RequisitesListResponse>("/teamlead/requisites");
      return response.items.map(toRequisite).filter((requisite) => filterRequisite(requisite, filters));
    },
    async save(input: {
      id?: number;
      phone: string;
      bankCode: string;
      proxy: string;
      employeeComment?: string;
      status: "active" | "archived";
    }) {
      if (input.id) {
        await apiClient.patch<RequisiteResponse>(`/teamlead/requisites/${input.id}`, {
          phone: input.phone,
          bankCode: input.bankCode,
          proxy: input.proxy,
          employeeComment: input.employeeComment,
          status: input.status,
        });
        return;
      }

      await apiClient.post<RequisiteResponse>("/teamlead/requisites", {
        phone: input.phone,
        bankCode: input.bankCode,
        proxy: input.proxy,
        employeeComment: input.employeeComment,
      });
    },
    async history(requisiteId: number) {
      const response = await apiClient.get<AssignmentHistoryResponse>(
        `/teamlead/requisites/${requisiteId}/assignment-history`,
      );
      return response.items.map(toAssignmentHistory);
    },
    async activity(): Promise<RequisiteAssignmentWorkRow[]> {
      const response = await apiClient.get<AssignmentRowsResponse>("/teamlead/requisites/activity");
      return response.items.map(toAssignmentWorkRow);
    },
    async plans(): Promise<RequisiteAssignmentWorkRow[]> {
      const response = await apiClient.get<AssignmentRowsResponse>("/teamlead/requisites/plans");
      return response.items.map(toAssignmentWorkRow);
    },
    async createPlan(input: {
      requisiteId: number;
      traderId: number;
      assignedForDate: string;
      targetTurnoverMinor: number;
      comment?: string;
    }) {
      await apiClient.post<AssignmentResponse>("/teamlead/requisites/plans", input);
    },
    async updatePlan(input: {
      assignmentId: number;
      requisiteId: number;
      traderId: number;
      assignedForDate: string;
      targetTurnoverMinor: number;
      comment?: string;
    }) {
      await apiClient.patch<AssignmentResponse>(`/teamlead/requisites/plans/${input.assignmentId}`, {
        requisiteId: input.requisiteId,
        traderId: input.traderId,
        assignedForDate: input.assignedForDate,
        targetTurnoverMinor: input.targetTurnoverMinor,
        comment: input.comment,
      });
    },
    async cancelPlan(assignmentId: number) {
      await apiClient.delete<AssignmentResponse>(`/teamlead/requisites/plans/${assignmentId}`);
    },
    async planEvents(assignmentId: number): Promise<RequisiteAssignmentEvent[]> {
      const response = await apiClient.get<AssignmentEventsResponse>(
        `/teamlead/requisites/plans/${assignmentId}/events`,
      );
      return response.items.map(toAssignmentEvent);
    },
    async report(requisiteId: number): Promise<RequisiteReport> {
      const response = await apiClient.get<RequisiteReportResponse>(`/teamlead/requisites/${requisiteId}/report`);
      return toRequisiteReport(response.report);
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
    async history(): Promise<ShiftReport[]> {
      const response = await apiClient.get<ShiftHistoryResponse>("/trader/shift/history");
      return response.items.map(toShiftReport);
    },
    async report(shiftId: number): Promise<ShiftReportDetails> {
      const response = await apiClient.get<ShiftReportDetailsResponse>(`/trader/shifts/${shiftId}/report`);
      return toShiftReportDetails(response.report);
    },
    async requisites() {
      const [assignedResponse, turnoversResponse] = await Promise.all([
        apiClient.get<ApiSchema<"AssignedRequisitesResponse">>("/trader/requisites"),
        apiClient.get<TurnoversResponse>("/trader/shift/current/turnovers").catch(() => ({ items: [] })),
      ]);
      const latestTurnovers = latestTurnoverByShiftRequisite(turnoversResponse.items);
      return assignedResponse.items.map((item) => toShiftRequisite(item, latestTurnovers));
    },
    async futureRequisites() {
      const response = await apiClient.get<ApiSchema<"AssignedRequisitesResponse">>("/trader/requisites/future");
      return response.items.map((item) => toShiftRequisite(item, new Map()));
    },
    async historicalRequisites() {
      const response = await apiClient.get<ApiSchema<"AssignedRequisitesResponse">>("/trader/requisites/history");
      return response.items.map((item) => toShiftRequisite(item, new Map()));
    },
    async reportRequisites(shiftId: number) {
      const response = await apiClient.get<ApiSchema<"AssignedRequisitesResponse">>(`/trader/shifts/${shiftId}/requisites`);
      return response.items.map((item) => toShiftRequisite(item, new Map()));
    },
    async reportReconciliation(shiftId: number, direction: OrderDirection): Promise<ReconciliationSummary | null> {
      try {
        const response = await apiClient.get<ReconciliationResponse>(`/trader/shifts/${shiftId}/reconciliation/${direction}`);
        return toReconciliation(response.run);
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return null;
        throw error;
      }
    },
    async reportReconciliationItems(shiftId: number, direction: OrderDirection): Promise<ReconciliationItem[]> {
      try {
        const response = await apiClient.get<ReconciliationItemsResponse>(`/trader/shifts/${shiftId}/reconciliation/${direction}/items`);
        return response.items.map(toReconciliationItem);
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return [];
        throw error;
      }
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
    async closeRequisite(input: {
      shiftRequisiteId: number;
      inboundTurnoverMinor: number;
      outboundTurnoverMinor: number;
      closingBalanceMinor: number;
      blocked: boolean;
      comment?: string;
    }) {
      await apiClient.post<ShiftRequisiteResponse>(`/trader/shift-requisites/${input.shiftRequisiteId}/close`, {
        inboundTurnoverMinor: input.inboundTurnoverMinor,
        outboundTurnoverMinor: input.outboundTurnoverMinor,
        closingBalanceMinor: input.closingBalanceMinor,
        blocked: input.blocked,
        comment: input.comment,
      });
    },
    async correctRequisite(input: {
      shiftRequisiteId: number;
      inboundTurnoverMinor: number;
      outboundTurnoverMinor: number;
      closingBalanceMinor: number;
      comment: string;
    }) {
      const response = await apiClient.post<ShiftRequisiteResponse>(`/trader/shift-requisites/${input.shiftRequisiteId}/correction`, {
        inboundTurnoverMinor: input.inboundTurnoverMinor,
        outboundTurnoverMinor: input.outboundTurnoverMinor,
        closingBalanceMinor: input.closingBalanceMinor,
        comment: input.comment,
      });
      return response.shiftRequisite;
    },
    async returnRequisiteToWork(shiftRequisiteId: number) {
      const response = await apiClient.post<ShiftRequisiteResponse>(
        `/trader/shift-requisites/${shiftRequisiteId}/return-to-work`,
      );
      return response.shiftRequisite;
    },
    async addTurnover(input: { shiftRequisiteId: number; amountMinor: number; comment?: string }) {
      await apiClient.post<TurnoverResponse>("/trader/shift/current/turnovers", input);
    },
    async turnovers(shiftRequisiteId: number) {
      const response = await apiClient.get<TurnoversResponse>(
        `/trader/shift-requisites/${shiftRequisiteId}/turnovers`,
      );
      return response.items.map(toTurnover);
    },
    async close(input?: { closeComment?: string }) {
      const response = await apiClient.post<CloseShiftResponse>("/trader/shift/current/close", {
        closeComment: input?.closeComment,
      });
      return response.shift;
    },
  },

  teamleadReports: {
    async history(): Promise<ShiftReport[]> {
      const response = await apiClient.get<ShiftHistoryResponse>("/teamlead/shift/history");
      return response.items.map(toShiftReport);
    },
    async report(shiftId: number): Promise<ShiftReportDetails> {
      const response = await apiClient.get<ShiftReportDetailsResponse>(`/teamlead/shifts/${shiftId}/report`);
      return toShiftReportDetails(response.report);
    },
    async reportRequisites(shiftId: number) {
      const response = await apiClient.get<ApiSchema<"AssignedRequisitesResponse">>(`/teamlead/shifts/${shiftId}/requisites`);
      return response.items.map((item) => toShiftRequisite(item, new Map()));
    },
    async reportReconciliation(shiftId: number, direction: OrderDirection): Promise<ReconciliationSummary | null> {
      try {
        const response = await apiClient.get<ReconciliationResponse>(`/teamlead/shifts/${shiftId}/reconciliation/${direction}`);
        return toReconciliation(response.run);
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return null;
        throw error;
      }
    },
    async reportReconciliationItems(shiftId: number, direction: OrderDirection): Promise<ReconciliationItem[]> {
      try {
        const response = await apiClient.get<ReconciliationItemsResponse>(`/teamlead/shifts/${shiftId}/reconciliation/${direction}/items`);
        return response.items.map(toReconciliationItem);
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return [];
        throw error;
      }
    },
    async periodReconciliation(periodId: number, direction: OrderDirection): Promise<ReconciliationSummary | null> {
      try {
        const response = await apiClient.get<ReconciliationResponse>(`/teamlead/periods/${periodId}/reconciliation/${direction}`);
        return toReconciliation(response.run);
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return null;
        throw error;
      }
    },
    async periodReconciliationItems(periodId: number, direction: OrderDirection): Promise<ReconciliationItem[]> {
      try {
        const response = await apiClient.get<ReconciliationItemsResponse>(`/teamlead/periods/${periodId}/reconciliation/${direction}/items`);
        return response.items.map(toReconciliationItem);
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return [];
        throw error;
      }
    },
  },

  payouts: {
    async list() {
      const response = await apiClient.get<PayoutsResponse>("/trader/payouts");
      return response.items.map(toPayout);
    },
    async history() {
      const response = await apiClient.get<PayoutsResponse>("/trader/payouts/history");
      return response.items.map(toPayout);
    },
    async transfers(payoutId: number) {
      const response = await apiClient.get<PayoutDetailsResponse>(`/trader/payouts/${payoutId}`);
      return response.transfers.map(toTransfer);
    },
    async create(input: { destinationBank: string; destinationRequisite: string; amountMinor: number }) {
      const response = await apiClient.post<PayoutResponse>("/trader/payouts", input);
      return toPayout(response.payout);
    },
    async update(input: { id: number; destinationBank: string; destinationRequisite: string; amountMinor: number }) {
      const response = await apiClient.patch<PayoutResponse>(`/trader/payouts/${input.id}`, {
        destinationBank: input.destinationBank,
        destinationRequisite: input.destinationRequisite,
        amountMinor: input.amountMinor,
      });
      return toPayout(response.payout);
    },
    async delete(payoutId: number) {
      await apiClient.delete<void>(`/trader/payouts/${payoutId}`);
    },
    async addTransfer(input: { payoutId: number; sourceShiftRequisiteId: number; amountMinor: number; comment?: string }) {
      await apiClient.post<TransferResponse>(`/trader/payouts/${input.payoutId}/transfers`, {
        sourceShiftRequisiteId: input.sourceShiftRequisiteId,
        amountMinor: input.amountMinor,
        comment: input.comment,
      });
    },
    async updateTransfer(input: {
      payoutId: number;
      transferId: number;
      sourceShiftRequisiteId: number;
      amountMinor: number;
      comment?: string;
    }) {
      const response = await apiClient.patch<TransferResponse>(`/trader/payouts/${input.payoutId}/transfers/${input.transferId}`, {
        sourceShiftRequisiteId: input.sourceShiftRequisiteId,
        amountMinor: input.amountMinor,
        comment: input.comment,
      });
      return toTransfer(response.transfer);
    },
    async deleteTransfer(input: { payoutId: number; transferId: number } | number) {
      if (typeof input === "number") {
        throw new Error("Для удаления перевода нужен payoutId");
      }
      await apiClient.delete<void>(`/trader/payouts/${input.payoutId}/transfers/${input.transferId}`);
    },
  },

  traderProfile: {
    async get(filters?: PeriodFilter): Promise<TraderProfile> {
      const response = await apiClient.get<TraderProfileResponse>(`/trader/profile${queryString(filters ?? {})}`);
      return response.profile;
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
    async dashboard(scope: "teamlead" | "trader", direction: OrderDirection, filters?: OrderFilters) {
      const response = await apiClient.get<DashboardResponse>(
        `/${scope}/${direction}/dashboard${queryString(filters ?? {})}`,
      );
      return response.dashboard;
    },
    async list(scope: "teamlead" | "trader", direction: OrderDirection, filters?: OrderFilters) {
      const response = await apiClient.get<OrdersListResponse>(
        `/${scope}/${direction}/orders${queryString({
          dateFrom: filters?.dateFrom,
          dateTo: filters?.dateTo,
          status: filters?.status,
          traderIds: filters?.traderIds,
          confirmedOnly: filters?.confirmedOnly,
        })}`,
      );
      return response.items.map(toOrder).filter((order) => filterOrder(order, filters));
    },
    async reconciliation(scope: "teamlead" | "trader", direction: OrderDirection) {
      try {
        const response = await apiClient.get<ReconciliationResponse>(`/${scope}/${direction}/reconciliation/latest`);
        return toReconciliation(response.run);
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) {
          return null;
        }
        throw error;
      }
    },
    async reconciliationItems(scope: "trader", direction: OrderDirection): Promise<ReconciliationItem[]> {
      try {
        const response = await apiClient.get<ReconciliationItemsResponse>(`/${scope}/${direction}/reconciliation/items`);
        return response.items.map(toReconciliationItem);
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) {
          return [];
        }
        throw error;
      }
    },
  },

  periods: {
    async list(): Promise<AccountingPeriod[]> {
      return apiClient.get<AccountingPeriodsResponse>("/teamlead/periods").then((response) => response.items);
    },
  },

  audit: {
    async list(): Promise<AuditLogEntry[]> {
      return apiClient.get<AuditLogResponse>("/teamlead/audit").then((response) => response.items);
    },
  },
};

function toCurrentUser(user: ApiSchema<"User">): CurrentUser {
  return {
    id: user.id,
    teamId: user.teamId,
    role: user.role,
    login: user.login,
    status: user.status === "disabled" ? "disabled" : "active",
  };
}

function toTrader(trader: BackendTrader): Trader {
  return {
    id: trader.id,
    login: trader.login,
    externalWorkerName: trader.externalWorkerName,
    salaryRateBps: trader.salaryRateBps,
    assignedRequisitesCount: 0,
    currentShiftStatus: "none",
    status: trader.status === "disabled" ? "disabled" : "active",
  };
}

function toRequisite(requisite: BackendRequisite): Requisite {
  return {
    id: requisite.id,
    phone: requisite.phone,
    methodType: requisite.methodType as Requisite["methodType"],
    bankCode: requisite.bankCode,
    bankName: requisite.bankName,
    proxy: requisite.proxy ?? "",
    employeeComment: requisite.employeeComment,
    holderName: requisite.holderName,
    cardNumber: requisite.cardNumber,
    detailsFilledAt: requisite.detailsFilledAt,
    detailsFilledBy: requisite.detailsFilledBy,
    assignedTraderId: requisite.assignedTraderId,
    assignedTraderLogin: requisite.assignedTraderLogin,
    assignmentStatus: requisite.assignmentStatus,
    assignedForDate: requisite.assignedForDate,
    targetTurnoverMinor: requisite.targetTurnoverMinor,
    status: requisite.status as Requisite["status"],
    updatedAt: requisite.updatedAt,
  };
}

function toAssignmentHistory(assignment: BackendAssignment): AssignmentHistoryItem {
  return {
    id: assignment.id,
    changedAt: assignment.assignedAt,
    traderId: assignment.traderId,
    status: assignment.status,
    assignedForDate: assignment.assignedForDate,
    targetTurnoverMinor: assignment.targetTurnoverMinor,
    unassignedAt: assignment.unassignedAt,
    startedAt: assignment.startedAt,
    completedAt: assignment.completedAt,
    cancelledAt: assignment.cancelledAt,
    wasReassign: assignment.wasReassign,
    changedBy: String(assignment.assignedBy),
    comment: assignment.comment ?? "",
  };
}

function toAssignmentWorkRow(item: BackendAssignmentWorkRow): RequisiteAssignmentWorkRow {
  return {
    assignmentId: item.assignmentId,
    requisiteId: item.requisiteId,
    phone: item.phone,
    bankCode: item.bankCode,
    bankName: item.bankName,
    proxy: item.proxy ?? "",
    traderId: item.traderId,
    traderLogin: item.traderLogin,
    status: item.status,
    assignedForDate: item.assignedForDate,
    targetTurnoverMinor: item.targetTurnoverMinor,
    inboundTurnoverMinor: item.inboundTurnoverMinor,
    outboundTurnoverMinor: item.outboundTurnoverMinor,
    closingBalanceMinor: item.closingBalanceMinor,
    cardNumber: item.cardNumber,
    holderName: item.holderName,
    takenAt: item.takenAt,
    releasedAt: item.releasedAt,
    comment: item.comment,
    assignedAt: item.assignedAt,
    startedAt: item.startedAt,
    completedAt: item.completedAt,
    updatedAt: item.updatedAt,
    shiftRequisiteId: item.shiftRequisiteId,
  };
}

function toAssignmentEvent(item: BackendAssignmentEvent): RequisiteAssignmentEvent {
  return {
    id: item.id,
    assignmentId: item.assignmentId,
    actorId: item.actorId,
    action: item.action,
    beforeJson: item.beforeJson,
    afterJson: item.afterJson,
    comment: item.comment,
    createdAt: item.createdAt,
  };
}

function toRequisiteReport(report: ApiSchema<"RequisiteReport">): RequisiteReport {
  return {
    summary: toRequisiteReportSummary(report.summary),
    shifts: report.shifts.map(toRequisiteReportShift),
  };
}

function toRequisiteReportSummary(item: BackendRequisiteReportSummary): RequisiteReportSummary {
  return {
    id: item.id,
    phone: item.phone,
    methodType: item.methodType as RequisiteReportSummary["methodType"],
    bankCode: item.bankCode,
    bankName: item.bankName,
    proxy: item.proxy ?? "",
    employeeComment: item.employeeComment,
    holderName: item.holderName,
    cardNumber: item.cardNumber,
    status: item.status,
    totalInboundTurnoverMinor: item.totalInboundTurnoverMinor,
    totalOutboundTurnoverMinor: item.totalOutboundTurnoverMinor,
    lastClosingBalanceMinor: item.lastClosingBalanceMinor,
    latestStatus: item.latestStatus,
    lastActivityAt: item.lastActivityAt,
    lastShiftRequisiteId: item.lastShiftRequisiteId,
  };
}

function toRequisiteReportShift(item: BackendRequisiteReportShift): RequisiteReportShift {
  return {
    shiftRequisiteId: item.shiftRequisiteId,
    shiftId: item.shiftId,
    traderId: item.traderId,
    traderLogin: item.traderLogin,
    shiftStartedAt: item.shiftStartedAt,
    shiftClosedAt: item.shiftClosedAt,
    shiftStatus: item.shiftStatus,
    takenAt: item.takenAt,
    releasedAt: item.releasedAt,
    requisiteStatus: item.requisiteStatus,
    inboundTurnoverMinor: item.inboundTurnoverMinor,
    outboundTurnoverMinor: item.outboundTurnoverMinor,
    targetTurnoverMinor: item.targetTurnoverMinor,
    closingBalanceMinor: item.closingBalanceMinor,
    cardNumber: item.cardNumber,
    holderName: item.holderName,
    assignedForDate: item.assignedForDate,
    assignmentStatus: item.assignmentStatus,
  };
}

function toShiftRequisite(item: BackendAssignedRequisite, latestTurnovers: Map<number, number>): ShiftRequisite {
  const shiftRequisiteId = item.shiftRequisiteId ?? item.id;
  return {
    id: shiftRequisiteId,
    requisiteId: item.id,
    phone: item.phone,
    methodType: item.methodType as ShiftRequisite["methodType"],
    bankCode: item.bankCode,
    bankName: item.bankName,
    proxy: item.proxy ?? "",
    employeeComment: item.employeeComment,
    cardNumber: item.cardNumber,
    holderName: item.holderName,
    inboundTurnoverMinor: item.inboundTurnoverMinor ?? 0,
    outboundTurnoverMinor: item.outboundTurnoverMinor ?? 0,
    closingBalanceMinor: item.closingBalanceMinor ?? 0,
    assignmentStatus: item.assignmentStatus,
    assignedForDate: item.assignedForDate,
    targetTurnoverMinor: item.targetTurnoverMinor,
    latestTurnoverMinor: (item.inboundTurnoverMinor ?? 0) || latestTurnovers.get(shiftRequisiteId) || 0,
    status: toShiftRequisiteStatus(item),
  };
}

function toShiftRequisiteStatus(item: BackendAssignedRequisite): ShiftRequisite["status"] {
  if (!item.shiftRequisiteId) {
    if (item.assignmentStatus === "blocked") return "blocked";
    if (item.assignmentStatus === "worked") return "worked_pending_review";
    return "assigned";
  }
  if (item.shiftRequisiteStatus === "active") return "in_work";
  if (item.shiftRequisiteStatus === "blocked") return "blocked";
  if (item.shiftRequisiteStatus === "correction") return "correction";
  if (item.shiftRequisiteStatus === "worked_pending_review") return "worked_pending_review";
  if (item.shiftRequisiteStatus === "worked_verified") return "worked_verified";
  if (item.shiftRequisiteStatus === "worked_discrepancy") return "worked_discrepancy";
  if (item.shiftRequisiteStatus === "worked") return "worked_pending_review";
  return "worked_pending_review";
}

function toShiftReport(item: ApiSchema<"Shift">): ShiftReport {
  return {
    id: item.id,
    traderId: item.traderId,
    startedAt: item.startedAt,
    endedAt: item.endedAt,
    closedAt: item.closedAt,
    status: item.status as ShiftReport["status"],
    inboundReconciliationStatus: item.inboundReconciliationStatus,
    outboundReconciliationStatus: item.outboundReconciliationStatus,
    closeComment: item.closeComment,
  };
}

function toShiftReportDetails(item: ShiftReportDetailsResponse["report"]): ShiftReportDetails {
  return {
    shift: toShiftReport(item.shift),
    inbound: item.inbound ? toShiftReportReconciliation(item.inbound) : undefined,
    outbound: item.outbound ? toShiftReportReconciliation(item.outbound) : undefined,
    rows: item.rows.map(toShiftReportRow),
  };
}

function toShiftReportReconciliation(item: BackendShiftReportReconciliation): ShiftReportReconciliation {
  return {
    id: item.id,
    status: item.status,
    expectedMinor: item.expectedAmountMinor,
    actualMinor: item.actualAmountMinor,
    diffMinor: item.diffAmountMinor,
    comment: item.comment,
    createdAt: item.createdAt,
  };
}

function toShiftReportRow(item: BackendShiftReportRow): ShiftReportRow {
  return {
    rowKey: item.rowKey,
    shiftRequisiteId: item.shiftRequisiteId,
    requisiteId: item.requisiteId,
    phone: item.phone,
    methodType: item.methodType,
    bankCode: item.bankCode,
    bankName: item.bankName,
    proxy: item.proxy,
    employeeComment: item.employeeComment,
    cardNumber: item.cardNumber,
    holderName: item.holderName,
    status: item.status,
    inboundTurnoverMinor: item.inboundTurnoverMinor,
    outboundTurnoverMinor: item.outboundTurnoverMinor,
    closingBalanceMinor: item.closingBalanceMinor,
    targetTurnoverMinor: item.targetTurnoverMinor,
    csvInboundMinor: item.csvInboundMinor,
    csvOutboundMinor: item.csvOutboundMinor,
    inboundDiffMinor: item.inboundDiffMinor,
    outboundDiffMinor: item.outboundDiffMinor,
    hasMismatch: item.hasMismatch,
    csvOnly: item.csvOnly,
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
    shiftId: item.shiftId,
    destinationBank: item.destinationBank,
    destinationRequisite: item.destinationRequisite,
    amountMinor: item.amountMinor,
    paidMinor: item.paidAmountMinor,
    status: item.status === "paid" ? "paid" : item.status === "cancelled" ? "cancelled" : "open",
    createdAt: item.createdAt,
    updatedAt: item.updatedAt,
    deletedAt: item.deletedAt,
  };
}

function toTransfer(item: BackendTransfer): PayoutTransfer {
  return {
    id: item.id,
    payoutId: item.manualPayoutOrderId,
    sourceShiftRequisiteId: item.sourceShiftRequisiteId,
    sourceRequisiteId: item.sourceRequisiteId,
    sourcePhone: item.sourcePhone,
    sourceBankName: item.sourceBankName,
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
    rawStatus: item.rawStatus,
    normalizedStatus: item.normalizedStatus,
    innerId: item.externalInnerId,
    externalId: item.externalId,
    importBatchId: item.importBatchId,
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

function toReconciliationItem(item: ApiSchema<"ReconciliationItem">): ReconciliationItem {
  return {
    id: item.id,
    reconciliationRunId: item.reconciliationRunId,
    issueType: item.issueType,
    externalInnerId: item.externalInnerId,
    teamleadValue: item.teamleadValue as Record<string, unknown> | undefined,
    traderValue: item.traderValue as Record<string, unknown> | undefined,
    message: item.message,
    createdAt: item.createdAt,
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

function filterRequisite(requisite: Requisite, filters?: { search?: string; bankCode?: string; status?: string; traderId?: string }) {
  const search = filters?.search?.trim().toLowerCase();
  const matchesSearch =
    !search ||
    [requisite.phone, requisite.bankName, requisite.proxy, requisite.employeeComment ?? "", requisite.holderName ?? "", requisite.cardNumber ?? ""].some(
      (value) => value.toLowerCase().includes(search),
    );
  const matchesBank = !filters?.bankCode || filters.bankCode === "all" || requisite.bankCode === filters.bankCode;
  const matchesStatus = !filters?.status || filters.status === "all" || requisite.status === filters.status;
  const matchesTrader =
    !filters?.traderId ||
    filters.traderId === "all" ||
    String(requisite.assignedTraderId ?? "unassigned") === filters.traderId;
  return matchesSearch && matchesBank && matchesStatus && matchesTrader;
}

function filterOrder(order: Order, filters?: { search?: string; status?: string }) {
  const search = filters?.search?.trim().toLowerCase();
  const matchesSearch =
    !search ||
    [order.id, order.trader, order.workerName, order.requisite, order.innerId].some((value) =>
      value.toLowerCase().includes(search),
    );
  const matchesStatus = !filters?.status || filters.status === "all" || order.normalizedStatus === filters.status;
  return matchesSearch && matchesStatus;
}

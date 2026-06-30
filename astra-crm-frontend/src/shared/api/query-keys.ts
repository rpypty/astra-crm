type QueryParams = Record<string, unknown>;

function withParams<const T extends readonly unknown[]>(base: T, params?: QueryParams) {
  return params ? ([...base, params] as const) : base;
}

export const queryKeys = {
  auth: {
    me: ["auth", "me"] as const,
  },
  banks: ["banks"] as const,
  teamlead: {
    traders: (params?: QueryParams) => withParams(["teamlead", "traders"] as const, params),
    requisites: (params?: QueryParams) => withParams(["teamlead", "requisites"] as const, params),
    requisite: (requisiteId: number | undefined) => ["teamlead", "requisites", requisiteId] as const,
    requisitePlans: (params?: QueryParams) => withParams(["teamlead", "requisites", "plans"] as const, params),
    requisiteActivity: (params?: QueryParams) => withParams(["teamlead", "requisites", "activity"] as const, params),
    requisiteReport: (requisiteId: number | undefined) => ["teamlead", "requisites", requisiteId, "report"] as const,
    requisitePlanEvents: (assignmentId: number | undefined) =>
      ["teamlead", "requisites", "plans", assignmentId, "events"] as const,
    shiftHistory: (params?: QueryParams) => withParams(["teamlead", "shift", "history"] as const, params),
    shiftReport: (shiftId: number) => ["teamlead", "shift", shiftId, "report"] as const,
    shiftReportRequisites: (shiftId: number, params?: QueryParams) =>
      withParams(["teamlead", "shift", shiftId, "requisites"] as const, params),
    shiftReportReconciliation: (shiftId: number, direction: "inbound" | "outbound") =>
      ["teamlead", "shift", shiftId, direction, "reconciliation"] as const,
    shiftReportReconciliationItems: (shiftId: number, direction: "inbound" | "outbound") =>
      ["teamlead", "shift", shiftId, direction, "reconciliation", "items"] as const,
    periods: (params?: QueryParams) => withParams(["teamlead", "periods"] as const, params),
    periodReconciliation: (periodId: number | undefined, direction: "inbound" | "outbound") =>
      ["teamlead", "period", periodId, direction, "reconciliation"] as const,
    periodReconciliationItems: (periodId: number | undefined, direction: "inbound" | "outbound", params?: QueryParams) =>
      withParams(["teamlead", "period", periodId, direction, "reconciliation", "items"] as const, params),
    dashboard: (direction: "inbound" | "outbound", params?: QueryParams) =>
      ["teamlead", direction, "dashboard", params] as const,
    orders: (direction: "inbound" | "outbound", params?: QueryParams) =>
      ["teamlead", direction, "orders", params] as const,
    reconciliation: (direction: "inbound" | "outbound") => ["teamlead", direction, "reconciliation"] as const,
    reconciliationItems: (direction: "inbound" | "outbound", params?: QueryParams) =>
      withParams(["teamlead", direction, "reconciliation", "items"] as const, params),
    reconciliationHistory: (direction: "inbound" | "outbound", params?: QueryParams) =>
      withParams(["teamlead", direction, "reconciliation", "history"] as const, params),
    reconciliationRun: (direction: "inbound" | "outbound", runId: number | undefined) =>
      ["teamlead", direction, "reconciliation", runId] as const,
    reconciliationRunItems: (direction: "inbound" | "outbound", runId: number | undefined, params?: QueryParams) =>
      withParams(["teamlead", direction, "reconciliation", runId, "items"] as const, params),
    reconciliationV2: (params?: QueryParams) => withParams(["teamlead", "reconciliations"] as const, params),
    reconciliationV2Run: (runId: number | undefined) => ["teamlead", "reconciliations", runId] as const,
    reconciliationV2Items: (runId: number | undefined, params?: QueryParams) =>
      withParams(["teamlead", "reconciliations", runId, "items"] as const, params),
    audit: (params?: QueryParams) => withParams(["teamlead", "audit"] as const, params),
  },
  trader: {
    profile: (params?: QueryParams) => ["trader", "profile", params] as const,
    currentShift: ["trader", "shift", "current"] as const,
    shiftHistory: (params?: QueryParams) => withParams(["trader", "shift", "history"] as const, params),
    shiftReport: (shiftId: number) => ["trader", "shift", shiftId, "report"] as const,
    shiftReportRequisites: (shiftId: number, params?: QueryParams) =>
      withParams(["trader", "shift", shiftId, "requisites"] as const, params),
    shiftReportReconciliation: (shiftId: number, direction: "inbound" | "outbound") =>
      ["trader", "shift", shiftId, direction, "reconciliation"] as const,
    shiftReportReconciliationItems: (shiftId: number, direction: "inbound" | "outbound") =>
      ["trader", "shift", shiftId, direction, "reconciliation", "items"] as const,
    requisites: (params?: QueryParams) => withParams(["trader", "requisites"] as const, params),
    futureRequisites: (params?: QueryParams) => withParams(["trader", "requisites", "future"] as const, params),
    historicalRequisites: (params?: QueryParams) => withParams(["trader", "requisites", "history"] as const, params),
    internalTransfers: (shiftRequisiteId: number | undefined, params?: QueryParams) =>
      withParams(["trader", "shift-requisites", shiftRequisiteId, "internal-transfers"] as const, params),
    payouts: (params?: QueryParams) => withParams(["trader", "payouts"] as const, params),
    payoutHistory: (params?: QueryParams) => withParams(["trader", "payouts", "history"] as const, params),
    dashboard: (direction: "inbound" | "outbound", params?: QueryParams) =>
      ["trader", direction, "dashboard", params] as const,
    orders: (direction: "inbound" | "outbound", params?: QueryParams) =>
      ["trader", direction, "orders", params] as const,
    reconciliation: (direction: "inbound" | "outbound") => ["trader", direction, "reconciliation"] as const,
    reconciliationItems: (direction: "inbound" | "outbound", params?: QueryParams) =>
      withParams(["trader", direction, "reconciliation", "items"] as const, params),
  },
};

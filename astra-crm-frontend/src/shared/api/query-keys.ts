export const queryKeys = {
  auth: {
    me: ["auth", "me"] as const,
  },
  banks: ["banks"] as const,
  teamlead: {
    traders: (params?: Record<string, unknown>) => ["teamlead", "traders", params] as const,
    requisites: (params?: Record<string, unknown>) => ["teamlead", "requisites", params] as const,
    requisitePlans: ["teamlead", "requisites", "plans"] as const,
    requisiteActivity: ["teamlead", "requisites", "activity"] as const,
    requisiteReport: (requisiteId: number | undefined) => ["teamlead", "requisites", requisiteId, "report"] as const,
    requisitePlanEvents: (assignmentId: number | undefined) =>
      ["teamlead", "requisites", "plans", assignmentId, "events"] as const,
    shiftHistory: ["teamlead", "shift", "history"] as const,
    shiftReport: (shiftId: number) => ["teamlead", "shift", shiftId, "report"] as const,
    shiftReportRequisites: (shiftId: number) => ["teamlead", "shift", shiftId, "requisites"] as const,
    shiftReportReconciliation: (shiftId: number, direction: "inbound" | "outbound") =>
      ["teamlead", "shift", shiftId, direction, "reconciliation"] as const,
    shiftReportReconciliationItems: (shiftId: number, direction: "inbound" | "outbound") =>
      ["teamlead", "shift", shiftId, direction, "reconciliation", "items"] as const,
    periods: ["teamlead", "periods"] as const,
    periodReconciliation: (periodId: number | undefined, direction: "inbound" | "outbound") =>
      ["teamlead", "period", periodId, direction, "reconciliation"] as const,
    periodReconciliationItems: (periodId: number | undefined, direction: "inbound" | "outbound") =>
      ["teamlead", "period", periodId, direction, "reconciliation", "items"] as const,
    dashboard: (direction: "inbound" | "outbound", params?: Record<string, unknown>) =>
      ["teamlead", direction, "dashboard", params] as const,
    orders: (direction: "inbound" | "outbound", params?: Record<string, unknown>) =>
      ["teamlead", direction, "orders", params] as const,
    audit: (params?: Record<string, unknown>) => ["teamlead", "audit", params] as const,
  },
  trader: {
    profile: (params?: Record<string, unknown>) => ["trader", "profile", params] as const,
    currentShift: ["trader", "shift", "current"] as const,
    shiftHistory: ["trader", "shift", "history"] as const,
    shiftReport: (shiftId: number) => ["trader", "shift", shiftId, "report"] as const,
    shiftReportRequisites: (shiftId: number) => ["trader", "shift", shiftId, "requisites"] as const,
    shiftReportReconciliation: (shiftId: number, direction: "inbound" | "outbound") =>
      ["trader", "shift", shiftId, direction, "reconciliation"] as const,
    shiftReportReconciliationItems: (shiftId: number, direction: "inbound" | "outbound") =>
      ["trader", "shift", shiftId, direction, "reconciliation", "items"] as const,
    requisites: (params?: Record<string, unknown>) => ["trader", "requisites", params] as const,
    futureRequisites: ["trader", "requisites", "future"] as const,
    historicalRequisites: ["trader", "requisites", "history"] as const,
    payouts: (params?: Record<string, unknown>) => ["trader", "payouts", params] as const,
    payoutHistory: ["trader", "payouts", "history"] as const,
    dashboard: (direction: "inbound" | "outbound", params?: Record<string, unknown>) =>
      ["trader", direction, "dashboard", params] as const,
    orders: (direction: "inbound" | "outbound", params?: Record<string, unknown>) =>
      ["trader", direction, "orders", params] as const,
    reconciliation: (direction: "inbound" | "outbound") => ["trader", direction, "reconciliation"] as const,
    reconciliationItems: (direction: "inbound" | "outbound") =>
      ["trader", direction, "reconciliation", "items"] as const,
  },
};

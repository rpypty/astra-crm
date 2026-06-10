/* eslint-disable */
/* This file is generated from docs/openapi.yaml. Run npm run openapi:generate. */

export type components = {
  schemas: {
    ErrorResponse: {
  error: {
  code: string;
  message: string;
  fields?: {
  [key: string]: string;
};
  details?: string[];
};
};
    User: {
  id: number;
  teamId: number;
  role: "teamlead" | "trader";
  login: string;
  status: "active" | "disabled" | "deleted";
};
    Trader: components["schemas"]["User"] & {
  salaryRateBps: number;
  externalWorkerName: string;
  createdAt: string;
  updatedAt: string;
};
    AuthResponse: {
  user: components["schemas"]["User"];
};
    TradersListResponse: {
  items: components["schemas"]["Trader"][];
};
    TraderResponse: {
  trader: components["schemas"]["Trader"];
};
    ResetTraderPasswordResponse: {
  trader: components["schemas"]["Trader"];
  temporaryPassword: string;
};
    Requisite: {
  id: number;
  teamId: number;
  phone: string;
  methodType: string;
  proxy?: string;
  status: "active" | "disabled" | "archived";
  assignedTraderId?: number;
  assignedTraderLogin?: string;
  createdAt: string;
  updatedAt: string;
};
    RequisiteAssignment: {
  id: number;
  teamId: number;
  requisiteId: number;
  traderId: number;
  assignedBy: number;
  assignedAt: string;
  unassignedAt?: string;
  comment?: string;
  wasReassign: boolean;
};
    RequisitesListResponse: {
  items: components["schemas"]["Requisite"][];
};
    RequisiteResponse: {
  requisite: components["schemas"]["Requisite"];
};
    AssignmentResponse: {
  assignment: components["schemas"]["RequisiteAssignment"];
};
    AssignmentHistoryResponse: {
  items: components["schemas"]["RequisiteAssignment"][];
};
    Shift: {
  id: number;
  teamId: number;
  traderId: number;
  startedAt: string;
  endedAt?: string;
  status: "open" | "closing" | "closed" | "closed_with_discrepancy";
  inboundReconciliationStatus: string;
  outboundReconciliationStatus: string;
  closeComment?: string;
  createdAt: string;
  updatedAt: string;
  closedAt?: string;
};
    CurrentShiftResponse: {
  shift?: components["schemas"]["Shift"];
};
    CloseShiftResponse: {
  shift: components["schemas"]["Shift"];
};
    AssignedRequisite: {
  id: number;
  teamId: number;
  phone: string;
  methodType: string;
  proxy?: string;
  status: "active" | "disabled" | "archived";
  assignmentId: number;
  shiftRequisiteId?: number;
  cardNumber?: string;
  holderName?: string;
  shiftRequisiteStatus?: "active" | "released";
  takenAt?: string;
  inWork: boolean;
};
    AssignedRequisitesResponse: {
  items: components["schemas"]["AssignedRequisite"][];
};
    ShiftRequisite: {
  id: number;
  teamId: number;
  shiftId: number;
  traderId: number;
  requisiteId: number;
  assignmentId?: number;
  cardNumber: string;
  holderName: string;
  takenAt: string;
  releasedAt?: string;
  status: "active" | "released";
  createdAt: string;
  updatedAt: string;
};
    ShiftRequisitesResponse: {
  items: components["schemas"]["ShiftRequisite"][];
};
    TakeRequisiteResponse: {
  shift: components["schemas"]["Shift"];
  shiftRequisite: components["schemas"]["ShiftRequisite"];
  shiftCreated: boolean;
};
    ShiftRequisiteResponse: {
  shiftRequisite: components["schemas"]["ShiftRequisite"];
};
    TurnoverEntry: {
  id: number;
  teamId: number;
  shiftId: number;
  shiftRequisiteId: number;
  requisiteId: number;
  traderId: number;
  amountMinor: number;
  createdBy: number;
  createdAt: string;
  comment?: string;
};
    TurnoversResponse: {
  items: components["schemas"]["TurnoverEntry"][];
};
    TurnoverResponse: {
  turnover: components["schemas"]["TurnoverEntry"];
};
    CloseChecklist: {
  shift: components["schemas"]["Shift"];
  inboundImported: boolean;
  inboundOk: boolean;
  outboundImported: boolean;
  outboundOk: boolean;
  allPayoutsFullyPaid: boolean;
  unpaidPayoutCount: number;
  canClose: boolean;
};
    ChecklistResponse: {
  checklist: components["schemas"]["CloseChecklist"];
};
    Payout: {
  id: number;
  teamId: number;
  shiftId: number;
  traderId: number;
  destinationBank: string;
  destinationRequisite: string;
  amountMinor: number;
  paidAmountMinor: number;
  remainingAmountMinor: number;
  status: "draft" | "in_progress" | "paid" | "cancelled";
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
};
    PayoutTransfer: {
  id: number;
  teamId: number;
  manualPayoutOrderId: number;
  shiftId: number;
  traderId: number;
  sourceShiftRequisiteId: number;
  sourceRequisiteId: number;
  amountMinor: number;
  createdBy: number;
  createdAt: string;
  comment?: string;
};
    PayoutsResponse: {
  items: components["schemas"]["Payout"][];
};
    PayoutResponse: {
  payout: components["schemas"]["Payout"];
};
    PayoutDetailsResponse: {
  payout: components["schemas"]["Payout"];
  transfers: components["schemas"]["PayoutTransfer"][];
};
    TransferResponse: {
  transfer: components["schemas"]["PayoutTransfer"];
};
    Order: {
  id: number;
  externalOrderId: number;
  externalId: string;
  externalInnerId: string;
  workerName: string;
  traderId?: number;
  traderLogin?: string;
  requisiteRaw?: string;
  requisitePhone?: string;
  methodType?: string;
  methodName?: string;
  amountMinor: number;
  currency: string;
  rawStatus: string;
  normalizedStatus: "success" | "corrected" | "failed" | "cancelled" | "unknown";
  createdAtExternal: string;
  importBatchId: number;
};
    OrdersListResponse: {
  items: components["schemas"]["Order"][];
  page: number;
  pageSize: number;
  total: number;
};
    OrderDashboardSummary: {
  totalAmountMinor: number;
  totalCount: number;
  successAmountMinor: number;
  successCount: number;
  failedAmountMinor: number;
  failedCount: number;
  unknownAmountMinor: number;
  unknownCount: number;
};
    OrderStatusBreakdownItem: {
  rawStatus: string;
  normalizedStatus: "success" | "corrected" | "failed" | "cancelled" | "unknown";
  amountMinor: number;
  count: number;
};
    OrderImportHistoryItem: {
  id: number;
  teamId: number;
  uploadedBy: number;
  scopeType: string;
  direction: "inbound" | "outbound";
  shiftId?: number;
  accountingPeriodId?: number;
  traderId?: number;
  fileName: string;
  rowsCount: number;
  status: string;
  supersededByBatchId?: number;
  errorMessage?: string;
  createdAt: string;
  appliedAt?: string;
};
    OrderDashboard: {
  summary: components["schemas"]["OrderDashboardSummary"];
  statusBreakdown: components["schemas"]["OrderStatusBreakdownItem"][];
  unknownStatuses: string[];
  recentImports: components["schemas"]["OrderImportHistoryItem"][];
};
    DashboardResponse: {
  dashboard: components["schemas"]["OrderDashboard"];
};
    ReconciliationRun: {
  id: number;
  teamId: number;
  type: string;
  scopeType: "trader_shift" | "teamlead_period";
  shiftId?: number;
  accountingPeriodId?: number;
  traderId?: number;
  importBatchId?: number;
  expectedAmountMinor: number;
  actualAmountMinor: number;
  diffAmountMinor: number;
  successAmountMinor?: number;
  successCount?: number;
  failedAmountMinor?: number;
  failedCount?: number;
  totalAmountMinor?: number;
  totalCount?: number;
  status: "matched" | "mismatch" | "accepted_with_comment";
  comment?: string;
  confirmedBy?: number;
  confirmedAt?: string;
  createdAt: string;
};
    ReconciliationResponse: {
  run: components["schemas"]["ReconciliationRun"];
};
    ReconciliationItem: {
  id: number;
  reconciliationRunId: number;
  issueType: string;
  externalOrderId?: number;
  externalInnerId?: string;
  teamleadValue?: {
  [key: string]: unknown;
};
  traderValue?: {
  [key: string]: unknown;
};
  message?: string;
  createdAt: string;
};
    ReconciliationItemsResponse: {
  items: components["schemas"]["ReconciliationItem"][];
};
    ImportResult: {
  importBatchId: number;
  status: string;
  rowsCount: number;
  createdOrders: number;
  updatedOrders: number;
  deactivatedScopeItems: number;
  activeScopeItems: number;
  supersededBatches: number;
  rawStatusCounts?: {
  [key: string]: number;
};
  normalizedStatusCounts?: {
  [key: string]: number;
};
  unknownStatuses?: string[];
};
    ImportResponse: {
  result: components["schemas"]["ImportResult"];
};
    AccountingPeriod: {
  id: number;
  title: string;
  dateRange: string;
  inboundStatus: "matched" | "mismatch" | "accepted_with_comment";
  outboundStatus: "matched" | "mismatch" | "accepted_with_comment";
  status: "open" | "closed" | "closed_with_discrepancy";
};
    AccountingPeriodsResponse: {
  items: components["schemas"]["AccountingPeriod"][];
};
    AuditLogEntry: {
  id: number;
  createdAt: string;
  actorLogin: string;
  action: string;
  entityType: string;
  entityId: string;
  comment?: string;
  maskedPayload: {
  [key: string]: unknown;
};
};
    AuditLogResponse: {
  items: components["schemas"]["AuditLogEntry"][];
};
    HealthResponse: {
  status: "ok" | "not_ready";
};
  };
};

export type ApiSchema<Name extends keyof components["schemas"]> = components["schemas"][Name];

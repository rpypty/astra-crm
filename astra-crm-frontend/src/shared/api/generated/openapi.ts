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
  page: number;
  pageSize: number;
  total: number;
};
    TraderResponse: {
  trader: components["schemas"]["Trader"];
};
    TraderProfile: {
  id: number;
  login: string;
  salaryRateBps: number;
  externalWorkerName: string;
  currentShiftSuccessInboundMinor: number;
  currentShiftSalaryMinor: number;
  periodId?: number;
  periodTitle?: string;
  periodSuccessInboundMinor: number;
  periodSalaryMinor: number;
};
    TraderProfileResponse: {
  profile: components["schemas"]["TraderProfile"];
};
    ResetTraderPasswordResponse: {
  trader: components["schemas"]["Trader"];
  temporaryPassword: string;
};
    Bank: {
  code: string;
  name: string;
};
    BanksListResponse: {
  items: components["schemas"]["Bank"][];
};
    Requisite: {
  id: number;
  teamId: number;
  phone: string;
  methodType: string;
  bankCode: string;
  bankName: string;
  proxy?: string;
  employeeComment?: string;
  holderName?: string;
  cardNumber?: string;
  detailsFilledAt?: string;
  detailsFilledBy?: number;
  status: "active" | "disabled" | "archived";
  assignedTraderId?: number;
  assignedTraderLogin?: string;
  assignmentStatus?: "planned" | "assigned" | "in_work" | "worked" | "blocked" | "cancelled" | "expired";
  assignedForDate?: string;
  targetTurnoverMinor: number;
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
  status: "planned" | "assigned" | "in_work" | "worked" | "blocked" | "cancelled" | "expired";
  assignedForDate: string;
  targetTurnoverMinor: number;
  startedAt?: string;
  completedAt?: string;
  cancelledAt?: string;
  shiftRequisiteId?: number;
  wasReassign: boolean;
};
    RequisiteAssignmentWorkRow: {
  assignmentId: number;
  teamId: number;
  requisiteId: number;
  phone: string;
  bankCode: string;
  bankName: string;
  proxy?: string;
  traderId: number;
  traderLogin: string;
  status: "planned" | "assigned" | "in_work" | "worked" | "blocked" | "cancelled" | "expired";
  assignedForDate: string;
  targetTurnoverMinor: number;
  inboundTurnoverMinor: number;
  outboundTurnoverMinor: number;
  closingBalanceMinor: number;
  cardNumber?: string;
  holderName?: string;
  takenAt?: string;
  releasedAt?: string;
  comment?: string;
  assignedAt: string;
  startedAt?: string;
  completedAt?: string;
  updatedAt: string;
  shiftRequisiteId?: number;
};
    RequisiteReportSummary: {
  id: number;
  teamId: number;
  phone: string;
  methodType: string;
  bankCode: string;
  bankName: string;
  proxy?: string;
  employeeComment?: string;
  holderName?: string;
  cardNumber?: string;
  status: "active" | "disabled" | "archived";
  totalInboundTurnoverMinor: number;
  totalOutboundTurnoverMinor: number;
  lastClosingBalanceMinor: number;
  latestStatus: string;
  lastActivityAt?: string;
  lastShiftRequisiteId?: number;
};
    RequisiteReportShift: {
  shiftRequisiteId: number;
  shiftId: number;
  teamId: number;
  requisiteId: number;
  traderId: number;
  traderLogin: string;
  shiftStartedAt: string;
  shiftClosedAt?: string;
  shiftStatus: "open" | "closing" | "closed" | "closed_with_discrepancy";
  takenAt: string;
  releasedAt?: string;
  requisiteStatus: "active" | "worked" | "worked_pending_review" | "worked_verified" | "worked_discrepancy" | "correction" | "released" | "blocked";
  inboundTurnoverMinor: number;
  outboundTurnoverMinor: number;
  targetTurnoverMinor: number;
  closingBalanceMinor: number;
  cardNumber?: string;
  holderName?: string;
  assignedForDate?: string;
  assignmentStatus: string;
};
    RequisiteReport: {
  summary: components["schemas"]["RequisiteReportSummary"];
  shifts: components["schemas"]["RequisiteReportShift"][];
};
    RequisiteAssignmentEvent: {
  id: number;
  teamId: number;
  assignmentId: number;
  actorId: number;
  action: string;
  beforeJson?: {
  [key: string]: unknown;
};
  afterJson?: {
  [key: string]: unknown;
};
  comment?: string;
  createdAt: string;
};
    PlanRequisiteRequest: {
  requisiteId: number;
  traderId: number;
  assignedForDate: string;
  targetTurnoverMinor: number;
  comment?: string;
};
    RequisitesListResponse: {
  items: components["schemas"]["Requisite"][];
  page: number;
  pageSize: number;
  total: number;
};
    RequisiteResponse: {
  requisite: components["schemas"]["Requisite"];
};
    AssignmentResponse: {
  assignment: components["schemas"]["RequisiteAssignment"];
};
    AssignmentHistoryResponse: {
  items: components["schemas"]["RequisiteAssignment"][];
  page: number;
  pageSize: number;
  total: number;
};
    AssignmentRowsResponse: {
  items: components["schemas"]["RequisiteAssignmentWorkRow"][];
  page: number;
  pageSize: number;
  total: number;
};
    RequisiteReportResponse: {
  report: components["schemas"]["RequisiteReport"];
};
    AssignmentEventsResponse: {
  items: components["schemas"]["RequisiteAssignmentEvent"][];
  page: number;
  pageSize: number;
  total: number;
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
  tlReconciliationStatus: "not_checked" | "confirmed_by_tl" | "updated_by_tl" | "tl_discrepancy" | "tl_accepted";
  lastTeamleadReconciliationId?: number;
  tlReconciledAt?: string;
  closeComment?: string;
  createdAt: string;
  updatedAt: string;
  closedAt?: string;
};
    CurrentShiftResponse: {
  shift?: components["schemas"]["Shift"];
};
    ShiftHistoryResponse: {
  items: components["schemas"]["Shift"][];
  page: number;
  pageSize: number;
  total: number;
};
    CloseShiftResponse: {
  shift: components["schemas"]["Shift"];
};
    AssignedRequisite: {
  id: number;
  teamId: number;
  phone: string;
  methodType: string;
  bankCode: string;
  bankName: string;
  proxy?: string;
  employeeComment?: string;
  status: "active" | "disabled" | "archived" | "blocked";
  assignmentId: number;
  assignmentStatus: "planned" | "assigned" | "in_work" | "worked" | "blocked" | "cancelled" | "expired";
  assignedForDate: string;
  targetTurnoverMinor: number;
  shiftRequisiteId?: number;
  cardNumber?: string;
  holderName?: string;
  shiftRequisiteStatus?: "active" | "worked" | "worked_pending_review" | "worked_verified" | "worked_discrepancy" | "correction" | "released" | "blocked";
  takenAt?: string;
  releasedAt?: string;
  inboundTurnoverMinor: number;
  outboundTurnoverMinor: number;
  closingBalanceMinor: number;
  inWork: boolean;
};
    AssignedRequisitesResponse: {
  items: components["schemas"]["AssignedRequisite"][];
  page: number;
  pageSize: number;
  total: number;
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
  status: "active" | "worked" | "worked_pending_review" | "worked_verified" | "worked_discrepancy" | "correction" | "released" | "blocked";
  inboundTurnoverMinor: number;
  outboundTurnoverMinor: number;
  closingBalanceMinor: number;
  createdAt: string;
  updatedAt: string;
};
    ShiftRequisitesResponse: {
  items: components["schemas"]["ShiftRequisite"][];
  page: number;
  pageSize: number;
  total: number;
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
    TurnoversListResponse: {
  items: components["schemas"]["TurnoverEntry"][];
  page: number;
  pageSize: number;
  total: number;
};
    TurnoverResponse: {
  turnover: components["schemas"]["TurnoverEntry"];
};
    CloseShiftRequisiteRequest: {
  inboundTurnoverMinor: number;
  outboundTurnoverMinor: number;
  closingBalanceMinor: number;
  releasedAt?: string;
  blocked: boolean;
  comment?: string;
};
    CorrectShiftRequisiteRequest: {
  inboundTurnoverMinor: number;
  outboundTurnoverMinor: number;
  closingBalanceMinor: number;
  comment: string;
};
    CreateInternalTransferRequest: {
  sourceShiftRequisiteId: number;
  destinationShiftRequisiteId: number;
  amountMinor: number;
  comment?: string;
};
    CloseChecklist: {
  shift: components["schemas"]["Shift"];
  inboundImported: boolean;
  inboundOk: boolean;
  outboundImported: boolean;
  outboundOk: boolean;
  allRequisitesClosed: boolean;
  openRequisiteCount: number;
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
  sourcePhone?: string;
  sourceBankName?: string;
  amountMinor: number;
  createdBy: number;
  createdAt: string;
  comment?: string;
};
    InternalTransfer: {
  id: number;
  teamId: number;
  shiftId: number;
  traderId: number;
  sourceShiftRequisiteId: number;
  sourceRequisiteId: number;
  sourcePhone: string;
  sourceBankCode: string;
  sourceBankName: string;
  destinationShiftRequisiteId: number;
  destinationRequisiteId: number;
  destinationPhone: string;
  destinationBankCode: string;
  destinationBankName: string;
  amountMinor: number;
  status: "active" | "cancelled";
  createdBy: number;
  createdAt: string;
  cancelledBy?: number;
  cancelledAt?: string;
  comment?: string;
};
    InternalTransfersResponse: {
  items: components["schemas"]["InternalTransfer"][];
  page: number;
  pageSize: number;
  total: number;
};
    InternalTransferResponse: {
  transfer: components["schemas"]["InternalTransfer"];
};
    PayoutsResponse: {
  items: components["schemas"]["Payout"][];
  page: number;
  pageSize: number;
  total: number;
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
  tlReconciliationStatus: "not_checked" | "confirmed_by_tl" | "updated_by_tl" | "tl_discrepancy" | "tl_accepted";
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
  blockedBalanceMinor: number;
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
    ReconciliationRunsResponse: {
  items: components["schemas"]["ReconciliationRun"][];
  page: number;
  pageSize: number;
  total: number;
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
  page: number;
  pageSize: number;
  total: number;
};
    TeamleadReconciliationPipelineStep: {
  stage: "normalization" | "matching" | "turnover_check" | "transaction_check" | "preview";
  status: "matched" | "mismatch";
  issuesCount: number;
  directionsCount: number;
};
    TeamleadReconciliationSummary: {
  rowsTotal?: number;
  rowsInPeriod?: number;
  successAmountMinor?: number;
  successCount?: number;
  failedAmountMinor?: number;
  failedCount?: number;
  totalAmountMinor?: number;
  totalCount?: number;
  crmAmountMinor?: number;
  crmCount?: number;
  diffAmountMinor?: number;
};
    TeamleadReconciliationPreview: {
  [key: string]: {
  createCount?: number;
  updateCount?: number;
  unchangedCount?: number;
  blockedCount?: number;
};
};
    TeamleadReconciliationApplyDirectionResult: {
  direction: "inbound" | "outbound";
  rowsApplied: number;
  createdOrders: number;
  updatedOrders: number;
  confirmedOrders: number;
  discrepancyOrders: number;
};
    TeamleadReconciliationApplyResult: {
  runId: number;
  createdOrders: number;
  updatedOrders: number;
  confirmedOrders: number;
  discrepancyOrders: number;
  directions: components["schemas"]["TeamleadReconciliationApplyDirectionResult"][];
};
    TeamleadReconciliationRun: {
  id: number;
  teamId: number;
  dateFrom: string;
  dateTo: string;
  status: "draft" | "analyzing" | "matched" | "mismatch" | "apply_queued" | "applying" | "applied" | "apply_failed" | "rejected";
  createdBy: number;
  confirmedBy?: number;
  rejectedBy?: number;
  inboundImportBatchId?: number;
  outboundImportBatchId?: number;
  comment?: string;
  mismatchCount: number;
  conflictCount: number;
  blockedCount: number;
  pipeline?: components["schemas"]["TeamleadReconciliationPipelineStep"][];
  inboundSummary?: components["schemas"]["TeamleadReconciliationSummary"];
  outboundSummary?: components["schemas"]["TeamleadReconciliationSummary"];
  preview?: components["schemas"]["TeamleadReconciliationPreview"];
  applyResult?: components["schemas"]["TeamleadReconciliationApplyResult"];
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
  analyzedAt?: string;
  confirmedAt?: string;
  rejectedAt?: string;
  applyQueuedAt?: string;
  appliedAt?: string;
};
    TeamleadReconciliationResponse: {
  reconciliation: components["schemas"]["TeamleadReconciliationRun"];
};
    TeamleadReconciliationsResponse: {
  items: components["schemas"]["TeamleadReconciliationRun"][];
  page: number;
  pageSize: number;
  total: number;
};
    TeamleadReconciliationItem: {
  id: number;
  teamleadReconciliationId: number;
  teamId: number;
  direction: "inbound" | "outbound";
  stage: string;
  issueType: string;
  severity: "info" | "warning" | "error" | "blocker";
  externalOrderId?: number;
  externalInnerId?: string;
  traderId?: number;
  requisiteId?: number;
  shiftId?: number;
  before?: {
  [key: string]: unknown;
};
  after?: {
  [key: string]: unknown;
};
  message?: string;
  isBlocking: boolean;
  appliedAt?: string;
  createdAt: string;
};
    TeamleadReconciliationItemsResponse: {
  items: components["schemas"]["TeamleadReconciliationItem"][];
  page: number;
  pageSize: number;
  total: number;
};
    TeamleadReconciliationDecisionRequest: {
  comment?: string;
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
  dateFrom: string;
  dateTo: string;
  dateRange: string;
  inboundStatus: "matched" | "mismatch" | "accepted_with_comment";
  outboundStatus: "matched" | "mismatch" | "accepted_with_comment";
  status: "open" | "closed" | "closed_with_discrepancy";
};
    AccountingPeriodsResponse: {
  items: components["schemas"]["AccountingPeriod"][];
  page: number;
  pageSize: number;
  total: number;
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
  page: number;
  pageSize: number;
  total: number;
};
    HealthResponse: {
  status: "ok" | "not_ready";
};
  };
};

export type ApiSchema<Name extends keyof components["schemas"]> = components["schemas"][Name];

export type UserRole = "teamlead" | "trader";
export type UserStatus = "active" | "disabled";

export type CurrentUser = {
  id: number;
  teamId: number;
  role: UserRole;
  login: string;
  status: UserStatus;
};

export type Trader = {
  id: number;
  login: string;
  externalWorkerName: string;
  salaryRateBps: number;
  assignedRequisitesCount: number;
  currentShiftStatus: "open" | "closed" | "closed_with_discrepancy" | "none";
  status: UserStatus;
};

export type Bank = {
  code: string;
  name: string;
};

export type Requisite = {
  id: number;
  phone: string;
  methodType: "SBP" | "C2C";
  bankCode: string;
  bankName: string;
  proxy: string;
  employeeComment?: string;
  holderName?: string;
  cardNumber?: string;
  detailsFilledAt?: string;
  detailsFilledBy?: number;
  assignedTraderId?: number;
  assignedTraderLogin?: string;
  assignmentStatus?: RequisiteAssignmentStatus;
  assignedForDate?: string;
  targetTurnoverMinor: number;
  status: "active" | "archived";
  updatedAt: string;
};

export type RequisiteAssignmentStatus = "planned" | "assigned" | "in_work" | "worked" | "blocked" | "cancelled" | "expired";

export type RequisiteAssignmentWorkRow = {
  assignmentId: number;
  requisiteId: number;
  phone: string;
  bankCode: string;
  bankName: string;
  proxy: string;
  traderId: number;
  traderLogin: string;
  status: RequisiteAssignmentStatus;
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

export type RequisiteReport = {
  summary: RequisiteReportSummary;
  shifts: RequisiteReportShift[];
};

export type RequisiteReportSummary = {
  id: number;
  phone: string;
  methodType: "SBP" | "C2C";
  bankCode: string;
  bankName: string;
  proxy: string;
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

export type RequisiteReportShift = {
  shiftRequisiteId: number;
  shiftId: number;
  traderId: number;
  traderLogin: string;
  shiftStartedAt: string;
  shiftClosedAt?: string;
  shiftStatus: "open" | "closing" | "closed" | "closed_with_discrepancy";
  takenAt: string;
  releasedAt?: string;
  requisiteStatus:
    | "active"
    | "worked"
    | "worked_pending_review"
    | "worked_verified"
    | "worked_discrepancy"
    | "correction"
    | "released"
    | "blocked";
  inboundTurnoverMinor: number;
  outboundTurnoverMinor: number;
  targetTurnoverMinor: number;
  closingBalanceMinor: number;
  cardNumber?: string;
  holderName?: string;
  assignedForDate?: string;
  assignmentStatus: string;
};

export type RequisiteAssignmentEvent = {
  id: number;
  assignmentId: number;
  actorId: number;
  action: string;
  beforeJson?: Record<string, unknown>;
  afterJson?: Record<string, unknown>;
  comment?: string;
  createdAt: string;
};

export type AssignmentHistoryItem = {
  id: number;
  changedAt: string;
  traderId: number;
  status: RequisiteAssignmentStatus;
  assignedForDate: string;
  targetTurnoverMinor: number;
  unassignedAt?: string;
  startedAt?: string;
  completedAt?: string;
  cancelledAt?: string;
  wasReassign: boolean;
  changedBy: string;
  comment: string;
};

export type ShiftRequisite = {
  id: number;
  requisiteId: number;
  phone: string;
  methodType: "SBP" | "C2C";
  bankCode: string;
  bankName: string;
  proxy: string;
  employeeComment?: string;
  cardNumber?: string;
  holderName?: string;
  latestTurnoverMinor: number;
  inboundTurnoverMinor: number;
  outboundTurnoverMinor: number;
  closingBalanceMinor: number;
  assignmentStatus: RequisiteAssignmentStatus;
  assignedForDate: string;
  takenAt?: string;
  releasedAt?: string;
  targetTurnoverMinor: number;
  status: "assigned" | "in_work" | "worked" | "worked_pending_review" | "worked_verified" | "worked_discrepancy" | "correction" | "blocked";
};

export type ShiftReport = {
  id: number;
  traderId: number;
  startedAt: string;
  endedAt?: string;
  closedAt?: string;
  status: "open" | "closing" | "closed" | "closed_with_discrepancy" | "draft";
  inboundReconciliationStatus: string;
  outboundReconciliationStatus: string;
  closeComment?: string;
};

export type ShiftReportReconciliation = {
  id: number;
  status: "matched" | "mismatch" | "accepted_with_comment";
  expectedMinor: number;
  actualMinor: number;
  diffMinor: number;
  comment?: string;
  createdAt: string;
};

export type ShiftReportRow = {
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

export type ShiftReportDetails = {
  shift: ShiftReport;
  inbound?: ShiftReportReconciliation;
  outbound?: ShiftReportReconciliation;
  rows: ShiftReportRow[];
};

export type TurnoverEntry = {
  id: number;
  shiftRequisiteId: number;
  amountMinor: number;
  comment?: string;
  createdAt: string;
};

export type Payout = {
  id: number;
  shiftId: number;
  destinationBank: string;
  destinationRequisite: string;
  amountMinor: number;
  paidMinor: number;
  status: "open" | "paid" | "cancelled";
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
};

export type PayoutTransfer = {
  id: number;
  payoutId: number;
  sourceShiftRequisiteId: number;
  sourceRequisiteId: number;
  sourcePhone?: string;
  sourceBankName?: string;
  amountMinor: number;
  comment?: string;
  createdAt: string;
};

export type OrderDirection = "inbound" | "outbound";

export type OrderFilters = {
  dateFrom?: string;
  dateTo?: string;
  search?: string;
  status?: string;
  traderIds?: number[];
  confirmedOnly?: boolean;
};

export type Order = {
  id: string;
  createdAt: string;
  closedAt?: string;
  trader: string;
  workerName: string;
  requisite: string;
  method: string;
  bankName: string;
  amountMinor: number;
  status: string;
  rawStatus: string;
  normalizedStatus: string;
  innerId: string;
  externalId: string;
  importBatchId: number;
};

export type OrderDashboardSummary = {
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

export type OrderStatusBreakdownItem = {
  rawStatus: string;
  normalizedStatus: string;
  amountMinor: number;
  count: number;
};

export type OrderImportHistoryItem = {
  id: number;
  fileName: string;
  rowsCount: number;
  status: string;
  createdAt: string;
  appliedAt?: string;
};

export type OrderDashboard = {
  summary: OrderDashboardSummary;
  statusBreakdown: OrderStatusBreakdownItem[];
  unknownStatuses: string[];
  recentImports: OrderImportHistoryItem[];
};

export type ImportIssue = {
  row: number;
  message: string;
};

export type ImportResult = {
  status: "matched" | "mismatch" | "failed";
  importedRows: number;
  successCount: number;
  failedCount: number;
  duplicateCount: number;
  expectedMinor: number;
  actualMinor: number;
  issues: ImportIssue[];
};

export type ReconciliationSummary = {
  status: "matched" | "mismatch" | "accepted_with_comment";
  expectedMinor: number;
  actualMinor: number;
  diffMinor: number;
  comment?: string;
  runId?: number;
  importBatchId?: number;
  confirmedAt?: string;
  createdAt?: string;
};

export type ReconciliationItem = {
  id: number;
  reconciliationRunId: number;
  issueType: string;
  externalInnerId?: string;
  teamleadValue?: Record<string, unknown>;
  traderValue?: Record<string, unknown>;
  message?: string;
  createdAt: string;
};

export type AccountingPeriod = {
  id: number;
  title: string;
  dateFrom: string;
  dateTo: string;
  dateRange: string;
  inboundStatus: "matched" | "mismatch" | "accepted_with_comment";
  outboundStatus: "matched" | "mismatch" | "accepted_with_comment";
  status: "open" | "closed" | "closed_with_discrepancy";
};

export type AuditLogEntry = {
  id: number;
  createdAt: string;
  actorLogin: string;
  action: string;
  entityType: string;
  entityId: string;
  comment?: string;
  maskedPayload: Record<string, unknown>;
};

export type TraderProfile = {
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

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
  oldTrader?: string;
  newTrader?: string;
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
  targetTurnoverMinor: number;
  status: "assigned" | "in_work" | "worked" | "correction" | "blocked";
};

export type ShiftReport = {
  id: number;
  traderId: number;
  startedAt: string;
  endedAt?: string;
  closedAt?: string;
  status: "closed" | "closed_with_discrepancy";
  inboundReconciliationStatus: string;
  outboundReconciliationStatus: string;
  closeComment?: string;
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
  destinationBank: string;
  destinationRequisite: string;
  amountMinor: number;
  paidMinor: number;
  status: "open" | "paid" | "cancelled";
  createdAt: string;
};

export type PayoutTransfer = {
  id: number;
  payoutId: number;
  sourceShiftRequisiteId: number;
  sourceRequisiteId: number;
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

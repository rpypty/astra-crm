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

export type Requisite = {
  id: number;
  phone: string;
  methodType: "SBP" | "C2C";
  proxy: string;
  assignedTraderId?: number;
  assignedTraderLogin?: string;
  status: "active" | "archived";
  updatedAt: string;
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
  proxy: string;
  cardNumber?: string;
  holderName?: string;
  latestTurnoverMinor: number;
  status: "assigned" | "in_work";
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
  amountMinor: number;
  comment?: string;
  createdAt: string;
};

export type OrderDirection = "inbound" | "outbound";

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

export type AccountingPeriod = {
  id: number;
  title: string;
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

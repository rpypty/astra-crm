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
  innerId: string;
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

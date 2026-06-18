import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { CalendarDays, Copy, Eye, FileText, History, Pencil, Plus, UserRound, X } from "lucide-react";
import { useCallback, useDeferredValue, useEffect, useMemo, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { Bar, CartesianGrid, Cell, ComposedChart, Line, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { z } from "zod";
import { DateTimeCell } from "@/shared/ui/date-time-cell";
import { EmptyState } from "@/shared/ui/empty-state";
import { FormField } from "@/shared/ui/form-field";
import { AcceptMismatchDialog, MismatchAlert } from "@/features/import-csv/ui/import-components";
import { MoneyCell } from "@/entities/order/ui/money-cell";
import { OrderDashboard } from "@/widgets/order-dashboard/ui/order-dashboard";
import { PageHeader } from "@/shared/ui/page-header";
import { PeriodFilterBar } from "@/features/period-filter/ui/period-filter-bar";
import { ReportReconciliationDetails } from "@/features/reconciliation/ui/report-reconciliation-details";
import { RequisiteCell } from "@/entities/requisite/ui/requisite-cell";
import { StatusBadge } from "@/entities/status/ui/status-badge";
import { UserCell } from "@/entities/user/ui/user-cell";
import { DataTable } from "@/shared/ui/data-table";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Input } from "@/shared/ui/input";
import { SearchableSelect, type SearchableSelectOption } from "@/shared/ui/searchable-select";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import type {
  AccountingPeriod,
  AuditLogEntry,
  Bank,
  Order,
  OrderDirection,
  Requisite,
  RequisiteAssignmentEvent,
  RequisiteAssignmentWorkRow,
  RequisiteReport,
  RequisiteReportShift,
  ReconciliationItem,
  ReconciliationSummary,
  ShiftReport,
  ShiftRequisite,
  Trader,
} from "@/shared/model/domain";
import { api } from "@/shared/api/api";
import { filterOrdersBySearch } from "@/shared/lib/order-filters";
import type { PeriodFilter } from "@/shared/lib/period-filter";
import { usePersistentPeriodFilter } from "@/shared/lib/period-filter";
import { queryKeys } from "@/shared/api/query-keys";
import {
  bpsToPercent,
  formatCardNumber,
  formatDateTime,
  formatMoneyMinor,
  formatRussianPhone,
  isValidRussianPhone,
  normalizeRussianPhone,
  parseMoneyToMinor,
  percentToBps,
} from "@/shared/lib/utils";
import { MetricCard } from "@/shared/ui/metric-card";
import { ReadOnlyField } from "@/shared/ui/read-only-field";

const traderSchema = z
  .object({
    id: z.number().optional(),
    login: z.string().min(1, "Введите логин"),
    password: z.string().optional(),
    externalWorkerName: z.string().min(1, "Введите external worker name"),
    salaryPercent: z.coerce.number().min(0, "Минимум 0").max(100, "Максимум 100"),
    status: z.enum(["active", "disabled"]),
  })
  .superRefine((values, context) => {
    if (!values.id && !values.password) {
      context.addIssue({ code: "custom", path: ["password"], message: "Пароль обязателен при создании" });
    }
  });

const requisiteSchema = z.object({
  id: z.number().optional(),
  phone: z.string().min(1, "Введите телефон").refine(isValidRussianPhone, "Введите телефон в формате +7 (XXX) XXX-XX-XX"),
  bankCode: z.string().min(1, "Выберите банк"),
  proxy: z.string().min(1, "Введите proxy"),
  employeeComment: z.string().optional(),
  assignedTraderId: z.string(),
  status: z.enum(["active", "archived"]),
});

const planSchema = z.object({
  assignmentId: z.number().optional(),
  requisiteId: z.string().min(1, "Выберите реквизит"),
  traderId: z.string().min(1, "Выберите трейдера"),
  assignedForDate: z.string().min(1, "Выберите дату"),
  targetTurnover: z.string().min(1, "Введите лимит").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
  comment: z.string().optional(),
});

type TraderForm = z.infer<typeof traderSchema>;
type RequisiteForm = z.infer<typeof requisiteSchema>;
type PlanForm = z.infer<typeof planSchema>;

type TeamleadRequisiteTab = "all" | "activity" | "planning";
type RequisiteReportTarget = {
  id: number;
  phone: string;
  bankName: string;
};

const TEAMLEAD_PERIOD_FILTER_STORAGE_KEY = "astra-crm:teamlead-period-filter";

export function TeamleadPeriodsPage() {
  const periodsQuery = useQuery({ queryKey: queryKeys.teamlead.periods, queryFn: api.periods.list });
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsPeriod, setDetailsPeriod] = useState<AccountingPeriod | null>(null);
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);
  const [selectedPeriodId, setSelectedPeriodId] = useState<number | undefined>(undefined);
  const periods = periodsQuery.data ?? [];
  useEffect(() => {
    if (!selectedPeriodId && periods.length) {
      setSelectedPeriodId(periods[0].id);
    }
  }, [periods, selectedPeriodId]);
  const selectedPeriod = periods.find((period) => period.id === selectedPeriodId);
  const inboundPeriodQuery = useQuery({
    queryKey: queryKeys.teamlead.periodReconciliation(selectedPeriodId, "inbound"),
    queryFn: () => api.teamleadReports.periodReconciliation(selectedPeriodId ?? 0, "inbound"),
    enabled: Boolean(selectedPeriodId),
  });
  const outboundPeriodQuery = useQuery({
    queryKey: queryKeys.teamlead.periodReconciliation(selectedPeriodId, "outbound"),
    queryFn: () => api.teamleadReports.periodReconciliation(selectedPeriodId ?? 0, "outbound"),
    enabled: Boolean(selectedPeriodId),
  });
  const inboundItemsQuery = useQuery({
    queryKey: queryKeys.teamlead.periodReconciliationItems(selectedPeriodId, "inbound"),
    queryFn: () => api.teamleadReports.periodReconciliationItems(selectedPeriodId ?? 0, "inbound"),
    enabled: Boolean(selectedPeriodId),
  });
  const outboundItemsQuery = useQuery({
    queryKey: queryKeys.teamlead.periodReconciliationItems(selectedPeriodId, "outbound"),
    queryFn: () => api.teamleadReports.periodReconciliationItems(selectedPeriodId ?? 0, "outbound"),
    enabled: Boolean(selectedPeriodId),
  });
  const periodIssueCount = (inboundItemsQuery.data?.length ?? 0) + (outboundItemsQuery.data?.length ?? 0);
  const columns = useMemo<ColumnDef<AccountingPeriod>[]>(
    () => [
      { accessorKey: "title", header: "Сверка" },
      { accessorKey: "dateRange", header: "Даты" },
      { accessorKey: "inboundStatus", header: "Инвойсы", cell: ({ row }) => <StatusBadge status={row.original.inboundStatus} /> },
      { accessorKey: "outboundStatus", header: "Выплаты", cell: ({ row }) => <StatusBadge status={row.original.outboundStatus} /> },
      { accessorKey: "status", header: "Статус", cell: ({ row }) => <StatusBadge status={row.original.status} /> },
    ],
    [],
  );
  return (
    <div className="space-y-6">
      <PageHeader
        title="Сверка"
        description="История месячных сверок CSV тимлида с оборотами CRM."
        actions={
          <Button type="button" onClick={() => setUploadDialogOpen(true)}>
            <FileText className="h-4 w-4" />
            Загрузить сверку
          </Button>
        }
      />
      <TeamleadReconciliationStatusCard
        period={selectedPeriod}
        inbound={inboundPeriodQuery.data}
        outbound={outboundPeriodQuery.data}
        issueCount={periodIssueCount}
        isLoading={periodsQuery.isLoading}
      />
      <TeamleadReconciliationHistoryCard
        periods={periods}
        columns={columns}
        pagination={pagination}
        onPaginationChange={setPagination}
        isLoading={periodsQuery.isLoading}
        error={periodsQuery.error instanceof Error ? periodsQuery.error.message : null}
        onOpenPeriod={setDetailsPeriod}
      />
      <PeriodDetailsDialog period={detailsPeriod} onClose={() => setDetailsPeriod(null)} />
      <TeamleadReconciliationUploadDialog
        open={uploadDialogOpen}
        onOpenChange={setUploadDialogOpen}
        periods={periods}
        selectedPeriod={selectedPeriod}
        selectedPeriodId={selectedPeriodId}
        onPeriodChange={setSelectedPeriodId}
        inbound={inboundPeriodQuery.data}
        outbound={outboundPeriodQuery.data}
        inboundItems={inboundItemsQuery.data ?? []}
        isLoading={periodsQuery.isLoading || inboundPeriodQuery.isLoading || outboundPeriodQuery.isLoading}
        error={
          periodsQuery.error instanceof Error
            ? periodsQuery.error.message
            : inboundPeriodQuery.error instanceof Error
              ? inboundPeriodQuery.error.message
              : outboundPeriodQuery.error instanceof Error
                ? outboundPeriodQuery.error.message
                : null
        }
      />
    </div>
  );
}

function TeamleadReconciliationStatusCard({
  period,
  inbound,
  outbound,
  issueCount,
  isLoading,
}: {
  period?: AccountingPeriod;
  inbound?: ReconciliationSummary | null;
  outbound?: ReconciliationSummary | null;
  issueCount: number;
  isLoading?: boolean;
}) {
  if (isLoading) return <EmptyState title="Загружаем сверки" />;
  if (!period) {
    return (
      <Card>
        <CardContent className="p-4">
          <EmptyState
            title="Истории сверок пока нет"
            description="Создайте accounting period и загрузите CSV тимлида через кнопку «Загрузить сверку»."
          />
        </CardContent>
      </Card>
    );
  }

  const hasMismatch = inbound?.status === "mismatch" || outbound?.status === "mismatch" || issueCount > 0;

  return (
    <Card className={hasMismatch ? "border-amber-200 bg-amber-50" : "border-emerald-200 bg-emerald-50"}>
      <CardContent className="grid gap-4 p-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-semibold">Последняя сверка: {period.title}</span>
            <StatusBadge status={period.status} />
            {hasMismatch ? <span className="text-sm font-medium text-amber-900">есть расхождения</span> : <span className="text-sm font-medium text-emerald-800">актуально</span>}
          </div>
          <div className="text-sm text-muted-foreground">
            Загрузка CSV тимлида актуализирует транзакции за большой период: статусы по innerId обновляются, затем пересчитывается оборот и расхождения по реквизитам.
          </div>
        </div>
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-1">
          <ChecklistLine ok={inbound?.status === "matched"} label="Инвойсы" detail={periodStatusDetail(inbound)} />
          <ChecklistLine ok={outbound?.status === "matched"} label="Выплаты" detail={periodStatusDetail(outbound)} />
          <ChecklistLine ok={issueCount === 0} label="Проблемные реквизиты" detail={issueCount ? `${issueCount} строк` : "нет"} />
        </div>
      </CardContent>
    </Card>
  );
}

function ChecklistLine({ ok, label, detail }: { ok: boolean; label: string; detail?: string }) {
  return (
    <div className={ok ? "rounded-md border border-emerald-200 bg-white/70 p-3" : "rounded-md border border-amber-200 bg-white/70 p-3"}>
      <div className="flex items-center justify-between gap-2">
        <span className={ok ? "text-sm font-medium text-emerald-900" : "text-sm font-medium text-amber-950"}>{label}</span>
        <StatusBadge status={ok ? "matched" : "mismatch"} />
      </div>
      {detail ? <div className="mt-1 text-xs text-muted-foreground">{detail}</div> : null}
    </div>
  );
}

function periodStatusDetail(summary?: ReconciliationSummary | null) {
  if (!summary) return "не запускалась";
  if (summary.status === "matched") return "сошлось";
  if (summary.status === "accepted_with_comment") return "принято с комментарием";
  return formatMoneyMinor(summary.diffMinor);
}

function TeamleadReconciliationHistoryCard({
  periods,
  columns,
  pagination,
  onPaginationChange,
  isLoading,
  error,
  onOpenPeriod,
}: {
  periods: AccountingPeriod[];
  columns: ColumnDef<AccountingPeriod>[];
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  isLoading?: boolean;
  error?: string | null;
  onOpenPeriod: (period: AccountingPeriod) => void;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>История сверок</CardTitle>
      </CardHeader>
      <CardContent>
        <DataTable
          columns={columns}
          data={periods}
          rowCount={periods.length}
          pagination={pagination}
          onPaginationChange={onPaginationChange}
          isLoading={isLoading}
          error={error}
          emptyTitle="Сверок пока нет"
          emptyDescription="Записи появятся после создания accounting period."
          onRowClick={onOpenPeriod}
          actions={[{ label: "Детали", onSelect: onOpenPeriod }]}
        />
      </CardContent>
    </Card>
  );
}

function TeamleadReconciliationUploadDialog({
  open,
  onOpenChange,
  periods,
  selectedPeriod,
  selectedPeriodId,
  onPeriodChange,
  inbound,
  outbound,
  inboundItems,
  isLoading,
  error,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  periods: AccountingPeriod[];
  selectedPeriod?: AccountingPeriod;
  selectedPeriodId?: number;
  onPeriodChange: (periodId: number | undefined) => void;
  inbound?: ReconciliationSummary | null;
  outbound?: ReconciliationSummary | null;
  inboundItems: ReconciliationItem[];
  isLoading?: boolean;
  error?: string | null;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[1240px] p-6">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Загрузить сверку</DialogTitle>
          <DialogDescription>Выберите период, загрузите CSV тимлида и проверьте расхождения по реквизитам.</DialogDescription>
        </DialogHeader>
        <div className="max-h-[76vh] overflow-y-auto pr-2">
          <TeamleadPeriodReconciliationPanel
            periods={periods}
            selectedPeriod={selectedPeriod}
            selectedPeriodId={selectedPeriodId}
            onPeriodChange={onPeriodChange}
            inbound={inbound}
            outbound={outbound}
            inboundItems={inboundItems}
            isLoading={isLoading}
            error={error}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}

function TeamleadPeriodReconciliationPanel({
  periods,
  selectedPeriod,
  selectedPeriodId,
  onPeriodChange,
  inbound,
  outbound,
  inboundItems,
  isLoading,
  error,
}: {
  periods: AccountingPeriod[];
  selectedPeriod?: AccountingPeriod;
  selectedPeriodId?: number;
  onPeriodChange: (periodId: number | undefined) => void;
  inbound?: ReconciliationSummary | null;
  outbound?: ReconciliationSummary | null;
  inboundItems: ReconciliationItem[];
  isLoading?: boolean;
  error?: string | null;
}) {
  const hasPeriod = Boolean(selectedPeriodId);

  return (
    <div className="space-y-5">
      <Card className="border-blue-200 bg-blue-50">
        <CardContent className="p-3 text-sm text-blue-950">
          CSV тимлида нужен не как отдельный отчет ради просмотра, а как переимпорт большого периода: строки обновляют транзакции по innerId, после чего пересчитываются обороты и остаются только реквизиты с расхождением.
        </CardContent>
      </Card>

      <section className="space-y-3">
        <h2 className="text-sm font-semibold">1. Период и CSV</h2>
        <div className="rounded-md border border-border bg-white p-4">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div className="min-w-64">
              <div className="mb-2 text-sm font-semibold">Месячный период</div>
              <Select
                value={selectedPeriodId ? String(selectedPeriodId) : ""}
                onChange={(event) => onPeriodChange(event.target.value ? Number(event.target.value) : undefined)}
                disabled={!periods.length}
              >
                {!periods.length ? <option value="">Периодов нет</option> : null}
                {periods.map((period) => (
                  <option key={period.id} value={period.id}>
                    {period.title} · {period.dateRange}
                  </option>
                ))}
              </Select>
            </div>

            <div className="grid min-w-[520px] flex-1 gap-3 md:grid-cols-2">
              <InlineTeamleadImportControl
                label="Входящие"
                direction="inbound"
                accountingPeriodId={selectedPeriodId}
                disabled={!selectedPeriodId}
              />
              <InlineTeamleadImportControl
                label="Выплаты"
                direction="outbound"
                accountingPeriodId={selectedPeriodId}
                disabled={!selectedPeriodId}
              />
            </div>
          </div>

          {selectedPeriod ? (
            <div className="mt-2 text-sm text-muted-foreground">
              {selectedPeriod.dateRange}. Повторная загрузка заменит активный CSV этого периода и запустит пересчет.
            </div>
          ) : null}

          {error ? <div className="mt-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{error}</div> : null}
          {!isLoading && !hasPeriod ? (
            <EmptyState title="Нет accounting period" description="Создайте месячный период, чтобы загрузить CSV и запустить сверку." />
          ) : null}
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="text-sm font-semibold">2. Результат сверки</h2>
        <div className="grid gap-3 xl:grid-cols-2">
          <div className="space-y-3">
            {inbound?.status === "mismatch" ? <MismatchAlert summary={inbound} title="Есть расхождение по входящим" /> : null}
            <ReportReconciliationDetails
              title="Входящие за месяц"
              summary={inbound}
              isLoading={isLoading}
              csvLabel="CSV тимлида"
              crmLabel="CRM"
              diffLabel="Расхождение"
            />
            <PeriodRequisiteMismatchTable items={inboundItems} isLoading={isLoading} />
          </div>
          <div className="space-y-3">
            {outbound?.status === "mismatch" ? <MismatchAlert summary={outbound} title="Есть расхождение по выплатам" /> : null}
            <ReportReconciliationDetails
              title="Выплаты за месяц"
              summary={outbound}
              isLoading={isLoading}
              csvLabel="CSV тимлида"
              crmLabel="Выплаты трейдеров"
              diffLabel="Расхождение"
            />
          </div>
        </div>
      </section>
    </div>
  );
}

function InlineTeamleadImportControl({
  label,
  direction,
  accountingPeriodId,
  disabled,
}: {
  label: string;
  direction: OrderDirection;
  accountingPeriodId?: number;
  disabled?: boolean;
}) {
  const queryClient = useQueryClient();
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const uploadMutation = useMutation({
    mutationFn: api.imports.upload,
    onSuccess: async (result) => {
      setSuccess(`Загружено строк: ${result.importedRows}`);
      setError(null);
      setFile(null);
      await queryClient.invalidateQueries({ queryKey: queryKeys.teamlead.periods });
      await queryClient.invalidateQueries({ queryKey: queryKeys.teamlead.periodReconciliation(accountingPeriodId, direction) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.teamlead.periodReconciliationItems(accountingPeriodId, direction) });
    },
    onError: (nextError) => {
      setSuccess(null);
      setError(nextError instanceof Error ? nextError.message : "Не удалось импортировать CSV");
    },
  });

  return (
    <div className="rounded-md border border-border bg-slate-50 p-3">
      <FormField label={`${label} CSV`} help="CSV с разделителем | из внешней админки.">
        <Input
          type="file"
          accept=".csv,text/csv"
          disabled={disabled || uploadMutation.isPending}
          onChange={(event) => {
            setFile(event.target.files?.[0] ?? null);
            setError(null);
            setSuccess(null);
          }}
        />
      </FormField>
      {error ? <div className="mt-2 rounded-md border border-red-200 bg-red-50 p-2 text-xs text-red-800">{error}</div> : null}
      {success ? <div className="mt-2 rounded-md border border-emerald-200 bg-emerald-50 p-2 text-xs text-emerald-800">{success}</div> : null}
      <div className="mt-3 flex justify-end">
        <Button
          type="button"
          size="sm"
          disabled={!file || disabled || uploadMutation.isPending}
          onClick={() => {
            if (!file || !accountingPeriodId) return;
            uploadMutation.mutate({ file, scope: "teamlead", direction, accountingPeriodId });
          }}
        >
          Загрузить и пересчитать
        </Button>
      </div>
    </div>
  );
}

type PeriodRequisiteMismatchRow = {
  id: number;
  requisite: string;
  trader: string;
  csvAmountMinor: number;
  crmAmountMinor: number;
  diffMinor: number;
  csvCount: number;
  crmCount: number;
  amountMismatch: boolean;
  countMismatch: boolean;
};

function PeriodRequisiteMismatchTable({ items, isLoading }: { items: ReconciliationItem[]; isLoading?: boolean }) {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const rows = useMemo(() => periodRequisiteMismatchRows(items), [items]);
  const columns = useMemo<ColumnDef<PeriodRequisiteMismatchRow>[]>(
    () => [
      {
        accessorKey: "requisite",
        header: "Реквизит",
        cell: ({ row }) => <span className="font-medium">{row.original.requisite}</span>,
      },
      {
        accessorKey: "trader",
        header: "Трейдер",
        cell: ({ row }) => <span className="text-muted-foreground">{row.original.trader}</span>,
      },
      {
        accessorKey: "csvAmountMinor",
        header: () => <div className="text-right">CSV тимлида</div>,
        cell: ({ row }) => <MismatchMoneyCell value={row.original.csvAmountMinor} warning={row.original.amountMismatch} />,
      },
      {
        accessorKey: "crmAmountMinor",
        header: () => <div className="text-right">CRM</div>,
        cell: ({ row }) => <MismatchMoneyCell value={row.original.crmAmountMinor} warning={row.original.amountMismatch} />,
      },
      {
        accessorKey: "diffMinor",
        header: () => <div className="text-right">Расхождение</div>,
        cell: ({ row }) => <MismatchMoneyCell value={row.original.diffMinor} warning />,
      },
      {
        id: "count",
        header: () => <div className="text-right">Кол-во</div>,
        cell: ({ row }) => (
          <div className={row.original.countMismatch ? "text-right font-semibold text-red-700" : "text-right text-muted-foreground"}>
            {row.original.csvCount} / {row.original.crmCount}
          </div>
        ),
      },
    ],
    [],
  );

  return (
    <DataTable
      columns={columns}
      data={rows}
      rowCount={rows.length}
      pagination={pagination}
      onPaginationChange={setPagination}
      isLoading={isLoading}
      emptyTitle="Расхождений по реквизитам нет"
      emptyDescription="В таблице показываются только реквизиты, где CSV тимлида отличается от CRM."
      getRowClassName={() => "bg-red-50/50"}
    />
  );
}

function MismatchMoneyCell({ value, warning }: { value: number; warning?: boolean }) {
  return (
    <div className={warning ? "text-right font-semibold text-red-700 tabular-nums" : "text-right tabular-nums"}>
      {formatMoneyMinor(value)}
    </div>
  );
}

function periodRequisiteMismatchRows(items: ReconciliationItem[]): PeriodRequisiteMismatchRow[] {
  return items
    .filter((item) => item.issueType === "requisite_amount_mismatch")
    .map((item) => {
      const csv = item.teamleadValue ?? {};
      const crm = item.traderValue ?? {};
      const csvAmountMinor = numberValue(csv.successAmountMinor);
      const crmAmountMinor = numberValue(crm.successAmountMinor);
      const csvCount = numberValue(csv.successCount);
      const crmCount = numberValue(crm.successCount);
      const requisite = stringValue(csv.requisitePhone) || stringValue(crm.requisitePhone) || "Без реквизита";
      const trader = stringValue(crm.traderLogin) || stringValue(crm.traderId) || "Не найден";

      return {
        id: item.id,
        requisite,
        trader,
        csvAmountMinor,
        crmAmountMinor,
        diffMinor: crmAmountMinor - csvAmountMinor,
        csvCount,
        crmCount,
        amountMismatch: csvAmountMinor !== crmAmountMinor,
        countMismatch: csvCount !== crmCount,
      };
    })
    .sort((left, right) => Math.abs(right.diffMinor) - Math.abs(left.diffMinor));
}

function numberValue(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function stringValue(value: unknown) {
  if (typeof value === "string") return value;
  if (typeof value === "number") return String(value);
  return "";
}

function PeriodDetailsDialog({ period, onClose }: { period: AccountingPeriod | null; onClose: () => void }) {
  const inboundQuery = useQuery({
    queryKey: queryKeys.teamlead.periodReconciliation(period?.id, "inbound"),
    queryFn: () => api.teamleadReports.periodReconciliation(period?.id ?? 0, "inbound"),
    enabled: Boolean(period?.id),
  });
  const outboundQuery = useQuery({
    queryKey: queryKeys.teamlead.periodReconciliation(period?.id, "outbound"),
    queryFn: () => api.teamleadReports.periodReconciliation(period?.id ?? 0, "outbound"),
    enabled: Boolean(period?.id),
  });
  const inboundItemsQuery = useQuery({
    queryKey: queryKeys.teamlead.periodReconciliationItems(period?.id, "inbound"),
    queryFn: () => api.teamleadReports.periodReconciliationItems(period?.id ?? 0, "inbound"),
    enabled: Boolean(period?.id),
  });

  return (
    <Dialog open={Boolean(period)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-[1240px] p-6">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{period?.title}</DialogTitle>
          <DialogDescription>Детали сверки по учетному периоду и реквизиты с расхождением.</DialogDescription>
        </DialogHeader>
        {period ? (
          <div className="max-h-[76vh] space-y-5 overflow-y-auto pr-2">
            <div className="grid gap-3 md:grid-cols-2">
              <ReadOnlyField label="Даты" value={period.dateRange} />
              <div className="rounded-md border border-border p-3">
                <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Статус периода</div>
                <StatusBadge status={period.status} />
              </div>
              <div className="rounded-md border border-border p-3">
                <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Инвойсы</div>
                <StatusBadge status={period.inboundStatus} />
              </div>
              <div className="rounded-md border border-border p-3">
                <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Выплаты</div>
                <StatusBadge status={period.outboundStatus} />
              </div>
            </div>

            <section className="space-y-3">
              <h2 className="text-sm font-semibold">Инвойсы</h2>
              {inboundQuery.data?.status === "mismatch" ? (
                <MismatchAlert summary={inboundQuery.data} title="Есть расхождение по входящим" />
              ) : null}
              <ReportReconciliationDetails
                title="Входящие за месяц"
                summary={inboundQuery.data}
                isLoading={inboundQuery.isLoading}
                csvLabel="CSV тимлида"
                crmLabel="CRM"
                diffLabel="Расхождение"
              />
              <PeriodRequisiteMismatchTable items={inboundItemsQuery.data ?? []} isLoading={inboundItemsQuery.isLoading} />
            </section>

            <section className="space-y-3">
              <h2 className="text-sm font-semibold">Выплаты</h2>
              {outboundQuery.data?.status === "mismatch" ? (
                <MismatchAlert summary={outboundQuery.data} title="Есть расхождение по выплатам" />
              ) : null}
              <ReportReconciliationDetails
                title="Выплаты за месяц"
                summary={outboundQuery.data}
                isLoading={outboundQuery.isLoading}
                csvLabel="CSV тимлида"
                crmLabel="Выплаты трейдеров"
                diffLabel="Расхождение"
              />
            </section>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

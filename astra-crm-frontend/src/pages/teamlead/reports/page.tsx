import { zodResolver } from "@hookform/resolvers/zod";
import { keepPreviousData, useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { ArrowDownLeft, ArrowUpRight, CalendarDays, Copy, Eye, FileText, History, Pencil, Plus, UserRound, X } from "lucide-react";
import { useCallback, useDeferredValue, useEffect, useMemo, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { Bar, CartesianGrid, Cell, ComposedChart, Line, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { z } from "zod";
import { DateTimeCell } from "@/shared/ui/date-time-cell";
import { EmptyState } from "@/shared/ui/empty-state";
import { FormField } from "@/shared/ui/form-field";
import { AcceptMismatchDialog } from "@/features/import-csv/ui/import-components";
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
import { Card, CardContent } from "@/shared/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
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
  AuditLogEntry,
  Bank,
  Order,
  OrderDirection,
  Requisite,
  RequisiteAssignmentEvent,
  RequisiteAssignmentWorkRow,
  RequisiteReport,
  RequisiteReportShift,
  ShiftReportReconciliation,
  ShiftReportRow,
  ShiftReport,
  ShiftRequisite,
  Trader,
} from "@/shared/model/domain";
import { api } from "@/shared/api/api";
import { filterOrdersBySearch } from "@/shared/lib/order-filters";
import type { PeriodFilter } from "@/shared/lib/period-filter";
import { usePersistentPeriodFilter } from "@/shared/lib/period-filter";
import { queryKeys } from "@/shared/api/query-keys";
import { paginationToQuery } from "@/shared/lib/pagination";
import {
  bpsToPercent,
  cn,
  formatCardNumber,
  formatDateTime,
  formatMoneyMinor,
  formatRussianPhone,
  isValidRussianPhone,
  normalizeRussianPhone,
  parseMoneyToMinor,
  percentToBps,
  phoneDigits,
} from "@/shared/lib/utils";
import { MetricCard } from "@/shared/ui/metric-card";

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

export function TeamleadReportsPage() {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 });
  const reportsQuery = useQuery({
    queryKey: queryKeys.teamlead.shiftHistory(paginationToQuery(pagination)),
    queryFn: () => api.teamleadReports.history(paginationToQuery(pagination)),
    placeholderData: keepPreviousData,
  });
  const tradersQuery = useQuery({
    queryKey: queryKeys.teamlead.traders({ status: "active" }),
    queryFn: () => api.traders.list({ status: "active", page: 1, pageSize: 200 }),
  });
  const [selectedReport, setSelectedReport] = useState<ShiftReport | null>(null);
  const tradersById = useMemo(() => new Map((tradersQuery.data?.items ?? []).map((trader) => [trader.id, trader])), [tradersQuery.data?.items]);
  const reports = reportsQuery.data?.items ?? [];
  const discrepancyCount = reports.filter((report) => report.status === "closed_with_discrepancy").length;
  const traderCount = new Set(reports.map((report) => report.traderId)).size;
  const columns = useMemo<ColumnDef<ShiftReport>[]>(
    () => [
      {
        accessorKey: "id",
        header: "Отчет",
        cell: ({ row }) => (
          <span className="inline-flex items-center gap-2 font-medium">
            <Eye className="h-4 w-4 text-muted-foreground" />
            #{row.original.id}
          </span>
        ),
      },
      {
        accessorKey: "traderId",
        header: "Трейдер",
        cell: ({ row }) => {
          const trader = tradersById.get(row.original.traderId);
          return <UserCell login={trader?.login ?? `ID ${row.original.traderId}`} secondary={trader?.externalWorkerName} />;
        },
      },
      {
        accessorKey: "startedAt",
        header: "Период работы",
        cell: ({ row }) => (
          <div>
            <DateTimeCell value={row.original.startedAt} />
            <div className="text-xs text-muted-foreground">закрыта {formatDateTime(row.original.closedAt ?? row.original.endedAt)}</div>
          </div>
        ),
      },
      {
        accessorKey: "inboundReconciliationStatus",
        header: "Инвойсы",
        cell: ({ row }) => <StatusBadge status={row.original.inboundReconciliationStatus} />,
      },
      {
        accessorKey: "outboundReconciliationStatus",
        header: "Выплаты",
        cell: ({ row }) => <StatusBadge status={row.original.outboundReconciliationStatus} />,
      },
      { accessorKey: "status", header: "Статус", cell: ({ row }) => <StatusBadge status={row.original.status} /> },
      {
        accessorKey: "closeComment",
        header: "Комментарий",
        cell: ({ row }) => (
          <span className="block max-w-xs truncate text-muted-foreground" title={row.original.closeComment ?? ""}>
            {row.original.closeComment || "—"}
          </span>
        ),
      },
    ],
    [tradersById],
  );

  return (
    <div className="space-y-6">
      <PageHeader title="Отчеты" description="История закрытых смен трейдеров и детали реквизитов, по которым они работали." />
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard label="Отчетов" value={String(reports.length)} />
        <MetricCard label="Трейдеров" value={String(traderCount)} />
        <MetricCard label="С расхождением" value={String(discrepancyCount)} warning={discrepancyCount > 0} />
      </div>
      <DataTable
        columns={columns}
        data={reports}
        rowCount={reportsQuery.data?.total ?? 0}
        pagination={pagination}
        onPaginationChange={setPagination}
        serverSidePagination
        isLoading={reportsQuery.isLoading || tradersQuery.isLoading}
        isFetching={reportsQuery.isFetching || tradersQuery.isFetching}
        error={reportsQuery.error instanceof Error ? reportsQuery.error.message : null}
        emptyTitle="Отчетов пока нет"
        emptyDescription="Закрытые смены трейдеров будут появляться здесь после сдачи отчетов."
        onRowClick={setSelectedReport}
        actions={[{ label: "Детали", onSelect: (row) => setSelectedReport(row) }]}
      />
      <TeamleadReportDetailsDialog
        report={selectedReport}
        trader={selectedReport ? tradersById.get(selectedReport.traderId) : undefined}
        onClose={() => setSelectedReport(null)}
      />
    </div>
  );
}

function TeamleadReportDetailsDialog({
  report,
  trader,
  onClose,
}: {
  report: ShiftReport | null;
  trader?: Trader;
  onClose: () => void;
}) {
  const reportQuery = useQuery({
    queryKey: report ? queryKeys.teamlead.shiftReport(report.id) : ["teamlead", "shift", "report", "empty"],
    queryFn: () => api.teamleadReports.report(report?.id ?? 0),
    enabled: Boolean(report),
  });
  const details = reportQuery.data;
  const rows = details?.rows ?? [];
  const shift = details?.shift ?? report;

  return (
    <Dialog open={Boolean(report)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-[1240px] p-6">
        <DialogHeader>
          <DialogTitle className="flex flex-wrap items-center gap-2 text-base font-semibold">
            <span>Детали отчета {report ? `#${report.id}` : ""}</span>
            <ReportStatusBadge label="Инвойсы" summary={details?.inbound} />
            <ReportStatusBadge label="Выплаты" summary={details?.outbound} />
          </DialogTitle>
          <DialogDescription>
            {shift
              ? `${trader?.login ?? `Трейдер ID ${shift.traderId}`} · ${formatDateTime(shift.startedAt)} - ${formatDateTime(shift.closedAt ?? shift.endedAt)}`
              : "Реквизиты, которые были в работе в выбранной смене."}
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[72vh] space-y-4 overflow-y-auto pr-2">
          <div className="grid gap-3 lg:grid-cols-2">
            <ReportReconciliationDetails
              title="Инвойсы"
              summary={details?.inbound}
              isLoading={reportQuery.isLoading}
              csvLabel="CSV инвойсы"
              crmLabel="CRM входы"
              diffLabel="Расхождение"
            />
            <ReportReconciliationDetails
              title="Выплаты"
              summary={details?.outbound}
              isLoading={reportQuery.isLoading}
              csvLabel="CSV выплаты"
              crmLabel="CRM выходы"
              diffLabel="Расхождение"
            />
          </div>

          {reportQuery.isLoading ? <EmptyState title="Загружаем отчет" /> : null}
          {reportQuery.error instanceof Error ? (
            <EmptyState title="Не удалось загрузить отчет" description={reportQuery.error.message} />
          ) : null}
          {!reportQuery.isLoading && !reportQuery.error && !rows.length ? (
            <EmptyState title="Реквизитов в отчете нет" description="В этой смене не найдено взятых в работу реквизитов." />
          ) : null}

          {rows.length ? <TeamleadReportRequisitesTable rows={rows} /> : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function TeamleadReportRequisitesTable({ rows }: { rows: ShiftReportRow[] }) {
  return (
    <div className="overflow-hidden rounded-md border border-border">
      <table className="w-full border-collapse text-sm">
        <thead className="bg-slate-50 text-left text-xs uppercase tracking-normal text-muted-foreground">
          <tr>
            <th className="h-10 border-b border-border px-3 font-medium">Реквизит</th>
            <th className="h-10 border-b border-border px-3 font-medium">Банк</th>
            <th className="h-10 border-b border-border px-3 font-medium">Статус</th>
            <th className="h-10 border-b border-border px-3 text-right font-medium">Оборот по CRM</th>
            <th className="h-10 border-b border-border px-3 text-right font-medium">CSV / переводы</th>
            <th className="h-10 border-b border-border px-3 text-right font-medium">Расхождение</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((item) => (
            <tr
              key={item.rowKey}
              className={cn(
                "border-b border-border last:border-0",
                item.hasMismatch ? "bg-red-50 text-red-950" : "bg-white",
              )}
            >
              <td className="px-3 py-3">
                <ReportRequisiteCell item={item} />
              </td>
              <td className="px-3 py-3">{item.bankName || "—"}</td>
              <td className="px-3 py-3">
                <StatusBadge status={item.status} />
              </td>
              <td className="px-3 py-3 text-right">
                <ReportCrmTurnoverCell item={item} />
              </td>
              <td className="px-3 py-3 text-right">
                <ReportCsvTurnoverCell item={item} />
              </td>
              <td className="px-3 py-3 text-right">
                <ReportDiffCell item={item} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ReportStatusBadge({ label, summary }: { label: string; summary?: ShiftReportReconciliation }) {
  const title = reconciliationStatusTitle(label, summary);
  return (
    <span title={title} className="inline-flex items-center gap-1">
      <span className="text-xs font-normal text-muted-foreground">{label}</span>
      <StatusBadge status={summary?.status ?? "unknown"} />
    </span>
  );
}

function reconciliationStatusTitle(label: string, summary?: ShiftReportReconciliation) {
  if (!summary) return `${label}: сверка не запускалась`;
  if (summary.status === "matched") return `${label}: CSV и CRM сходятся`;
  if (summary.status === "accepted_with_comment") {
    return `${label}: расхождение подтверждено${summary.comment ? `. Комментарий: ${summary.comment}` : ""}`;
  }
  return `${label}: есть расхождение ${formatMoneyMinor(summary.diffMinor)}`;
}

function ReportRequisiteCell({ item }: { item: ShiftReportRow }) {
  const copyPhone = phoneDigits(normalizeRussianPhone(item.phone));
  const canCopyPhone = /^7\d{10}$/.test(copyPhone);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" className="block max-w-[220px] truncate text-left font-medium hover:text-primary">
          {formatRussianPhone(item.phone)}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-72">
        <DropdownMenuLabel>Реквизит</DropdownMenuLabel>
        <CopyDropdownItem label="Телефон" value={formatRussianPhone(item.phone)} copyValue={canCopyPhone ? copyPhone : item.phone} />
        <CopyDropdownItem label="Карта" value={formatCardNumber(item.cardNumber)} copyValue={item.cardNumber} />
        <CopyDropdownItem label="ФИО" value={item.holderName || "—"} copyValue={item.holderName} />
        {item.proxy ? <CopyDropdownItem label="Proxy" value={item.proxy} copyValue={item.proxy} /> : null}
        {item.csvOnly ? <DropdownMenuItem className="text-red-700">Есть в CSV, но нет в смене</DropdownMenuItem> : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function ReportCrmTurnoverCell({ item }: { item: ShiftReportRow }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" className="ml-auto block text-right tabular-nums hover:text-primary">
          <ReportAmountStack
            inboundValue={item.inboundTurnoverMinor}
            outboundValue={item.outboundTurnoverMinor}
          />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuLabel>Оборот по CRM</DropdownMenuLabel>
        <AmountDropdownItem label="Входы" value={item.inboundTurnoverMinor} />
        <AmountDropdownItem label="Выходы" value={item.outboundTurnoverMinor} />
        <AmountDropdownItem label="Остаток" value={item.closingBalanceMinor} />
        <AmountDropdownItem label="Лимит" value={item.targetTurnoverMinor} />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function ReportCsvTurnoverCell({ item }: { item: ShiftReportRow }) {
  return (
    <ReportAmountStack
      inboundValue={item.csvInboundMinor}
      outboundValue={item.csvOutboundMinor}
    />
  );
}

function ReportDiffCell({ item }: { item: ShiftReportRow }) {
  const hasInboundDiff = item.inboundDiffMinor !== 0;
  const hasOutboundDiff = item.outboundDiffMinor !== 0;
  if (!hasInboundDiff && !hasOutboundDiff) return <span className="text-muted-foreground">—</span>;

  return (
    <div className="flex min-h-[44px] flex-col justify-center gap-1 tabular-nums">
      {hasInboundDiff ? <ReportAmountLine type="inbound" value={item.inboundDiffMinor} tone="danger" /> : null}
      {hasOutboundDiff ? <ReportAmountLine type="outbound" value={item.outboundDiffMinor} tone="danger" /> : null}
    </div>
  );
}

function ReportAmountStack({ inboundValue, outboundValue }: { inboundValue: number; outboundValue: number }) {
  return (
    <div className="flex min-h-[44px] flex-col justify-center gap-1 tabular-nums">
      <ReportAmountLine type="inbound" value={inboundValue} />
      <ReportAmountLine type="outbound" value={outboundValue} />
    </div>
  );
}

function ReportAmountLine({
  type,
  value,
  tone = "default",
}: {
  type: "inbound" | "outbound";
  value: number;
  tone?: "default" | "danger";
}) {
  const Icon = type === "inbound" ? ArrowDownLeft : ArrowUpRight;
  const title = type === "inbound" ? "Входящие: инвойсы" : "Исходящие: выплаты";
  return (
    <div className={cn("flex items-center justify-end gap-2 text-sm font-medium", tone === "danger" ? "text-red-700" : undefined)}>
      <span
        aria-label={title}
        title={title}
        className={cn("inline-flex h-4 w-4 shrink-0 items-center justify-center", tone === "danger" ? "text-red-600" : "text-muted-foreground")}
      >
        <Icon aria-hidden="true" className="h-4 w-4" />
      </span>
      <span>{formatMoneyMinor(value)}</span>
    </div>
  );
}

function CopyDropdownItem({ label, value, copyValue }: { label: string; value: string; copyValue?: string }) {
  return (
    <DropdownMenuItem
      onSelect={() => {
        if (copyValue) void navigator.clipboard?.writeText(copyValue);
      }}
      className="justify-between gap-3"
    >
      <span className="text-muted-foreground">{label}</span>
      <span className="truncate text-right font-medium">{value}</span>
    </DropdownMenuItem>
  );
}

function AmountDropdownItem({ label, value }: { label: string; value: number }) {
  return (
    <DropdownMenuItem className="justify-between gap-3">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium tabular-nums">{formatMoneyMinor(value)}</span>
    </DropdownMenuItem>
  );
}

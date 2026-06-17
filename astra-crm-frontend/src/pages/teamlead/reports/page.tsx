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
import { AcceptMismatchDialog, ImportCsvDialog, MismatchAlert } from "@/features/import-csv/ui/import-components";
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
  targetTurnover: z.string().min(1, "Введите целевой оборот").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
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
  const reportsQuery = useQuery({ queryKey: queryKeys.teamlead.shiftHistory, queryFn: api.teamleadReports.history });
  const tradersQuery = useQuery({
    queryKey: queryKeys.teamlead.traders({ status: "active" }),
    queryFn: () => api.traders.list({ status: "active" }),
  });
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 });
  const [selectedReport, setSelectedReport] = useState<ShiftReport | null>(null);
  const tradersById = useMemo(() => new Map((tradersQuery.data ?? []).map((trader) => [trader.id, trader])), [tradersQuery.data]);
  const reports = reportsQuery.data ?? [];
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
      <PageHeader title="Отчеты" description="История смен трейдеров и детали реквизитов, по которым они работали." />
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard label="Отчетов" value={String(reports.length)} />
        <MetricCard label="Трейдеров" value={String(traderCount)} />
        <MetricCard label="С расхождением" value={String(discrepancyCount)} warning={discrepancyCount > 0} />
      </div>
      <DataTable
        columns={columns}
        data={reports}
        rowCount={reports.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        isLoading={reportsQuery.isLoading || tradersQuery.isLoading}
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
  const requisitesQuery = useQuery({
    queryKey: report ? queryKeys.teamlead.shiftReportRequisites(report.id) : ["teamlead", "shift", "report", "requisites", "empty"],
    queryFn: () => api.teamleadReports.reportRequisites(report?.id ?? 0),
    enabled: Boolean(report),
  });
  const inboundReconciliationQuery = useQuery({
    queryKey: report ? queryKeys.teamlead.shiftReportReconciliation(report.id, "inbound") : ["teamlead", "shift", "report", "inbound", "empty"],
    queryFn: () => api.teamleadReports.reportReconciliation(report?.id ?? 0, "inbound"),
    enabled: Boolean(report),
  });
  const outboundReconciliationQuery = useQuery({
    queryKey: report ? queryKeys.teamlead.shiftReportReconciliation(report.id, "outbound") : ["teamlead", "shift", "report", "outbound", "empty"],
    queryFn: () => api.teamleadReports.reportReconciliation(report?.id ?? 0, "outbound"),
    enabled: Boolean(report),
  });
  const inboundItemsQuery = useQuery({
    queryKey: report ? queryKeys.teamlead.shiftReportReconciliationItems(report.id, "inbound") : ["teamlead", "shift", "report", "inbound", "items", "empty"],
    queryFn: () => api.teamleadReports.reportReconciliationItems(report?.id ?? 0, "inbound"),
    enabled: Boolean(report),
  });
  const outboundItemsQuery = useQuery({
    queryKey: report ? queryKeys.teamlead.shiftReportReconciliationItems(report.id, "outbound") : ["teamlead", "shift", "report", "outbound", "items", "empty"],
    queryFn: () => api.teamleadReports.reportReconciliationItems(report?.id ?? 0, "outbound"),
    enabled: Boolean(report),
  });
  const requisites = requisitesQuery.data ?? [];
  const inboundTotal = requisites.reduce((sum, item) => sum + item.inboundTurnoverMinor, 0);
  const outboundTotal = requisites.reduce((sum, item) => sum + item.outboundTurnoverMinor, 0);
  const balanceTotal = requisites.reduce((sum, item) => sum + item.closingBalanceMinor, 0);

  return (
    <Dialog open={Boolean(report)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-[1100px] p-6">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Детали отчета {report ? `#${report.id}` : ""}</DialogTitle>
          <DialogDescription>
            {report
              ? `${trader?.login ?? `Трейдер ID ${report.traderId}`} · ${formatDateTime(report.startedAt)} - ${formatDateTime(report.closedAt ?? report.endedAt)}`
              : "Реквизиты, которые были в работе в выбранной смене."}
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[72vh] space-y-4 overflow-y-auto pr-2">
          {report ? (
            <div className="grid gap-3 sm:grid-cols-3">
              <MetricCard label="Оплаты" value={formatMoneyMinor(inboundTotal)} />
              <MetricCard label="Выплаты" value={formatMoneyMinor(outboundTotal)} />
              <MetricCard label="Остаток" value={formatMoneyMinor(balanceTotal)} />
            </div>
          ) : null}

          {requisitesQuery.isLoading ? <EmptyState title="Загружаем реквизиты" /> : null}
          {requisitesQuery.error instanceof Error ? (
            <EmptyState title="Не удалось загрузить реквизиты" description={requisitesQuery.error.message} />
          ) : null}
          {!requisitesQuery.isLoading && !requisitesQuery.error && !requisites.length ? (
            <EmptyState title="Реквизитов в отчете нет" description="В этой смене не найдено взятых в работу реквизитов." />
          ) : null}

          {requisites.length ? <TeamleadReportRequisitesTable requisites={requisites} /> : null}

          <div className="grid gap-4 lg:grid-cols-2">
            <ReportReconciliationDetails
              title="Инвойсы"
              summary={inboundReconciliationQuery.data}
              items={inboundItemsQuery.data ?? []}
              isLoading={inboundReconciliationQuery.isLoading || inboundItemsQuery.isLoading}
            />
            <ReportReconciliationDetails
              title="Выплаты"
              summary={outboundReconciliationQuery.data}
              items={outboundItemsQuery.data ?? []}
              isLoading={outboundReconciliationQuery.isLoading || outboundItemsQuery.isLoading}
            />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function TeamleadReportRequisitesTable({ requisites }: { requisites: ShiftRequisite[] }) {
  return (
    <div className="overflow-hidden rounded-md border border-border">
      <table className="w-full border-collapse text-sm">
        <thead className="bg-slate-50 text-left text-xs uppercase tracking-normal text-muted-foreground">
          <tr>
            <th className="h-10 border-b border-border px-3 font-medium">Реквизит</th>
            <th className="h-10 border-b border-border px-3 font-medium">Карта</th>
            <th className="h-10 border-b border-border px-3 font-medium">Держатель</th>
            <th className="h-10 border-b border-border px-3 font-medium">Статус</th>
            <th className="h-10 border-b border-border px-3 text-right font-medium">Оплаты</th>
            <th className="h-10 border-b border-border px-3 text-right font-medium">Выплаты</th>
            <th className="h-10 border-b border-border px-3 text-right font-medium">Остаток</th>
          </tr>
        </thead>
        <tbody>
          {requisites.map((item) => (
            <tr key={item.id} className="border-b border-border last:border-0">
              <td className="px-3 py-3">
                <RequisiteCell phone={item.phone} method={item.bankName} proxy={item.proxy} />
                {item.employeeComment ? (
                  <div className="mt-1 max-w-xs truncate text-xs text-muted-foreground" title={item.employeeComment}>
                    {item.employeeComment}
                  </div>
                ) : null}
              </td>
              <td className="px-3 py-3 font-mono text-xs">{formatCardNumber(item.cardNumber)}</td>
              <td className="px-3 py-3">{item.holderName || "—"}</td>
              <td className="px-3 py-3">
                <StatusBadge status={item.status} />
              </td>
              <td className="px-3 py-3 text-right">
                <MoneyCell valueMinor={item.inboundTurnoverMinor} />
              </td>
              <td className="px-3 py-3 text-right">
                <MoneyCell valueMinor={item.outboundTurnoverMinor} />
              </td>
              <td className="px-3 py-3 text-right">
                <MoneyCell valueMinor={item.closingBalanceMinor} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}


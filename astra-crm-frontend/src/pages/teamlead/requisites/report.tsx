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
import type { CopyHandler, RequisiteReportTarget } from "./types";
import { assignmentEventLabel, formatDateOnly } from "./planning";

export function RequisiteReportDialog({
  requisite,
  onClose,
  onCopy,
}: {
  requisite: RequisiteReportTarget | null;
  onClose: () => void;
  onCopy: CopyHandler;
}) {
  const reportQuery = useQuery({
    queryKey: queryKeys.teamlead.requisiteReport(requisite?.id),
    queryFn: () => api.requisites.report(requisite?.id ?? 0),
    enabled: Boolean(requisite),
  });
  const report = reportQuery.data;

  return (
    <Dialog open={Boolean(requisite)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-[1180px] p-6">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Отчет по реквизиту</DialogTitle>
          <DialogDescription>{requisite ? requisite.bankName : "История активности реквизита"}</DialogDescription>
        </DialogHeader>

        <div className="max-h-[72vh] space-y-4 overflow-y-auto pr-2">
          {reportQuery.isLoading ? <EmptyState title="Загружаем отчет" /> : null}
          {reportQuery.error instanceof Error ? (
            <EmptyState title="Не удалось загрузить отчет" description={reportQuery.error.message} />
          ) : null}
          {report ? (
            <>
              <RequisiteReportSummaryPanel report={report} onCopy={onCopy} />
              <RequisiteReportShiftCards report={report} />
            </>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function RequisiteReportSummaryPanel({ report, onCopy }: { report: RequisiteReport; onCopy: CopyHandler }) {
  const { summary } = report;
  const identityMeta = [
    { label: "Телефон", value: formatRussianPhone(summary.phone), copyValue: normalizeRussianPhone(summary.phone) },
    summary.cardNumber ? { label: "Карта", value: formatCardNumber(summary.cardNumber), copyValue: summary.cardNumber } : null,
    summary.holderName ? { label: "Держатель", value: summary.holderName, copyValue: summary.holderName } : null,
    summary.proxy ? { label: "Прокси", value: summary.proxy, copyValue: summary.proxy } : null,
  ].filter((item): item is { label: string; value: string; copyValue: string } => Boolean(item));

  return (
    <section className="space-y-3">
      {identityMeta.length > 0 ? (
        <div className="flex flex-wrap gap-2">
          {identityMeta.map((item) => (
            <ReportCopyButton key={item.label} item={item} onCopy={onCopy} />
          ))}
        </div>
      ) : null}
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
        <ReportMetric label="Вход" value={formatMoneyMinor(summary.totalInboundTurnoverMinor)} />
        <ReportMetric label="Выход" value={formatMoneyMinor(summary.totalOutboundTurnoverMinor)} />
        <ReportMetric label="Остаток" value={formatMoneyMinor(summary.lastClosingBalanceMinor)} />
        <ReportMetric label="Смен" value={String(report.shifts.length)} />
        <div className="rounded-md border border-border bg-white px-3 py-2">
          <div className="text-[11px] font-medium uppercase tracking-normal text-muted-foreground">Последний статус</div>
          <div className="mt-1.5 flex items-center justify-between gap-2">
            <StatusBadge status={summary.latestStatus || summary.status} />
            <span className="whitespace-nowrap text-xs text-muted-foreground">{formatDateTime(summary.lastActivityAt)}</span>
          </div>
        </div>
      </div>
    </section>
  );
}

function ReportCopyButton({
  item,
  onCopy,
}: {
  item: { label: string; value: string; copyValue: string };
  onCopy: CopyHandler;
}) {
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      className="h-auto max-w-[300px] justify-start gap-1.5 px-2 py-1.5 text-left"
      title={`Скопировать ${item.label.toLowerCase()}: ${item.value}`}
      onClick={() => onCopy(item.copyValue, item.label)}
    >
      <Copy className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      <span className="text-xs text-muted-foreground">{item.label}:</span>
      <span className="min-w-0 truncate text-sm font-medium tabular-nums">{item.value}</span>
    </Button>
  );
}

function ReportMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-border bg-white px-3 py-2">
      <div className="text-[11px] font-medium uppercase tracking-normal text-muted-foreground">{label}</div>
      <div className="mt-1 whitespace-nowrap text-base font-semibold tabular-nums">{value}</div>
    </div>
  );
}

function RequisiteReportShiftCards({ report }: { report: RequisiteReport }) {
  if (report.shifts.length === 0) {
    return <EmptyState title="Активности по реквизиту пока нет" description="Смены появятся после взятия реквизита в работу." />;
  }

  return (
    <section className="space-y-2">
      <h3 className="text-sm font-semibold">Смены работы реквизита</h3>
      <div className="flex gap-3 overflow-x-auto pb-2">
        {report.shifts.map((shift) => (
          <RequisiteReportShiftCard key={shift.shiftRequisiteId} shift={shift} />
        ))}
      </div>
    </section>
  );
}

function RequisiteReportShiftCard({ shift }: { shift: RequisiteReportShift }) {
  return (
    <article className="min-w-[304px] max-w-[332px] shrink-0 rounded-md border border-border bg-white p-3 shadow-sm">
      <div className="min-w-0">
        <div className="text-sm font-semibold">{formatDateOnly(shift.assignedForDate ?? shift.takenAt)}</div>
        <div className="mt-2 flex items-center gap-2 rounded-md border border-border bg-slate-50 px-2 py-1.5" title={shift.traderLogin}>
          <UserRound className="h-4 w-4 shrink-0 text-muted-foreground" />
          <div className="min-w-0">
            <div className="text-[11px] font-medium uppercase tracking-normal text-muted-foreground">Трейдер</div>
            <div className="truncate text-sm font-medium">{shift.traderLogin}</div>
          </div>
        </div>
      </div>

      <div className="mt-3 flex items-center justify-between gap-3">
        <span className="text-xs text-muted-foreground">Статус</span>
        <StatusBadge status={shift.assignmentStatus || shift.requisiteStatus} />
      </div>

      <div className="mt-3 space-y-1.5">
        <ReportShiftAmount label="Вход" value={shift.inboundTurnoverMinor} title={`Обработал: ${shift.traderLogin}`} strong />
        <ReportShiftAmount label="Выход" value={shift.outboundTurnoverMinor} />
        <ReportShiftAmount label="Лимит" value={shift.targetTurnoverMinor} />
        <ReportShiftAmount label="Остаток" value={shift.closingBalanceMinor} />
      </div>

      <div className="mt-3 grid grid-cols-2 gap-2 border-t border-border pt-3 text-xs">
        <div>
          <div className="text-muted-foreground">Взял</div>
          <div className="mt-0.5 font-medium tabular-nums">{formatDateTime(shift.takenAt)}</div>
        </div>
        <div>
          <div className="text-muted-foreground">Сдал</div>
          <div className="mt-0.5 font-medium tabular-nums">{formatDateTime(shift.releasedAt)}</div>
        </div>
      </div>
    </article>
  );
}

function ReportShiftAmount({ label, value, title, strong = false }: { label: string; value: number; title?: string; strong?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3" title={title}>
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className={strong ? "whitespace-nowrap text-base font-semibold tabular-nums" : "whitespace-nowrap text-sm font-medium tabular-nums"}>
        {formatMoneyMinor(value)}
      </span>
    </div>
  );
}

function AssignmentEventItem({ event }: { event: RequisiteAssignmentEvent }) {
  return (
    <Card>
      <CardContent className="space-y-1 p-3 text-sm">
        <div className="flex items-center justify-between gap-3">
          <span className="font-medium">{assignmentEventLabel(event.action)}</span>
          <DateTimeCell value={event.createdAt} />
        </div>
        {event.comment ? <div className="text-muted-foreground">{event.comment}</div> : null}
      </CardContent>
    </Card>
  );
}

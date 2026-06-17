import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { AlertTriangle, CalendarDays, CheckCircle2, Eye, FileText, History, Plus, RefreshCw, Upload } from "lucide-react";
import { useDeferredValue, useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { AcceptMismatchDialog, ImportCsvDialog, MismatchAlert } from "@/features/import-csv/ui/import-components";
import { ConfirmDialog } from "@/shared/ui/confirm-dialog";
import { DateTimeCell } from "@/shared/ui/date-time-cell";
import { EmptyState } from "@/shared/ui/empty-state";
import { FormField } from "@/shared/ui/form-field";
import { MoneyCell } from "@/entities/order/ui/money-cell";
import { OrderDashboard } from "@/widgets/order-dashboard/ui/order-dashboard";
import { PageHeader } from "@/shared/ui/page-header";
import { PeriodFilterBar } from "@/features/period-filter/ui/period-filter-bar";
import { ReportReconciliationDetails } from "@/features/reconciliation/ui/report-reconciliation-details";
import { RequisiteCell } from "@/entities/requisite/ui/requisite-cell";
import { StatusBadge } from "@/entities/status/ui/status-badge";
import { DataTable } from "@/shared/ui/data-table";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { SearchableSelect, type SearchableSelectOption } from "@/shared/ui/searchable-select";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import type { OrderDirection, Payout, PayoutTransfer, ReconciliationItem, ReconciliationSummary, ShiftReport, ShiftRequisite } from "@/shared/model/domain";
import { api } from "@/shared/api/api";
import { filterOrdersBySearch } from "@/shared/lib/order-filters";
import type { PeriodFilter } from "@/shared/lib/period-filter";
import { usePersistentPeriodFilter } from "@/shared/lib/period-filter";
import { queryKeys } from "@/shared/api/query-keys";
import {

  formatCardNumber,
  formatDateTime,
  formatMoneyMinor,
  formatRussianPhone,
  normalizeCardNumber,
  parseMoneyToMinor,
} from "@/shared/lib/utils";
import { MetricCard } from "@/shared/ui/metric-card";

const takeSchema = z.object({
  cardNumber: z.string().min(8, "Введите номер карты"),
  holderName: z.string().min(1, "Введите держателя"),
});

const closeRequisiteSchema = z.object({
  inboundTurnover: z.string().min(1, "Введите оборот по оплатам").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  outboundTurnover: z.string().min(1, "Введите оборот по выплатам").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  closingBalance: z.string().min(1, "Введите остаток").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  blocked: z.boolean(),
  comment: z.string().optional(),
});

const payoutSchema = z.object({
  destinationBank: z.string().min(1, "Введите банк"),
  destinationRequisite: z.string().min(1, "Введите реквизит получателя"),
  amount: z.string().min(1, "Введите сумму").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
});

const transferSchema = z.object({
  sourceShiftRequisiteId: z.coerce.number().min(1, "Выберите источник"),
  amount: z.string().min(1, "Введите сумму").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
  comment: z.string().optional(),
});

const TRADER_PERIOD_FILTER_STORAGE_KEY = "astra-crm:trader-period-filter";
type TraderRequisiteTab = "current" | "future" | "history";

export function TraderReportsPage() {
  const shiftQuery = useQuery({ queryKey: queryKeys.trader.currentShift, queryFn: api.traderShift.current });
  const historyQuery = useQuery({ queryKey: queryKeys.trader.shiftHistory, queryFn: api.traderShift.history });
  const inboundReconciliationQuery = useQuery({
    queryKey: queryKeys.trader.reconciliation("inbound"),
    queryFn: () => api.orders.reconciliation("trader", "inbound"),
  });
  const outboundReconciliationQuery = useQuery({
    queryKey: queryKeys.trader.reconciliation("outbound"),
    queryFn: () => api.orders.reconciliation("trader", "outbound"),
  });

  const checklist = shiftQuery.data?.checklist;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Отчеты"
        description="Сдача смены: загрузка CSV, сверка, принятие расхождений и финальное закрытие."
        actions={<SubmitShiftReportDialog />}
      />

      <ShiftReportStatusCard
        checklist={checklist}
        isLoading={shiftQuery.isLoading}
        inboundReconciliation={inboundReconciliationQuery.data}
        outboundReconciliation={outboundReconciliationQuery.data}
      />

      <ShiftReportHistoryCard reports={historyQuery.data ?? []} isLoading={historyQuery.isLoading} />
    </div>
  );
}

function SubmitShiftReportDialog() {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [inboundFile, setInboundFile] = useState<File | null>(null);
  const [outboundFile, setOutboundFile] = useState<File | null>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [closeComment, setCloseComment] = useState("");
  const shiftQuery = useQuery({ queryKey: queryKeys.trader.currentShift, queryFn: api.traderShift.current, enabled: open });
  const requisitesQuery = useQuery({ queryKey: queryKeys.trader.requisites(), queryFn: api.traderShift.requisites, enabled: open });
  const payoutsQuery = useQuery({ queryKey: queryKeys.trader.payouts(), queryFn: api.payouts.list, enabled: open });
  const inboundReconciliationQuery = useQuery({
    queryKey: queryKeys.trader.reconciliation("inbound"),
    queryFn: () => api.orders.reconciliation("trader", "inbound"),
    enabled: open,
  });
  const outboundReconciliationQuery = useQuery({
    queryKey: queryKeys.trader.reconciliation("outbound"),
    queryFn: () => api.orders.reconciliation("trader", "outbound"),
    enabled: open,
  });
  const inboundItemsQuery = useQuery({
    queryKey: queryKeys.trader.reconciliationItems("inbound"),
    queryFn: () => api.orders.reconciliationItems("trader", "inbound"),
    enabled: open && Boolean(inboundReconciliationQuery.data?.runId),
  });
  const outboundItemsQuery = useQuery({
    queryKey: queryKeys.trader.reconciliationItems("outbound"),
    queryFn: () => api.orders.reconciliationItems("trader", "outbound"),
    enabled: open && Boolean(outboundReconciliationQuery.data?.runId),
  });
  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!inboundFile || !outboundFile) {
        throw new Error("Прикрепите CSV инвойсов и CSV выплат");
      }

      await api.imports.upload({ file: inboundFile, scope: "trader", direction: "inbound" });
      await api.imports.upload({ file: outboundFile, scope: "trader", direction: "outbound" });
    },
    onSuccess: async () => {
      setUploadError(null);
      setInboundFile(null);
      setOutboundFile(null);
      await queryClient.invalidateQueries({ queryKey: ["trader"] });
    },
    onError: (error) => setUploadError(error instanceof Error ? error.message : "Не удалось загрузить отчеты"),
  });
  const closeMutation = useMutation({
    mutationFn: () => api.traderShift.close({ closeComment: closeComment.trim() || undefined }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["trader"] });
      setOpen(false);
      setCloseComment("");
    },
  });

  const checklist = shiftQuery.data?.checklist;
  const openRequisites = (requisitesQuery.data ?? []).filter((item) => item.status === "in_work" || item.status === "correction");
  const unpaidPayouts = (payoutsQuery.data ?? []).filter((payout) => payout.status === "open");
  const canUpload = Boolean(inboundFile && outboundFile) && !uploadMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button">
          <FileText className="h-4 w-4" />
          Сдать смену
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-[1200px] p-6">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Сдать отчет и закрыть смену</DialogTitle>
          <DialogDescription>
            Загрузите CSV инвойсов и выплат, проверьте сверку и закройте смену после подтверждения.
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[76vh] space-y-5 overflow-y-auto pr-2">
          <Card className="border-blue-200 bg-blue-50">
            <CardContent className="p-3 text-sm text-blue-950">
              Если закрыть диалог после загрузки CSV, батчи останутся в истории и активном scope. Повторная загрузка CSV сбросит текущий результат сверки и пересчитает отчет заново.
            </CardContent>
          </Card>

          <section className="space-y-3">
            <h2 className="text-sm font-semibold">1. Загрузка отчетов</h2>
            <div className="grid gap-4 md:grid-cols-2">
              <FormField label="CSV инвойсов" help="Файл входящих ордеров за смену.">
                <Input type="file" accept=".csv,text/csv" onChange={(event) => setInboundFile(event.target.files?.[0] ?? null)} />
              </FormField>
              <FormField label="CSV выплат" help="Файл выплат за смену.">
                <Input type="file" accept=".csv,text/csv" onChange={(event) => setOutboundFile(event.target.files?.[0] ?? null)} />
              </FormField>
            </div>
            {uploadError ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{uploadError}</div> : null}
            <Button type="button" disabled={!canUpload} onClick={() => uploadMutation.mutate()}>
              {uploadMutation.isPending ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
              Начать сверку
            </Button>
          </section>

          <section className="space-y-3">
            <h2 className="text-sm font-semibold">2. Готовность смены</h2>
            <CloseChecklistPanel checklist={checklist} openRequisites={openRequisites} unpaidPayouts={unpaidPayouts} />
          </section>

          <section className="space-y-3">
            <h2 className="text-sm font-semibold">3. Результат сверки</h2>
            <div className="grid gap-4 lg:grid-cols-2">
              <ReconciliationReportCard
                title="Инвойсы"
                direction="inbound"
                summary={inboundReconciliationQuery.data}
                items={inboundItemsQuery.data ?? []}
              />
              <ReconciliationReportCard
                title="Выплаты"
                direction="outbound"
                summary={outboundReconciliationQuery.data}
                items={outboundItemsQuery.data ?? []}
              />
            </div>
          </section>

          <section className="space-y-3">
            <h2 className="text-sm font-semibold">4. Финал</h2>
            <Textarea
              value={closeComment}
              onChange={(event) => setCloseComment(event.target.value)}
              placeholder="Комментарий к закрытию смены, если нужен"
            />
            {closeMutation.error instanceof Error ? (
              <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{closeMutation.error.message}</div>
            ) : null}
          </section>
        </div>

        <div className="flex flex-wrap justify-end gap-2 border-t border-border pt-4">
          <CancelReportDialog onConfirm={() => setOpen(false)} />
          <ConfirmDialog
            trigger={
              <Button type="button" disabled={!checklist?.canClose || closeMutation.isPending}>
                Сдать отчет и закрыть смену
              </Button>
            }
            title="Закрыть смену?"
            description="Действие необратимо. Смена будет закрыта, а загруженные отчеты и результат сверки останутся в истории."
            confirmText="Закрыть смену"
            onConfirm={() => closeMutation.mutate()}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ShiftReportStatusCard({
  checklist,
  isLoading,
  inboundReconciliation,
  outboundReconciliation,
}: {
  checklist?: Awaited<ReturnType<typeof api.traderShift.current>>["checklist"];
  isLoading?: boolean;
  inboundReconciliation?: ReconciliationSummary | null;
  outboundReconciliation?: ReconciliationSummary | null;
}) {
  if (isLoading) return <EmptyState title="Загружаем смену" />;
  if (!checklist) {
    return (
      <Card>
        <CardContent className="p-4">
          <EmptyState title="Открытой смены нет" description="Смена начнется автоматически после взятия первого реквизита в работу." />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={checklist.canClose ? "border-emerald-200 bg-emerald-50" : undefined}>
      <CardContent className="grid gap-4 p-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-semibold">Текущая смена #{checklist.shift.id}</span>
            <StatusBadge status={checklist.shift.status} />
            {checklist.canClose ? <span className="text-sm font-medium text-emerald-800">готова к закрытию</span> : null}
          </div>
          <div className="text-sm text-muted-foreground">
            Началась {formatDateTime(checklist.shift.startedAt)}. Финальное закрытие доступно после закрытия реквизитов, импорта двух CSV и сверки.
          </div>
        </div>
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-1">
          <ChecklistLine ok={checklist.allRequisitesClosed} label="Реквизиты закрыты" detail={checklist.openRequisiteCount ? `${checklist.openRequisiteCount} открыто` : "готово"} />
          <ChecklistLine ok={checklist.inboundImported && checklist.inboundOk} label="Инвойсы сверены" detail={statusDetail(inboundReconciliation)} />
          <ChecklistLine ok={checklist.outboundImported && checklist.outboundOk} label="Выплаты сверены" detail={statusDetail(outboundReconciliation)} />
          <ChecklistLine ok={checklist.allPayoutsFullyPaid} label="Ручные выплаты оплачены" detail={checklist.unpaidPayoutCount ? `${checklist.unpaidPayoutCount} открыто` : "готово"} />
        </div>
      </CardContent>
    </Card>
  );
}

function CloseChecklistPanel({
  checklist,
  openRequisites,
  unpaidPayouts,
}: {
  checklist?: Awaited<ReturnType<typeof api.traderShift.current>>["checklist"];
  openRequisites: ShiftRequisite[];
  unpaidPayouts: Payout[];
}) {
  if (!checklist) return <EmptyState title="Нет открытой смены" />;

  return (
    <div className="grid gap-3 md:grid-cols-2">
      <ChecklistLine ok={checklist.allRequisitesClosed} label="Все реквизиты закрыты" detail={checklist.openRequisiteCount ? `${checklist.openRequisiteCount} открыто` : "готово"} />
      <ChecklistLine ok={checklist.allPayoutsFullyPaid} label="Ручные выплаты закрыты" detail={checklist.unpaidPayoutCount ? `${checklist.unpaidPayoutCount} открыто` : "готово"} />
      <ChecklistLine ok={checklist.inboundImported && checklist.inboundOk} label="CSV инвойсов загружен и сверен" detail={checklist.inboundImported ? "импорт есть" : "нужен CSV"} />
      <ChecklistLine ok={checklist.outboundImported && checklist.outboundOk} label="CSV выплат загружен и сверен" detail={checklist.outboundImported ? "импорт есть" : "нужен CSV"} />
      {openRequisites.length ? (
        <IssueList title="Открытые реквизиты" items={openRequisites.map((item) => `${formatRussianPhone(item.phone)} · ${item.bankName}`)} />
      ) : null}
      {unpaidPayouts.length ? (
        <IssueList title="Неоплаченные выплаты" items={unpaidPayouts.map((payout) => `${payout.destinationBank} · ${formatMoneyMinor(payout.amountMinor - payout.paidMinor)}`)} />
      ) : null}
    </div>
  );
}

function ReconciliationReportCard({
  title,
  direction,
  summary,
  items,
}: {
  title: string;
  direction: OrderDirection;
  summary?: ReconciliationSummary | null;
  items: ReconciliationItem[];
}) {
  if (!summary) {
    return (
      <Card>
        <CardContent className="p-4">
          <EmptyState title={`${title}: сверка не запускалась`} />
        </CardContent>
      </Card>
    );
  }

  const isMismatch = summary.status === "mismatch";

  return (
    <Card className={isMismatch ? "border-red-200 bg-red-50" : summary.status === "accepted_with_comment" ? "border-amber-200 bg-amber-50" : "border-emerald-200 bg-emerald-50"}>
      <CardContent className="space-y-4 p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="font-semibold">{title}</div>
            <div className="mt-1">
              <StatusBadge status={summary.status} />
            </div>
          </div>
          {isMismatch && summary.runId ? <AcceptMismatchDialog scope="trader" direction={direction} runId={summary.runId} /> : null}
        </div>
        <div className="grid gap-2 text-sm sm:grid-cols-3">
          <AmountBox label="CSV" value={summary.expectedMinor} />
          <AmountBox label="CRM" value={summary.actualMinor} />
          <AmountBox label="Diff" value={summary.diffMinor} />
        </div>
        {summary.comment ? <div className="rounded-md border border-border/70 p-3 text-sm">Комментарий: {summary.comment}</div> : null}
        {items.length ? (
          <div className="space-y-2">
            {items.map((item) => (
              <ReconciliationIssueItem key={item.id} item={item} />
            ))}
          </div>
        ) : isMismatch ? (
          <div className="rounded-md border border-red-200 bg-white/70 p-3 text-sm text-red-900">
            Детализация по строкам не сформирована. Проверьте финальные обороты реквизитов, ручные выплаты и суммы CSV.
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function ShiftReportHistoryCard({ reports, isLoading }: { reports: ShiftReport[]; isLoading?: boolean }) {
  const [selectedReport, setSelectedReport] = useState<ShiftReport | null>(null);

  const openReport = (report: ShiftReport) => setSelectedReport(report);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">История отчетов</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? <EmptyState title="Загружаем историю" /> : null}
        {!isLoading && !reports.length ? (
          <EmptyState title="Отчетов пока нет" description="Закрытые смены будут появляться здесь после сдачи отчета." />
        ) : null}
        {reports.length ? (
          <div className="hidden overflow-hidden rounded-md border border-border md:block">
            <table className="w-full border-collapse text-sm">
              <thead className="bg-slate-50 text-left text-xs uppercase tracking-normal text-muted-foreground">
                <tr>
                  <th className="h-10 border-b border-border px-3 font-medium">Смена</th>
                  <th className="h-10 border-b border-border px-3 font-medium">Период работы</th>
                  <th className="h-10 border-b border-border px-3 font-medium">Инвойсы</th>
                  <th className="h-10 border-b border-border px-3 font-medium">Выплаты</th>
                  <th className="h-10 border-b border-border px-3 font-medium">Статус</th>
                  <th className="h-10 border-b border-border px-3 font-medium">Комментарий</th>
                </tr>
              </thead>
              <tbody>
                {reports.map((report) => (
                  <tr
                    key={report.id}
                    className="cursor-pointer border-b border-border hover:bg-accent/50 focus:bg-accent/50 focus:outline-none last:border-0"
                    tabIndex={0}
                    role="button"
                    onClick={() => openReport(report)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        openReport(report);
                      }
                    }}
                  >
                    <td className="px-3 py-3 font-medium">
                      <span className="inline-flex items-center gap-2">
                        <Eye className="h-4 w-4 text-muted-foreground" />
                        #{report.id}
                      </span>
                    </td>
                    <td className="px-3 py-3">
                      <div>{formatDateTime(report.startedAt)}</div>
                      <div className="text-xs text-muted-foreground">закрыта {formatDateTime(report.closedAt ?? report.endedAt)}</div>
                    </td>
                    <td className="px-3 py-3">
                      <StatusBadge status={report.inboundReconciliationStatus} />
                    </td>
                    <td className="px-3 py-3">
                      <StatusBadge status={report.outboundReconciliationStatus} />
                    </td>
                    <td className="px-3 py-3">
                      <StatusBadge status={report.status} />
                    </td>
                    <td className="max-w-md px-3 py-3 text-muted-foreground">
                      <span className="block truncate" title={report.closeComment ?? ""}>
                        {report.closeComment || "—"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
        {reports.length ? (
          <div className="mt-3 space-y-2 md:hidden">
            {reports.map((report) => (
              <button
                key={report.id}
                type="button"
                className="w-full rounded-md border border-border p-3 text-left text-sm hover:bg-accent/50 focus:bg-accent/50 focus:outline-none"
                onClick={() => openReport(report)}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="inline-flex items-center gap-2 font-medium">
                    <Eye className="h-4 w-4 text-muted-foreground" />
                    Смена #{report.id}
                  </span>
                  <StatusBadge status={report.status} />
                </div>
                <div className="mt-2 text-muted-foreground">{formatDateTime(report.startedAt)} - {formatDateTime(report.closedAt ?? report.endedAt)}</div>
                <div className="mt-3 flex flex-wrap gap-2">
                  <StatusBadge status={report.inboundReconciliationStatus} />
                  <StatusBadge status={report.outboundReconciliationStatus} />
                </div>
                {report.closeComment ? <div className="mt-2 text-muted-foreground">{report.closeComment}</div> : null}
              </button>
            ))}
          </div>
        ) : null}
      </CardContent>
      <ShiftReportDetailsDialog report={selectedReport} onClose={() => setSelectedReport(null)} />
    </Card>
  );
}

function ShiftReportDetailsDialog({ report, onClose }: { report: ShiftReport | null; onClose: () => void }) {
  const requisitesQuery = useQuery({
    queryKey: report ? queryKeys.trader.shiftReportRequisites(report.id) : ["trader", "shift", "report", "requisites", "empty"],
    queryFn: () => api.traderShift.reportRequisites(report?.id ?? 0),
    enabled: Boolean(report),
  });
  const inboundReconciliationQuery = useQuery({
    queryKey: report ? queryKeys.trader.shiftReportReconciliation(report.id, "inbound") : ["trader", "shift", "report", "inbound", "empty"],
    queryFn: () => api.traderShift.reportReconciliation(report?.id ?? 0, "inbound"),
    enabled: Boolean(report),
  });
  const outboundReconciliationQuery = useQuery({
    queryKey: report ? queryKeys.trader.shiftReportReconciliation(report.id, "outbound") : ["trader", "shift", "report", "outbound", "empty"],
    queryFn: () => api.traderShift.reportReconciliation(report?.id ?? 0, "outbound"),
    enabled: Boolean(report),
  });
  const inboundItemsQuery = useQuery({
    queryKey: report ? queryKeys.trader.shiftReportReconciliationItems(report.id, "inbound") : ["trader", "shift", "report", "inbound", "items", "empty"],
    queryFn: () => api.traderShift.reportReconciliationItems(report?.id ?? 0, "inbound"),
    enabled: Boolean(report),
  });
  const outboundItemsQuery = useQuery({
    queryKey: report ? queryKeys.trader.shiftReportReconciliationItems(report.id, "outbound") : ["trader", "shift", "report", "outbound", "items", "empty"],
    queryFn: () => api.traderShift.reportReconciliationItems(report?.id ?? 0, "outbound"),
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
          <DialogTitle className="text-base font-semibold">Реквизиты в отчете {report ? `#${report.id}` : ""}</DialogTitle>
          <DialogDescription>
            {report
              ? `Смена ${formatDateTime(report.startedAt)} - ${formatDateTime(report.closedAt ?? report.endedAt)}`
              : "Реквизиты, которые были в работе в выбранной смене."}
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[72vh] space-y-4 overflow-y-auto pr-2">
          {report ? (
            <div className="grid gap-3 sm:grid-cols-3">
              <MetricCard layout="header" label="Оплаты" value={formatMoneyMinor(inboundTotal)} />
              <MetricCard layout="header" label="Выплаты" value={formatMoneyMinor(outboundTotal)} />
              <MetricCard layout="header" label="Остаток" value={formatMoneyMinor(balanceTotal)} />
            </div>
          ) : null}

          {requisitesQuery.isLoading ? <EmptyState title="Загружаем реквизиты" /> : null}
          {requisitesQuery.error instanceof Error ? (
            <EmptyState title="Не удалось загрузить реквизиты" description={requisitesQuery.error.message} />
          ) : null}
          {!requisitesQuery.isLoading && !requisitesQuery.error && !requisites.length ? (
            <EmptyState title="Реквизитов в отчете нет" description="В этой смене не найдено взятых в работу реквизитов." />
          ) : null}

          {requisites.length ? (
            <div className="hidden overflow-hidden rounded-md border border-border md:block">
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
          ) : null}

          {requisites.length ? (
            <div className="space-y-2 md:hidden">
              {requisites.map((item) => (
                <div key={item.id} className="rounded-md border border-border p-3 text-sm">
                  <div className="flex items-start justify-between gap-3">
                    <RequisiteCell phone={item.phone} method={item.bankName} proxy={item.proxy} />
                    <StatusBadge status={item.status} />
                  </div>
                  <div className="mt-3 grid gap-2 text-xs text-muted-foreground">
                    <div>Карта: <span className="font-mono text-foreground">{formatCardNumber(item.cardNumber)}</span></div>
                    <div>Держатель: <span className="text-foreground">{item.holderName || "—"}</span></div>
                    <div className="grid grid-cols-3 gap-2 pt-1 text-foreground">
                      <div>
                        <div className="text-muted-foreground">Оплаты</div>
                        <MoneyCell valueMinor={item.inboundTurnoverMinor} />
                      </div>
                      <div>
                        <div className="text-muted-foreground">Выплаты</div>
                        <MoneyCell valueMinor={item.outboundTurnoverMinor} />
                      </div>
                      <div>
                        <div className="text-muted-foreground">Остаток</div>
                        <MoneyCell valueMinor={item.closingBalanceMinor} />
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : null}

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

function ChecklistLine({ ok, label, detail }: { ok: boolean; label: string; detail: string }) {
  return (
    <div className={ok ? "rounded-md border border-emerald-200 bg-emerald-50 p-3" : "rounded-md border border-amber-200 bg-amber-50 p-3"}>
      <div className="flex items-start gap-2">
        {ok ? <CheckCircle2 className="mt-0.5 h-4 w-4 text-emerald-700" /> : <AlertTriangle className="mt-0.5 h-4 w-4 text-amber-700" />}
        <div className="min-w-0">
          <div className="text-sm font-medium">{label}</div>
          <div className="text-xs text-muted-foreground">{detail}</div>
        </div>
      </div>
    </div>
  );
}

function IssueList({ title, items }: { title: string; items: string[] }) {
  return (
    <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-950 md:col-span-2">
      <div className="font-medium">{title}</div>
      <ul className="mt-2 list-inside list-disc space-y-1">
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
  );
}

function AmountBox({ label, value }: { label: string; value: number }) {
  return (
    <div className="min-w-0 rounded-md border border-border/70 bg-white/70 p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 break-words text-sm font-semibold tabular-nums">
        {formatMoneyMinor(value)}
      </div>
    </div>
  );
}

function ReconciliationIssueItem({ item }: { item: ReconciliationItem }) {
  return (
    <div className="rounded-md border border-border/70 bg-white/80 p-3 text-sm">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="font-medium">{issueTypeLabel(item.issueType)}</span>
        {item.externalInnerId ? <span className="text-xs text-muted-foreground">innerId: {item.externalInnerId}</span> : null}
      </div>
      {item.message ? <div className="mt-1 text-muted-foreground">{item.message}</div> : null}
      <div className="mt-2 grid gap-2 md:grid-cols-2">
        {item.teamleadValue ? <JsonValueBox label="CSV" value={item.teamleadValue} /> : null}
        {item.traderValue ? <JsonValueBox label="CRM" value={item.traderValue} /> : null}
      </div>
    </div>
  );
}

function JsonValueBox({ label, value }: { label: string; value: Record<string, unknown> }) {
  return (
    <div className="rounded-md bg-slate-50 p-2">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="mt-1 break-words text-xs">{JSON.stringify(value)}</div>
    </div>
  );
}

function CancelReportDialog({ onConfirm }: { onConfirm: () => void }) {
  return (
    <ConfirmDialog
      trigger={
        <Button type="button" variant="outline">
          Отмена
        </Button>
      }
      title="Закрыть сдачу отчета?"
      description="Если CSV уже загружены, они останутся в истории и активном scope. Незавершенный результат можно продолжить или пересчитать повторной загрузкой."
      confirmText="Закрыть"
      onConfirm={onConfirm}
    />
  );
}

function statusDetail(summary?: ReconciliationSummary | null) {
  if (!summary) return "не запускалась";
  if (summary.status === "matched") return "сошлось";
  if (summary.status === "accepted_with_comment") return "принято с комментарием";
  return `расхождение ${formatMoneyMinor(summary.diffMinor)}`;
}

function issueTypeLabel(issueType: string) {
  const labels: Record<string, string> = {
    payout_not_fully_paid: "Ручная выплата оплачена не полностью",
    missing_manual_payout_order: "Не найдена ручная выплата",
    manual_payout_not_fully_paid: "Ручная выплата оплачена не полностью",
    total_mismatch: "Итоговая сумма не сходится",
    order_mismatch: "Ордер не сходится",
  };

  return labels[issueType] ?? issueType;
}

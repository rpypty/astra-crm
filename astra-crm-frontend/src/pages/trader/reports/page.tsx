import { zodResolver } from "@hookform/resolvers/zod";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { AlertTriangle, ArrowDownLeft, ArrowUpRight, CalendarDays, CheckCircle2, Eye, FileText, History, Plus, RefreshCw, Upload } from "lucide-react";
import { useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Input } from "@/shared/ui/input";
import { SearchableSelect, type SearchableSelectOption } from "@/shared/ui/searchable-select";
import { Select } from "@/shared/ui/select";
import { LoadingSkeleton } from "@/shared/ui/loading-skeleton";
import { Textarea } from "@/shared/ui/textarea";
import type { OrderDirection, OrderImportHistoryItem, Payout, PayoutTransfer, ReconciliationItem, ReconciliationSummary, ShiftReport, ShiftReportReconciliation, ShiftReportRow, ShiftRequisite } from "@/shared/model/domain";
import { api } from "@/shared/api/api";
import { filterOrdersBySearch } from "@/shared/lib/order-filters";
import type { PeriodFilter } from "@/shared/lib/period-filter";
import { usePersistentPeriodFilter } from "@/shared/lib/period-filter";
import { queryKeys } from "@/shared/api/query-keys";
import { paginationToQuery } from "@/shared/lib/pagination";
import {

  cn,
  formatCardNumber,
  formatDateTime,
  formatMoneyMinor,
  formatRussianPhone,
  normalizeCardNumber,
  normalizeRussianPhone,
  phoneDigits,
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
type TraderShiftChecklist = NonNullable<Awaited<ReturnType<typeof api.traderShift.current>>["checklist"]>;
type TraderShiftSummary = TraderShiftChecklist["shift"];

export function TraderReportsPage() {
  const shiftQuery = useQuery({ queryKey: queryKeys.trader.currentShift, queryFn: api.traderShift.current });
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

      <ShiftReportHistoryCard />
    </div>
  );
}

function toDraftShiftReport(shift: TraderShiftSummary): ShiftReport {
  return {
    id: shift.id,
    traderId: shift.traderId,
    startedAt: shift.startedAt,
    endedAt: shift.endedAt,
    closedAt: shift.closedAt,
    status: "draft",
    inboundReconciliationStatus: shift.inboundReconciliationStatus ?? "unknown",
    outboundReconciliationStatus: shift.outboundReconciliationStatus ?? "unknown",
    tlReconciliationStatus: "not_checked",
    closeComment: shift.closeComment,
  };
}

function SubmitShiftReportDialog() {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [inboundFile, setInboundFile] = useState<File | null>(null);
  const [outboundFile, setOutboundFile] = useState<File | null>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [closeComment, setCloseComment] = useState("");
  const shiftQuery = useQuery({ queryKey: queryKeys.trader.currentShift, queryFn: api.traderShift.current, enabled: open });
  const requisitesQuery = useQuery({
    queryKey: queryKeys.trader.requisites({ reportDialog: true, statuses: ["in_work", "correction"], page: 1, pageSize: 200 }),
    queryFn: () => api.traderShift.requisites({ statuses: ["in_work", "correction"], page: 1, pageSize: 200 }),
    enabled: open,
  });
  const payoutsQuery = useQuery({
    queryKey: queryKeys.trader.payouts({ reportDialog: true, status: "open", page: 1, pageSize: 200 }),
    queryFn: () => api.payouts.list({ status: "open", page: 1, pageSize: 200 }),
    enabled: open,
  });
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
  const checklist = shiftQuery.data?.checklist;
  const currentShiftId = checklist?.shift.id;
  const inboundDashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard("inbound", { reportFilesForShift: currentShiftId }),
    queryFn: () => api.orders.dashboard("trader", "inbound"),
    enabled: open && Boolean(currentShiftId),
  });
  const outboundDashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard("outbound", { reportFilesForShift: currentShiftId }),
    queryFn: () => api.orders.dashboard("trader", "outbound"),
    enabled: open && Boolean(currentShiftId),
  });
  const reportDetailsQuery = useQuery({
    queryKey: currentShiftId ? queryKeys.trader.shiftReport(currentShiftId) : ["trader", "shift", "current", "report", "empty"],
    queryFn: () => api.traderShift.report(currentShiftId ?? 0),
    enabled: open && Boolean(currentShiftId) && Boolean(inboundReconciliationQuery.data || outboundReconciliationQuery.data),
  });
  const latestInboundImport = latestSavedImport(inboundDashboardQuery.data?.recentImports);
  const latestOutboundImport = latestSavedImport(outboundDashboardQuery.data?.recentImports);
  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!inboundFile && !outboundFile) {
        throw new Error("Выберите новый CSV инвойсов или выплат для перезагрузки");
      }

      if (!inboundFile && !latestInboundImport) {
        throw new Error("Прикрепите CSV инвойсов");
      }
      if (!outboundFile && !latestOutboundImport) {
        throw new Error("Прикрепите CSV выплат");
      }

      if (inboundFile) {
        await api.imports.upload({ file: inboundFile, scope: "trader", direction: "inbound" });
      }
      if (outboundFile) {
        await api.imports.upload({ file: outboundFile, scope: "trader", direction: "outbound" });
      }
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

  const openRequisites = requisitesQuery.data?.items ?? [];
  const unpaidPayouts = payoutsQuery.data?.items ?? [];
  const hasFilesToUpload = Boolean(inboundFile || outboundFile);
  const hasRequiredReports = Boolean(inboundFile || latestInboundImport) && Boolean(outboundFile || latestOutboundImport);
  const canUpload = hasFilesToUpload && hasRequiredReports && !uploadMutation.isPending;
  const reportRows = reportDetailsQuery.data?.rows ?? [];
  const inboundMismatchRows = reportRows.filter((item) => item.inboundDiffMinor !== 0);
  const outboundMismatchRows = reportRows.filter((item) => item.outboundDiffMinor !== 0);

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
              <ReportFileDropzone
                label="CSV инвойсов"
                help="Файл входящих ордеров за смену."
                selectedFile={inboundFile}
                savedImport={latestInboundImport}
                isLoading={inboundDashboardQuery.isLoading}
                onFileChange={setInboundFile}
              />
              <ReportFileDropzone
                label="CSV выплат"
                help="Файл выплат за смену."
                selectedFile={outboundFile}
                savedImport={latestOutboundImport}
                isLoading={outboundDashboardQuery.isLoading}
                onFileChange={setOutboundFile}
              />
            </div>
            {uploadError ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{uploadError}</div> : null}
            <Button type="button" disabled={!canUpload} onClick={() => uploadMutation.mutate()}>
              {uploadMutation.isPending ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
              {hasFilesToUpload ? "Перезагрузить и пересчитать" : "Выберите CSV для сверки"}
            </Button>
            {!hasRequiredReports ? (
              <div className="text-sm text-muted-foreground">
                Для закрытия смены нужны обе выписки. Уже загруженный файл сохраняется, можно заменить только изменившуюся выписку.
              </div>
            ) : null}
          </section>

          <section className="space-y-3">
            <h2 className="text-sm font-semibold">2. Готовность смены</h2>
            <CloseChecklistPanel checklist={checklist} openRequisites={openRequisites} unpaidPayouts={unpaidPayouts} />
          </section>

          <section className="space-y-3">
            <h2 className="text-sm font-semibold">3. Результат сверки</h2>
            <div className="space-y-4">
              <ReconciliationReportCard
                title="Инвойсы"
                direction="inbound"
                summary={inboundReconciliationQuery.data}
                mismatchRows={inboundMismatchRows}
                isRowsLoading={reportDetailsQuery.isFetching && !reportDetailsQuery.data}
              />
              <ReconciliationReportCard
                title="Выплаты"
                direction="outbound"
                summary={outboundReconciliationQuery.data}
                mismatchRows={outboundMismatchRows}
                isRowsLoading={reportDetailsQuery.isFetching && !reportDetailsQuery.data}
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

function ReportFileDropzone({
  label,
  help,
  selectedFile,
  savedImport,
  isLoading,
  onFileChange,
}: {
  label: string;
  help: string;
  selectedFile: File | null;
  savedImport?: OrderImportHistoryItem;
  isLoading?: boolean;
  onFileChange: (file: File | null) => void;
}) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const fileName = selectedFile?.name ?? savedImport?.fileName;
  const statusText = selectedFile
    ? "Будет загружен и пересчитан после запуска сверки"
    : savedImport
      ? `Загружен ${formatDateTime(savedImport.appliedAt ?? savedImport.createdAt)} · строк: ${savedImport.rowsCount}`
      : "Файл еще не загружен";

  const handleFile = (file?: File) => {
    if (!file) return;
    onFileChange(file);
  };

  return (
    <div className="space-y-2">
      <div>
        <div className="text-sm font-medium">{label}</div>
        <div className="text-xs text-muted-foreground">{help}</div>
      </div>
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        onDragOver={(event) => {
          event.preventDefault();
          setIsDragging(true);
        }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={(event) => {
          event.preventDefault();
          setIsDragging(false);
          handleFile(event.dataTransfer.files?.[0]);
        }}
        className={cn(
          "flex min-h-[126px] w-full items-center justify-between gap-4 rounded-md border border-dashed border-border bg-white p-4 text-left transition hover:border-primary hover:bg-primary/5",
          isDragging ? "border-primary bg-primary/10" : undefined,
        )}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".csv,text/csv"
          className="sr-only"
          onChange={(event) => handleFile(event.target.files?.[0])}
        />
        <span className="min-w-0 space-y-1">
          <span className="flex items-center gap-2 text-sm font-semibold">
            <FileText className="h-4 w-4 text-primary" />
            {isLoading ? "Проверяем сохраненный файл" : fileName || "Перетащите CSV сюда"}
          </span>
          <span className="block text-xs text-muted-foreground">
            {fileName ? statusText : "Можно выбрать файл кнопкой или перетащить его в эту область."}
          </span>
        </span>
        <span className="inline-flex shrink-0 items-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground">
          <Upload className="h-4 w-4" />
          {savedImport || selectedFile ? "Заменить" : "Выбрать"}
        </span>
      </button>
    </div>
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
  mismatchRows,
  isRowsLoading,
}: {
  title: string;
  direction: OrderDirection;
  summary?: ReconciliationSummary | null;
  mismatchRows: ShiftReportRow[];
  isRowsLoading?: boolean;
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
    <Card className={isMismatch ? "border-red-200" : summary.status === "accepted_with_comment" ? "border-amber-200" : "border-emerald-200"}>
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
          <AmountBox label={direction === "inbound" ? "CSV инвойсы" : "CSV выплаты"} value={summary.expectedMinor} />
          <AmountBox label={direction === "inbound" ? "CRM входы" : "CRM выходы"} value={summary.actualMinor} />
          <AmountBox label="Расхождение" value={summary.diffMinor} />
        </div>
        {summary.comment ? <div className="rounded-md border border-border/70 p-3 text-sm">Комментарий: {summary.comment}</div> : null}
        {isMismatch || summary.status === "accepted_with_comment" ? (
          <div className="space-y-3">
            <div className="text-sm font-medium">Реквизиты с расхождением</div>
            <TraderReportRowsTable
              rows={mismatchRows}
              isLoading={isRowsLoading}
              emptyTitle="Расхождений по реквизитам нет"
              emptyDescription="Итоговая сверка расходится, но в таблице реквизитов нет строк по этому направлению."
              pageSize={8}
              resetKey={`${direction}-${summary.runId ?? summary.status}`}
            />
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function ShiftReportHistoryCard() {
  const [selectedReport, setSelectedReport] = useState<ShiftReport | null>(null);
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const historyQuery = useQuery({
    queryKey: queryKeys.trader.shiftHistory(paginationToQuery(pagination)),
    queryFn: () => api.traderShift.history(paginationToQuery(pagination)),
    placeholderData: keepPreviousData,
  });
  const reports = historyQuery.data?.items ?? [];

  const openReport = (report: ShiftReport) => setSelectedReport(report);
  const columns = useMemo<ColumnDef<ShiftReport>[]>(
    () => [
      {
        accessorKey: "id",
        header: "Смена",
        cell: ({ row }) => (
          <span className="inline-flex items-center gap-2 font-medium">
            <Eye className="h-4 w-4 text-muted-foreground" />
            #{row.original.id}
          </span>
        ),
      },
      {
        accessorKey: "startedAt",
        header: "Период работы",
        cell: ({ row }) => (
          <div>
            <DateTimeCell value={row.original.startedAt} />
            <div className="text-xs text-muted-foreground">
              {row.original.status === "draft"
                ? "черновик: смена еще не закрыта"
                : `закрыта ${formatDateTime(row.original.closedAt ?? row.original.endedAt)}`}
            </div>
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
      {
        accessorKey: "tlReconciliationStatus",
        header: "TL",
        cell: ({ row }) => <StatusBadge status={row.original.tlReconciliationStatus} />,
      },
      { accessorKey: "status", header: "Статус", cell: ({ row }) => <StatusBadge status={row.original.status} /> },
      {
        accessorKey: "closeComment",
        header: "Комментарий",
        cell: ({ row }) => (
          <span className="block max-w-md truncate text-muted-foreground" title={row.original.closeComment ?? ""}>
            {row.original.closeComment || "—"}
          </span>
        ),
      },
    ],
    [],
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">История отчетов</CardTitle>
      </CardHeader>
      <CardContent>
        <DataTable
          columns={columns}
          data={reports}
          rowCount={historyQuery.data?.total ?? 0}
          pagination={pagination}
          onPaginationChange={setPagination}
          serverSidePagination
          isLoading={historyQuery.isLoading}
          isFetching={historyQuery.isFetching}
          emptyTitle="Отчетов пока нет"
          emptyDescription="Закрытые смены будут появляться здесь после сдачи отчета."
          onRowClick={openReport}
          actions={[{ label: "Детали", onSelect: openReport }]}
        />
      </CardContent>
      <ShiftReportDetailsDialog report={selectedReport} onClose={() => setSelectedReport(null)} />
    </Card>
  );
}

function TraderReportRowsTable({
  rows,
  isLoading,
  emptyTitle,
  emptyDescription,
  pageSize = 15,
  resetKey,
}: {
  rows: ShiftReportRow[];
  isLoading?: boolean;
  emptyTitle: string;
  emptyDescription: string;
  pageSize?: number;
  resetKey?: string | number;
}) {
  const columns = useTraderReportRowsColumns();
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize });

  useEffect(() => {
    setPagination({ pageIndex: 0, pageSize });
  }, [pageSize, resetKey]);

  return (
    <DataTable
      columns={columns}
      data={rows}
      rowCount={rows.length}
      pagination={pagination}
      onPaginationChange={setPagination}
      isLoading={isLoading}
      emptyTitle={emptyTitle}
      emptyDescription={emptyDescription}
      pageSizeOptions={[8, 15, 25, 50, 100]}
      getRowClassName={(item) => (item.hasMismatch ? "bg-red-50 text-red-950 hover:bg-red-100" : undefined)}
    />
  );
}

function useTraderReportRowsColumns() {
  return useMemo<ColumnDef<ShiftReportRow>[]>(
    () => [
      {
        accessorKey: "phone",
        header: "Реквизит",
        cell: ({ row }) => <TraderReportRequisiteCell item={row.original} />,
      },
      {
        accessorKey: "bankName",
        header: "Банк",
        cell: ({ row }) => row.original.bankName || "—",
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        accessorKey: "tlReconciliationStatus",
        header: "TL",
        cell: ({ row }) => <StatusBadge status={row.original.tlReconciliationStatus} />,
      },
      {
        accessorKey: "inboundTurnoverMinor",
        header: "Оборот по CRM",
        cell: ({ row }) => (
          <div className="text-right">
            <TraderReportCrmTurnoverCell item={row.original} />
          </div>
        ),
      },
      {
        accessorKey: "csvInboundMinor",
        header: "CSV / переводы",
        cell: ({ row }) => (
          <div className="text-right">
            <TraderReportCsvTurnoverCell item={row.original} />
          </div>
        ),
      },
      {
        accessorKey: "inboundDiffMinor",
        header: "Расхождение",
        cell: ({ row }) => (
          <div className="text-right">
            <TraderReportDiffCell item={row.original} />
          </div>
        ),
      },
    ],
    [],
  );
}

function ShiftReportDetailsDialog({ report, onClose }: { report: ShiftReport | null; onClose: () => void }) {
  const reportQuery = useQuery({
    queryKey: report ? queryKeys.trader.shiftReport(report.id) : ["trader", "shift", "report", "empty"],
    queryFn: () => api.traderShift.report(report?.id ?? 0),
    enabled: Boolean(report),
  });
  const details = reportQuery.data;
  const rows = details?.rows ?? [];
  const shift = details?.shift ?? report;
  const isInitialLoading = reportQuery.isLoading && !details;

  return (
    <Dialog open={Boolean(report)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-[1240px] p-6">
        <DialogHeader>
          <DialogTitle className="flex flex-wrap items-center gap-2 text-base font-semibold">
            <span>Реквизиты в отчете {report ? `#${report.id}` : ""}</span>
            {!isInitialLoading ? (
              <>
                <TraderReportStatusBadge label="Инвойсы" summary={details?.inbound} />
                <TraderReportStatusBadge label="Выплаты" summary={details?.outbound} />
              </>
            ) : null}
          </DialogTitle>
          <DialogDescription>
            {shift
              ? `Смена ${formatDateTime(shift.startedAt)} - ${formatDateTime(shift.closedAt ?? shift.endedAt)}`
              : "Реквизиты, которые были в работе в выбранной смене."}
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[72vh] space-y-4 overflow-y-auto pr-2">
          {isInitialLoading ? <ReportDetailsLoadingState /> : null}
          {reportQuery.error instanceof Error ? (
            <EmptyState title="Не удалось загрузить отчет" description={reportQuery.error.message} />
          ) : null}

          {!isInitialLoading && !reportQuery.error ? (
            <>
              <div className="grid gap-3 lg:grid-cols-2">
                <ReportReconciliationDetails
                  title="Инвойсы"
                  summary={details?.inbound}
                  isLoading={reportQuery.isFetching}
                  csvLabel="CSV инвойсы"
                  crmLabel="CRM входы"
                  diffLabel="Расхождение"
                />
                <ReportReconciliationDetails
                  title="Выплаты"
                  summary={details?.outbound}
                  isLoading={reportQuery.isFetching}
                  csvLabel="CSV выплаты"
                  crmLabel="CRM выходы"
                  diffLabel="Расхождение"
                />
              </div>

              <TraderReportRowsTable
                rows={rows}
                isLoading={reportQuery.isFetching && !details}
                emptyTitle="Реквизитов в отчете нет"
                emptyDescription="В этой смене не найдено взятых в работу реквизитов."
                pageSize={15}
                resetKey={report?.id}
              />
            </>
          ) : null}

        </div>
      </DialogContent>
    </Dialog>
  );
}

function ReportDetailsLoadingState() {
  return (
    <Card>
      <CardContent className="space-y-4 p-4">
        <div className="flex items-center gap-3 text-sm text-muted-foreground">
          <RefreshCw className="h-4 w-4 animate-spin text-primary" />
          <span>Загружаем тяжелый отчет, данные появятся здесь после ответа сервера</span>
        </div>
        <LoadingSkeleton rows={8} />
      </CardContent>
    </Card>
  );
}

function TraderReportStatusBadge({ label, summary }: { label: string; summary?: ShiftReportReconciliation }) {
  return (
    <span title={traderReconciliationStatusTitle(label, summary)} className="inline-flex items-center gap-1">
      <span className="text-xs font-normal text-muted-foreground">{label}</span>
      <StatusBadge status={summary?.status ?? "unknown"} />
    </span>
  );
}

function traderReconciliationStatusTitle(label: string, summary?: ShiftReportReconciliation) {
  if (!summary) return `${label}: сверка не запускалась`;
  if (summary.status === "matched") return `${label}: CSV и CRM сходятся`;
  if (summary.status === "accepted_with_comment") {
    return `${label}: расхождение подтверждено${summary.comment ? `. Комментарий: ${summary.comment}` : ""}`;
  }
  return `${label}: есть расхождение ${formatMoneyMinor(summary.diffMinor)}`;
}

function TraderReportRequisiteCell({ item }: { item: ShiftReportRow }) {
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
        <TraderCopyDropdownItem label="Телефон" value={formatRussianPhone(item.phone)} copyValue={canCopyPhone ? copyPhone : item.phone} />
        <TraderCopyDropdownItem label="Карта" value={formatCardNumber(item.cardNumber)} copyValue={item.cardNumber} />
        <TraderCopyDropdownItem label="ФИО" value={item.holderName || "—"} copyValue={item.holderName} />
        {item.proxy ? <TraderCopyDropdownItem label="Proxy" value={item.proxy} copyValue={item.proxy} /> : null}
        {item.csvOnly ? <DropdownMenuItem className="text-red-700">Есть в CSV, но нет в смене</DropdownMenuItem> : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function TraderReportCrmTurnoverCell({ item }: { item: ShiftReportRow }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" className="ml-auto block text-right tabular-nums hover:text-primary">
          <TraderReportAmountStack
            inboundValue={item.inboundTurnoverMinor}
            outboundValue={item.outboundTurnoverMinor}
          />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuLabel>Оборот по CRM</DropdownMenuLabel>
        <TraderAmountDropdownItem label="Входы" value={item.inboundTurnoverMinor} />
        <TraderAmountDropdownItem label="Выходы" value={item.outboundTurnoverMinor} />
        <TraderAmountDropdownItem label="Остаток" value={item.closingBalanceMinor} />
        <TraderAmountDropdownItem label="Лимит" value={item.targetTurnoverMinor} />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function TraderReportCsvTurnoverCell({ item }: { item: ShiftReportRow }) {
  return (
    <TraderReportAmountStack
      inboundValue={item.csvInboundMinor}
      outboundValue={item.csvOutboundMinor}
    />
  );
}

function TraderReportDiffCell({ item }: { item: ShiftReportRow }) {
  const hasInboundDiff = item.inboundDiffMinor !== 0;
  const hasOutboundDiff = item.outboundDiffMinor !== 0;
  if (!hasInboundDiff && !hasOutboundDiff) return <span className="text-muted-foreground">—</span>;

  return (
    <div className="flex min-h-[44px] flex-col justify-center gap-1 tabular-nums">
      {hasInboundDiff ? <TraderReportAmountLine type="inbound" value={item.inboundDiffMinor} tone="danger" /> : null}
      {hasOutboundDiff ? <TraderReportAmountLine type="outbound" value={item.outboundDiffMinor} tone="danger" /> : null}
    </div>
  );
}

function TraderReportAmountStack({ inboundValue, outboundValue }: { inboundValue: number; outboundValue: number }) {
  return (
    <div className="flex min-h-[44px] flex-col justify-center gap-1 tabular-nums">
      <TraderReportAmountLine type="inbound" value={inboundValue} />
      <TraderReportAmountLine type="outbound" value={outboundValue} />
    </div>
  );
}

function TraderReportAmountLine({
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

function TraderCopyDropdownItem({ label, value, copyValue }: { label: string; value: string; copyValue?: string }) {
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

function TraderAmountDropdownItem({ label, value }: { label: string; value: number }) {
  return (
    <DropdownMenuItem className="justify-between gap-3">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium tabular-nums">{formatMoneyMinor(value)}</span>
    </DropdownMenuItem>
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

function latestSavedImport(items?: OrderImportHistoryItem[]) {
  return items?.find((item) => item.status !== "failed");
}

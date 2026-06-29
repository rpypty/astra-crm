import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { AlertTriangle, ArrowDownLeft, ArrowUpRight, CalendarDays, CheckCircle2, ChevronDown, Eye, FileText, History, Plus, RefreshCw, Trash2, Upload } from "lucide-react";
import { useCallback, useDeferredValue, useEffect, useMemo, useState } from "react";
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
import { CopyToast, RequisitePhoneMenu } from "@/entities/requisite/ui/requisite-phone-menu";
import { StatusBadge } from "@/entities/status/ui/status-badge";
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
import type { InternalTransfer, OrderDirection, Payout, PayoutTransfer, ReconciliationItem, ReconciliationSummary, ShiftReport, ShiftRequisite } from "@/shared/model/domain";
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
import { ReadOnlyField } from "@/shared/ui/read-only-field";
import { CurrentRequisitesTab, FutureRequisitesTab, HistoricalRequisitesTab } from "./tabs";

const takeSchema = z.object({
  cardNumber: z.string().min(8, "Введите номер карты"),
  holderName: z.string().min(1, "Введите держателя"),
});

const closeRequisiteSchema = z.object({
  inboundTurnover: z.string().min(1, "Введите оборот по оплатам").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  outboundTurnover: z.string().min(1, "Введите оборот по выплатам").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  closingBalance: z.string().min(1, "Введите остаток").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  releasedAt: z.string().min(1, "Выберите дату закрытия").refine((value) => value <= todayDateInputValue(), "Дата закрытия не может быть в будущем"),
  blocked: z.boolean(),
  comment: z.string().optional(),
});

const correctRequisiteSchema = z.object({
  inboundTurnover: z.string().min(1, "Введите оборот по оплатам").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  outboundTurnover: z.string().min(1, "Введите оборот по выплатам").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  closingBalance: z.string().min(1, "Введите остаток").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
  comment: z.string().trim().min(1, "Укажите причину корректировки"),
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

const internalTransferSchema = z.object({
  direction: z.enum(["incoming", "outgoing"]),
  counterpartyShiftRequisiteId: z.string().min(1, "Выберите реквизит"),
  amount: z.string().min(1, "Введите сумму").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
  comment: z.string().optional(),
});

const TRADER_PERIOD_FILTER_STORAGE_KEY = "astra-crm:trader-period-filter";
type TraderRequisiteTab = "current" | "future" | "history";

export function TraderRequisitesPage() {
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<TraderRequisiteTab>("current");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [futurePagination, setFuturePagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [historyPagination, setHistoryPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [selectedRequisite, setSelectedRequisite] = useState<ShiftRequisite | null>(null);
  const [copiedMessage, setCopiedMessage] = useState<string | null>(null);
  const requisitesQuery = useQuery({ queryKey: queryKeys.trader.requisites(), queryFn: api.traderShift.requisites });
  const futureRequisitesQuery = useQuery({
    queryKey: queryKeys.trader.futureRequisites,
    queryFn: api.traderShift.futureRequisites,
    enabled: activeTab === "future",
  });
  const historicalRequisitesQuery = useQuery({
    queryKey: queryKeys.trader.historicalRequisites,
    queryFn: api.traderShift.historicalRequisites,
    enabled: activeTab === "history",
  });
  const copyToClipboard = useCallback((value: string | undefined | null, label: string) => {
    if (!value) return;
    void navigator.clipboard?.writeText(value);
    setCopiedMessage(`${label} скопирован`);
  }, []);

  useEffect(() => {
    if (!copiedMessage) return;
    const timeout = window.setTimeout(() => setCopiedMessage(null), 1800);
    return () => window.clearTimeout(timeout);
  }, [copiedMessage]);

  useEffect(() => {
    void queryClient.invalidateQueries({ queryKey: ["trader", "requisites"] });
  }, [activeTab, queryClient]);

  const currentColumns = useMemo<ColumnDef<ShiftRequisite>[]>(
    () => [
      {
        accessorKey: "phone",
        header: "Реквизит",
        cell: ({ row }) => <RequisitePhoneMenu item={row.original} onCopy={copyToClipboard} />,
      },
      { accessorKey: "bankName", header: "Банк" },
      {
        accessorKey: "employeeComment",
        header: () => <div className="w-24">Комментарий</div>,
        cell: ({ row }) => {
          const comment = row.original.employeeComment;

          return comment ? (
            <span className="block w-24 truncate text-sm text-muted-foreground" title={comment}>
              {comment}
            </span>
          ) : (
            <span className="block w-24 text-muted-foreground">—</span>
          );
        },
      },
      {
        accessorKey: "targetTurnoverMinor",
        header: () => <div className="text-right">Лимит</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.targetTurnoverMinor} />,
      },
      {
        id: "progress",
        header: "Прогресс",
        cell: ({ row }) => shiftRequisiteProgressLabel(row.original),
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        id: "work",
        header: "",
        cell: ({ row }) => <ShiftRequisiteActions item={row.original} />,
      },
    ],
    [copyToClipboard],
  );
  const futureColumns = useMemo<ColumnDef<ShiftRequisite>[]>(
    () => [
      {
        accessorKey: "assignedForDate",
        header: "Дата",
        cell: ({ row }) => formatDateOnly(row.original.assignedForDate),
      },
      {
        accessorKey: "phone",
        header: "Реквизит",
        cell: ({ row }) => <RequisitePhoneMenu item={row.original} onCopy={copyToClipboard} />,
      },
      {
        accessorKey: "employeeComment",
        header: "Комментарий",
        cell: ({ row }) => (
          <span className="block max-w-[260px] truncate text-sm text-muted-foreground" title={row.original.employeeComment ?? ""}>
            {row.original.employeeComment || "—"}
          </span>
        ),
      },
      {
        accessorKey: "targetTurnoverMinor",
        header: () => <div className="text-right">Лимит</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.targetTurnoverMinor} />,
      },
      {
        accessorKey: "assignmentStatus",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.assignmentStatus} />,
      },
    ],
    [copyToClipboard],
  );
  const historyColumns = useMemo<ColumnDef<ShiftRequisite>[]>(
    () => [
      {
        accessorKey: "releasedAt",
        header: "Дата",
        cell: ({ row }) => formatDateOnly(row.original.releasedAt ?? row.original.assignedForDate),
      },
      {
        accessorKey: "phone",
        header: "Реквизит",
        cell: ({ row }) => <RequisitePhoneMenu item={row.original} onCopy={copyToClipboard} />,
      },
      {
        accessorKey: "inboundTurnoverMinor",
        header: () => <div className="text-right">Оплаты</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.inboundTurnoverMinor} />,
      },
      {
        accessorKey: "targetTurnoverMinor",
        header: () => <div className="text-right">Лимит</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.targetTurnoverMinor} />,
      },
      {
        accessorKey: "outboundTurnoverMinor",
        header: () => <div className="text-right">Выплаты</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.outboundTurnoverMinor} />,
      },
      {
        accessorKey: "closingBalanceMinor",
        header: () => <div className="text-right">Остаток</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.closingBalanceMinor} />,
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        id: "work",
        header: "",
        cell: ({ row }) => <ShiftRequisiteActions item={row.original} />,
      },
    ],
    [copyToClipboard],
  );

  return (
    <div className="space-y-6">
      <PageHeader title="Мои реквизиты" description="Текущая работа, будущие назначения и история закрытых или заблокированных реквизитов." />
      <TraderRequisiteTabs value={activeTab} onChange={setActiveTab} />
      {activeTab === "current" ? (
        <CurrentRequisitesTab
          columns={currentColumns}
          data={requisitesQuery.data ?? []}
          pagination={pagination}
          onPaginationChange={setPagination}
          isLoading={requisitesQuery.isLoading}
          error={requisitesQuery.error instanceof Error ? requisitesQuery.error.message : null}
          onRowClick={setSelectedRequisite}
        />
      ) : null}
      {activeTab === "future" ? (
        <FutureRequisitesTab
          columns={futureColumns}
          data={futureRequisitesQuery.data ?? []}
          pagination={futurePagination}
          onPaginationChange={setFuturePagination}
          isLoading={futureRequisitesQuery.isLoading}
          error={futureRequisitesQuery.error instanceof Error ? futureRequisitesQuery.error.message : null}
        />
      ) : null}
      {activeTab === "history" ? (
        <HistoricalRequisitesTab
          columns={historyColumns}
          data={historicalRequisitesQuery.data ?? []}
          pagination={historyPagination}
          onPaginationChange={setHistoryPagination}
          isLoading={historicalRequisitesQuery.isLoading}
          error={historicalRequisitesQuery.error instanceof Error ? historicalRequisitesQuery.error.message : null}
        />
      ) : null}
      <ShiftRequisiteInteractionDialog item={selectedRequisite} onClose={() => setSelectedRequisite(null)} />
      <CopyToast message={copiedMessage} />
    </div>
  );
}

function TraderRequisiteTabs({
  value,
  onChange,
}: {
  value: TraderRequisiteTab;
  onChange: (value: TraderRequisiteTab) => void;
}) {
  const tabs: { value: TraderRequisiteTab; label: string; icon: typeof FileText }[] = [
    { value: "current", label: "Текущие", icon: FileText },
    { value: "future", label: "Будущие", icon: CalendarDays },
    { value: "history", label: "История", icon: History },
  ];

  return (
    <div className="inline-flex rounded-lg border border-border bg-white p-1">
      {tabs.map((tab) => {
        const Icon = tab.icon;
        return (
          <Button
            key={tab.value}
            type="button"
            variant={value === tab.value ? "default" : "ghost"}
            size="sm"
            onClick={() => onChange(tab.value)}
          >
            <Icon className="h-4 w-4" />
            {tab.label}
          </Button>
        );
      })}
    </div>
  );
}

function shiftRequisiteProgressLabel(row: ShiftRequisite) {
  if (row.targetTurnoverMinor <= 0) return "—";
  const fact = row.inboundTurnoverMinor || row.latestTurnoverMinor;
  return `${Math.round((fact / row.targetTurnoverMinor) * 100)}%`;
}

function formatDateOnly(value: string | null | undefined) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("ru-RU", { day: "2-digit", month: "2-digit", year: "numeric" }).format(date);
}

function moneyInputValue(valueMinor: number) {
  return (valueMinor / 100).toFixed(2).replace(".", ",");
}

function todayDateInputValue() {
  const date = new Date();
  date.setMinutes(date.getMinutes() - date.getTimezoneOffset());
  return date.toISOString().slice(0, 10);
}

function dateInputToCurrentTimeISO(value: string) {
  const [year, month, day] = value.split("-").map(Number);
  const now = new Date();
  return new Date(year, month - 1, day, now.getHours(), now.getMinutes(), now.getSeconds()).toISOString();
}

function canCorrectTurnovers(item: ShiftRequisite) {
  return item.status === "worked_pending_review" || item.status === "worked_discrepancy";
}

function canReturnToWork(item: ShiftRequisite) {
  return item.status === "worked_pending_review" || item.status === "worked_discrepancy" || item.status === "blocked" || item.status === "correction";
}

function ShiftRequisiteActions({ item }: { item: ShiftRequisite }) {
  return (
    <div className="flex justify-end gap-2">
      {item.status === "assigned" ? <TakeRequisiteDialog item={item} /> : null}
      {item.status === "in_work" || item.status === "correction" ? <EditDetailsDialog item={item} /> : null}
      {item.status === "in_work" || item.status === "correction" ? <CloseRequisiteDialog item={item} /> : null}
      {canCorrectTurnovers(item) ? <CorrectRequisiteDialog item={item} /> : null}
      {canReturnToWork(item) ? <ReturnRequisiteToWorkButton item={item} /> : null}
    </div>
  );
}

function ShiftRequisiteInteractionDialog({ item, onClose }: { item: ShiftRequisite | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(item)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{item ? formatRussianPhone(item.phone) : "Реквизит"}</DialogTitle>
          <DialogDescription>
            {item?.bankName}
            {item?.proxy ? ` · ${item.proxy}` : ""}
          </DialogDescription>
        </DialogHeader>
        {item ? (
          <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <ReadOnlyField label="Карта" value={formatCardNumber(item.cardNumber)} />
              <ReadOnlyField label="Держатель" value={item.holderName ?? "—"} />
              <ReadOnlyField label="Банк" value={item.bankName} />
              <ReadOnlyField label="Комментарий" value={item.employeeComment ?? "—"} />
              <ReadOnlyField label="Оплаты" value={formatMoneyMinor(item.inboundTurnoverMinor || item.latestTurnoverMinor)} />
              <ReadOnlyField label="Выплаты" value={formatMoneyMinor(item.outboundTurnoverMinor)} />
              <ReadOnlyField label="Остаток" value={formatMoneyMinor(item.closingBalanceMinor)} />
              <div className="rounded-md border border-border p-3">
                <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Статус</div>
                <StatusBadge status={item.status} />
              </div>
            </div>
            <ShiftRequisiteActions item={item} />
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function TakeRequisiteDialog({ item }: { item: ShiftRequisite }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const form = useForm<z.infer<typeof takeSchema>>({
    resolver: zodResolver(takeSchema),
    values: { cardNumber: item.cardNumber ? formatCardNumber(item.cardNumber) : "", holderName: item.holderName ?? "" },
  });
  const mutation = useMutation({
    mutationFn: api.traderShift.takeRequisite,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["trader"] });
      setOpen(false);
    },
  });
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" size="sm">
          В работу
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Взять реквизит в работу</DialogTitle>
          <DialogDescription>
            {item.bankName}
            {item.employeeComment ? ` · ${item.employeeComment}` : ""}
          </DialogDescription>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) =>
            mutation.mutate({ shiftRequisiteId: item.id, cardNumber: normalizeCardNumber(values.cardNumber), holderName: values.holderName }),
          )}
        >
          <FormField label="Номер карты" error={form.formState.errors.cardNumber?.message}>
            <Input
              {...form.register("cardNumber")}
              onBlur={(event) => form.setValue("cardNumber", formatCardNumber(event.target.value), { shouldValidate: true })}
            />
          </FormField>
          <FormField label="Держатель" error={form.formState.errors.holderName?.message}>
            <Input {...form.register("holderName")} />
          </FormField>
          <Button type="submit" disabled={mutation.isPending}>
            Сохранить
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function EditDetailsDialog({ item }: { item: ShiftRequisite }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const form = useForm<z.infer<typeof takeSchema>>({
    resolver: zodResolver(takeSchema),
    values: { cardNumber: item.cardNumber ? formatCardNumber(item.cardNumber) : "", holderName: item.holderName ?? "" },
  });
  const mutation = useMutation({
    mutationFn: api.traderShift.updateDetails,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["trader"] });
      setOpen(false);
    },
  });
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          Детали
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Daily details</DialogTitle>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) =>
            mutation.mutate({ shiftRequisiteId: item.id, cardNumber: normalizeCardNumber(values.cardNumber), holderName: values.holderName }),
          )}
        >
          <FormField label="Номер карты" error={form.formState.errors.cardNumber?.message}>
            <Input
              {...form.register("cardNumber")}
              onBlur={(event) => form.setValue("cardNumber", formatCardNumber(event.target.value), { shouldValidate: true })}
            />
          </FormField>
          <FormField label="Держатель" error={form.formState.errors.holderName?.message}>
            <Input {...form.register("holderName")} />
          </FormField>
          <Button type="submit" disabled={mutation.isPending}>
            Сохранить
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function CloseRequisiteDialog({ item }: { item: ShiftRequisite }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [transfersCollapsed, setTransfersCollapsed] = useState(false);
  const form = useForm<z.infer<typeof closeRequisiteSchema>>({
    resolver: zodResolver(closeRequisiteSchema),
    defaultValues: { inboundTurnover: "", outboundTurnover: "", closingBalance: "", releasedAt: todayDateInputValue(), blocked: false, comment: "" },
  });
  const transferForm = useForm<z.infer<typeof internalTransferSchema>>({
    resolver: zodResolver(internalTransferSchema),
    defaultValues: { direction: "outgoing", counterpartyShiftRequisiteId: "", amount: "", comment: "" },
  });
  const currentRequisitesQuery = useQuery({
    queryKey: queryKeys.trader.requisites(),
    queryFn: api.traderShift.requisites,
    enabled: open,
  });
  const internalTransfersQuery = useQuery({
    queryKey: queryKeys.trader.internalTransfers(item.id),
    queryFn: () => api.traderShift.internalTransfers(item.id),
    enabled: open && item.id > 0,
  });
  const transferOptions = useMemo<SearchableSelectOption[]>(
    () =>
      (currentRequisitesQuery.data ?? [])
        .filter((requisite) => requisite.id !== item.id && (requisite.status === "in_work" || requisite.status === "correction"))
        .map((requisite) => ({
          value: String(requisite.id),
          label: `${formatRussianPhone(requisite.phone)} · ${requisite.bankName}`,
          searchText: `${requisite.phone} ${requisite.bankName} ${requisite.cardNumber ?? ""}`,
        })),
    [currentRequisitesQuery.data, item.id],
  );
  const internalTransfers = internalTransfersQuery.data ?? [];
  const incomingTotalMinor = internalTransfers
    .filter((transfer) => transfer.destinationShiftRequisiteId === item.id)
    .reduce((sum, transfer) => sum + transfer.amountMinor, 0);
  const outgoingTotalMinor = internalTransfers
    .filter((transfer) => transfer.sourceShiftRequisiteId === item.id)
    .reduce((sum, transfer) => sum + transfer.amountMinor, 0);
  const mutation = useMutation({
    mutationFn: api.traderShift.closeRequisite,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["trader"] });
      setOpen(false);
      form.reset();
      transferForm.reset();
    },
  });
  const createTransferMutation = useMutation({
    mutationFn: api.traderShift.createInternalTransfer,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.trader.internalTransfers(item.id) });
      transferForm.reset({ direction: transferForm.getValues("direction"), counterpartyShiftRequisiteId: "", amount: "", comment: "" });
    },
  });
  const cancelTransferMutation = useMutation({
    mutationFn: api.traderShift.cancelInternalTransfer,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.trader.internalTransfers(item.id) });
    },
  });
  const addInternalTransfer = transferForm.handleSubmit((values) => {
    const counterpartyShiftRequisiteId = Number(values.counterpartyShiftRequisiteId);
    createTransferMutation.mutate({
      sourceShiftRequisiteId: values.direction === "outgoing" ? item.id : counterpartyShiftRequisiteId,
      destinationShiftRequisiteId: values.direction === "outgoing" ? counterpartyShiftRequisiteId : item.id,
      amountMinor: parseMoneyToMinor(values.amount),
      comment: values.comment?.trim() || undefined,
    });
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          Закрыть
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[calc(100vh-32px)] max-w-[560px] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Закрыть реквизит</DialogTitle>
          <DialogDescription>Укажите финальный оборот на момент завершения работы по реквизиту.</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <FormField label="Оборот по оплатам" error={form.formState.errors.inboundTurnover?.message}>
            <Input {...form.register("inboundTurnover")} />
          </FormField>
          <FormField label="Оборот по выплатам" error={form.formState.errors.outboundTurnover?.message}>
            <Input {...form.register("outboundTurnover")} />
          </FormField>
          <FormField label="Остаток" error={form.formState.errors.closingBalance?.message}>
            <Input {...form.register("closingBalance")} />
          </FormField>
          <div className="space-y-3 rounded-md border border-border p-3">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="text-sm font-medium">Внутренние переливы</div>
                <div className="mt-1 grid grid-cols-2 gap-3 text-xs text-muted-foreground">
                  <span>Приход: {formatMoneyMinor(incomingTotalMinor)}</span>
                  <span>Отправка: {formatMoneyMinor(outgoingTotalMinor)}</span>
                </div>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-expanded={!transfersCollapsed}
                title={transfersCollapsed ? "Развернуть внутренние переливы" : "Свернуть внутренние переливы"}
                onClick={() => setTransfersCollapsed((value) => !value)}
              >
                <ChevronDown className={`h-4 w-4 transition-transform ${transfersCollapsed ? "-rotate-90" : ""}`} />
              </Button>
            </div>

            {!transfersCollapsed ? (
              <div className="space-y-3">
                {internalTransfers.length > 0 ? (
                  <div className="space-y-2">
                    {internalTransfers.map((transfer) => (
                      <InternalTransferLine
                        key={transfer.id}
                        transfer={transfer}
                        currentShiftRequisiteId={item.id}
                        onCancel={() => cancelTransferMutation.mutate(transfer.id)}
                        isCancelling={cancelTransferMutation.isPending}
                      />
                    ))}
                  </div>
                ) : (
                  <div className="rounded-md bg-muted/50 px-3 py-2 text-sm text-muted-foreground">Переливов по реквизиту нет</div>
                )}

                <div className="grid gap-3 sm:grid-cols-[140px_1fr]">
                  <FormField label="Тип" error={transferForm.formState.errors.direction?.message}>
                    <Select {...transferForm.register("direction")}>
                      <option value="outgoing">Отправка</option>
                      <option value="incoming">Приход</option>
                    </Select>
                  </FormField>
                  <FormField label="Реквизит" error={transferForm.formState.errors.counterpartyShiftRequisiteId?.message}>
                    <SearchableSelect
                      value={transferForm.watch("counterpartyShiftRequisiteId")}
                      options={transferOptions}
                      onValueChange={(value) => transferForm.setValue("counterpartyShiftRequisiteId", value, { shouldValidate: true })}
                      placeholder="Выберите реквизит"
                      searchPlaceholder="Найти реквизит"
                      emptyText="Нет доступных реквизитов"
                    />
                  </FormField>
                </div>
                <div className="grid gap-3 sm:grid-cols-[1fr_auto]">
                  <FormField label="Сумма" error={transferForm.formState.errors.amount?.message}>
                    <Input {...transferForm.register("amount")} />
                  </FormField>
                  <div className="flex items-end">
                    <Button type="button" variant="outline" onClick={() => void addInternalTransfer()} disabled={createTransferMutation.isPending}>
                      <Plus className="h-4 w-4" />
                      Добавить
                    </Button>
                  </div>
                </div>
                <FormField label="Комментарий">
                  <Textarea {...transferForm.register("comment")} />
                </FormField>
              </div>
            ) : null}
          </div>
          <FormField label="Дата закрытия" error={form.formState.errors.releasedAt?.message}>
            <Input type="date" max={todayDateInputValue()} {...form.register("releasedAt")} />
          </FormField>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" className="h-4 w-4 rounded border-border" {...form.register("blocked")} />
            Карта заблокирована
          </label>
          <FormField label="Комментарий">
            <Textarea {...form.register("comment")} />
          </FormField>
          <Button
            type="button"
            disabled={mutation.isPending}
            onClick={form.handleSubmit((values) =>
              mutation.mutate({
                shiftRequisiteId: item.id,
                inboundTurnoverMinor: parseMoneyToMinor(values.inboundTurnover),
                outboundTurnoverMinor: parseMoneyToMinor(values.outboundTurnover),
                closingBalanceMinor: parseMoneyToMinor(values.closingBalance),
                releasedAt: dateInputToCurrentTimeISO(values.releasedAt),
                blocked: values.blocked,
                comment: values.comment,
              }),
            )}
          >
            Закрыть реквизит
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function InternalTransferLine({
  transfer,
  currentShiftRequisiteId,
  onCancel,
  isCancelling,
}: {
  transfer: InternalTransfer;
  currentShiftRequisiteId: number;
  onCancel: () => void;
  isCancelling: boolean;
}) {
  const incoming = transfer.destinationShiftRequisiteId === currentShiftRequisiteId;
  const counterpartyPhone = incoming ? transfer.sourcePhone : transfer.destinationPhone;
  const counterpartyBank = incoming ? transfer.sourceBankName : transfer.destinationBankName;

  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-border bg-background px-3 py-2">
      <div className="flex min-w-0 items-center gap-2">
        {incoming ? (
          <ArrowDownLeft className="h-4 w-4 shrink-0 text-emerald-600" />
        ) : (
          <ArrowUpRight className="h-4 w-4 shrink-0 text-amber-600" />
        )}
        <div className="min-w-0">
          <div className="truncate text-sm font-medium">
            {incoming ? "Приход с" : "Отправка на"} {formatRussianPhone(counterpartyPhone)}
          </div>
          <div className="truncate text-xs text-muted-foreground">{counterpartyBank}</div>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <span className="text-sm font-medium">{formatMoneyMinor(transfer.amountMinor)}</span>
        <Button type="button" variant="ghost" size="icon" onClick={onCancel} disabled={isCancelling} title="Отменить перелив">
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

function CorrectRequisiteDialog({ item }: { item: ShiftRequisite }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const form = useForm<z.infer<typeof correctRequisiteSchema>>({
    resolver: zodResolver(correctRequisiteSchema),
    defaultValues: {
      inboundTurnover: moneyInputValue(item.inboundTurnoverMinor || item.latestTurnoverMinor),
      outboundTurnover: moneyInputValue(item.outboundTurnoverMinor),
      closingBalance: moneyInputValue(item.closingBalanceMinor),
      comment: "",
    },
  });
  const mutation = useMutation({
    mutationFn: api.traderShift.correctRequisite,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["trader"] });
      setOpen(false);
      form.reset();
    },
  });
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          Коррекция
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Корректировка оборотов</DialogTitle>
          <DialogDescription>Исправьте финальные значения по реквизиту и укажите причину корректировки.</DialogDescription>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) =>
            mutation.mutate({
              shiftRequisiteId: item.id,
              inboundTurnoverMinor: parseMoneyToMinor(values.inboundTurnover),
              outboundTurnoverMinor: parseMoneyToMinor(values.outboundTurnover),
              closingBalanceMinor: parseMoneyToMinor(values.closingBalance),
              comment: values.comment.trim(),
            }),
          )}
        >
          <FormField label="Оборот по оплатам" error={form.formState.errors.inboundTurnover?.message}>
            <Input {...form.register("inboundTurnover")} />
          </FormField>
          <FormField label="Оборот по выплатам" error={form.formState.errors.outboundTurnover?.message}>
            <Input {...form.register("outboundTurnover")} />
          </FormField>
          <FormField label="Остаток" error={form.formState.errors.closingBalance?.message}>
            <Input {...form.register("closingBalance")} />
          </FormField>
          <FormField label="Комментарий" error={form.formState.errors.comment?.message}>
            <Textarea {...form.register("comment")} />
          </FormField>
          <Button type="submit" disabled={mutation.isPending}>
            Сохранить корректировку
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function ReturnRequisiteToWorkButton({ item }: { item: ShiftRequisite }) {
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: api.traderShift.returnRequisiteToWork,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["trader"] });
    },
  });

  return (
    <ConfirmDialog
      trigger={
        <Button type="button" variant="outline" size="sm" disabled={mutation.isPending}>
          В работу
        </Button>
      }
      title="Вернуть реквизит в работу"
      description="Реквизит снова станет активным в текущей смене. Его можно будет закрыть заново после правок."
      confirmText="Вернуть"
      onConfirm={() => mutation.mutate(item.id)}
    />
  );
}

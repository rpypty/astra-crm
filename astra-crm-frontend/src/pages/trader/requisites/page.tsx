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

export function TraderRequisitesPage() {
  const [activeTab, setActiveTab] = useState<TraderRequisiteTab>("current");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [futurePagination, setFuturePagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [historyPagination, setHistoryPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [selectedRequisite, setSelectedRequisite] = useState<ShiftRequisite | null>(null);
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

  const currentColumns = useMemo<ColumnDef<ShiftRequisite>[]>(
    () => [
      {
        accessorKey: "phone",
        header: "Реквизит",
        cell: ({ row }) => (
          <RequisiteCell phone={row.original.phone} method={row.original.bankName} proxy={row.original.proxy} />
        ),
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
      { accessorKey: "cardNumber", header: "Карта", cell: ({ row }) => formatCardNumber(row.original.cardNumber) },
      { accessorKey: "holderName", header: "Держатель", cell: ({ row }) => row.original.holderName ?? "—" },
      {
        accessorKey: "inboundTurnoverMinor",
        header: () => <div className="text-right">Оплаты</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.inboundTurnoverMinor || row.original.latestTurnoverMinor} />,
      },
      {
        accessorKey: "targetTurnoverMinor",
        header: () => <div className="text-right">Цель</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.targetTurnoverMinor} />,
      },
      {
        id: "progress",
        header: "Прогресс",
        cell: ({ row }) => shiftRequisiteProgressLabel(row.original),
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
    [],
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
        cell: ({ row }) => (
          <RequisiteCell phone={row.original.phone} method={row.original.bankName} proxy={row.original.proxy} />
        ),
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
        header: () => <div className="text-right">Целевой оборот</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.targetTurnoverMinor} />,
      },
      {
        accessorKey: "assignmentStatus",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.assignmentStatus} />,
      },
    ],
    [],
  );
  const historyColumns = useMemo<ColumnDef<ShiftRequisite>[]>(
    () => [
      {
        accessorKey: "assignedForDate",
        header: "Дата",
        cell: ({ row }) => formatDateOnly(row.original.assignedForDate),
      },
      {
        accessorKey: "phone",
        header: "Реквизит",
        cell: ({ row }) => (
          <RequisiteCell phone={row.original.phone} method={row.original.bankName} proxy={row.original.proxy} />
        ),
      },
      { accessorKey: "cardNumber", header: "Карта", cell: ({ row }) => formatCardNumber(row.original.cardNumber) },
      { accessorKey: "holderName", header: "Держатель", cell: ({ row }) => row.original.holderName ?? "—" },
      {
        accessorKey: "inboundTurnoverMinor",
        header: () => <div className="text-right">Оборот</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.inboundTurnoverMinor} />,
      },
      {
        accessorKey: "targetTurnoverMinor",
        header: () => <div className="text-right">Цель</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.targetTurnoverMinor} />,
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
    ],
    [],
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

function ShiftRequisiteActions({ item }: { item: ShiftRequisite }) {
  return (
    <div className="flex justify-end gap-2">
      {item.status === "assigned" ? <TakeRequisiteDialog item={item} /> : null}
      {item.status === "in_work" || item.status === "correction" ? <EditDetailsDialog item={item} /> : null}
      {item.status === "in_work" || item.status === "correction" ? <CloseRequisiteDialog item={item} /> : null}
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
  const form = useForm<z.infer<typeof closeRequisiteSchema>>({
    resolver: zodResolver(closeRequisiteSchema),
    defaultValues: { inboundTurnover: "", outboundTurnover: "", closingBalance: "", blocked: false, comment: "" },
  });
  const mutation = useMutation({
    mutationFn: api.traderShift.closeRequisite,
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
          Закрыть
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Закрыть реквизит</DialogTitle>
          <DialogDescription>Укажите финальный оборот на момент завершения работы по реквизиту.</DialogDescription>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) =>
            mutation.mutate({
              shiftRequisiteId: item.id,
              inboundTurnoverMinor: parseMoneyToMinor(values.inboundTurnover),
              outboundTurnoverMinor: parseMoneyToMinor(values.outboundTurnover),
              closingBalanceMinor: parseMoneyToMinor(values.closingBalance),
              blocked: values.blocked,
              comment: values.comment,
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
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" className="h-4 w-4 rounded border-border" {...form.register("blocked")} />
            Карта заблокирована
          </label>
          <FormField label="Комментарий">
            <Textarea {...form.register("comment")} />
          </FormField>
          <Button type="submit" disabled={mutation.isPending}>
            Закрыть реквизит
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}

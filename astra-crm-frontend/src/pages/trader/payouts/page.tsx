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

export function TraderPayoutsPage() {
  const queryClient = useQueryClient();
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsPayout, setDetailsPayout] = useState<Payout | null>(null);
  const payoutsQuery = useQuery({ queryKey: queryKeys.trader.payouts(), queryFn: api.payouts.list });
  const createMutation = useMutation({
    mutationFn: api.payouts.create,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["trader", "payouts"] }),
  });
  const data = payoutsQuery.data ?? [];
  const total = data.reduce((sum, payout) => sum + payout.amountMinor, 0);
  const paid = data.reduce((sum, payout) => sum + payout.paidMinor, 0);
  const unpaidCount = data.filter((payout) => payout.status === "open").length;
  const columns = useMemo<ColumnDef<Payout>[]>(
    () => [
      { accessorKey: "createdAt", header: "Создана", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
      { accessorKey: "destinationBank", header: "Банк" },
      { accessorKey: "destinationRequisite", header: "Получатель" },
      { accessorKey: "amountMinor", header: () => <div className="text-right">Сумма</div>, cell: ({ row }) => <MoneyCell valueMinor={row.original.amountMinor} /> },
      { accessorKey: "paidMinor", header: () => <div className="text-right">Оплачено</div>, cell: ({ row }) => <MoneyCell valueMinor={row.original.paidMinor} /> },
      {
        id: "remaining",
        header: () => <div className="text-right">Остаток</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.amountMinor - row.original.paidMinor} />,
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
      <PageHeader
        title="Ручные выплаты"
        description="Ручные выплаты и промежуточные переводы."
        actions={<CreatePayoutDialog onSubmit={(values) => createMutation.mutate(values)} />}
      />
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard layout="header" label="Всего выплат" value={formatMoneyMinor(total)} />
        <MetricCard layout="header" label="Оплачено" value={formatMoneyMinor(paid)} />
        <MetricCard layout="header" label="Блокеры закрытия" value={String(unpaidCount)} warning={unpaidCount > 0} />
      </div>
      {unpaidCount > 0 ? (
        <Card className="border-amber-200 bg-amber-50">
          <CardContent className="p-4 text-sm text-amber-900">
            Смена не может быть закрыта, пока есть не полностью оплаченные ручные выплаты.
          </CardContent>
        </Card>
      ) : null}
      <DataTable
        columns={columns}
        data={data}
        rowCount={data.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        isLoading={payoutsQuery.isLoading}
        onRowClick={setDetailsPayout}
        actions={[{ label: "Детали", onSelect: (row) => setDetailsPayout(row) }]}
      />
      <PayoutDetailsDialog payout={detailsPayout} onClose={() => setDetailsPayout(null)} />
    </div>
  );
}

function CreatePayoutDialog({ onSubmit }: { onSubmit: (values: { destinationBank: string; destinationRequisite: string; amountMinor: number }) => void }) {
  const [open, setOpen] = useState(false);
  const form = useForm<z.infer<typeof payoutSchema>>({
    resolver: zodResolver(payoutSchema),
    defaultValues: { destinationBank: "", destinationRequisite: "", amount: "" },
  });
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button">
          <Plus className="h-4 w-4" />
          Добавить выплату
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Ручная выплата</DialogTitle>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) => {
            onSubmit({
              destinationBank: values.destinationBank,
              destinationRequisite: values.destinationRequisite,
              amountMinor: parseMoneyToMinor(values.amount),
            });
            setOpen(false);
            form.reset();
          })}
        >
          <FormField label="Банк" error={form.formState.errors.destinationBank?.message}>
            <Input {...form.register("destinationBank")} />
          </FormField>
          <FormField label="Реквизит получателя" error={form.formState.errors.destinationRequisite?.message}>
            <Input {...form.register("destinationRequisite")} />
          </FormField>
          <FormField label="Сумма" error={form.formState.errors.amount?.message}>
            <Input {...form.register("amount")} />
          </FormField>
          <Button type="submit">Создать</Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function PayoutDetailsDialog({ payout, onClose }: { payout: Payout | null; onClose: () => void }) {
  const queryClient = useQueryClient();
  const transfersQuery = useQuery({
    queryKey: ["trader", "payouts", payout?.id, "transfers"],
    queryFn: () => api.payouts.transfers(payout?.id ?? 0),
    enabled: Boolean(payout),
  });
  const requisitesQuery = useQuery({ queryKey: queryKeys.trader.requisites(), queryFn: api.traderShift.requisites });
  const addTransferMutation = useMutation({
    mutationFn: api.payouts.addTransfer,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["trader", "payouts"] }),
  });
  const deleteTransferMutation = useMutation({
    mutationFn: api.payouts.deleteTransfer,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["trader", "payouts"] }),
  });
  const remaining = payout ? payout.amountMinor - payout.paidMinor : 0;
  const sourceByShiftRequisiteId = new Map(
    (requisitesQuery.data ?? []).map((item) => [item.id, `${formatRussianPhone(item.phone)} · ${item.bankName}`]),
  );

  return (
    <Dialog open={Boolean(payout)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="left-auto right-0 top-0 h-screen w-[min(620px,100vw)] translate-x-0 translate-y-0 rounded-none">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Детали выплаты</DialogTitle>
          <DialogDescription>Остаток: {formatMoneyMinor(remaining)}</DialogDescription>
        </DialogHeader>
        {payout ? (
          <div className="space-y-5">
            <AddTransferForm
              payout={payout}
              shiftRequisites={requisitesQuery.data ?? []}
              onSubmit={(values) => addTransferMutation.mutate(values)}
            />
            <div className="space-y-2">
              <div className="text-sm font-semibold">Переводы</div>
              {(transfersQuery.data ?? []).map((transfer) => (
                <TransferRow
                  key={transfer.id}
                  transfer={transfer}
                  sourceLabel={sourceByShiftRequisiteId.get(transfer.sourceShiftRequisiteId) ?? `Реквизит #${transfer.sourceRequisiteId}`}
                  onDelete={() => deleteTransferMutation.mutate({ payoutId: payout.id, transferId: transfer.id })}
                />
              ))}
              {!transfersQuery.data?.length ? <EmptyState title="Переводов пока нет" /> : null}
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function AddTransferForm({
  payout,
  shiftRequisites,
  onSubmit,
}: {
  payout: Payout;
  shiftRequisites: ShiftRequisite[];
  onSubmit: (values: { payoutId: number; sourceShiftRequisiteId: number; amountMinor: number; comment?: string }) => void;
}) {
  const remaining = payout.amountMinor - payout.paidMinor;
  const schema = transferSchema.refine((values) => parseMoneyToMinor(values.amount) <= remaining, {
    path: ["amount"],
    message: "Сумма перевода не может быть больше остатка",
  });
  const form = useForm<z.infer<typeof transferSchema>>({
    resolver: zodResolver(schema),
    defaultValues: { sourceShiftRequisiteId: shiftRequisites[0]?.id ?? 0, amount: "", comment: "" },
  });
  const sourceOptions = useMemo<SearchableSelectOption[]>(
    () =>
      shiftRequisites.map((item) => ({
        value: String(item.id),
        label: `${formatRussianPhone(item.phone)} · ${item.bankName}`,
        searchText: `${item.proxy} ${item.holderName ?? ""} ${item.cardNumber ?? ""}`,
      })),
    [shiftRequisites],
  );
  return (
    <form
      className="space-y-4 rounded-lg border border-border p-4"
      onSubmit={form.handleSubmit((values) => {
        onSubmit({
          payoutId: payout.id,
          sourceShiftRequisiteId: values.sourceShiftRequisiteId,
          amountMinor: parseMoneyToMinor(values.amount),
          comment: values.comment,
        });
        form.reset();
      })}
    >
      <FormField label="Источник" error={form.formState.errors.sourceShiftRequisiteId?.message}>
        <SearchableSelect
          value={String(form.watch("sourceShiftRequisiteId") || "")}
          options={sourceOptions}
          onValueChange={(value) =>
            form.setValue("sourceShiftRequisiteId", Number(value), { shouldDirty: true, shouldValidate: true })
          }
          placeholder="Выберите источник"
          searchPlaceholder="Найти реквизит"
        />
      </FormField>
      <FormField label="Сумма перевода" error={form.formState.errors.amount?.message} help={`Остаток: ${formatMoneyMinor(remaining)}`}>
        <Input {...form.register("amount")} />
      </FormField>
      <FormField label="Комментарий">
        <Textarea {...form.register("comment")} />
      </FormField>
      <Button type="submit" disabled={remaining <= 0}>
        Добавить перевод
      </Button>
    </form>
  );
}

function TransferRow({ transfer, sourceLabel, onDelete }: { transfer: PayoutTransfer; sourceLabel: string; onDelete: () => void }) {
  return (
    <Card>
      <CardContent className="flex items-center justify-between gap-3 p-3">
        <div>
          <DateTimeCell value={transfer.createdAt} />
          <div className="text-sm font-medium">{sourceLabel}</div>
          {transfer.comment ? <div className="text-sm text-muted-foreground">{transfer.comment}</div> : null}
        </div>
        <div className="flex items-center gap-3">
          <MoneyCell valueMinor={transfer.amountMinor} />
          <ConfirmDialog
            trigger={
              <Button type="button" variant="outline" size="sm">
                Удалить
              </Button>
            }
            title="Удалить перевод?"
            description="Сумма выплаты будет пересчитана. Действие попадет в аудит."
            confirmText="Удалить"
            destructive
            onConfirm={onDelete}
          />
        </div>
      </CardContent>
    </Card>
  );
}


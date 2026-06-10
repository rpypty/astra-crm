import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { History, Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { AcceptMismatchDialog, ImportCsvDialog, MismatchAlert } from "@/components/crm/import-components";
import { ConfirmDialog } from "@/components/crm/confirm-dialog";
import { DateTimeCell } from "@/components/crm/date-time-cell";
import { EmptyState } from "@/components/crm/empty-state";
import { FormField } from "@/components/crm/form-field";
import { MoneyCell } from "@/components/crm/money-cell";
import { OrderDashboard } from "@/components/crm/order-dashboard";
import { PageHeader } from "@/components/crm/page-header";
import { RequisiteCell } from "@/components/crm/requisite-cell";
import { StatusBadge } from "@/components/crm/status-badge";
import { DataTable } from "@/components/table/data-table";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { Payout, PayoutTransfer, ShiftRequisite, TurnoverEntry } from "@/lib/domain";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { formatMoneyMinor, parseMoneyToMinor } from "@/lib/utils";

const takeSchema = z.object({
  cardNumber: z.string().min(8, "Введите номер карты"),
  holderName: z.string().min(1, "Введите держателя"),
});

const turnoverSchema = z.object({
  amount: z.string().min(1, "Введите сумму").refine((value) => parseMoneyToMinor(value) >= 0, "Некорректная сумма"),
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

export function TraderRequisitesPage() {
  const queryClient = useQueryClient();
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [lastClosedStatus, setLastClosedStatus] = useState<"closed" | "closed_with_discrepancy" | null>(null);
  const shiftQuery = useQuery({ queryKey: queryKeys.trader.currentShift, queryFn: api.traderShift.current });
  const requisitesQuery = useQuery({ queryKey: queryKeys.trader.requisites(), queryFn: api.traderShift.requisites });
  const closeMutation = useMutation({
    mutationFn: api.traderShift.close,
    onSuccess: async (shift) => {
      if (shift.status === "closed" || shift.status === "closed_with_discrepancy") {
        setLastClosedStatus(shift.status);
      }
      await queryClient.invalidateQueries({ queryKey: ["trader"] });
    },
  });

  const columns = useMemo<ColumnDef<ShiftRequisite>[]>(
    () => [
      {
        accessorKey: "phone",
        header: "Реквизит",
        cell: ({ row }) => (
          <RequisiteCell phone={row.original.phone} method={row.original.methodType} proxy={row.original.proxy} />
        ),
      },
      { accessorKey: "cardNumber", header: "Карта", cell: ({ row }) => row.original.cardNumber ?? "—" },
      { accessorKey: "holderName", header: "Держатель", cell: ({ row }) => row.original.holderName ?? "—" },
      {
        accessorKey: "latestTurnoverMinor",
        header: () => <div className="text-right">Оборот</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.latestTurnoverMinor} />,
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

  const checklist = shiftQuery.data?.checklist;
  const blockers = checklist
    ? [
        !checklist.inboundImported && "Не импортированы входы",
        !checklist.inboundOk && "Есть неподтвержденное расхождение по входам",
        !checklist.outboundImported && "Не импортированы выплаты",
        !checklist.outboundOk && "Есть неподтвержденное расхождение по выплатам",
        !checklist.allPayoutsFullyPaid && "Есть не полностью оплаченные ручные выплаты",
      ].filter(Boolean)
    : [];

  return (
    <div className="space-y-6">
      <PageHeader title="Мои реквизиты" description="Реквизиты в смене, daily details и накопленные обороты." />
      <Card className={blockers.length ? "border-amber-200 bg-amber-50" : undefined}>
        <CardContent className="flex flex-wrap items-center justify-between gap-4 p-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <span className="font-semibold">Текущая смена</span>
              {shiftQuery.data?.shift ? <StatusBadge status={shiftQuery.data.shift.status} /> : null}
              {!shiftQuery.data?.shift && lastClosedStatus ? <StatusBadge status={lastClosedStatus} /> : null}
            </div>
            <div className="text-sm text-muted-foreground">
              Смена стартует автоматически, когда трейдер берет первый назначенный реквизит в работу.
            </div>
            {blockers.length ? (
              <ul className="list-inside list-disc text-sm text-amber-900">
                {blockers.map((blocker) => (
                  <li key={String(blocker)}>{blocker}</li>
                ))}
              </ul>
            ) : null}
          </div>
          <CloseShiftDialog blockers={blockers as string[]} canClose={Boolean(checklist?.canClose)} onClose={() => closeMutation.mutate()} />
        </CardContent>
      </Card>
      <DataTable
        columns={columns}
        data={requisitesQuery.data ?? []}
        rowCount={requisitesQuery.data?.length ?? 0}
        pagination={pagination}
        onPaginationChange={setPagination}
        isLoading={requisitesQuery.isLoading}
        error={requisitesQuery.error instanceof Error ? requisitesQuery.error.message : null}
        emptyTitle="Нет назначенных реквизитов"
      />
    </div>
  );
}

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
        title="Выплаты"
        description="Ручные выплаты и промежуточные переводы."
        actions={<CreatePayoutDialog onSubmit={(values) => createMutation.mutate(values)} />}
      />
      <div className="grid gap-4 md:grid-cols-3">
        <SummaryCard label="Всего выплат" value={formatMoneyMinor(total)} />
        <SummaryCard label="Оплачено" value={formatMoneyMinor(paid)} />
        <SummaryCard label="Блокеры закрытия" value={String(unpaidCount)} warning={unpaidCount > 0} />
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
        actions={[{ label: "Детали", onSelect: (row) => setDetailsPayout(row) }]}
      />
      <PayoutDetailsDialog payout={detailsPayout} onClose={() => setDetailsPayout(null)} />
    </div>
  );
}

export function TraderOrdersPage({ direction }: { direction: "inbound" | "outbound" }) {
  const dashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard(direction),
    queryFn: () => api.orders.dashboard("trader", direction),
  });
  const reconciliationQuery = useQuery({
    queryKey: ["trader", direction, "reconciliation"],
    queryFn: () => api.orders.reconciliation("trader", direction),
  });

  return (
    <div className="space-y-6">
      <PageHeader
        title={direction === "inbound" ? "Входы" : "Исходящие ордера"}
        description="Импорт CSV, сверка и ордера текущей смены."
        actions={
          <ImportCsvDialog
            scopeLabel={direction === "inbound" ? "Входы текущей смены" : "Выплаты текущей смены"}
            scope="trader"
            direction={direction}
          />
        }
      />
      <OrderDashboard
        dashboard={dashboardQuery.data}
        direction={direction}
        isLoading={dashboardQuery.isLoading}
        error={dashboardQuery.error instanceof Error ? dashboardQuery.error : null}
      />
      {reconciliationQuery.data ? <MismatchAlert summary={reconciliationQuery.data} /> : null}
      {reconciliationQuery.data?.status === "mismatch" && reconciliationQuery.data.runId ? (
        <AcceptMismatchDialog scope="trader" direction={direction} runId={reconciliationQuery.data.runId} />
      ) : null}
      <TraderOrdersTable direction={direction} />
    </div>
  );
}

export function TraderAnalyticsPage() {
  const inboundDashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard("inbound"),
    queryFn: () => api.orders.dashboard("trader", "inbound"),
  });
  const outboundDashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard("outbound"),
    queryFn: () => api.orders.dashboard("trader", "outbound"),
  });

  return (
    <div className="space-y-6">
      <PageHeader title="Аналитика" description="Показатели текущей и прошлых смен." />
      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Входы</h2>
        <OrderDashboard
          dashboard={inboundDashboardQuery.data}
          direction="inbound"
          isLoading={inboundDashboardQuery.isLoading}
          error={inboundDashboardQuery.error instanceof Error ? inboundDashboardQuery.error : null}
        />
      </section>
      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Выплаты</h2>
        <OrderDashboard
          dashboard={outboundDashboardQuery.data}
          direction="outbound"
          isLoading={outboundDashboardQuery.isLoading}
          error={outboundDashboardQuery.error instanceof Error ? outboundDashboardQuery.error : null}
        />
      </section>
    </div>
  );
}

function ShiftRequisiteActions({ item }: { item: ShiftRequisite }) {
  return (
    <div className="flex justify-end gap-2">
      {item.status === "assigned" ? <TakeRequisiteDialog item={item} /> : <EditDetailsDialog item={item} />}
      <AddTurnoverDialog item={item} />
      <TurnoverHistoryDialog shiftRequisiteId={item.id} />
    </div>
  );
}

function TakeRequisiteDialog({ item }: { item: ShiftRequisite }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const form = useForm<z.infer<typeof takeSchema>>({
    resolver: zodResolver(takeSchema),
    defaultValues: { cardNumber: "", holderName: "" },
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
          <DialogDescription>Если открытой смены нет, она будет создана автоматически.</DialogDescription>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) => mutation.mutate({ shiftRequisiteId: item.id, ...values }))}
        >
          <FormField label="Номер карты" error={form.formState.errors.cardNumber?.message}>
            <Input {...form.register("cardNumber")} />
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
    values: { cardNumber: item.cardNumber ?? "", holderName: item.holderName ?? "" },
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
          onSubmit={form.handleSubmit((values) => mutation.mutate({ shiftRequisiteId: item.id, ...values }))}
        >
          <FormField label="Номер карты" error={form.formState.errors.cardNumber?.message}>
            <Input {...form.register("cardNumber")} />
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

function AddTurnoverDialog({ item }: { item: ShiftRequisite }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const form = useForm<z.infer<typeof turnoverSchema>>({
    resolver: zodResolver(turnoverSchema),
    defaultValues: { amount: "", comment: "" },
  });
  const mutation = useMutation({
    mutationFn: api.traderShift.addTurnover,
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
          Оборот
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Добавить оборот</DialogTitle>
          <DialogDescription>Введите текущий накопленный оборот по реквизиту, а не прибавку.</DialogDescription>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) =>
            mutation.mutate({
              shiftRequisiteId: item.id,
              amountMinor: parseMoneyToMinor(values.amount),
              comment: values.comment,
            }),
          )}
        >
          <FormField label="Накопленный оборот" error={form.formState.errors.amount?.message}>
            <Input {...form.register("amount")} />
          </FormField>
          <FormField label="Комментарий">
            <Textarea {...form.register("comment")} />
          </FormField>
          <Button type="submit" disabled={mutation.isPending}>
            Добавить
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function TurnoverHistoryDialog({ shiftRequisiteId }: { shiftRequisiteId: number }) {
  const historyQuery = useQuery({
    queryKey: ["trader", "turnovers", shiftRequisiteId],
    queryFn: () => api.traderShift.turnovers(shiftRequisiteId),
    enabled: false,
  });
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button type="button" variant="ghost" size="icon" onClick={() => void historyQuery.refetch()} aria-label="История оборотов">
          <History className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">История оборотов</DialogTitle>
        </DialogHeader>
        <TurnoverList entries={historyQuery.data ?? []} />
      </DialogContent>
    </Dialog>
  );
}

function TurnoverList({ entries }: { entries: TurnoverEntry[] }) {
  if (!entries.length) return <EmptyState title="Оборотов пока нет" />;
  return (
    <div className="space-y-2">
      {entries.map((entry) => (
        <Card key={entry.id}>
          <CardContent className="flex items-center justify-between p-3">
            <div>
              <DateTimeCell value={entry.createdAt} />
              {entry.comment ? <div className="text-sm text-muted-foreground">{entry.comment}</div> : null}
            </div>
            <MoneyCell valueMinor={entry.amountMinor} />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function CloseShiftDialog({ blockers, canClose, onClose }: { blockers: string[]; canClose: boolean; onClose: () => void }) {
  const [open, setOpen] = useState(false);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" variant={canClose ? "default" : "outline"}>
          Закрыть смену
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Чеклист закрытия смены</DialogTitle>
          <DialogDescription>Закрытую смену нельзя открыть повторно.</DialogDescription>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          {blockers.length ? (
            blockers.map((blocker) => (
              <div key={blocker} className="rounded-md border border-amber-200 bg-amber-50 p-2 text-amber-900">
                {blocker}
              </div>
            ))
          ) : (
            <div className="rounded-md border border-emerald-200 bg-emerald-50 p-2 text-emerald-800">Все проверки пройдены.</div>
          )}
        </div>
        <Button
          type="button"
          disabled={!canClose}
          onClick={() => {
            onClose();
            setOpen(false);
          }}
        >
          Закрыть смену
        </Button>
      </DialogContent>
    </Dialog>
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
        <Select {...form.register("sourceShiftRequisiteId", { valueAsNumber: true })}>
          {shiftRequisites.map((item) => (
            <option key={item.id} value={item.id}>
              {item.phone}
            </option>
          ))}
        </Select>
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

function TransferRow({ transfer, onDelete }: { transfer: PayoutTransfer; onDelete: () => void }) {
  return (
    <Card>
      <CardContent className="flex items-center justify-between gap-3 p-3">
        <div>
          <DateTimeCell value={transfer.createdAt} />
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

function SummaryCard({ label, value, warning }: { label: string; value: string; warning?: boolean }) {
  return (
    <Card className={warning ? "border-amber-200 bg-amber-50" : undefined}>
      <CardHeader>
        <CardTitle className="text-xs uppercase text-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-semibold">{value}</div>
      </CardContent>
    </Card>
  );
}

function TraderOrdersTable({ direction }: { direction: "inbound" | "outbound" }) {
  const [search, setSearch] = useState("");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const ordersQuery = useQuery({
    queryKey: queryKeys.trader.orders(direction, { search }),
    queryFn: () => api.orders.list("trader", direction, { search }),
  });
  const columns = useMemo<ColumnDef<Awaited<ReturnType<typeof api.orders.list>>[number]>[]>(
    () => [
      { accessorKey: "createdAt", header: "Время", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
      { accessorKey: "requisite", header: "Реквизит", cell: ({ row }) => <RequisiteCell phone={row.original.requisite} method={row.original.method} /> },
      { accessorKey: "bankName", header: "Банк" },
      { accessorKey: "amountMinor", header: () => <div className="text-right">Сумма</div>, cell: ({ row }) => <MoneyCell valueMinor={row.original.amountMinor} /> },
      {
        accessorKey: "normalizedStatus",
        header: "Статус",
        cell: ({ row }) => (
          <div className="space-y-1">
            <StatusBadge status={row.original.normalizedStatus} />
            {row.original.rawStatus !== row.original.normalizedStatus ? (
              <div className="text-xs text-muted-foreground">{row.original.rawStatus}</div>
            ) : null}
          </div>
        ),
      },
      { accessorKey: "innerId", header: "innerId" },
    ],
    [],
  );
  return (
    <DataTable
      columns={columns}
      data={ordersQuery.data ?? []}
      rowCount={ordersQuery.data?.length ?? 0}
      pagination={pagination}
      onPaginationChange={setPagination}
      search={search}
      onSearchChange={setSearch}
      isLoading={ordersQuery.isLoading}
    />
  );
}

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { AlertCircle, FileText, History, MoreHorizontal, Plus, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { DateTimeCell } from "@/shared/ui/date-time-cell";
import { EmptyState } from "@/shared/ui/empty-state";
import { FormField } from "@/shared/ui/form-field";
import { MoneyCell } from "@/entities/order/ui/money-cell";
import { PageHeader } from "@/shared/ui/page-header";
import { StatusBadge } from "@/entities/status/ui/status-badge";
import { DataTable } from "@/shared/ui/data-table";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import {
  Dialog,
  DialogClose,
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
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Input } from "@/shared/ui/input";
import { SearchableSelect, type SearchableSelectOption } from "@/shared/ui/searchable-select";
import { Textarea } from "@/shared/ui/textarea";
import type { Bank, Payout, PayoutTransfer, ShiftRequisite } from "@/shared/model/domain";
import { api } from "@/shared/api/api";
import { queryKeys } from "@/shared/api/query-keys";
import {
  formatMoneyMinor,
  formatRussianPhone,
  isValidRussianPhone,
  normalizeRussianPhone,
  parseMoneyToMinor,
} from "@/shared/lib/utils";
import { MetricCard } from "@/shared/ui/metric-card";

const payoutSchema = z.object({
  destinationBank: z.string().min(1, "Выберите банк"),
  destinationRequisite: z.string().min(1, "Введите номер телефона").refine(isValidRussianPhone, "Введите корректный номер телефона"),
  amount: z.string().min(1, "Введите сумму").regex(/^\d+$/, "Введите сумму цифрами").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
});

const transferSchema = z.object({
  sourceShiftRequisiteId: z.coerce.number().min(1, "Выберите источник"),
  amount: z.string().min(1, "Введите сумму").regex(/^\d+$/, "Введите сумму цифрами").refine((value) => parseMoneyToMinor(value) > 0, "Сумма должна быть больше 0"),
  comment: z.string().optional(),
});

function digitsOnly(value: string) {
  return value.replace(/\D/g, "");
}

function formatLiveRussianPhone(value: string) {
  const rawDigits = digitsOnly(value);
  if (!rawDigits) return "";

  const normalizedDigits = (rawDigits.startsWith("7") || rawDigits.startsWith("8") ? rawDigits.slice(1) : rawDigits).slice(0, 10);
  if (!normalizedDigits) return "+7";

  const code = normalizedDigits.slice(0, 3);
  const first = normalizedDigits.slice(3, 6);
  const second = normalizedDigits.slice(6, 8);
  const third = normalizedDigits.slice(8, 10);

  if (normalizedDigits.length <= 3) return `+7 (${code}`;
  if (normalizedDigits.length <= 6) return `+7 (${code}) ${first}`;
  if (normalizedDigits.length <= 8) return `+7 (${code}) ${first}-${second}`;
  return `+7 (${code}) ${first}-${second}-${third}`;
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "Не удалось создать выплату";
}

type PayoutTab = "current" | "history";

export function TraderPayoutsPage() {
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<PayoutTab>("current");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [historyPagination, setHistoryPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsPayout, setDetailsPayout] = useState<Payout | null>(null);
  const [editingPayout, setEditingPayout] = useState<Payout | null>(null);
  const [deletingPayout, setDeletingPayout] = useState<Payout | null>(null);
  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const payoutsQuery = useQuery({ queryKey: queryKeys.trader.payouts(), queryFn: api.payouts.list });
  const payoutHistoryQuery = useQuery({
    queryKey: queryKeys.trader.payoutHistory,
    queryFn: api.payouts.history,
    enabled: activeTab === "history",
  });
  const banksQuery = useQuery({ queryKey: queryKeys.banks, queryFn: api.banks.list });
  const invalidatePayouts = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: queryKeys.trader.payouts() }),
      queryClient.invalidateQueries({ queryKey: queryKeys.trader.payoutHistory }),
    ]);
  const createMutation = useMutation({
    mutationFn: api.payouts.create,
    onSuccess: async () => {
      setToastMessage(null);
      await invalidatePayouts();
    },
    onError: (error) => setToastMessage(errorMessage(error)),
  });
  const updateMutation = useMutation({
    mutationFn: api.payouts.update,
    onSuccess: async (updatedPayout) => {
      setToastMessage(null);
      setEditingPayout(null);
      setDetailsPayout((current) => (current?.id === updatedPayout.id ? updatedPayout : current));
      await invalidatePayouts();
    },
    onError: (error) => setToastMessage(errorMessage(error)),
  });
  const deleteMutation = useMutation({
    mutationFn: api.payouts.delete,
    onSuccess: async (_, payoutId) => {
      setToastMessage(null);
      setDeletingPayout(null);
      setDetailsPayout((current) => (current?.id === payoutId ? null : current));
      await invalidatePayouts();
    },
    onError: (error) => setToastMessage(errorMessage(error)),
  });
  const data = payoutsQuery.data ?? [];
  const total = data.reduce((sum, payout) => sum + payout.amountMinor, 0);
  const paid = data.reduce((sum, payout) => sum + payout.paidMinor, 0);
  const unpaidCount = data.filter((payout) => payout.amountMinor > payout.paidMinor).length;
  const activeDetailsPayout = detailsPayout ? data.find((payout) => payout.id === detailsPayout.id) ?? detailsPayout : null;
  const currentColumns = useMemo<ColumnDef<Payout>[]>(
    () => [
      { accessorKey: "createdAt", header: "Создана", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
      { accessorKey: "destinationBank", header: "Банк" },
      {
        accessorKey: "destinationRequisite",
        header: "Телефон",
        cell: ({ row }) => <span className="tabular-nums">{formatRussianPhone(row.original.destinationRequisite)}</span>,
      },
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
  const historyColumns = useMemo<ColumnDef<Payout>[]>(
    () => [
      {
        accessorKey: "shiftId",
        header: "Смена",
        cell: ({ row }) => <span className="tabular-nums">#{row.original.shiftId}</span>,
      },
      { accessorKey: "createdAt", header: "Создана", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
      { accessorKey: "updatedAt", header: "Обновлена", cell: ({ row }) => <DateTimeCell value={row.original.deletedAt ?? row.original.updatedAt} /> },
      { accessorKey: "destinationBank", header: "Банк" },
      {
        accessorKey: "destinationRequisite",
        header: "Телефон",
        cell: ({ row }) => <span className="tabular-nums">{formatRussianPhone(row.original.destinationRequisite)}</span>,
      },
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
        description="Текущие ручные выплаты, промежуточные переводы и история."
        actions={
          <CreatePayoutDialog
            banks={banksQuery.data ?? []}
            isSaving={createMutation.isPending}
            onSubmit={(values) => createMutation.mutateAsync(values)}
          />
        }
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
      <PayoutTabs value={activeTab} onChange={setActiveTab} />
      {activeTab === "current" ? (
        <DataTable
          columns={currentColumns}
          data={data}
          rowCount={data.length}
          pagination={pagination}
          onPaginationChange={setPagination}
          isLoading={payoutsQuery.isLoading}
          error={payoutsQuery.error instanceof Error ? payoutsQuery.error.message : null}
          emptyTitle="Нет текущих выплат"
          emptyDescription="Созданные и не отмененные выплаты текущей смены появятся здесь."
          onRowClick={setDetailsPayout}
          actions={[
            { label: "Детали", onSelect: (row) => setDetailsPayout(row) },
            { label: "Редактировать", onSelect: (row) => setEditingPayout(row) },
            { label: "Удалить", onSelect: (row) => setDeletingPayout(row), destructive: true },
          ]}
        />
      ) : null}
      {activeTab === "history" ? (
        <DataTable
          columns={historyColumns}
          data={payoutHistoryQuery.data ?? []}
          rowCount={payoutHistoryQuery.data?.length ?? 0}
          pagination={historyPagination}
          onPaginationChange={setHistoryPagination}
          isLoading={payoutHistoryQuery.isLoading}
          error={payoutHistoryQuery.error instanceof Error ? payoutHistoryQuery.error.message : null}
          emptyTitle="История пуста"
          emptyDescription="Отмененные выплаты и выплаты прошлых смен появятся здесь."
          onRowClick={setDetailsPayout}
          actions={[{ label: "Детали", onSelect: (row) => setDetailsPayout(row) }]}
        />
      ) : null}
      <EditPayoutDialog
        payout={editingPayout}
        banks={banksQuery.data ?? []}
        isSaving={updateMutation.isPending}
        onClose={() => setEditingPayout(null)}
        onSubmit={(values) => updateMutation.mutateAsync(values)}
      />
      <DeletePayoutDialog
        payout={deletingPayout}
        isDeleting={deleteMutation.isPending}
        onClose={() => setDeletingPayout(null)}
        onConfirm={(payoutId) => deleteMutation.mutate(payoutId)}
      />
      <PayoutDetailsDialog payout={activeDetailsPayout} readOnly={activeTab === "history"} onClose={() => setDetailsPayout(null)} />
      <PayoutErrorToast message={toastMessage} onClose={() => setToastMessage(null)} />
    </div>
  );
}

function PayoutTabs({ value, onChange }: { value: PayoutTab; onChange: (value: PayoutTab) => void }) {
  const tabs: { value: PayoutTab; label: string; icon: typeof FileText }[] = [
    { value: "current", label: "Текущие", icon: FileText },
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

function CreatePayoutDialog({
  banks,
  isSaving,
  onSubmit,
}: {
  banks: Bank[];
  isSaving: boolean;
  onSubmit: (values: { destinationBank: string; destinationRequisite: string; amountMinor: number }) => Promise<unknown>;
}) {
  const [open, setOpen] = useState(false);
  const form = useForm<z.infer<typeof payoutSchema>>({
    resolver: zodResolver(payoutSchema),
    mode: "onChange",
    defaultValues: { destinationBank: "", destinationRequisite: "", amount: "" },
  });
  const bankOptions = useMemo<SearchableSelectOption[]>(
    () => banks.map((bank) => ({ value: bank.name, label: bank.name, searchText: bank.code })),
    [banks],
  );
  const destinationRequisiteField = form.register("destinationRequisite");
  const amountField = form.register("amount");
  const destinationRequisiteValue = form.watch("destinationRequisite");
  const amountValue = form.watch("amount");
  const closeDialog = () => {
    setOpen(false);
    form.reset();
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => (nextOpen ? setOpen(true) : closeDialog())}>
      <DialogTrigger asChild>
        <Button type="button">
          <Plus className="h-4 w-4" />
          Добавить выплату
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Ручная выплата</DialogTitle>
          <DialogDescription>
            Укажите реквизиты получателя выплаты. Реквизит, с которого трейдер отправляет деньги, выбирается позже в частичных переводах.
          </DialogDescription>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={form.handleSubmit(async (values) => {
            try {
              await onSubmit({
                destinationBank: values.destinationBank,
                destinationRequisite: normalizeRussianPhone(values.destinationRequisite),
                amountMinor: parseMoneyToMinor(values.amount),
              });
              closeDialog();
            } catch {
              // Mutation onError renders the toast; keep the dialog open for correction/retry.
            }
          })}
        >
          <FormField
            label="Банк получателя"
            error={form.formState.errors.destinationBank?.message}
            labelInfo="Банк реквизита получателя, на который нужно отправить выплату."
            help="Это банк получателя из выплатного ордера, а не наш реквизит-источник."
          >
            <SearchableSelect
              value={form.watch("destinationBank")}
              options={bankOptions}
              onValueChange={(value) => form.setValue("destinationBank", value, { shouldDirty: true, shouldValidate: true })}
              placeholder="Выберите банк"
              searchPlaceholder="Найти банк"
              emptyText="Банк не найден"
              disabled={banks.length === 0}
            />
          </FormField>
          <FormField
            label="Телефон получателя"
            error={form.formState.errors.destinationRequisite?.message}
            labelInfo="Телефон или реквизит назначения, куда должна уйти выплата."
            help="Введите реквизит получателя из выплатного ордера. Этот номер не используется как наш источник списания."
          >
            <Input
              {...destinationRequisiteField}
              value={destinationRequisiteValue}
              inputMode="numeric"
              autoComplete="tel"
              placeholder="+7 (999) 123-45-67"
              onChange={(event) =>
                form.setValue("destinationRequisite", formatLiveRussianPhone(event.target.value), {
                  shouldDirty: true,
                  shouldValidate: true,
                })
              }
            />
          </FormField>
          <FormField
            label="Сумма выплаты"
            error={form.formState.errors.amount?.message}
            labelInfo="Полная сумма, которую нужно отправить получателю по этой выплате."
            help="Если трейдер платит частями, отдельные переводы добавляются в деталях выплаты после создания."
          >
            <Input
              {...amountField}
              value={amountValue}
              inputMode="numeric"
              pattern="[0-9]*"
              onChange={(event) => form.setValue("amount", digitsOnly(event.target.value), { shouldDirty: true, shouldValidate: true })}
            />
          </FormField>
          <Button type="submit" disabled={isSaving || banks.length === 0}>
            Создать
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function EditPayoutDialog({
  payout,
  banks,
  isSaving,
  onClose,
  onSubmit,
}: {
  payout: Payout | null;
  banks: Bank[];
  isSaving: boolean;
  onClose: () => void;
  onSubmit: (values: { id: number; destinationBank: string; destinationRequisite: string; amountMinor: number }) => Promise<unknown>;
}) {
  const form = useForm<z.infer<typeof payoutSchema>>({
    resolver: zodResolver(payoutSchema),
    mode: "onChange",
    defaultValues: { destinationBank: "", destinationRequisite: "", amount: "" },
  });
  const bankOptions = useMemo<SearchableSelectOption[]>(
    () => banks.map((bank) => ({ value: bank.name, label: bank.name, searchText: bank.code })),
    [banks],
  );
  const destinationRequisiteField = form.register("destinationRequisite");
  const amountField = form.register("amount");
  const destinationRequisiteValue = form.watch("destinationRequisite");
  const amountValue = form.watch("amount");

  useEffect(() => {
    if (!payout) return;

    form.reset({
      destinationBank: payout.destinationBank,
      destinationRequisite: formatLiveRussianPhone(payout.destinationRequisite),
      amount: String(Math.trunc(payout.amountMinor / 100)),
    });
  }, [form, payout]);

  return (
    <Dialog open={Boolean(payout)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Редактировать выплату</DialogTitle>
          <DialogDescription>
            Можно изменить банк, телефон получателя и сумму. Сумма не может быть меньше уже добавленных переводов.
          </DialogDescription>
        </DialogHeader>
        {payout ? (
          <form
            className="space-y-4"
            onSubmit={form.handleSubmit(async (values) => {
              const amountMinor = parseMoneyToMinor(values.amount);
              if (amountMinor < payout.paidMinor) {
                form.setError("amount", { message: "Сумма не может быть меньше уже оплаченной" });
                return;
              }

              try {
                await onSubmit({
                  id: payout.id,
                  destinationBank: values.destinationBank,
                  destinationRequisite: normalizeRussianPhone(values.destinationRequisite),
                  amountMinor,
                });
              } catch {
                // Mutation onError renders the toast; keep the dialog open for correction/retry.
              }
            })}
          >
            <FormField
              label="Банк получателя"
              error={form.formState.errors.destinationBank?.message}
              help="Банк получателя из выплатного ордера."
            >
              <SearchableSelect
                value={form.watch("destinationBank")}
                options={bankOptions}
                onValueChange={(value) => form.setValue("destinationBank", value, { shouldDirty: true, shouldValidate: true })}
                placeholder="Выберите банк"
                searchPlaceholder="Найти банк"
                emptyText="Банк не найден"
                disabled={banks.length === 0}
              />
            </FormField>
            <FormField
              label="Телефон получателя"
              error={form.formState.errors.destinationRequisite?.message}
              help="Реквизит назначения, куда должна уйти выплата."
            >
              <Input
                {...destinationRequisiteField}
                value={destinationRequisiteValue}
                inputMode="numeric"
                autoComplete="tel"
                placeholder="+7 (999) 123-45-67"
                onChange={(event) =>
                  form.setValue("destinationRequisite", formatLiveRussianPhone(event.target.value), {
                    shouldDirty: true,
                    shouldValidate: true,
                  })
                }
              />
            </FormField>
            <FormField
              label="Сумма выплаты"
              error={form.formState.errors.amount?.message}
              help={`Уже оплачено: ${formatMoneyMinor(payout.paidMinor)}`}
            >
              <Input
                {...amountField}
                value={amountValue}
                inputMode="numeric"
                pattern="[0-9]*"
                onChange={(event) => form.setValue("amount", digitsOnly(event.target.value), { shouldDirty: true, shouldValidate: true })}
              />
            </FormField>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={onClose}>
                Отмена
              </Button>
              <Button type="submit" disabled={isSaving || banks.length === 0}>
                Сохранить
              </Button>
            </div>
          </form>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function DeletePayoutDialog({
  payout,
  isDeleting,
  onClose,
  onConfirm,
}: {
  payout: Payout | null;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: (payoutId: number) => void;
}) {
  return (
    <Dialog open={Boolean(payout)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Удалить выплату?</DialogTitle>
          <DialogDescription className="text-sm text-muted-foreground">
            Выплата будет отменена и пропадет из текущего списка. Действие попадет в аудит.
          </DialogDescription>
        </DialogHeader>
        {payout ? (
          <div className="rounded-md border border-border bg-slate-50 p-3 text-sm">
            <div className="font-medium">{payout.destinationBank}</div>
            <div className="text-muted-foreground">{formatRussianPhone(payout.destinationRequisite)}</div>
            <div className="mt-2 font-semibold">{formatMoneyMinor(payout.amountMinor)}</div>
          </div>
        ) : null}
        <div className="flex justify-end gap-2">
          <DialogClose asChild>
            <Button type="button" variant="outline">
              Отмена
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            disabled={!payout || isDeleting}
            onClick={() => payout && onConfirm(payout.id)}
          >
            Удалить
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function PayoutErrorToast({ message, onClose }: { message: string | null; onClose: () => void }) {
  if (!message) return null;

  return (
    <div
      role="alert"
      className="fixed bottom-5 right-5 z-50 flex max-w-sm items-start gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 shadow-lg"
    >
      <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
      <div className="min-w-0 flex-1 font-medium">{message}</div>
      <button
        type="button"
        className="rounded-sm p-0.5 text-red-700 hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500"
        aria-label="Закрыть уведомление"
        onClick={onClose}
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}

function PayoutDetailsDialog({
  payout,
  readOnly = false,
  onClose,
}: {
  payout: Payout | null;
  readOnly?: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [editingTransfer, setEditingTransfer] = useState<PayoutTransfer | null>(null);
  const [deletingTransfer, setDeletingTransfer] = useState<PayoutTransfer | null>(null);
  const transfersQuery = useQuery({
    queryKey: ["trader", "payouts", payout?.id, "transfers"],
    queryFn: () => api.payouts.transfers(payout?.id ?? 0),
    enabled: Boolean(payout),
  });
  const requisitesQuery = useQuery({
    queryKey: queryKeys.trader.requisites(),
    queryFn: api.traderShift.requisites,
    enabled: Boolean(payout) && !readOnly,
  });
  const addTransferMutation = useMutation({
    mutationFn: api.payouts.addTransfer,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["trader", "payouts"] }),
  });
  const updateTransferMutation = useMutation({
    mutationFn: api.payouts.updateTransfer,
    onSuccess: async () => {
      setEditingTransfer(null);
      await queryClient.invalidateQueries({ queryKey: ["trader", "payouts"] });
    },
  });
  const deleteTransferMutation = useMutation({
    mutationFn: api.payouts.deleteTransfer,
    onSuccess: async () => {
      setDeletingTransfer(null);
      await queryClient.invalidateQueries({ queryKey: ["trader", "payouts"] });
    },
  });
  const remaining = payout ? payout.amountMinor - payout.paidMinor : 0;
  const sourceByShiftRequisiteId = new Map(
    (!readOnly ? requisitesQuery.data ?? [] : []).map((item) => [item.id, `${formatRussianPhone(item.phone)} · ${item.bankName}`]),
  );

  return (
    <Dialog open={Boolean(payout)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="left-auto right-0 top-0 h-screen w-[min(620px,100vw)] translate-x-0 translate-y-0 rounded-none">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Детали выплаты</DialogTitle>
          <DialogDescription>
            Сумма: {formatMoneyMinor(payout?.amountMinor ?? 0)} · Оплачено: {formatMoneyMinor(payout?.paidMinor ?? 0)} · Остаток:{" "}
            {formatMoneyMinor(remaining)}
          </DialogDescription>
        </DialogHeader>
        {payout ? (
          <div className="space-y-5">
            {!readOnly ? (
              <AddTransferForm
                payout={payout}
                shiftRequisites={requisitesQuery.data ?? []}
                onSubmit={(values) => addTransferMutation.mutate(values)}
              />
            ) : null}
            <div className="space-y-2">
              <div className="text-sm font-semibold">Переводы</div>
              {(transfersQuery.data ?? []).map((transfer) => (
                <TransferRow
                  key={transfer.id}
                  transfer={transfer}
                  sourceLabel={transferSourceLabel(transfer, sourceByShiftRequisiteId)}
                  onEdit={readOnly ? undefined : () => setEditingTransfer(transfer)}
                  onDelete={readOnly ? undefined : () => setDeletingTransfer(transfer)}
                />
              ))}
              {!transfersQuery.data?.length ? <EmptyState title="Переводов пока нет" /> : null}
            </div>
            {!readOnly ? (
              <>
                <EditTransferDialog
                  payout={payout}
                  transfer={editingTransfer}
                  shiftRequisites={requisitesQuery.data ?? []}
                  isSaving={updateTransferMutation.isPending}
                  onClose={() => setEditingTransfer(null)}
                  onSubmit={(values) => updateTransferMutation.mutate(values)}
                />
                <DeleteTransferDialog
                  transfer={deletingTransfer}
                  sourceLabel={deletingTransfer ? transferSourceLabel(deletingTransfer, sourceByShiftRequisiteId) : ""}
                  isDeleting={deleteTransferMutation.isPending}
                  onClose={() => setDeletingTransfer(null)}
                  onConfirm={(transfer) => deleteTransferMutation.mutate({ payoutId: payout.id, transferId: transfer.id })}
                />
              </>
            ) : null}
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function transferSourceLabel(transfer: PayoutTransfer, fallbackByShiftRequisiteId: Map<number, string>) {
  if (transfer.sourcePhone || transfer.sourceBankName) {
    const phone = transfer.sourcePhone ? formatRussianPhone(transfer.sourcePhone) : `Реквизит #${transfer.sourceRequisiteId}`;
    return transfer.sourceBankName ? `${phone} · ${transfer.sourceBankName}` : phone;
  }

  return fallbackByShiftRequisiteId.get(transfer.sourceShiftRequisiteId) ?? `Реквизит #${transfer.sourceRequisiteId}`;
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
    mode: "onChange",
    defaultValues: { sourceShiftRequisiteId: shiftRequisites[0]?.id ?? 0, amount: "", comment: "" },
  });
  const amountField = form.register("amount");
  const amountValue = form.watch("amount");
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
        <Input
          {...amountField}
          value={amountValue}
          inputMode="numeric"
          pattern="[0-9]*"
          onChange={(event) => form.setValue("amount", digitsOnly(event.target.value), { shouldDirty: true, shouldValidate: true })}
        />
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

function EditTransferDialog({
  payout,
  transfer,
  shiftRequisites,
  isSaving,
  onClose,
  onSubmit,
}: {
  payout: Payout;
  transfer: PayoutTransfer | null;
  shiftRequisites: ShiftRequisite[];
  isSaving: boolean;
  onClose: () => void;
  onSubmit: (values: { payoutId: number; transferId: number; sourceShiftRequisiteId: number; amountMinor: number; comment?: string }) => void;
}) {
  const editableRemaining = transfer ? payout.amountMinor - payout.paidMinor + transfer.amountMinor : 0;
  const schema = transferSchema.refine((values) => parseMoneyToMinor(values.amount) <= editableRemaining, {
    path: ["amount"],
    message: "Сумма перевода не может быть больше доступного остатка",
  });
  const form = useForm<z.infer<typeof transferSchema>>({
    resolver: zodResolver(schema),
    mode: "onChange",
    defaultValues: { sourceShiftRequisiteId: 0, amount: "", comment: "" },
  });
  const amountField = form.register("amount");
  const amountValue = form.watch("amount");
  const sourceOptions = useMemo<SearchableSelectOption[]>(() => {
    const options = shiftRequisites.map((item) => ({
      value: String(item.id),
      label: `${formatRussianPhone(item.phone)} · ${item.bankName}`,
      searchText: `${item.proxy} ${item.holderName ?? ""} ${item.cardNumber ?? ""}`,
    }));
    if (transfer && !options.some((option) => option.value === String(transfer.sourceShiftRequisiteId))) {
      options.unshift({
        value: String(transfer.sourceShiftRequisiteId),
        label: transferSourceLabel(transfer, new Map()),
        searchText: "",
      });
    }
    return options;
  }, [shiftRequisites, transfer]);

  useEffect(() => {
    if (!transfer) return;
    form.reset({
      sourceShiftRequisiteId: transfer.sourceShiftRequisiteId,
      amount: String(Math.trunc(transfer.amountMinor / 100)),
      comment: transfer.comment ?? "",
    });
  }, [form, transfer]);

  return (
    <Dialog open={Boolean(transfer)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Редактировать перевод</DialogTitle>
          <DialogDescription>Измените источник, сумму или комментарий перевода.</DialogDescription>
        </DialogHeader>
        {transfer ? (
          <form
            className="space-y-4"
            onSubmit={form.handleSubmit((values) =>
              onSubmit({
                payoutId: payout.id,
                transferId: transfer.id,
                sourceShiftRequisiteId: values.sourceShiftRequisiteId,
                amountMinor: parseMoneyToMinor(values.amount),
                comment: values.comment,
              }),
            )}
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
            <FormField
              label="Сумма перевода"
              error={form.formState.errors.amount?.message}
              help={`Доступно с учетом текущего перевода: ${formatMoneyMinor(editableRemaining)}`}
            >
              <Input
                {...amountField}
                value={amountValue}
                inputMode="numeric"
                pattern="[0-9]*"
                onChange={(event) => form.setValue("amount", digitsOnly(event.target.value), { shouldDirty: true, shouldValidate: true })}
              />
            </FormField>
            <FormField label="Комментарий">
              <Textarea {...form.register("comment")} />
            </FormField>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={onClose}>
                Отмена
              </Button>
              <Button type="submit" disabled={isSaving}>
                Сохранить
              </Button>
            </div>
          </form>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function DeleteTransferDialog({
  transfer,
  sourceLabel,
  isDeleting,
  onClose,
  onConfirm,
}: {
  transfer: PayoutTransfer | null;
  sourceLabel: string;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: (transfer: PayoutTransfer) => void;
}) {
  return (
    <Dialog open={Boolean(transfer)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Удалить перевод?</DialogTitle>
          <DialogDescription className="text-sm text-muted-foreground">
            Сумма выплаты будет пересчитана. Действие попадет в аудит.
          </DialogDescription>
        </DialogHeader>
        {transfer ? (
          <div className="rounded-md border border-border bg-slate-50 p-3 text-sm">
            <div className="font-medium">{sourceLabel}</div>
            <div className="mt-2 font-semibold">{formatMoneyMinor(transfer.amountMinor)}</div>
          </div>
        ) : null}
        <div className="flex justify-end gap-2">
          <DialogClose asChild>
            <Button type="button" variant="outline">
              Отмена
            </Button>
          </DialogClose>
          <Button type="button" variant="destructive" disabled={!transfer || isDeleting} onClick={() => transfer && onConfirm(transfer)}>
            Удалить
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function TransferRow({
  transfer,
  sourceLabel,
  onEdit,
  onDelete,
}: {
  transfer: PayoutTransfer;
  sourceLabel: string;
  onEdit?: () => void;
  onDelete?: () => void;
}) {
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
          {onEdit || onDelete ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button type="button" variant="ghost" size="icon" aria-label="Действия перевода">
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {onEdit ? <DropdownMenuItem onSelect={onEdit}>Редактировать</DropdownMenuItem> : null}
                {onDelete ? (
                  <DropdownMenuItem className="text-red-700 focus:text-red-700" onSelect={onDelete}>
                    Удалить
                  </DropdownMenuItem>
                ) : null}
              </DropdownMenuContent>
            </DropdownMenu>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}

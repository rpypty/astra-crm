import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { Eye, History, Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { DateTimeCell } from "@/components/crm/date-time-cell";
import { EmptyState } from "@/components/crm/empty-state";
import { FormField } from "@/components/crm/form-field";
import { AcceptMismatchDialog, ImportCsvDialog, MismatchAlert } from "@/components/crm/import-components";
import { MoneyCell } from "@/components/crm/money-cell";
import { PageHeader } from "@/components/crm/page-header";
import { RequisiteCell } from "@/components/crm/requisite-cell";
import { StatusBadge } from "@/components/crm/status-badge";
import { UserCell } from "@/components/crm/user-cell";
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
import type { AccountingPeriod, AuditLogEntry, Order, Requisite, Trader } from "@/lib/domain";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { bpsToPercent, formatMoneyMinor, percentToBps } from "@/lib/utils";

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
  phone: z.string().min(1, "Введите телефон"),
  methodType: z.enum(["SBP", "C2C"]),
  proxy: z.string().min(1, "Введите proxy"),
  assignedTraderId: z.string(),
  status: z.enum(["active", "archived"]),
});

type TraderForm = z.infer<typeof traderSchema>;
type RequisiteForm = z.infer<typeof requisiteSchema>;

export function TeamleadTradersPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [editingTrader, setEditingTrader] = useState<Trader | null>(null);
  const [archiveTrader, setArchiveTrader] = useState<Trader | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [generatedPassword, setGeneratedPassword] = useState<string | null>(null);

  const tradersQuery = useQuery({
    queryKey: queryKeys.teamlead.traders({ search, status }),
    queryFn: () => api.traders.list({ search, status }),
  });

  const saveMutation = useMutation({
    mutationFn: api.traders.save,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["teamlead", "traders"] });
      setFormOpen(false);
      setEditingTrader(null);
    },
  });

  const resetPasswordMutation = useMutation({
    mutationFn: api.traders.resetPassword,
    onSuccess: (response) => setGeneratedPassword(response.password),
  });

  const archiveMutation = useMutation({
    mutationFn: api.traders.archive,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["teamlead", "traders"] }),
  });

  const columns = useMemo<ColumnDef<Trader>[]>(
    () => [
      {
        accessorKey: "login",
        header: "Логин",
        cell: ({ row }) => <UserCell login={row.original.login} secondary={row.original.externalWorkerName} />,
      },
      {
        accessorKey: "salaryRateBps",
        header: "Ставка",
        cell: ({ row }) => <span className="tabular-nums">{bpsToPercent(row.original.salaryRateBps)}%</span>,
      },
      {
        accessorKey: "assignedRequisitesCount",
        header: "Реквизиты",
      },
      {
        accessorKey: "currentShiftStatus",
        header: "Смена",
        cell: ({ row }) =>
          row.original.currentShiftStatus === "none" ? "—" : <StatusBadge status={row.original.currentShiftStatus} />,
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
    ],
    [],
  );

  const data = tradersQuery.data ?? [];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Сотрудники"
        description="Трейдеры, ставки, статусы и текущие смены."
        actions={
          <Button
            type="button"
            onClick={() => {
              setEditingTrader(null);
              setFormOpen(true);
            }}
          >
            <Plus className="h-4 w-4" />
            Добавить трейдера
          </Button>
        }
      />
      <DataTable
        columns={columns}
        data={data}
        rowCount={data.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        search={search}
        onSearchChange={setSearch}
        toolbarFilters={
          <Select className="w-40" value={status} onChange={(event) => setStatus(event.target.value)}>
            <option value="all">Все статусы</option>
            <option value="active">Активные</option>
            <option value="disabled">Отключенные</option>
          </Select>
        }
        isLoading={tradersQuery.isLoading}
        error={tradersQuery.error instanceof Error ? tradersQuery.error.message : null}
        emptyTitle="Сотрудников пока нет"
        emptyDescription="Добавьте первого трейдера для работы со сменами."
        actions={[
          {
            label: "Редактировать",
            onSelect: (row) => {
              setEditingTrader(row);
              setFormOpen(true);
            },
          },
          { label: "Сбросить пароль", onSelect: (row) => resetPasswordMutation.mutate(row.id) },
          { label: "Отключить", destructive: true, onSelect: (row) => setArchiveTrader(row) },
        ]}
      />
      <ConfirmActionDialog
        open={Boolean(archiveTrader)}
        onOpenChange={(open) => !open && setArchiveTrader(null)}
        title="Отключить трейдера?"
        description="Трейдер потеряет доступ к CRM. Действие будет записано в аудит."
        confirmText="Отключить"
        onConfirm={() => {
          if (archiveTrader) archiveMutation.mutate(archiveTrader.id);
          setArchiveTrader(null);
        }}
      />
      <TraderFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        trader={editingTrader}
        isSaving={saveMutation.isPending}
        onSubmit={(values) =>
          saveMutation.mutate({
            id: values.id,
            login: values.login,
            password: values.password,
            externalWorkerName: values.externalWorkerName,
            salaryRateBps: percentToBps(values.salaryPercent),
            status: values.status,
          })
        }
      />
      <GeneratedPasswordDialog password={generatedPassword} onClose={() => setGeneratedPassword(null)} />
    </div>
  );
}

export function TeamleadRequisitesPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [methodType, setMethodType] = useState("all");
  const [status, setStatus] = useState("all");
  const [traderId, setTraderId] = useState("all");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [editingRequisite, setEditingRequisite] = useState<Requisite | null>(null);
  const [archiveRequisite, setArchiveRequisite] = useState<Requisite | null>(null);
  const [historyRequisite, setHistoryRequisite] = useState<Requisite | null>(null);
  const [formOpen, setFormOpen] = useState(false);

  const requisitesQuery = useQuery({
    queryKey: queryKeys.teamlead.requisites({ search, methodType, status, traderId }),
    queryFn: () => api.requisites.list({ search, methodType, status, traderId }),
  });
  const tradersQuery = useQuery({
    queryKey: queryKeys.teamlead.traders({ status: "active" }),
    queryFn: () => api.traders.list({ status: "active" }),
  });

  const saveMutation = useMutation({
    mutationFn: api.requisites.save,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["teamlead", "requisites"] });
      setFormOpen(false);
      setEditingRequisite(null);
    },
  });
  const archiveMutation = useMutation({
    mutationFn: api.requisites.archive,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["teamlead", "requisites"] }),
  });

  const columns = useMemo<ColumnDef<Requisite>[]>(
    () => [
      {
        accessorKey: "phone",
        header: "Реквизит",
        cell: ({ row }) => (
          <RequisiteCell phone={row.original.phone} method={row.original.methodType} proxy={row.original.proxy} />
        ),
      },
      {
        accessorKey: "assignedTraderLogin",
        header: "Трейдер",
        cell: ({ row }) =>
          row.original.assignedTraderLogin ? (
            <UserCell login={row.original.assignedTraderLogin} />
          ) : (
            <span className="text-muted-foreground">Не назначен</span>
          ),
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        accessorKey: "updatedAt",
        header: "Обновлен",
        cell: ({ row }) => <DateTimeCell value={row.original.updatedAt} />,
      },
    ],
    [],
  );
  const data = requisitesQuery.data ?? [];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Реквизиты"
        description="Базовые реквизиты команды и история назначений. Карта и держатель здесь не хранятся."
        actions={
          <Button
            type="button"
            onClick={() => {
              setEditingRequisite(null);
              setFormOpen(true);
            }}
          >
            <Plus className="h-4 w-4" />
            Добавить реквизит
          </Button>
        }
      />
      <DataTable
        columns={columns}
        data={data}
        rowCount={data.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        search={search}
        onSearchChange={setSearch}
        toolbarFilters={
          <div className="flex gap-2">
            <Select className="w-32" value={methodType} onChange={(event) => setMethodType(event.target.value)}>
              <option value="all">Метод</option>
              <option value="SBP">SBP</option>
              <option value="C2C">C2C</option>
            </Select>
            <Select className="w-36" value={status} onChange={(event) => setStatus(event.target.value)}>
              <option value="all">Статус</option>
              <option value="active">Активные</option>
              <option value="archived">Архив</option>
            </Select>
            <Select className="w-44" value={traderId} onChange={(event) => setTraderId(event.target.value)}>
              <option value="all">Все трейдеры</option>
              <option value="unassigned">Не назначены</option>
              {(tradersQuery.data ?? []).map((trader) => (
                <option key={trader.id} value={trader.id}>
                  {trader.login}
                </option>
              ))}
            </Select>
          </div>
        }
        isLoading={requisitesQuery.isLoading}
        error={requisitesQuery.error instanceof Error ? requisitesQuery.error.message : null}
        emptyTitle="Реквизитов пока нет"
        emptyDescription="Добавьте первый реквизит, чтобы назначить его трейдеру."
        actions={[
          {
            label: "Редактировать",
            onSelect: (row) => {
              setEditingRequisite(row);
              setFormOpen(true);
            },
          },
          { label: "История", onSelect: (row) => setHistoryRequisite(row) },
          { label: "Архивировать", destructive: true, onSelect: (row) => setArchiveRequisite(row) },
        ]}
      />
      <ConfirmActionDialog
        open={Boolean(archiveRequisite)}
        onOpenChange={(open) => !open && setArchiveRequisite(null)}
        title="Архивировать реквизит?"
        description="Использованный реквизит не удаляется физически, а переносится в архив."
        confirmText="Архивировать"
        onConfirm={() => {
          if (archiveRequisite) archiveMutation.mutate(archiveRequisite.id);
          setArchiveRequisite(null);
        }}
      />
      <RequisiteFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        requisite={editingRequisite}
        traders={tradersQuery.data ?? []}
        isSaving={saveMutation.isPending}
        onSubmit={(values) =>
          saveMutation.mutate({
            id: values.id,
            phone: values.phone,
            methodType: values.methodType,
            proxy: values.proxy,
            assignedTraderId: values.assignedTraderId === "unassigned" ? undefined : Number(values.assignedTraderId),
            status: values.status,
          })
        }
      />
      <AssignmentHistoryViewer requisite={historyRequisite} onClose={() => setHistoryRequisite(null)} />
    </div>
  );
}

export function TeamleadDashboardPage() {
  return (
    <div className="space-y-6">
      <PageHeader title="Дашборд" description="Сводка по активному периоду, импортам и расхождениям." />
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {[
          { label: "Успешный оборот", value: "265 300 ₽", detail: "1 284 ордера" },
          { label: "Неуспешный оборот", value: "18 900 ₽", detail: "42 ордера" },
          { label: "Конверсия", value: "96.8%", detail: "hand_success + corrected" },
          { label: "Расхождения", value: "3", detail: "требуют комментария", warning: true },
        ].map((metric) => (
          <Card key={metric.label} className={metric.warning ? "border-red-200 bg-red-50" : undefined}>
            <CardHeader>
              <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">{metric.label}</div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-semibold">{metric.value}</div>
              <div className="mt-1 text-sm text-muted-foreground">{metric.detail}</div>
            </CardContent>
          </Card>
        ))}
      </div>
      <OrdersPage direction="inbound" scope="teamlead" embedded />
    </div>
  );
}

export function OrdersPage({
  direction,
  scope,
  embedded,
}: {
  direction: "inbound" | "outbound";
  scope: "teamlead" | "trader";
  embedded?: boolean;
}) {
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const ordersQuery = useQuery({
    queryKey:
      scope === "teamlead"
        ? queryKeys.teamlead.orders(direction, { search, status })
        : queryKeys.trader.orders(direction, { search, status }),
    queryFn: () => api.orders.list(scope, direction, { search, status }),
  });
  const reconciliationQuery = useQuery({
    queryKey: [scope, direction, "reconciliation"],
    queryFn: () => api.orders.reconciliation(scope, direction),
    enabled: scope === "trader" || direction === "inbound",
  });
  const columns = useMemo<ColumnDef<Order>[]>(
    () => [
      { accessorKey: "createdAt", header: "Время", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
      { accessorKey: "trader", header: "Трейдер", cell: ({ row }) => <UserCell login={row.original.trader} secondary={row.original.workerName} /> },
      { accessorKey: "requisite", header: "Реквизит", cell: ({ row }) => <RequisiteCell phone={row.original.requisite} method={row.original.method} /> },
      { accessorKey: "bankName", header: "Банк" },
      { accessorKey: "amountMinor", header: () => <div className="text-right">Сумма</div>, cell: ({ row }) => <MoneyCell valueMinor={row.original.amountMinor} /> },
      { accessorKey: "status", header: "Статус", cell: ({ row }) => <StatusBadge status={row.original.status} /> },
      { accessorKey: "innerId", header: "innerId" },
    ],
    [],
  );
  const data = ordersQuery.data ?? [];
  const title = direction === "inbound" ? "Входы" : "Выплаты";

  return (
    <div className="space-y-6">
      {!embedded ? (
        <PageHeader
          title={title}
          description="Ордера, импорт CSV и состояние сверки."
          actions={
            <ImportCsvDialog
              scopeLabel={direction === "inbound" ? "Период тимлида: входы" : "Период тимлида: выплаты"}
              scope={scope}
              direction={direction}
            />
          }
        />
      ) : null}
      {reconciliationQuery.data ? <MismatchAlert summary={reconciliationQuery.data} /> : null}
      {scope === "trader" && reconciliationQuery.data?.status === "mismatch" && reconciliationQuery.data.runId ? (
        <AcceptMismatchDialog scope={scope} direction={direction} runId={reconciliationQuery.data.runId} />
      ) : null}
      {reconciliationQuery.data?.status !== "mismatch" && reconciliationQuery.data ? (
        <Card>
          <CardContent className="flex flex-wrap items-center justify-between gap-3 p-4">
            <div className="flex items-center gap-3">
              <StatusBadge status={reconciliationQuery.data.status} />
              <span className="text-sm text-muted-foreground">
                Ожидалось {formatMoneyMinor(reconciliationQuery.data.expectedMinor)}, факт{" "}
                {formatMoneyMinor(reconciliationQuery.data.actualMinor)}
              </span>
            </div>
            <span className="text-sm font-medium">Diff: {formatMoneyMinor(reconciliationQuery.data.diffMinor)}</span>
          </CardContent>
        </Card>
      ) : null}
      <DataTable
        columns={columns}
        data={data}
        rowCount={data.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        search={search}
        onSearchChange={setSearch}
        toolbarFilters={
          <Select className="w-44" value={status} onChange={(event) => setStatus(event.target.value)}>
            <option value="all">Все статусы</option>
            <option value="hand_success">Успех</option>
            <option value="corrected">Исправлен</option>
            <option value="mismatch">Расхождение</option>
          </Select>
        }
        isLoading={ordersQuery.isLoading}
        error={ordersQuery.error instanceof Error ? ordersQuery.error.message : null}
      />
    </div>
  );
}

export function TeamleadPeriodsPage() {
  const periodsQuery = useQuery({ queryKey: ["teamlead", "periods"], queryFn: api.periods.list });
  const columns = useMemo<ColumnDef<AccountingPeriod>[]>(
    () => [
      { accessorKey: "title", header: "Период" },
      { accessorKey: "dateRange", header: "Даты" },
      { accessorKey: "inboundStatus", header: "Входы", cell: ({ row }) => <StatusBadge status={row.original.inboundStatus} /> },
      { accessorKey: "outboundStatus", header: "Выплаты", cell: ({ row }) => <StatusBadge status={row.original.outboundStatus} /> },
      { accessorKey: "status", header: "Статус", cell: ({ row }) => <StatusBadge status={row.original.status} /> },
    ],
    [],
  );
  return (
    <div className="space-y-6">
      <PageHeader title="Периоды" description="Учетные периоды и итоговая сверка." />
      <DataTable
        columns={columns}
        data={periodsQuery.data ?? []}
        rowCount={periodsQuery.data?.length ?? 0}
        pagination={{ pageIndex: 0, pageSize: 8 }}
        onPaginationChange={() => undefined}
        isLoading={periodsQuery.isLoading}
      />
    </div>
  );
}

export function TeamleadAuditPage() {
  const auditQuery = useQuery({ queryKey: queryKeys.teamlead.audit(), queryFn: api.audit.list });
  const columns = useMemo<ColumnDef<AuditLogEntry>[]>(
    () => [
      { accessorKey: "createdAt", header: "Время", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
      { accessorKey: "actorLogin", header: "Автор" },
      { accessorKey: "action", header: "Действие" },
      { accessorKey: "entityType", header: "Сущность" },
      {
        id: "details",
        header: "",
        cell: ({ row }) => (
          <Dialog>
            <DialogTrigger asChild>
              <Button type="button" variant="outline" size="sm">
                <Eye className="h-4 w-4" />
                Детали
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle className="text-base font-semibold">Аудит #{row.original.id}</DialogTitle>
                <DialogDescription>Чувствительные значения отображаются только если backend уже вернул их замаскированными.</DialogDescription>
              </DialogHeader>
              <pre className="overflow-auto rounded-md bg-slate-950 p-3 text-xs text-slate-50">
                {JSON.stringify(row.original.maskedPayload, null, 2)}
              </pre>
            </DialogContent>
          </Dialog>
        ),
      },
    ],
    [],
  );
  return (
    <div className="space-y-6">
      <PageHeader title="Аудит" description="Журнал изменений по команде." />
      <DataTable
        columns={columns}
        data={auditQuery.data ?? []}
        rowCount={auditQuery.data?.length ?? 0}
        pagination={{ pageIndex: 0, pageSize: 8 }}
        onPaginationChange={() => undefined}
        isLoading={auditQuery.isLoading}
      />
    </div>
  );
}

function TraderFormDialog({
  open,
  onOpenChange,
  trader,
  isSaving,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  trader: Trader | null;
  isSaving: boolean;
  onSubmit: (values: TraderForm) => void;
}) {
  const form = useForm<TraderForm>({
    resolver: zodResolver(traderSchema),
    values: trader
      ? {
          id: trader.id,
          login: trader.login,
          password: "",
          externalWorkerName: trader.externalWorkerName,
          salaryPercent: bpsToPercent(trader.salaryRateBps),
          status: trader.status,
        }
      : { login: "", password: "", externalWorkerName: "", salaryPercent: 0.5, status: "active" },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="left-auto right-0 top-0 h-screen w-[min(520px,100vw)] translate-x-0 translate-y-0 rounded-none">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{trader ? "Редактировать трейдера" : "Добавить трейдера"}</DialogTitle>
          <DialogDescription>{trader ? "Пароль на форме редактирования не показывается." : "Пароль нужен только при создании."}</DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
          <FormField label="Логин" error={form.formState.errors.login?.message}>
            <Input {...form.register("login")} />
          </FormField>
          {!trader ? (
            <FormField label="Пароль" error={form.formState.errors.password?.message}>
              <Input type="password" {...form.register("password")} />
            </FormField>
          ) : null}
          <FormField label="External worker name" error={form.formState.errors.externalWorkerName?.message}>
            <Input {...form.register("externalWorkerName")} />
          </FormField>
          <FormField label="Ставка, %" error={form.formState.errors.salaryPercent?.message}>
            <Input type="number" step="0.01" {...form.register("salaryPercent")} />
          </FormField>
          <FormField label="Статус">
            <Select {...form.register("status")}>
              <option value="active">Активен</option>
              <option value="disabled">Отключен</option>
            </Select>
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" disabled={isSaving}>
              Сохранить
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function RequisiteFormDialog({
  open,
  onOpenChange,
  requisite,
  traders,
  isSaving,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  requisite: Requisite | null;
  traders: Trader[];
  isSaving: boolean;
  onSubmit: (values: RequisiteForm) => void;
}) {
  const form = useForm<RequisiteForm>({
    resolver: zodResolver(requisiteSchema),
    values: requisite
      ? {
          id: requisite.id,
          phone: requisite.phone,
          methodType: requisite.methodType,
          proxy: requisite.proxy,
          assignedTraderId: String(requisite.assignedTraderId ?? "unassigned"),
          status: requisite.status,
        }
      : { phone: "", methodType: "SBP", proxy: "", assignedTraderId: "unassigned", status: "active" },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="left-auto right-0 top-0 h-screen w-[min(560px,100vw)] translate-x-0 translate-y-0 rounded-none">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{requisite ? "Редактировать реквизит" : "Добавить реквизит"}</DialogTitle>
          <DialogDescription>Card number и holder name относятся к смене трейдера, не к базовому реквизиту.</DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
          <FormField label="Телефон" error={form.formState.errors.phone?.message}>
            <Input {...form.register("phone")} />
          </FormField>
          <FormField label="Метод">
            <Select {...form.register("methodType")}>
              <option value="SBP">SBP</option>
              <option value="C2C">C2C</option>
            </Select>
          </FormField>
          <FormField label="Proxy" error={form.formState.errors.proxy?.message}>
            <Input {...form.register("proxy")} />
          </FormField>
          <FormField label="Назначенный трейдер">
            <Select {...form.register("assignedTraderId")}>
              <option value="unassigned">Не назначен</option>
              {traders.map((trader) => (
                <option key={trader.id} value={trader.id}>
                  {trader.login}
                </option>
              ))}
            </Select>
          </FormField>
          <FormField label="Статус">
            <Select {...form.register("status")}>
              <option value="active">Активен</option>
              <option value="archived">Архив</option>
            </Select>
          </FormField>
          {requisite ? <AssignmentHistoryDialog requisiteId={requisite.id} /> : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" disabled={isSaving}>
              Сохранить
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function AssignmentHistoryDialog({ requisiteId }: { requisiteId: number }) {
  const historyQuery = useQuery({
    queryKey: ["teamlead", "requisites", requisiteId, "history"],
    queryFn: () => api.requisites.history(requisiteId),
    enabled: false,
  });
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button type="button" variant="outline" size="sm" onClick={() => void historyQuery.refetch()}>
          <History className="h-4 w-4" />
          История назначений
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">История назначений</DialogTitle>
        </DialogHeader>
        <div className="space-y-2">
          {(historyQuery.data ?? []).map((item) => (
            <Card key={item.id}>
              <CardContent className="space-y-1 p-3 text-sm">
                <DateTimeCell value={item.changedAt} />
                <div>
                  {item.oldTrader ?? "—"} → {item.newTrader ?? "—"}
                </div>
                <div className="text-muted-foreground">{item.comment}</div>
              </CardContent>
            </Card>
          ))}
          {!historyQuery.isLoading && historyQuery.data?.length === 0 ? <EmptyState title="Истории пока нет" /> : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function AssignmentHistoryViewer({ requisite, onClose }: { requisite: Requisite | null; onClose: () => void }) {
  const historyQuery = useQuery({
    queryKey: ["teamlead", "requisites", requisite?.id, "history"],
    queryFn: () => api.requisites.history(requisite?.id ?? 0),
    enabled: Boolean(requisite),
  });

  return (
    <Dialog open={Boolean(requisite)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">История назначений</DialogTitle>
          <DialogDescription>{requisite?.phone}</DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          {(historyQuery.data ?? []).map((item) => (
            <Card key={item.id}>
              <CardContent className="space-y-1 p-3 text-sm">
                <DateTimeCell value={item.changedAt} />
                <div>
                  {item.oldTrader ?? "—"} → {item.newTrader ?? "—"}
                </div>
                <div className="text-muted-foreground">{item.comment}</div>
              </CardContent>
            </Card>
          ))}
          {!historyQuery.isLoading && historyQuery.data?.length === 0 ? <EmptyState title="Истории пока нет" /> : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function GeneratedPasswordDialog({ password, onClose }: { password: string | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(password)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Новый пароль</DialogTitle>
          <DialogDescription>Пароль показывается один раз. После закрытия он не будет доступен в интерфейсе.</DialogDescription>
        </DialogHeader>
        <div className="rounded-md border border-border bg-slate-50 p-3 font-mono text-sm">{password}</div>
        <Button type="button" onClick={() => void navigator.clipboard?.writeText(password ?? "")}>
          Скопировать
        </Button>
      </DialogContent>
    </Dialog>
  );
}

function ConfirmActionDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmText,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  confirmText: string;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Отмена
          </Button>
          <Button type="button" variant="destructive" onClick={onConfirm}>
            {confirmText}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

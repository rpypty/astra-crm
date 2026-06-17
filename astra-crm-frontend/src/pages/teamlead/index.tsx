import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { CalendarDays, Eye, History, Pencil, Plus, X } from "lucide-react";
import { useDeferredValue, useEffect, useMemo, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { Bar, CartesianGrid, Cell, ComposedChart, Line, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { z } from "zod";
import { DateTimeCell } from "@/components/crm/date-time-cell";
import { EmptyState } from "@/components/crm/empty-state";
import { FormField } from "@/components/crm/form-field";
import { AcceptMismatchDialog, ImportCsvDialog, MismatchAlert } from "@/components/crm/import-components";
import { MoneyCell } from "@/components/crm/money-cell";
import { OrderDashboard } from "@/components/crm/order-dashboard";
import { PageHeader } from "@/components/crm/page-header";
import { PeriodFilterBar } from "@/components/crm/period-filter-bar";
import { RequisiteCell } from "@/components/crm/requisite-cell";
import { StatusBadge } from "@/components/crm/status-badge";
import { UserCell } from "@/components/crm/user-cell";
import { DataTable } from "@/components/table/data-table";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type {
  AccountingPeriod,
  AuditLogEntry,
  Bank,
  Order,
  OrderDirection,
  Requisite,
  RequisiteAssignmentEvent,
  RequisiteAssignmentWorkRow,
  ShiftReport,
  ShiftRequisite,
  Trader,
} from "@/lib/domain";
import { api } from "@/lib/api";
import { filterOrdersBySearch } from "@/lib/order-filters";
import type { PeriodFilter } from "@/lib/period-filter";
import { usePersistentPeriodFilter } from "@/lib/period-filter";
import { queryKeys } from "@/lib/query-keys";
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
} from "@/lib/utils";

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

const TEAMLEAD_PERIOD_FILTER_STORAGE_KEY = "astra-crm:teamlead-period-filter";

export function TeamleadTradersPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const [status, setStatus] = useState("all");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [editingTrader, setEditingTrader] = useState<Trader | null>(null);
  const [archiveTrader, setArchiveTrader] = useState<Trader | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [generatedPassword, setGeneratedPassword] = useState<string | null>(null);

  const tradersQuery = useQuery({
    queryKey: queryKeys.teamlead.traders({ search: deferredSearch, status }),
    queryFn: () => api.traders.list({ search: deferredSearch, status }),
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
        onRowClick={(row) => {
          setEditingTrader(row);
          setFormOpen(true);
        }}
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
  const [activeTab, setActiveTab] = useState<TeamleadRequisiteTab>("all");
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const [bankCode, setBankCode] = useState("all");
  const [status, setStatus] = useState("all");
  const [traderId, setTraderId] = useState("all");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [activityPagination, setActivityPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [planningPagination, setPlanningPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [editingRequisite, setEditingRequisite] = useState<Requisite | null>(null);
  const [commentRequisite, setCommentRequisite] = useState<Requisite | null>(null);
  const [archiveRequisite, setArchiveRequisite] = useState<Requisite | null>(null);
  const [historyRequisite, setHistoryRequisite] = useState<Requisite | null>(null);
  const [editingPlan, setEditingPlan] = useState<RequisiteAssignmentWorkRow | null>(null);
  const [eventsPlan, setEventsPlan] = useState<RequisiteAssignmentWorkRow | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [planOpen, setPlanOpen] = useState(false);

  const requisitesQuery = useQuery({
    queryKey: queryKeys.teamlead.requisites({ search: deferredSearch, bankCode, status, traderId }),
    queryFn: () => api.requisites.list({ search: deferredSearch, bankCode, status, traderId }),
  });
  const activityQuery = useQuery({
    queryKey: queryKeys.teamlead.requisiteActivity,
    queryFn: api.requisites.activity,
    enabled: activeTab === "activity",
  });
  const plansQuery = useQuery({
    queryKey: queryKeys.teamlead.requisitePlans,
    queryFn: api.requisites.plans,
    enabled: activeTab === "planning",
  });
  const banksQuery = useQuery({
    queryKey: queryKeys.banks,
    queryFn: api.banks.list,
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
  const savePlanMutation = useMutation({
    mutationFn: (values: PlanForm) => {
      const payload = {
        requisiteId: Number(values.requisiteId),
        traderId: Number(values.traderId),
        assignedForDate: values.assignedForDate,
        targetTurnoverMinor: parseMoneyToMinor(values.targetTurnover),
        comment: values.comment,
      };
      return values.assignmentId
        ? api.requisites.updatePlan({ assignmentId: values.assignmentId, ...payload })
        : api.requisites.createPlan(payload);
    },
    onSuccess: async () => {
      await invalidateRequisiteWorkQueries(queryClient);
      setPlanOpen(false);
      setEditingPlan(null);
    },
  });
  const cancelPlanMutation = useMutation({
    mutationFn: api.requisites.cancelPlan,
    onSuccess: () => invalidateRequisiteWorkQueries(queryClient),
  });
  const archiveMutation = useMutation({
    mutationFn: api.requisites.archive,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["teamlead", "requisites"] }),
  });
  const commentMutation = useMutation({
    mutationFn: api.requisites.save,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["teamlead", "requisites"] });
      setCommentRequisite(null);
    },
  });

  const columns = useMemo<ColumnDef<Requisite>[]>(
    () => [
      {
        accessorKey: "phone",
        header: "Реквизит",
        cell: ({ row }) => (
          <RequisiteCell phone={row.original.phone} method={row.original.bankName} proxy={row.original.proxy} />
        ),
      },
      {
        accessorKey: "bankName",
        header: "Банк",
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
        accessorKey: "employeeComment",
        header: "Комментарий",
        cell: ({ row }) => (
          <div className="flex min-w-0 items-center gap-2">
            <span className="max-w-[220px] truncate text-sm text-muted-foreground">
              {row.original.employeeComment || "—"}
            </span>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0"
              onClick={(event) => {
                event.stopPropagation();
                setCommentRequisite(row.original);
              }}
              title="Редактировать комментарий"
            >
              <Pencil className="h-4 w-4" />
            </Button>
          </div>
        ),
      },
      {
        accessorKey: "holderName",
        header: "ФИО",
        cell: ({ row }) => row.original.holderName || <span className="text-muted-foreground">—</span>,
      },
      {
        accessorKey: "cardNumber",
        header: "Карта",
        cell: ({ row }) => row.original.cardNumber || <span className="text-muted-foreground">—</span>,
      },
      {
        accessorKey: "status",
        header: "Состояние",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        accessorKey: "assignmentStatus",
        header: "Работа",
        cell: ({ row }) =>
          row.original.assignmentStatus ? (
            <StatusBadge status={row.original.assignmentStatus} />
          ) : (
            <span className="text-muted-foreground">—</span>
          ),
      },
      {
        accessorKey: "updatedAt",
        header: "Обновлен",
        cell: ({ row }) => <DateTimeCell value={row.original.updatedAt} />,
      },
    ],
    [],
  );
  const activityColumns = useMemo<ColumnDef<RequisiteAssignmentWorkRow>[]>(
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
        accessorKey: "traderLogin",
        header: "Трейдер",
        cell: ({ row }) => <UserCell login={row.original.traderLogin} />,
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
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
      { accessorKey: "holderName", header: "ФИО", cell: ({ row }) => row.original.holderName || "—" },
      { accessorKey: "cardNumber", header: "Карта", cell: ({ row }) => row.original.cardNumber || "—" },
    ],
    [],
  );
  const planningColumns = useMemo<ColumnDef<RequisiteAssignmentWorkRow>[]>(
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
        accessorKey: "traderLogin",
        header: "Трейдер",
        cell: ({ row }) => <UserCell login={row.original.traderLogin} />,
      },
      {
        accessorKey: "targetTurnoverMinor",
        header: () => <div className="text-right">Целевой оборот</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.targetTurnoverMinor} />,
      },
      {
        accessorKey: "inboundTurnoverMinor",
        header: () => <div className="text-right">Факт</div>,
        cell: ({ row }) => <MoneyCell valueMinor={row.original.inboundTurnoverMinor} />,
      },
      {
        id: "progress",
        header: "Прогресс",
        cell: ({ row }) => assignmentProgressLabel(row.original),
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        accessorKey: "comment",
        header: "Комментарий",
        cell: ({ row }) => (
          <span className="block max-w-[220px] truncate text-sm text-muted-foreground" title={row.original.comment ?? ""}>
            {row.original.comment || "—"}
          </span>
        ),
      },
      {
        id: "planActions",
        header: "",
        cell: ({ row }) => (
          <div className="flex justify-end gap-1" onClick={(event) => event.stopPropagation()}>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              title="История изменений"
              onClick={() => setEventsPlan(row.original)}
            >
              <History className="h-4 w-4" />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              title="Редактировать план"
              onClick={() => {
                setEditingPlan(row.original);
                setPlanOpen(true);
              }}
            >
              <Pencil className="h-4 w-4" />
            </Button>
            {isPlanCancellable(row.original) ? (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-8 w-8 text-red-600 hover:text-red-700"
                title="Отменить план"
                onClick={() => cancelPlanMutation.mutate(row.original.assignmentId)}
              >
                <X className="h-4 w-4" />
              </Button>
            ) : null}
          </div>
        ),
      },
    ],
    [cancelPlanMutation],
  );
  const data = requisitesQuery.data ?? [];
  const pageAction =
    activeTab === "planning" ? (
      <Button
        type="button"
        onClick={() => {
          setEditingPlan(null);
          setPlanOpen(true);
        }}
      >
        <CalendarDays className="h-4 w-4" />
        Запланировать
      </Button>
    ) : activeTab === "all" ? (
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
    ) : null;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Реквизиты"
        description="База реквизитов отдельно от рабочих назначений, планов и фактической активности трейдеров."
        actions={pageAction}
      />
      <TeamleadRequisiteTabs value={activeTab} onChange={setActiveTab} />
      {activeTab === "all" ? (
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
              <Select className="w-44" value={bankCode} onChange={(event) => setBankCode(event.target.value)}>
                <option value="all">Все банки</option>
                {(banksQuery.data ?? []).map((bank) => (
                  <option key={bank.code} value={bank.code}>
                    {bank.name}
                  </option>
                ))}
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
          onRowClick={(row) => {
            setEditingRequisite(row);
            setFormOpen(true);
          }}
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
      ) : null}
      {activeTab === "activity" ? (
        <DataTable
          columns={activityColumns}
          data={activityQuery.data ?? []}
          rowCount={activityQuery.data?.length ?? 0}
          pagination={activityPagination}
          onPaginationChange={setActivityPagination}
          isLoading={activityQuery.isLoading}
          error={activityQuery.error instanceof Error ? activityQuery.error.message : null}
          emptyTitle="Активности пока нет"
          emptyDescription="Здесь появятся фактические взятия в работу, закрытия, блоки и зафиксированные обороты."
        />
      ) : null}
      {activeTab === "planning" ? (
        <DataTable
          columns={planningColumns}
          data={plansQuery.data ?? []}
          rowCount={plansQuery.data?.length ?? 0}
          pagination={planningPagination}
          onPaginationChange={setPlanningPagination}
          isLoading={plansQuery.isLoading}
          error={plansQuery.error instanceof Error ? plansQuery.error.message : null}
          emptyTitle="Планов пока нет"
          emptyDescription="Запланируйте дату, трейдера, реквизит и целевой оборот."
          onRowClick={(row) => {
            setEditingPlan(row);
            setPlanOpen(true);
          }}
        />
      ) : null}
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
        banks={banksQuery.data ?? []}
        isSaving={saveMutation.isPending}
        error={saveMutation.error instanceof Error ? saveMutation.error.message : null}
        onSubmit={(values) =>
          saveMutation.mutate({
            id: values.id,
            phone: normalizeRussianPhone(values.phone),
            bankCode: values.bankCode,
            proxy: values.proxy,
            employeeComment: values.employeeComment,
            assignedTraderId: values.assignedTraderId === "unassigned" ? undefined : Number(values.assignedTraderId),
            currentAssignedTraderId: editingRequisite?.assignedTraderId,
            status: values.status,
          })
        }
      />
      <RequisiteCommentDialog
        requisite={commentRequisite}
        isSaving={commentMutation.isPending}
        error={commentMutation.error instanceof Error ? commentMutation.error.message : null}
        onClose={() => setCommentRequisite(null)}
        onSubmit={(employeeComment) => {
          if (!commentRequisite) return;
          commentMutation.mutate({
            id: commentRequisite.id,
            phone: commentRequisite.phone,
            bankCode: commentRequisite.bankCode,
            proxy: commentRequisite.proxy,
            employeeComment,
            assignedTraderId: commentRequisite.assignedTraderId,
            currentAssignedTraderId: commentRequisite.assignedTraderId,
            status: commentRequisite.status,
          });
        }}
      />
      <AssignmentHistoryViewer requisite={historyRequisite} onClose={() => setHistoryRequisite(null)} />
      <PlanRequisiteDialog
        open={planOpen}
        onOpenChange={(open) => {
          setPlanOpen(open);
          if (!open) setEditingPlan(null);
        }}
        plan={editingPlan}
        requisites={(requisitesQuery.data ?? []).filter((requisite) => requisite.status === "active")}
        traders={tradersQuery.data ?? []}
        isSaving={savePlanMutation.isPending}
        error={savePlanMutation.error instanceof Error ? savePlanMutation.error.message : null}
        onSubmit={(values) => savePlanMutation.mutate(values)}
      />
      <PlanEventsDialog plan={eventsPlan} onClose={() => setEventsPlan(null)} />
    </div>
  );
}

export function TeamleadDashboardPage() {
  const [periodFilter, setPeriodFilter] = usePersistentPeriodFilter(TEAMLEAD_PERIOD_FILTER_STORAGE_KEY);
  const inboundDashboardQuery = useQuery({
    queryKey: queryKeys.teamlead.dashboard("inbound", periodFilter),
    queryFn: () => api.orders.dashboard("teamlead", "inbound", periodFilter),
  });
  const outboundDashboardQuery = useQuery({
    queryKey: queryKeys.teamlead.dashboard("outbound", periodFilter),
    queryFn: () => api.orders.dashboard("teamlead", "outbound", periodFilter),
  });

  return (
    <div className="space-y-6">
      <PageHeader title="Аналитика" description="Сводка по выбранному периоду, инвойсам и выплатам." />
      <PeriodFilterBar value={periodFilter} onChange={setPeriodFilter} />
      <div className="grid gap-4 xl:grid-cols-2">
        <OrderDashboard
          title="Инвойсы"
          dashboard={inboundDashboardQuery.data}
          direction="inbound"
          isLoading={inboundDashboardQuery.isLoading}
          error={inboundDashboardQuery.error instanceof Error ? inboundDashboardQuery.error : null}
        />
        <OrderDashboard
          title="Выплаты"
          dashboard={outboundDashboardQuery.data}
          direction="outbound"
          isLoading={outboundDashboardQuery.isLoading}
          error={outboundDashboardQuery.error instanceof Error ? outboundDashboardQuery.error : null}
        />
      </div>
      <ProfitLossPlaceholder periodFilter={periodFilter} />
    </div>
  );
}

function ProfitLossPlaceholder({ periodFilter }: { periodFilter: PeriodFilter }) {
  const data = useMemo(() => buildProfitLossPlaceholder(periodFilter), [periodFilter]);
  const totalMinor = data.length > 0 ? data[data.length - 1].cumulativeMinor : 0;
  const isProfit = totalMinor >= 0;
  const profitableDays = data.filter((item) => item.amountMinor > 0).length;
  const lossDays = data.filter((item) => item.amountMinor < 0).length;
  const bestDay = data.reduce((max, item) => Math.max(max, item.amountMinor), 0);
  const worstDay = data.reduce((min, item) => Math.min(min, item.amountMinor), 0);

  return (
    <Card className="overflow-hidden">
      <CardContent className="space-y-5 p-4">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <div className="text-sm font-medium text-muted-foreground">P&L</div>
              <span className="rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700">заглушка</span>
            </div>
            <div className={isProfit ? "mt-2 text-3xl font-semibold text-emerald-700" : "mt-2 text-3xl font-semibold text-red-700"}>
              {formatMoneyMinor(totalMinor)}
            </div>
            <div className="mt-1 text-sm text-muted-foreground">Прибыль/убыток за выбранный период</div>
          </div>
          <div className="grid w-full gap-2 sm:w-auto sm:grid-cols-3">
            <ProfitLossMetric label="Прибыльных дней" value={String(profitableDays)} />
            <ProfitLossMetric label="Лучший день" value={formatMoneyMinor(bestDay)} tone="positive" />
            <ProfitLossMetric label="Худший день" value={formatMoneyMinor(worstDay)} tone="negative" />
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-2">
            <span className="h-2.5 w-2.5 rounded-sm bg-emerald-600" />
            Дневная прибыль
          </span>
          <span className="inline-flex items-center gap-2">
            <span className="h-2.5 w-2.5 rounded-sm bg-red-500" />
            Дневной убыток
          </span>
          <span className="inline-flex items-center gap-2">
            <span className="h-0.5 w-5 rounded-full bg-amber-500" />
            Накопленный P&L
          </span>
          <span>Убыточных дней: {lossDays}</span>
        </div>
        <div className="h-80 min-w-0 rounded-md border bg-white p-3">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart data={data} margin={{ top: 8, right: 14, left: 0, bottom: 0 }}>
              <CartesianGrid stroke="#e5e7eb" strokeDasharray="3 3" vertical={false} />
              <ReferenceLine y={0} stroke="#94a3b8" strokeWidth={1.5} />
              <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={24} tick={{ fill: "#64748b", fontSize: 12 }} />
              <YAxis
                tickLine={false}
                axisLine={false}
                width={88}
                tick={{ fill: "#64748b", fontSize: 12 }}
                tickFormatter={(value) => formatMoneyAxis(Number(value))}
              />
              <Tooltip
                cursor={{ fill: "rgba(148, 163, 184, 0.14)" }}
                contentStyle={{ borderRadius: 8, borderColor: "#dbe3ee", boxShadow: "0 10px 30px rgba(15, 23, 42, 0.12)" }}
                formatter={(value, name) => [
                  formatMoneyMinor(Number(value)),
                  name === "amountMinor" ? "Дневной P&L" : "Накопленный P&L",
                ]}
                labelFormatter={(label) => `Дата: ${label}`}
              />
              <Bar dataKey="amountMinor" barSize={18} radius={[3, 3, 0, 0]}>
                {data.map((item) => (
                  <Cell key={item.date} fill={item.amountMinor >= 0 ? "#059669" : "#dc2626"} />
                ))}
              </Bar>
              <Line
                type="monotone"
                dataKey="cumulativeMinor"
                stroke="#f59e0b"
                strokeWidth={2.5}
                dot={false}
                activeDot={{ r: 4, stroke: "#f59e0b", strokeWidth: 2, fill: "#ffffff" }}
              />
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}

function ProfitLossMetric({
  label,
  value,
  tone = "neutral",
}: {
  label: string;
  value: string;
  tone?: "neutral" | "positive" | "negative";
}) {
  const valueClass =
    tone === "positive" ? "text-emerald-700" : tone === "negative" ? "text-red-700" : "text-foreground";

  return (
    <div className="rounded-md border bg-muted/20 px-3 py-2">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className={`mt-1 whitespace-nowrap text-sm font-semibold tabular-nums ${valueClass}`}>{value}</div>
    </div>
  );
}

function buildProfitLossPlaceholder(periodFilter: PeriodFilter) {
  const today = startOfLocalDay(new Date());
  const end = parseISODate(periodFilter.dateTo) ?? today;
  const start = parseISODate(periodFilter.dateFrom) ?? addDays(end, -13);
  const normalizedStart = start > end ? end : start;
  const daysCount = Math.min(diffDays(normalizedStart, end) + 1, 45);
  const firstDay = addDays(end, -(daysCount - 1));
  let cumulativeMinor = 0;

  return Array.from({ length: daysCount }, (_, index) => {
    const date = addDays(firstDay, index);
    const direction = index % 6 === 1 || index % 6 === 4 ? -1 : 1;
    const baseMinor = (180_000 + ((index * 37) % 420_000)) * 100;
    const amountMinor = direction * baseMinor;
    cumulativeMinor += amountMinor;

    return {
      date: toISODate(date),
      label: new Intl.DateTimeFormat("ru-RU", { day: "2-digit", month: "2-digit" }).format(date),
      amountMinor,
      cumulativeMinor,
    };
  });
}

function formatMoneyAxis(value: number) {
  const absolute = Math.abs(value);
  const sign = value < 0 ? "-" : "";

  if (absolute >= 100_000_000) {
    return `${sign}${(absolute / 100_000_000).toFixed(1).replace(".", ",")} млн ₽`;
  }

  if (absolute >= 100_000) {
    return `${sign}${Math.round(absolute / 100_000)} тыс ₽`;
  }

  return formatMoneyMinor(value).replace(",00", "");
}

function parseISODate(value?: string) {
  if (!value) return undefined;
  const [year, month, day] = value.split("-").map(Number);
  if (!year || !month || !day) return undefined;
  const date = new Date(year, month - 1, day);
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) return undefined;
  return startOfLocalDay(date);
}

function startOfLocalDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function addDays(date: Date, days: number) {
  const next = new Date(date);
  next.setDate(next.getDate() + days);
  return next;
}

function diffDays(start: Date, end: Date) {
  return Math.max(0, Math.round((end.getTime() - start.getTime()) / 86_400_000));
}

function toISODate(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}

export function TeamleadTransactionsPage({ initialDirection = "inbound" }: { initialDirection?: OrderDirection }) {
  const [periodFilter, setPeriodFilter] = usePersistentPeriodFilter(TEAMLEAD_PERIOD_FILTER_STORAGE_KEY);
  const [activeOrdersDirection, setActiveOrdersDirection] = useState<OrderDirection>(initialDirection);
  const [, startOrdersTabTransition] = useTransition();
  const activeLabel = activeOrdersDirection === "inbound" ? "Инвойсы" : "Выплаты";

  useEffect(() => {
    setActiveOrdersDirection(initialDirection);
  }, [initialDirection]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Транзакции"
        description="Ордера по инвойсам и выплатам в одном разделе."
        actions={
          <ImportCsvDialog
            scopeLabel={`Сверка тимлида: ${activeLabel.toLowerCase()}`}
            scope="teamlead"
            direction={activeOrdersDirection}
          />
        }
      />
      <PeriodFilterBar value={periodFilter} onChange={setPeriodFilter} />
      <section className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-lg font-semibold">{activeLabel}</h2>
          <div className="inline-flex rounded-md border border-border bg-card p-1">
            <Button
              type="button"
              variant={activeOrdersDirection === "inbound" ? "default" : "ghost"}
              size="sm"
              onClick={() => startOrdersTabTransition(() => setActiveOrdersDirection("inbound"))}
            >
              Инвойсы
            </Button>
            <Button
              type="button"
              variant={activeOrdersDirection === "outbound" ? "default" : "ghost"}
              size="sm"
              onClick={() => startOrdersTabTransition(() => setActiveOrdersDirection("outbound"))}
            >
              Выплаты
            </Button>
          </div>
        </div>
        <OrdersPage
          direction={activeOrdersDirection}
          scope="teamlead"
          embedded
          periodFilter={periodFilter}
          showReconciliation={false}
        />
      </section>
    </div>
  );
}

export function OrdersPage({
  direction,
  scope,
  embedded,
  periodFilter: externalPeriodFilter,
  showReconciliation = true,
}: {
  direction: "inbound" | "outbound";
  scope: "teamlead" | "trader";
  embedded?: boolean;
  periodFilter?: PeriodFilter;
  showReconciliation?: boolean;
}) {
  const [storedPeriodFilter, setStoredPeriodFilter] = usePersistentPeriodFilter(
    scope === "teamlead" ? TEAMLEAD_PERIOD_FILTER_STORAGE_KEY : "astra-crm:trader-period-filter",
  );
  const periodFilter = externalPeriodFilter ?? storedPeriodFilter;
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const [status, setStatus] = useState("all");
  const [selectedTraderIds, setSelectedTraderIds] = useState<number[]>([]);
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsOrder, setDetailsOrder] = useState<Order | null>(null);
  const dashboardFilters = useMemo(
    () => ({ ...periodFilter, traderIds: selectedTraderIds }),
    [periodFilter, selectedTraderIds],
  );
  const orderServerFilters = useMemo(
    () => ({ status, ...periodFilter, traderIds: selectedTraderIds }),
    [periodFilter, selectedTraderIds, status],
  );
  const tradersQuery = useQuery({
    queryKey: queryKeys.teamlead.traders({ status: "active" }),
    queryFn: () => api.traders.list({ status: "active" }),
    enabled: scope === "teamlead",
  });
  const dashboardQuery = useQuery({
    queryKey:
      scope === "teamlead"
        ? queryKeys.teamlead.dashboard(direction, dashboardFilters)
        : queryKeys.trader.dashboard(direction, dashboardFilters),
    queryFn: () => api.orders.dashboard(scope, direction, dashboardFilters),
    enabled: !embedded,
  });
  const ordersQuery = useQuery({
    queryKey:
      scope === "teamlead"
        ? queryKeys.teamlead.orders(direction, orderServerFilters)
        : queryKeys.trader.orders(direction, orderServerFilters),
    queryFn: () => api.orders.list(scope, direction, orderServerFilters),
  });
  const reconciliationQuery = useQuery({
    queryKey: [scope, direction, "reconciliation"],
    queryFn: () => api.orders.reconciliation(scope, direction),
    enabled: showReconciliation,
  });
  useEffect(() => {
    setPagination((current) => (current.pageIndex === 0 ? current : { ...current, pageIndex: 0 }));
  }, [deferredSearch, direction, orderServerFilters]);
  const columns = useMemo<ColumnDef<Order>[]>(
    () => [
      { accessorKey: "createdAt", header: "Время", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
      { accessorKey: "trader", header: "Трейдер", cell: ({ row }) => <UserCell login={row.original.trader} secondary={row.original.workerName} /> },
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
  const data = useMemo(
    () => filterOrdersBySearch(ordersQuery.data ?? [], deferredSearch),
    [deferredSearch, ordersQuery.data],
  );
  const title = direction === "inbound" ? "Инвойсы" : "Выплаты";

  return (
    <div className="space-y-6">
      {!embedded ? (
        <PageHeader
          title={title}
          description="Ордера, импорт CSV и состояние сверки."
          actions={
            <ImportCsvDialog
              scopeLabel={direction === "inbound" ? "Сверка тимлида: инвойсы" : "Сверка тимлида: выплаты"}
              scope={scope}
              direction={direction}
            />
          }
        />
      ) : null}
      {!embedded ? <PeriodFilterBar value={periodFilter} onChange={setStoredPeriodFilter} /> : null}
      {!embedded ? (
        <OrderDashboard
          dashboard={dashboardQuery.data}
          direction={direction}
          isLoading={dashboardQuery.isLoading}
          error={dashboardQuery.error instanceof Error ? dashboardQuery.error : null}
        />
      ) : null}
      {showReconciliation && reconciliationQuery.data ? <MismatchAlert summary={reconciliationQuery.data} /> : null}
      {showReconciliation && scope === "trader" && reconciliationQuery.data?.status === "mismatch" && reconciliationQuery.data.runId ? (
        <AcceptMismatchDialog scope={scope} direction={direction} runId={reconciliationQuery.data.runId} />
      ) : null}
      {showReconciliation && reconciliationQuery.data?.status !== "mismatch" && reconciliationQuery.data ? (
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
          <div className="flex flex-wrap items-center gap-2">
            <Select className="w-44" value={status} onChange={(event) => setStatus(event.target.value)}>
              <option value="all">Все статусы</option>
              <option value="success">Успех</option>
              <option value="corrected">Исправлен</option>
              <option value="failed">Неуспех</option>
              <option value="cancelled">Отменен</option>
              <option value="unknown">Неизвестно</option>
            </Select>
            {scope === "teamlead" ? (
              <TraderFilterDropdown
                traders={tradersQuery.data ?? []}
                selectedTraderIds={selectedTraderIds}
                isLoading={tradersQuery.isLoading}
                onChange={setSelectedTraderIds}
              />
            ) : null}
          </div>
        }
        isLoading={ordersQuery.isLoading}
        error={ordersQuery.error instanceof Error ? ordersQuery.error.message : null}
        emptyTitle="Ордеров пока нет"
        emptyDescription="После CSV-импорта активного scope здесь появятся ордера."
        onRowClick={setDetailsOrder}
        actions={[{ label: "Детали", onSelect: (row) => setDetailsOrder(row) }]}
      />
      <OrderDetailsDialog order={detailsOrder} onClose={() => setDetailsOrder(null)} />
    </div>
  );
}

function TraderFilterDropdown({
  traders,
  selectedTraderIds,
  isLoading,
  onChange,
}: {
  traders: Trader[];
  selectedTraderIds: number[];
  isLoading?: boolean;
  onChange: (traderIds: number[]) => void;
}) {
  const [search, setSearch] = useState("");
  const selected = useMemo(() => new Set(selectedTraderIds), [selectedTraderIds]);
  const filteredTraders = useMemo(() => {
    const normalizedSearch = search.trim().toLowerCase();
    if (!normalizedSearch) {
      return traders;
    }

    return traders.filter((trader) =>
      [trader.login, trader.externalWorkerName].some((value) => value.toLowerCase().includes(normalizedSearch)),
    );
  }, [search, traders]);
  const label = selectedTraderIds.length === 0 ? "Все трейдеры" : `Трейдеры: ${selectedTraderIds.length}`;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="outline" className="min-w-44 justify-between">
          {label}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="flex max-h-[min(28rem,var(--radix-dropdown-menu-content-available-height))] w-80 max-w-[calc(100vw-2rem)] flex-col overflow-hidden"
      >
        <div className="space-y-2 border-b border-border p-2">
          <DropdownMenuLabel className="px-0 py-0">Трейдеры</DropdownMenuLabel>
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            onKeyDown={(event) => event.stopPropagation()}
            placeholder="Найти трейдера"
          />
          {selectedTraderIds.length > 0 ? (
            <Button type="button" variant="ghost" size="sm" className="w-full justify-start px-2" onClick={() => onChange([])}>
              Сбросить выбор
            </Button>
          ) : null}
        </div>
        <div className="min-h-0 overflow-y-auto p-1">
          {isLoading ? <DropdownMenuItem disabled>Загрузка...</DropdownMenuItem> : null}
          {!isLoading && traders.length === 0 ? <DropdownMenuItem disabled>Нет активных трейдеров</DropdownMenuItem> : null}
          {!isLoading && traders.length > 0 && filteredTraders.length === 0 ? (
            <DropdownMenuItem disabled>Ничего не найдено</DropdownMenuItem>
          ) : null}
          {filteredTraders.map((trader) => (
            <DropdownMenuCheckboxItem
              key={trader.id}
              checked={selected.has(trader.id)}
              className="max-w-full"
              onCheckedChange={(checked) => {
                if (checked) {
                  onChange([...selectedTraderIds, trader.id]);
                  return;
                }
                onChange(selectedTraderIds.filter((traderId) => traderId !== trader.id));
              }}
              onSelect={(event) => event.preventDefault()}
            >
              <span className="min-w-0 truncate" title={trader.login}>
                {trader.login}
              </span>
            </DropdownMenuCheckboxItem>
          ))}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function TeamleadPeriodsPage() {
  const periodsQuery = useQuery({ queryKey: ["teamlead", "periods"], queryFn: api.periods.list });
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsPeriod, setDetailsPeriod] = useState<AccountingPeriod | null>(null);
  const periods = periodsQuery.data ?? [];
  const openCount = periods.filter((period) => period.status === "open").length;
  const mismatchCount = periods.filter(
    (period) => period.inboundStatus === "mismatch" || period.outboundStatus === "mismatch",
  ).length;
  const discrepancyCount = periods.filter((period) => period.status === "closed_with_discrepancy").length;
  const columns = useMemo<ColumnDef<AccountingPeriod>[]>(
    () => [
      { accessorKey: "title", header: "Сверка" },
      { accessorKey: "dateRange", header: "Даты" },
      { accessorKey: "inboundStatus", header: "Инвойсы", cell: ({ row }) => <StatusBadge status={row.original.inboundStatus} /> },
      { accessorKey: "outboundStatus", header: "Выплаты", cell: ({ row }) => <StatusBadge status={row.original.outboundStatus} /> },
      { accessorKey: "status", header: "Статус", cell: ({ row }) => <StatusBadge status={row.original.status} /> },
    ],
    [],
  );
  return (
    <div className="space-y-6">
      <PageHeader title="Сверка" description="Инвойсы и выплаты по учетным периодам." />
      <div className="grid gap-4 md:grid-cols-3">
        <ReadMetricCard label="Открытые периоды" value={String(openCount)} />
        <ReadMetricCard label="Расхождения" value={String(mismatchCount)} warning={mismatchCount > 0} />
        <ReadMetricCard label="Закрыты с расхождением" value={String(discrepancyCount)} warning={discrepancyCount > 0} />
      </div>
      <DataTable
        columns={columns}
        data={periods}
        rowCount={periods.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        isLoading={periodsQuery.isLoading}
        error={periodsQuery.error instanceof Error ? periodsQuery.error.message : null}
        emptyTitle="Сверок пока нет"
        emptyDescription="Записи появятся после создания accounting period."
        onRowClick={setDetailsPeriod}
        actions={[{ label: "Детали", onSelect: (row) => setDetailsPeriod(row) }]}
      />
      <PeriodDetailsDialog period={detailsPeriod} onClose={() => setDetailsPeriod(null)} />
    </div>
  );
}

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
        <ReadMetricCard label="Отчетов" value={String(reports.length)} />
        <ReadMetricCard label="Трейдеров" value={String(traderCount)} />
        <ReadMetricCard label="С расхождением" value={String(discrepancyCount)} warning={discrepancyCount > 0} />
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
              <ReadMetricCard label="Оплаты" value={formatMoneyMinor(inboundTotal)} />
              <ReadMetricCard label="Выплаты" value={formatMoneyMinor(outboundTotal)} />
              <ReadMetricCard label="Остаток" value={formatMoneyMinor(balanceTotal)} />
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

export function TeamleadAuditPage() {
  const auditQuery = useQuery({ queryKey: queryKeys.teamlead.audit(), queryFn: api.audit.list });
  const [search, setSearch] = useState("");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsEntry, setDetailsEntry] = useState<AuditLogEntry | null>(null);
  const auditItems = auditQuery.data ?? [];
  const normalizedSearch = search.trim().toLowerCase();
  const filteredAuditItems = normalizedSearch
    ? auditItems.filter((item) =>
        [item.actorLogin, item.action, item.entityType, item.entityId, item.comment ?? ""].some((value) =>
          value.toLowerCase().includes(normalizedSearch),
        ),
      )
    : auditItems;
  const actorsCount = new Set(auditItems.map((item) => item.actorLogin)).size;
  const columns = useMemo<ColumnDef<AuditLogEntry>[]>(
    () => [
      { accessorKey: "createdAt", header: "Время", cell: ({ row }) => <DateTimeCell value={row.original.createdAt} /> },
      { accessorKey: "actorLogin", header: "Автор" },
      { accessorKey: "action", header: "Действие" },
      { accessorKey: "entityType", header: "Сущность" },
      { accessorKey: "entityId", header: "ID" },
      { accessorKey: "comment", header: "Комментарий", cell: ({ row }) => row.original.comment ?? "—" },
    ],
    [],
  );
  return (
    <div className="space-y-6">
      <PageHeader title="Аудит" description="Журнал изменений по команде." />
      <div className="grid gap-4 md:grid-cols-3">
        <ReadMetricCard label="События" value={String(auditItems.length)} />
        <ReadMetricCard label="Авторы" value={String(actorsCount)} />
        <ReadMetricCard label="Последнее событие" value={auditItems[0] ? formatDateTime(auditItems[0].createdAt) : "—"} />
      </div>
      <DataTable
        columns={columns}
        data={filteredAuditItems}
        rowCount={filteredAuditItems.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        search={search}
        onSearchChange={setSearch}
        isLoading={auditQuery.isLoading}
        error={auditQuery.error instanceof Error ? auditQuery.error.message : null}
        emptyTitle="Событий аудита нет"
        emptyDescription="Мутации в системе будут отображаться здесь."
        onRowClick={setDetailsEntry}
        actions={[{ label: "Детали", onSelect: (row) => setDetailsEntry(row) }]}
      />
      <AuditDetailsDialog entry={detailsEntry} onClose={() => setDetailsEntry(null)} />
    </div>
  );
}

function OrderDetailsDialog({ order, onClose }: { order: Order | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(order)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Ордер {order?.innerId}</DialogTitle>
          <DialogDescription>Данные активного import scope из backend.</DialogDescription>
        </DialogHeader>
        {order ? (
          <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <ReadOnlyField label="Время" value={formatDateTime(order.createdAt)} />
              <ReadOnlyField label="Сумма" value={formatMoneyMinor(order.amountMinor)} />
              <ReadOnlyField label="Трейдер" value={order.trader} />
              <ReadOnlyField label="Worker" value={order.workerName} />
              <ReadOnlyField label="Реквизит" value={order.requisite || "—"} />
              <ReadOnlyField label="Метод/банк" value={order.bankName || order.method || "—"} />
              <ReadOnlyField label="External ID" value={order.externalId} />
              <ReadOnlyField label="Import batch" value={String(order.importBatchId)} />
            </div>
            <div className="rounded-md border border-border p-3">
              <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Статус</div>
              <div className="flex flex-wrap items-center gap-3">
                <StatusBadge status={order.normalizedStatus} />
                <span className="text-sm text-muted-foreground">raw: {order.rawStatus}</span>
              </div>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function PeriodDetailsDialog({ period, onClose }: { period: AccountingPeriod | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(period)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{period?.title}</DialogTitle>
          <DialogDescription>Состояние итоговой сверки по учетному периоду.</DialogDescription>
        </DialogHeader>
        {period ? (
          <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <ReadOnlyField label="Даты" value={period.dateRange} />
              <div className="rounded-md border border-border p-3">
                <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Статус периода</div>
                <StatusBadge status={period.status} />
              </div>
              <div className="rounded-md border border-border p-3">
                <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Инвойсы</div>
                <StatusBadge status={period.inboundStatus} />
              </div>
              <div className="rounded-md border border-border p-3">
                <div className="mb-2 text-xs font-medium uppercase text-muted-foreground">Выплаты</div>
                <StatusBadge status={period.outboundStatus} />
              </div>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function AuditDetailsDialog({ entry, onClose }: { entry: AuditLogEntry | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(entry)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Аудит #{entry?.id}</DialogTitle>
          <DialogDescription>Payload отображается в том виде, в котором backend вернул read model.</DialogDescription>
        </DialogHeader>
        {entry ? (
          <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <ReadOnlyField label="Время" value={formatDateTime(entry.createdAt)} />
              <ReadOnlyField label="Автор" value={entry.actorLogin} />
              <ReadOnlyField label="Действие" value={entry.action} />
              <ReadOnlyField label="Сущность" value={`${entry.entityType} #${entry.entityId}`} />
            </div>
            {entry.comment ? <ReadOnlyField label="Комментарий" value={entry.comment} /> : null}
            <pre className="max-h-96 overflow-auto rounded-md bg-slate-950 p-3 text-xs text-slate-50">
              {JSON.stringify(entry.maskedPayload, null, 2)}
            </pre>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function ReadMetricCard({ label, value, warning }: { label: string; value: string; warning?: boolean }) {
  return (
    <Card className={warning ? "border-amber-200 bg-amber-50" : undefined}>
      <CardContent className="p-4">
        <div className="text-xs font-medium uppercase tracking-normal text-muted-foreground">{label}</div>
        <div className="mt-2 text-2xl font-semibold">{value}</div>
      </CardContent>
    </Card>
  );
}

function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-border p-3">
      <div className="mb-1 text-xs font-medium uppercase text-muted-foreground">{label}</div>
      <div className="break-words text-sm font-medium">{value}</div>
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

function TeamleadRequisiteTabs({
  value,
  onChange,
}: {
  value: TeamleadRequisiteTab;
  onChange: (value: TeamleadRequisiteTab) => void;
}) {
  const tabs: { value: TeamleadRequisiteTab; label: string }[] = [
    { value: "all", label: "Все реквизиты" },
    { value: "activity", label: "Активность" },
    { value: "planning", label: "Планирование" },
  ];

  return (
    <div className="inline-flex rounded-lg border border-border bg-white p-1">
      {tabs.map((tab) => (
        <Button
          key={tab.value}
          type="button"
          variant={value === tab.value ? "default" : "ghost"}
          size="sm"
          onClick={() => onChange(tab.value)}
        >
          {tab.label}
        </Button>
      ))}
    </div>
  );
}

function PlanRequisiteDialog({
  open,
  onOpenChange,
  plan,
  requisites,
  traders,
  isSaving,
  error,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  plan: RequisiteAssignmentWorkRow | null;
  requisites: Requisite[];
  traders: Trader[];
  isSaving: boolean;
  error?: string | null;
  onSubmit: (values: PlanForm) => void;
}) {
  const formValues = useMemo<PlanForm>(
    () =>
      plan
        ? {
            assignmentId: plan.assignmentId,
            requisiteId: String(plan.requisiteId),
            traderId: String(plan.traderId),
            assignedForDate: toDateInputValue(plan.assignedForDate),
            targetTurnover: moneyMinorToInput(plan.targetTurnoverMinor),
            comment: plan.comment ?? "",
          }
        : {
            requisiteId: requisites[0]?.id ? String(requisites[0].id) : "",
            traderId: traders[0]?.id ? String(traders[0].id) : "",
            assignedForDate: tomorrowDateInputValue(),
            targetTurnover: "",
            comment: "",
          },
    [plan, requisites, traders],
  );
  const form = useForm<PlanForm>({
    resolver: zodResolver(planSchema),
    values: formValues,
  });
  const closeWithoutValidation = () => {
    form.reset(formValues);
    form.clearErrors();
    onOpenChange(false);
  };

  useEffect(() => {
    if (open) {
      form.reset(formValues);
      form.clearErrors();
    }
  }, [form, formValues, open]);

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => (nextOpen ? onOpenChange(true) : closeWithoutValidation())}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">
            {plan ? "Редактировать план" : "Запланировать реквизит"}
          </DialogTitle>
          <DialogDescription>Назначение реквизита на дату с целевым оборотом для трейдера.</DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
          <FormField label="Дата" error={form.formState.errors.assignedForDate?.message}>
            <Input type="date" {...form.register("assignedForDate")} />
          </FormField>
          <FormField label="Реквизит" error={form.formState.errors.requisiteId?.message}>
            <Select {...form.register("requisiteId")}>
              <option value="">Выберите реквизит</option>
              {requisites.map((requisite) => (
                <option key={requisite.id} value={requisite.id}>
                  {formatRussianPhone(requisite.phone)} · {requisite.bankName}
                </option>
              ))}
            </Select>
          </FormField>
          <FormField label="Трейдер" error={form.formState.errors.traderId?.message}>
            <Select {...form.register("traderId")}>
              <option value="">Выберите трейдера</option>
              {traders.map((trader) => (
                <option key={trader.id} value={trader.id}>
                  {trader.login}
                </option>
              ))}
            </Select>
          </FormField>
          <FormField label="Целевой оборот" error={form.formState.errors.targetTurnover?.message}>
            <Input inputMode="decimal" placeholder="500000" {...form.register("targetTurnover")} />
          </FormField>
          <FormField label="Комментарий">
            <Textarea rows={3} {...form.register("comment")} />
          </FormField>
          {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div> : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onMouseDown={(event) => event.preventDefault()} onClick={closeWithoutValidation}>
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

function PlanEventsDialog({ plan, onClose }: { plan: RequisiteAssignmentWorkRow | null; onClose: () => void }) {
  const eventsQuery = useQuery({
    queryKey: queryKeys.teamlead.requisitePlanEvents(plan?.assignmentId),
    queryFn: () => api.requisites.planEvents(plan?.assignmentId ?? 0),
    enabled: Boolean(plan),
  });

  return (
    <Dialog open={Boolean(plan)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">История изменений плана</DialogTitle>
          <DialogDescription>
            {plan ? `${formatRussianPhone(plan.phone)} · ${plan.bankName} · ${formatDateOnly(plan.assignedForDate)}` : ""}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          {(eventsQuery.data ?? []).map((event) => (
            <AssignmentEventItem key={event.id} event={event} />
          ))}
          {!eventsQuery.isLoading && eventsQuery.data?.length === 0 ? <EmptyState title="Истории пока нет" /> : null}
        </div>
      </DialogContent>
    </Dialog>
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

async function invalidateRequisiteWorkQueries(queryClient: QueryClient) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["teamlead", "requisites"] }),
    queryClient.invalidateQueries({ queryKey: queryKeys.teamlead.requisitePlans }),
    queryClient.invalidateQueries({ queryKey: queryKeys.teamlead.requisiteActivity }),
    queryClient.invalidateQueries({ queryKey: ["trader", "requisites"] }),
  ]);
}

function assignmentProgressLabel(row: RequisiteAssignmentWorkRow) {
  if (row.targetTurnoverMinor <= 0) return "—";
  const percent = Math.round((row.inboundTurnoverMinor / row.targetTurnoverMinor) * 100);
  return `${percent}%`;
}

function isPlanCancellable(row: RequisiteAssignmentWorkRow) {
  return row.status === "planned" || row.status === "assigned";
}

function assignmentEventLabel(action: string) {
  switch (action) {
    case "created":
      return "Создан";
    case "updated":
      return "Изменен";
    case "cancelled":
      return "Отменен";
    default:
      return action;
  }
}

function toDateInputValue(value: string) {
  return value.slice(0, 10);
}

function tomorrowDateInputValue() {
  const date = new Date();
  date.setDate(date.getDate() + 1);
  return date.toISOString().slice(0, 10);
}

function formatDateOnly(value: string | null | undefined) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("ru-RU", { day: "2-digit", month: "2-digit", year: "numeric" }).format(date);
}

function moneyMinorToInput(value: number) {
  if (!value) return "";
  return String(value / 100);
}

function RequisiteFormDialog({
  open,
  onOpenChange,
  requisite,
  traders,
  banks,
  isSaving,
  error,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  requisite: Requisite | null;
  traders: Trader[];
  banks: Bank[];
  isSaving: boolean;
  error?: string | null;
  onSubmit: (values: RequisiteForm) => void;
}) {
  const formValues = useMemo<RequisiteForm>(
    () =>
      requisite
        ? {
            id: requisite.id,
            phone: formatRussianPhone(requisite.phone),
            bankCode: requisite.bankCode,
            proxy: requisite.proxy,
            employeeComment: requisite.employeeComment ?? "",
            assignedTraderId: String(requisite.assignedTraderId ?? "unassigned"),
            status: requisite.status,
          }
        : {
            phone: "",
            bankCode: banks[0]?.code ?? "",
            proxy: "",
            employeeComment: "",
            assignedTraderId: "unassigned",
            status: "active",
          },
    [banks, requisite],
  );
  const form = useForm<RequisiteForm>({
    resolver: zodResolver(requisiteSchema),
    values: formValues,
  });
  const closeWithoutValidation = () => {
    form.reset(formValues);
    form.clearErrors();
    onOpenChange(false);
  };

  useEffect(() => {
    if (open) {
      form.reset(formValues);
      form.clearErrors();
    }
  }, [form, formValues, open]);

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => (nextOpen ? onOpenChange(true) : closeWithoutValidation())}>
      <DialogContent className="left-auto right-0 top-0 h-screen w-[min(560px,100vw)] translate-x-0 translate-y-0 rounded-none">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">{requisite ? "Редактировать реквизит" : "Добавить реквизит"}</DialogTitle>
          <DialogDescription>ФИО и карта заполняются трейдером при первом взятии реквизита в работу.</DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
          <FormField label="Телефон" error={form.formState.errors.phone?.message}>
            <Input
              {...form.register("phone")}
              onBlur={(event) => form.setValue("phone", formatRussianPhone(event.target.value), { shouldValidate: true })}
            />
          </FormField>
          <FormField label="Банк" error={form.formState.errors.bankCode?.message}>
            <Select {...form.register("bankCode")}>
              <option value="">Выберите банк</option>
              {banks.map((bank) => (
                <option key={bank.code} value={bank.code}>
                  {bank.name}
                </option>
              ))}
            </Select>
          </FormField>
          <FormField label="Proxy" error={form.formState.errors.proxy?.message}>
            <Input {...form.register("proxy")} />
          </FormField>
          <FormField label="Комментарий для сотрудника">
            <Textarea rows={3} {...form.register("employeeComment")} />
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
          {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div> : null}
          {requisite ? <AssignmentHistoryDialog requisiteId={requisite.id} /> : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onMouseDown={(event) => event.preventDefault()} onClick={closeWithoutValidation}>
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

function RequisiteCommentDialog({
  requisite,
  isSaving,
  error,
  onClose,
  onSubmit,
}: {
  requisite: Requisite | null;
  isSaving: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (employeeComment: string) => void;
}) {
  const form = useForm<{ employeeComment: string }>({
    values: { employeeComment: requisite?.employeeComment ?? "" },
  });

  return (
    <Dialog open={Boolean(requisite)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Комментарий для сотрудника</DialogTitle>
          <DialogDescription>
            {requisite ? `${formatRussianPhone(requisite.phone)} · ${requisite.bankName}` : ""}
          </DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={form.handleSubmit((values) => onSubmit(values.employeeComment))}>
          <FormField label="Комментарий">
            <Textarea rows={4} autoFocus {...form.register("employeeComment")} />
          </FormField>
          {error ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div> : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose}>
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

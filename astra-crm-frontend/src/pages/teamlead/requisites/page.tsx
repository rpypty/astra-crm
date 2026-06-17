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
import { AllRequisitesTab, RequisiteActivityTab, RequisitePlanningTab } from "./tabs";
import {
  AssignmentHistoryViewer,
  ConfirmActionDialog,
  CopyToast,
  CopyableInlineValue,
  PlanEventsDialog,
  PlanRequisiteDialog,
  ReportStatusButton,
  RequisiteCommentDialog,
  RequisiteFormDialog,
  RequisitePhoneMenu,
  RequisiteReportDialog,
  TeamleadRequisiteTabs,
  assignmentProgressLabel,
  formatDateOnly,
  invalidateRequisiteWorkQueries,
  isPlanCancellable,
  workRowReportTarget,
  type RequisiteReportTarget,
} from "./widgets";

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
  const [reportRequisite, setReportRequisite] = useState<RequisiteReportTarget | null>(null);
  const [editingPlan, setEditingPlan] = useState<RequisiteAssignmentWorkRow | null>(null);
  const [eventsPlan, setEventsPlan] = useState<RequisiteAssignmentWorkRow | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [planOpen, setPlanOpen] = useState(false);
  const [copiedMessage, setCopiedMessage] = useState<string | null>(null);

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

  const columns = useMemo<ColumnDef<Requisite>[]>(
    () => [
      {
        accessorKey: "phone",
        header: "Реквизит",
        cell: ({ row }) => <RequisitePhoneMenu item={row.original} onCopy={copyToClipboard} />,
      },
      {
        accessorKey: "bankName",
        header: "Банк",
      },
      {
        accessorKey: "proxy",
        header: "Прокси",
        cell: ({ row }) => (
          <CopyableInlineValue label="Прокси" value={row.original.proxy} onCopy={copyToClipboard} />
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
        accessorKey: "status",
        header: "Состояние",
        cell: ({ row }) => (
          <ReportStatusButton
            status={row.original.status}
            target={row.original}
            onOpen={setReportRequisite}
          />
        ),
      },
      {
        accessorKey: "assignmentStatus",
        header: "Работа",
        cell: ({ row }) =>
          row.original.assignmentStatus ? (
            <ReportStatusButton
              status={row.original.assignmentStatus}
              target={row.original}
              onOpen={setReportRequisite}
            />
          ) : (
            <span className="text-muted-foreground">—</span>
          ),
      },
      {
        id: "report",
        header: "",
        cell: ({ row }) => (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-8 px-2"
            title="Отчет по реквизиту"
            onClick={(event) => {
              event.stopPropagation();
              setReportRequisite(row.original);
            }}
          >
            <FileText className="h-4 w-4" />
            Отч
          </Button>
        ),
      },
    ],
    [copyToClipboard],
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
        cell: ({ row }) => <RequisitePhoneMenu item={row.original} onCopy={copyToClipboard} />,
      },
      {
        accessorKey: "traderLogin",
        header: "Трейдер",
        cell: ({ row }) => <UserCell login={row.original.traderLogin} />,
      },
      {
        accessorKey: "status",
        header: "Статус",
        cell: ({ row }) => (
          <ReportStatusButton
            status={row.original.status}
            target={workRowReportTarget(row.original)}
            onOpen={setReportRequisite}
          />
        ),
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
    ],
    [copyToClipboard],
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
        cell: ({ row }) => (
          <ReportStatusButton
            status={row.original.status}
            target={workRowReportTarget(row.original)}
            onOpen={setReportRequisite}
          />
        ),
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
  const bankFilterOptions = useMemo<SearchableSelectOption[]>(
    () => [
      { value: "all", label: "Все банки" },
      ...(banksQuery.data ?? []).map((bank) => ({ value: bank.code, label: bank.name })),
    ],
    [banksQuery.data],
  );
  const traderFilterOptions = useMemo<SearchableSelectOption[]>(
    () => [
      { value: "all", label: "Все трейдеры" },
      { value: "unassigned", label: "Не назначены" },
      ...(tradersQuery.data ?? []).map((trader) => ({
        value: String(trader.id),
        label: trader.login,
        searchText: trader.externalWorkerName,
      })),
    ],
    [tradersQuery.data],
  );
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
      <div className="flex flex-wrap items-center justify-between gap-3">
        <TeamleadRequisiteTabs value={activeTab} onChange={setActiveTab} />
        {pageAction}
      </div>
      {activeTab === "all" ? (
        <AllRequisitesTab
          columns={columns}
          data={data}
          pagination={pagination}
          onPaginationChange={setPagination}
          search={search}
          onSearchChange={setSearch}
          toolbarFilters={
            <div className="flex gap-2">
              <SearchableSelect
                className="w-44"
                value={bankCode}
                options={bankFilterOptions}
                onValueChange={setBankCode}
                placeholder="Все банки"
                searchPlaceholder="Найти банк"
              />
              <Select className="w-36" value={status} onChange={(event) => setStatus(event.target.value)}>
                <option value="all">Статус</option>
                <option value="active">Активные</option>
                <option value="archived">Архив</option>
              </Select>
              <SearchableSelect
                className="w-44"
                value={traderId}
                options={traderFilterOptions}
                onValueChange={setTraderId}
                placeholder="Все трейдеры"
                searchPlaceholder="Найти трейдера"
              />
            </div>
          }
          isLoading={requisitesQuery.isLoading}
          error={requisitesQuery.error instanceof Error ? requisitesQuery.error.message : null}
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
            { label: "Отчет", onSelect: (row) => setReportRequisite(row) },
            { label: "Архивировать", destructive: true, onSelect: (row) => setArchiveRequisite(row) },
          ]}
        />
      ) : null}
      {activeTab === "activity" ? (
        <RequisiteActivityTab
          columns={activityColumns}
          data={activityQuery.data ?? []}
          pagination={activityPagination}
          onPaginationChange={setActivityPagination}
          isLoading={activityQuery.isLoading}
          error={activityQuery.error instanceof Error ? activityQuery.error.message : null}
        />
      ) : null}
      {activeTab === "planning" ? (
        <RequisitePlanningTab
          columns={planningColumns}
          data={plansQuery.data ?? []}
          pagination={planningPagination}
          onPaginationChange={setPlanningPagination}
          isLoading={plansQuery.isLoading}
          error={plansQuery.error instanceof Error ? plansQuery.error.message : null}
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
      <RequisiteReportDialog requisite={reportRequisite} onClose={() => setReportRequisite(null)} onCopy={copyToClipboard} />
      <CopyToast message={copiedMessage} />
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

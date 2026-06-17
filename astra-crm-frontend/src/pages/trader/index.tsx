import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, PaginationState } from "@tanstack/react-table";
import { AlertTriangle, CalendarDays, CheckCircle2, Eye, FileText, History, Plus, RefreshCw, Upload } from "lucide-react";
import { useDeferredValue, useEffect, useMemo, useState } from "react";
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
import { PeriodFilterBar } from "@/components/crm/period-filter-bar";
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
import type { OrderDirection, Payout, PayoutTransfer, ReconciliationItem, ReconciliationSummary, ShiftReport, ShiftRequisite } from "@/lib/domain";
import { api } from "@/lib/api";
import { filterOrdersBySearch } from "@/lib/order-filters";
import type { PeriodFilter } from "@/lib/period-filter";
import { usePersistentPeriodFilter } from "@/lib/period-filter";
import { queryKeys } from "@/lib/query-keys";
import {
  formatCardNumber,
  formatDateTime,
  formatMoneyMinor,
  formatRussianPhone,
  normalizeCardNumber,
  parseMoneyToMinor,
} from "@/lib/utils";

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
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<TraderRequisiteTab>("current");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [futurePagination, setFuturePagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [historyPagination, setHistoryPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [selectedRequisite, setSelectedRequisite] = useState<ShiftRequisite | null>(null);
  const [lastClosedStatus, setLastClosedStatus] = useState<"closed" | "closed_with_discrepancy" | null>(null);
  const shiftQuery = useQuery({ queryKey: queryKeys.trader.currentShift, queryFn: api.traderShift.current });
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
  const closeMutation = useMutation({
    mutationFn: () => api.traderShift.close(),
    onSuccess: async (shift) => {
      if (shift.status === "closed" || shift.status === "closed_with_discrepancy") {
        setLastClosedStatus(shift.status);
      }
      await queryClient.invalidateQueries({ queryKey: ["trader"] });
    },
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

  const checklist = shiftQuery.data?.checklist;
  const blockers = checklist
    ? [
        !checklist.inboundImported && "Не импортированы инвойсы",
        !checklist.inboundOk && "Есть неподтвержденное расхождение по инвойсам",
        !checklist.outboundImported && "Не импортированы выплаты",
        !checklist.outboundOk && "Есть неподтвержденное расхождение по выплатам",
        !checklist.allPayoutsFullyPaid && "Есть не полностью оплаченные ручные выплаты",
      ].filter(Boolean)
    : [];

  return (
    <div className="space-y-6">
      <PageHeader title="Мои реквизиты" description="Текущая работа, будущие назначения и история закрытых или заблокированных реквизитов." />
      <TraderRequisiteTabs value={activeTab} onChange={setActiveTab} />
      {activeTab === "current" ? (
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
      ) : null}
      {activeTab === "current" ? (
        <DataTable
          columns={currentColumns}
          data={requisitesQuery.data ?? []}
          rowCount={requisitesQuery.data?.length ?? 0}
          pagination={pagination}
          onPaginationChange={setPagination}
          isLoading={requisitesQuery.isLoading}
          error={requisitesQuery.error instanceof Error ? requisitesQuery.error.message : null}
          emptyTitle="Нет текущих реквизитов"
          onRowClick={setSelectedRequisite}
        />
      ) : null}
      {activeTab === "future" ? (
        <DataTable
          columns={futureColumns}
          data={futureRequisitesQuery.data ?? []}
          rowCount={futureRequisitesQuery.data?.length ?? 0}
          pagination={futurePagination}
          onPaginationChange={setFuturePagination}
          isLoading={futureRequisitesQuery.isLoading}
          error={futureRequisitesQuery.error instanceof Error ? futureRequisitesQuery.error.message : null}
          emptyTitle="Будущих реквизитов нет"
          emptyDescription="Здесь появятся назначения на будущие даты."
        />
      ) : null}
      {activeTab === "history" ? (
        <DataTable
          columns={historyColumns}
          data={historicalRequisitesQuery.data ?? []}
          rowCount={historicalRequisitesQuery.data?.length ?? 0}
          pagination={historyPagination}
          onPaginationChange={setHistoryPagination}
          isLoading={historicalRequisitesQuery.isLoading}
          error={historicalRequisitesQuery.error instanceof Error ? historicalRequisitesQuery.error.message : null}
          emptyTitle="Истории реквизитов нет"
          emptyDescription="Закрытые и заблокированные реквизиты появятся после отработки."
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
        onRowClick={setDetailsPayout}
        actions={[{ label: "Детали", onSelect: (row) => setDetailsPayout(row) }]}
      />
      <PayoutDetailsDialog payout={detailsPayout} onClose={() => setDetailsPayout(null)} />
    </div>
  );
}

export function TraderOrdersPage({ direction }: { direction: "inbound" | "outbound" }) {
  return <TraderTransactionsPage initialDirection={direction} />;
}

export function TraderTransactionsPage({ initialDirection = "inbound" }: { initialDirection?: OrderDirection }) {
  const [activeOrdersDirection, setActiveOrdersDirection] = useState<OrderDirection>(initialDirection);
  const [periodFilter, setPeriodFilter] = usePersistentPeriodFilter(TRADER_PERIOD_FILTER_STORAGE_KEY);
  const activeLabel = activeOrdersDirection === "inbound" ? "Инвойсы" : "Выплаты";

  useEffect(() => {
    setActiveOrdersDirection(initialDirection);
  }, [initialDirection]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Транзакции"
        description="Импорт CSV, сверка и ордера текущей смены."
        actions={
          <ImportCsvDialog
            scopeLabel={`${activeLabel} текущей смены`}
            scope="trader"
            direction={activeOrdersDirection}
          />
        }
      />
      <PeriodFilterBar value={periodFilter} onChange={setPeriodFilter} />
      <div className="flex justify-end">
        <div className="inline-flex rounded-md border border-border bg-card p-1">
          <Button
            type="button"
            variant={activeOrdersDirection === "inbound" ? "default" : "ghost"}
            size="sm"
            onClick={() => setActiveOrdersDirection("inbound")}
          >
            Инвойсы
          </Button>
          <Button
            type="button"
            variant={activeOrdersDirection === "outbound" ? "default" : "ghost"}
            size="sm"
            onClick={() => setActiveOrdersDirection("outbound")}
          >
            Выплаты
          </Button>
        </div>
      </div>
      <TraderOrdersDirectionContent direction={activeOrdersDirection} periodFilter={periodFilter} />
    </div>
  );
}

function TraderOrdersDirectionContent({ direction, periodFilter }: { direction: OrderDirection; periodFilter: PeriodFilter }) {
  const dashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard(direction, periodFilter),
    queryFn: () => api.orders.dashboard("trader", direction, periodFilter),
  });
  const reconciliationQuery = useQuery({
    queryKey: ["trader", direction, "reconciliation"],
    queryFn: () => api.orders.reconciliation("trader", direction),
  });

  return (
    <div className="space-y-6">
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
      <TraderOrdersTable direction={direction} periodFilter={periodFilter} />
    </div>
  );
}

export function TraderAnalyticsPage() {
  const [periodFilter, setPeriodFilter] = usePersistentPeriodFilter(TRADER_PERIOD_FILTER_STORAGE_KEY);
  const profileQuery = useQuery({
    queryKey: queryKeys.trader.profile(periodFilter),
    queryFn: () => api.traderProfile.get(periodFilter),
  });
  const inboundDashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard("inbound", periodFilter),
    queryFn: () => api.orders.dashboard("trader", "inbound", periodFilter),
  });
  const outboundDashboardQuery = useQuery({
    queryKey: queryKeys.trader.dashboard("outbound", periodFilter),
    queryFn: () => api.orders.dashboard("trader", "outbound", periodFilter),
  });

  return (
    <div className="space-y-6">
      <PageHeader title="Аналитика" description="Показатели текущей и прошлых смен." />
      <PeriodFilterBar value={periodFilter} onChange={setPeriodFilter} />
      <TraderSalaryAnalytics
        profile={profileQuery.data}
        isLoading={profileQuery.isLoading}
        error={profileQuery.error instanceof Error ? profileQuery.error.message : null}
      />
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
    </div>
  );
}

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
              <SummaryCard label="Оплаты" value={formatMoneyMinor(inboundTotal)} />
              <SummaryCard label="Выплаты" value={formatMoneyMinor(outboundTotal)} />
              <SummaryCard label="Остаток" value={formatMoneyMinor(balanceTotal)} />
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

function TraderSalaryAnalytics({
  profile,
  isLoading,
  error,
}: {
  profile?: Awaited<ReturnType<typeof api.traderProfile.get>>;
  isLoading?: boolean;
  error?: string | null;
}) {
  if (isLoading) return <EmptyState title="Загружаем ЗП" />;
  if (error) return <EmptyState title="Не удалось загрузить ЗП" description={error} />;
  if (!profile) return null;

  return (
    <section>
      <div className="grid gap-4 md:grid-cols-3">
        <Card className="border-emerald-200 bg-emerald-50 md:col-span-2">
          <CardContent className="flex min-h-32 flex-col justify-center p-4">
            <div className="text-xs font-medium uppercase tracking-normal text-emerald-800">ЗП за выбранный период</div>
            <div className="mt-3 text-3xl font-semibold text-emerald-700 tabular-nums">
              {formatMoneyMinor(profile.periodSalaryMinor)}
            </div>
            <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-emerald-900">
              <span>
                Сегодня: <span className="font-semibold tabular-nums">{formatMoneyMinor(profile.currentShiftSalaryMinor)}</span>
              </span>
              <span>Ставка {profile.salaryRateBps / 100}%</span>
            </div>
          </CardContent>
        </Card>
        <SummaryCard label="Ставка" value={`${profile.salaryRateBps / 100}%`} />
      </div>
    </section>
  );
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
              <ReadOnlyValue label="Карта" value={formatCardNumber(item.cardNumber)} />
              <ReadOnlyValue label="Держатель" value={item.holderName ?? "—"} />
              <ReadOnlyValue label="Банк" value={item.bankName} />
              <ReadOnlyValue label="Комментарий" value={item.employeeComment ?? "—"} />
              <ReadOnlyValue label="Оплаты" value={formatMoneyMinor(item.inboundTurnoverMinor || item.latestTurnoverMinor)} />
              <ReadOnlyValue label="Выплаты" value={formatMoneyMinor(item.outboundTurnoverMinor)} />
              <ReadOnlyValue label="Остаток" value={formatMoneyMinor(item.closingBalanceMinor)} />
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
              {formatRussianPhone(item.phone)} · {item.bankName}
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

function ReadOnlyValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-border p-3">
      <div className="mb-1 text-xs font-medium uppercase text-muted-foreground">{label}</div>
      <div className="text-sm font-medium">{value}</div>
    </div>
  );
}

function TraderOrdersTable({ direction, periodFilter }: { direction: "inbound" | "outbound"; periodFilter: PeriodFilter }) {
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 8 });
  const [detailsOrder, setDetailsOrder] = useState<Awaited<ReturnType<typeof api.orders.list>>[number] | null>(null);
  const ordersQuery = useQuery({
    queryKey: queryKeys.trader.orders(direction, periodFilter),
    queryFn: () => api.orders.list("trader", direction, periodFilter),
  });
  const data = useMemo(
    () => filterOrdersBySearch(ordersQuery.data ?? [], deferredSearch),
    [deferredSearch, ordersQuery.data],
  );
  useEffect(() => {
    setPagination((current) => (current.pageIndex === 0 ? current : { ...current, pageIndex: 0 }));
  }, [deferredSearch, direction, periodFilter]);
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
    <>
      <DataTable
        columns={columns}
        data={data}
        rowCount={data.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        search={search}
        onSearchChange={setSearch}
        isLoading={ordersQuery.isLoading}
        onRowClick={setDetailsOrder}
        actions={[{ label: "Детали", onSelect: (row) => setDetailsOrder(row) }]}
      />
      <TraderOrderDetailsDialog order={detailsOrder} onClose={() => setDetailsOrder(null)} />
    </>
  );
}

function TraderOrderDetailsDialog({ order, onClose }: { order: Awaited<ReturnType<typeof api.orders.list>>[number] | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(order)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-base font-semibold">Ордер {order?.innerId}</DialogTitle>
          <DialogDescription>Данные активного import scope.</DialogDescription>
        </DialogHeader>
        {order ? (
          <div className="grid gap-3 md:grid-cols-2">
            <ReadOnlyValue label="Время" value={new Date(order.createdAt).toLocaleString("ru-RU")} />
            <ReadOnlyValue label="Сумма" value={formatMoneyMinor(order.amountMinor)} />
            <ReadOnlyValue label="Реквизит" value={formatRussianPhone(order.requisite)} />
            <ReadOnlyValue label="Метод/банк" value={order.bankName || order.method || "—"} />
            <ReadOnlyValue label="Статус" value={order.rawStatus} />
            <ReadOnlyValue label="innerId" value={order.innerId} />
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
